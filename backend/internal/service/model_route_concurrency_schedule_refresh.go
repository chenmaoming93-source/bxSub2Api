package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const routeScheduleRedisKeyPrefix = "concurrency:route-schedule:"

var ErrModelRouteConcurrencyScheduleRefreshInProgress = errors.New("model route concurrency schedule refresh is already running")

type ModelRouteConcurrencyScheduleRefreshResult struct {
	TaskID         string
	Trigger        string
	Skipped        bool
	CandidateCount int
	UpdatedCount   int
	DeletedCount   int
	FailedCount    int
	ErrorRouteKey  string
	StartedAt      time.Time
	FinishedAt     time.Time
}

// ModelRouteConcurrencyScheduleRefresher materializes the effective value for
// the current minute. The request path only reads Redis and never depends on
// this service being present during a request.
type ModelRouteConcurrencyScheduleRefresher struct {
	repo        ModelRouteConcurrencyScheduleRefreshRepository
	concurrency *ConcurrencyService
	location    *time.Location
	lockTTL     time.Duration
	renewEvery  time.Duration
	now         func() time.Time
	instance    string

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewModelRouteConcurrencyScheduleRefresher(
	repo ModelRouteConcurrencyScheduleRefreshRepository,
	concurrency *ConcurrencyService,
	cfg *config.Config,
) *ModelRouteConcurrencyScheduleRefresher {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	lockTTL := 5 * time.Minute
	renewEvery := 30 * time.Second
	if cfg != nil {
		if name := strings.TrimSpace(cfg.Timezone); name != "" {
			if loaded, err := time.LoadLocation(name); err == nil {
				location = loaded
			}
		}
		if cfg.Gateway.ModelRouteSchedule.RefreshLockTTLSeconds > 0 {
			lockTTL = time.Duration(cfg.Gateway.ModelRouteSchedule.RefreshLockTTLSeconds) * time.Second
		}
		if cfg.Gateway.ModelRouteSchedule.RefreshLockRenewIntervalSeconds > 0 {
			renewEvery = time.Duration(cfg.Gateway.ModelRouteSchedule.RefreshLockRenewIntervalSeconds) * time.Second
		}
	}
	return &ModelRouteConcurrencyScheduleRefresher{
		repo: repo, concurrency: concurrency, location: location,
		lockTTL: lockTTL, renewEvery: renewEvery, now: time.Now,
		instance: refreshInstanceID(),
	}
}

func refreshInstanceID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s/%d", host, os.Getpid())
}

// SetNowForTest makes the minute calculation deterministic in unit tests.
func (s *ModelRouteConcurrencyScheduleRefresher) SetNowForTest(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *ModelRouteConcurrencyScheduleRefresher) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *ModelRouteConcurrencyScheduleRefresher) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		s.wg.Wait()
	}
}

func (s *ModelRouteConcurrencyScheduleRefresher) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		timer := time.NewTimer(durationUntilNextMinute(s.now(), s.location))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
		_, _ = s.Refresh(ctx, "scheduled")
	}
}

func durationUntilNextMinute(now time.Time, location *time.Location) time.Duration {
	local := now.In(location)
	next := local.Truncate(time.Minute).Add(time.Minute)
	d := next.Sub(local)
	if d <= 0 {
		return time.Minute
	}
	return d
}

func (s *ModelRouteConcurrencyScheduleRefresher) Refresh(ctx context.Context, trigger string) (ModelRouteConcurrencyScheduleRefreshResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, token := s.newRefreshResult(trigger)
	acquired, err := s.concurrency.TryAcquireRouteScheduleRefreshLock(ctx, token, s.lockTTL)
	if err != nil {
		return s.finishRefresh(result, fmt.Errorf("acquire refresh lock: %w", err))
	}
	if !acquired {
		result.Skipped = true
		return s.finishRefresh(result, ErrModelRouteConcurrencyScheduleRefreshInProgress)
	}
	return s.runOwned(ctx, result, token)
}

// StartImmediate acquires the same global lock synchronously, then starts the
// actual refresh in the background so the admin API can return immediately.
func (s *ModelRouteConcurrencyScheduleRefresher) StartImmediate(ctx context.Context) (ModelRouteConcurrencyScheduleRefreshResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, token := s.newRefreshResult("manual")
	acquired, err := s.concurrency.TryAcquireRouteScheduleRefreshLock(ctx, token, s.lockTTL)
	if err != nil {
		return s.finishRefresh(result, fmt.Errorf("acquire refresh lock: %w", err))
	}
	if !acquired {
		result.Skipped = true
		return s.finishRefresh(result, ErrModelRouteConcurrencyScheduleRefreshInProgress)
	}
	go func() {
		_, _ = s.runOwned(context.Background(), result, token)
	}()
	return result, nil
}

func (s *ModelRouteConcurrencyScheduleRefresher) newRefreshResult(trigger string) (ModelRouteConcurrencyScheduleRefreshResult, string) {
	if trigger == "" {
		trigger = "manual"
	}
	started := s.now()
	result := ModelRouteConcurrencyScheduleRefreshResult{TaskID: uuid.NewString(), Trigger: trigger, StartedAt: started}
	logger.L().With(zap.String("component", "service.model_route_schedule")).Info("refresh start",
		zap.String("task_id", result.TaskID),
		zap.String("trigger", trigger),
		zap.String("instance", s.instance),
		zap.Time("started_at", started),
	)
	return result, uuid.NewString()
}

func (s *ModelRouteConcurrencyScheduleRefresher) runOwned(ctx context.Context, result ModelRouteConcurrencyScheduleRefreshResult, token string) (ModelRouteConcurrencyScheduleRefreshResult, error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if releaseErr := s.concurrency.ReleaseRouteScheduleRefreshLock(releaseCtx, token); releaseErr != nil {
			logger.LegacyPrintf("service.model_route_schedule", "refresh release lock failed task_id=%s trigger=%s err=%v", result.TaskID, result.Trigger, releaseErr)
		}
	}()

	var leaseLost atomic.Bool
	var renewWG sync.WaitGroup
	renewWG.Add(1)
	go func() {
		defer renewWG.Done()
		ticker := time.NewTicker(s.renewEvery)
		defer ticker.Stop()
		for {
			select {
			case <-taskCtx.Done():
				return
			case <-ticker.C:
				ok, renewErr := s.concurrency.RenewRouteScheduleRefreshLock(taskCtx, token, s.lockTTL)
				if renewErr != nil || !ok {
					leaseLost.Store(true)
					logger.LegacyPrintf("service.model_route_schedule", "refresh lease lost task_id=%s trigger=%s instance=%s renewed=%t err=%v", result.TaskID, result.Trigger, s.instance, ok, renewErr)
					cancel()
					return
				}
			}
		}
	}()

	finish := func(err error) (ModelRouteConcurrencyScheduleRefreshResult, error) {
		cancel()
		renewWG.Wait()
		if leaseLost.Load() && err == nil {
			err = errors.New("refresh lock lease lost before completion")
		}
		return s.finishRefresh(result, err)
	}

	candidates, err := s.repo.ListModelRouteConcurrencyScheduleCandidates(taskCtx)
	if err != nil {
		return finish(fmt.Errorf("load schedule candidates: %w", err))
	}
	result.CandidateCount = len(candidates)
	if leaseLost.Load() {
		return finish(errors.New("refresh lock lease lost after loading candidates"))
	}

	minute := s.now().In(s.location).Hour()*60 + s.now().In(s.location).Minute()
	limits := make(map[string]*int, len(candidates))
	for _, candidate := range candidates {
		if leaseLost.Load() {
			return finish(errors.New("refresh lock lease lost before writing limits"))
		}
		limit := candidate.DefaultMaxConcurrency
		for _, schedule := range candidate.Schedules {
			if minute >= schedule.StartMinute && minute < schedule.EndMinute {
				limit = schedule.MaxConcurrency
				break
			}
		}
		groupID := candidate.GroupID
		limits[routeConcurrencyKey(&groupID, candidate.RouteAlias, candidate.AccountID)] = limit
	}
	if err := s.concurrency.SetRouteScheduleConcurrencyLimits(taskCtx, limits, result.StartedAt); err != nil {
		result.FailedCount = len(limits)
		result.ErrorRouteKey = firstRouteKey(limits)
		return finish(fmt.Errorf("write effective limits: %w", err))
	}
	result.UpdatedCount = len(limits)

	existingKeys, err := s.concurrency.ListRouteScheduleKeys(taskCtx)
	if err != nil {
		return finish(fmt.Errorf("list stale schedule keys: %w", err))
	}
	active := make(map[string]struct{}, len(limits))
	for key := range limits {
		active[routeScheduleRedisKeyPrefix+key] = struct{}{}
	}
	stale := make([]string, 0)
	for _, fullKey := range existingKeys {
		if _, ok := active[fullKey]; !ok && strings.HasPrefix(fullKey, routeScheduleRedisKeyPrefix) {
			stale = append(stale, strings.TrimPrefix(fullKey, routeScheduleRedisKeyPrefix))
		}
	}
	if err := s.concurrency.DeleteRouteScheduleConcurrencyLimits(taskCtx, stale); err != nil {
		return finish(fmt.Errorf("delete stale schedule keys: %w", err))
	}
	result.DeletedCount = len(stale)
	return finish(nil)
}

func (s *ModelRouteConcurrencyScheduleRefresher) finishRefresh(result ModelRouteConcurrencyScheduleRefreshResult, err error) (ModelRouteConcurrencyScheduleRefreshResult, error) {
	result.FinishedAt = s.now()
	fields := []zap.Field{
		zap.String("task_id", result.TaskID),
		zap.String("trigger", result.Trigger),
		zap.String("instance", s.instance),
		zap.Time("finished_at", result.FinishedAt),
		zap.Int64("duration_ms", result.FinishedAt.Sub(result.StartedAt).Milliseconds()),
		zap.Bool("skipped", result.Skipped),
		zap.Int("candidates", result.CandidateCount),
		zap.Int("updated", result.UpdatedCount),
		zap.Int("deleted", result.DeletedCount),
		zap.Int("failed", result.FailedCount),
		zap.String("route_key", result.ErrorRouteKey),
	}
	log := logger.L().With(zap.String("component", "service.model_route_schedule"))
	if err != nil {
		log.Warn("refresh end", append(fields, zap.Error(err))...)
	} else {
		log.Info("refresh end", fields...)
	}
	return result, err
}

func firstRouteKey(limits map[string]*int) string {
	for key := range limits {
		return key
	}
	return ""
}
