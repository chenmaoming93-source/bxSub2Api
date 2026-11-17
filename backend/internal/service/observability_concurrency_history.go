package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

const (
	ConcurrencySampleInterval      = 20 * time.Second
	concurrencySampleLeaderTTL     = 45 * time.Second
	concurrencySampleLeaderLock    = "observability:concurrency-sampler:leader"
	concurrencyAccountBatchSize    = 200
	concurrencySamplerSourceTTL    = 30 * time.Second
	concurrencySamplerRouteTTL     = 5 * time.Minute
	ConcurrencySampleKindAccount   = observability.ConcurrencySampleKindAccount
	ConcurrencySampleKindRoute     = observability.ConcurrencySampleKindRoute
	concurrencyMinuteRetention     = 24 * time.Hour
	concurrencyFiveMinuteRetention = 7 * 24 * time.Hour
	concurrencyHourlyRetention     = 7 * 24 * time.Hour
)

type ConcurrencyHistoryRow = observability.ConcurrencyAggregate
type ConcurrencyCurrentValue = observability.ConcurrencySample

type ConcurrencyHistoryFilter struct {
	Start         time.Time
	End           time.Time
	WindowSeconds int
	AccountID     int64
	GroupID       int64
	RouteAlias    string
}

type ConcurrencySamplerState struct {
	LastSampleAt       *time.Time
	LastFlushAt        *time.Time
	LastMinuteStart    *time.Time
	SamplesAccumulated int64
	WriteCount         int64
	FailureCount       int64
	QualityStatus      string
}

type ConcurrencyHistoryResult struct {
	Rows          []ConcurrencyHistoryRow
	Available     bool
	QualityStatus string
}

type ConcurrencyHistoryRepository interface {
	FlushConcurrencyHistory(ctx context.Context, minuteStart time.Time, rows []ConcurrencyHistoryRow) error
	GetConcurrencySamplerState(ctx context.Context) (ConcurrencySamplerState, error)
	RecordConcurrencySample(ctx context.Context, sampledAt time.Time, accumulated int64, failed bool) error
	QueryConcurrencyHistory(ctx context.Context, filter ConcurrencyHistoryFilter) ([]ConcurrencyHistoryRow, error)
	CleanupConcurrencyHistory(ctx context.Context, now time.Time) error
}

type ConcurrencyHistoryService struct {
	repo        ConcurrencyHistoryRepository
	accountRepo AccountRepository
	groupRepo   GroupRepository
	concurrency *ConcurrencyService
	timingWheel *TimingWheelService
	lockCache   LeaderLockCache
	db          *sql.DB
	instanceID  string

	accumulator *observability.ConcurrencyAccumulator

	sourceMu        sync.Mutex
	accounts        []Account
	accountsExpire  time.Time
	routeRefs       []ModelRouteReference
	routeRefsExpire time.Time

	running     int32
	stopped     int32
	lastCleanup atomic.Int64
}

func NewConcurrencyHistoryService(
	repo ConcurrencyHistoryRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	concurrency *ConcurrencyService,
	timingWheel *TimingWheelService,
	lockCache LeaderLockCache,
	db *sql.DB,
) *ConcurrencyHistoryService {
	history := &ConcurrencyHistoryService{
		repo:        repo,
		accountRepo: accountRepo,
		groupRepo:   groupRepo,
		concurrency: concurrency,
		timingWheel: timingWheel,
		lockCache:   lockCache,
		db:          db,
		accumulator: observability.NewConcurrencyAccumulator(),
	}
	if concurrency != nil {
		concurrency.SetAccountWaitObserver(history)
	}
	return history
}

func (s *ConcurrencyHistoryService) Start() {
	if s == nil || s.repo == nil || s.concurrency == nil || s.timingWheel == nil {
		return
	}
	s.instanceID = newConcurrencySamplerID()
	s.timingWheel.Schedule("observability:concurrency-sampler:initial", 2*time.Second, s.sampleOnce)
	s.timingWheel.ScheduleRecurring("observability:concurrency-sampler", ConcurrencySampleInterval, s.sampleOnce)
}

func (s *ConcurrencyHistoryService) Stop() {
	if s == nil {
		return
	}
	atomic.StoreInt32(&s.stopped, 1)
	if s.timingWheel != nil {
		s.timingWheel.Cancel("observability:concurrency-sampler")
		s.timingWheel.Cancel("observability:concurrency-sampler:initial")
	}
}

func (s *ConcurrencyHistoryService) sampleOnce() {
	if s == nil || atomic.LoadInt32(&s.stopped) != 0 || !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&s.running, 0)
	ctx, cancel := context.WithTimeout(context.Background(), ConcurrencySampleInterval-2*time.Second)
	defer cancel()

	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, concurrencySampleLeaderLock, s.instanceID, concurrencySampleLeaderTTL)
	if !ok {
		return
	}
	defer release()

	sampledAt := time.Now().UTC()
	rows, err := s.collectSnapshot(ctx, sampledAt)
	if err != nil {
		_ = s.repo.RecordConcurrencySample(ctx, sampledAt, 0, true)
		return
	}
	accumulated := s.accumulate(ctx, sampledAt, rows)
	_ = s.repo.RecordConcurrencySample(ctx, sampledAt, accumulated, false)
	lastCleanup := s.lastCleanup.Load()
	if lastCleanup == 0 || sampledAt.Sub(time.Unix(0, lastCleanup)) >= time.Hour {
		if s.lastCleanup.CompareAndSwap(lastCleanup, sampledAt.UnixNano()) {
			_ = s.repo.CleanupConcurrencyHistory(ctx, sampledAt)
		}
	}
}

func (s *ConcurrencyHistoryService) accumulate(ctx context.Context, sampledAt time.Time, rows []ConcurrencyCurrentValue) int64 {
	flushed, accumulated := s.accumulator.Add(sampledAt, rows)
	if len(flushed) > 0 {
		_ = s.repo.FlushConcurrencyHistory(ctx, flushed[0].BucketStart, flushed)
	}
	return int64(accumulated)
}

func (s *ConcurrencyHistoryService) RecordAccountWait(accountID int64, wait time.Duration) {
	if s == nil || accountID <= 0 || wait <= 0 {
		return
	}
	flushed, _ := s.accumulator.AddWait(time.Now().UTC(), accountID, wait)
	if len(flushed) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.repo.FlushConcurrencyHistory(ctx, flushed[0].BucketStart, flushed)
}

func (s *ConcurrencyHistoryService) collectSnapshot(ctx context.Context, sampledAt time.Time) ([]ConcurrencyCurrentValue, error) {
	// Kept as a single code path so current API reads and persisted samples use
	// the same Redis batch methods.
	return s.CollectCurrent(ctx, sampledAt)
}

func (s *ConcurrencyHistoryService) CollectCurrent(ctx context.Context, sampledAt time.Time) ([]ConcurrencyCurrentValue, error) {
	if s == nil || s.concurrency == nil {
		return nil, fmt.Errorf("concurrency history service is unavailable")
	}
	accounts, err := s.schedulableAccounts(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]ConcurrencyCurrentValue, 0, len(accounts))
	accountWaits := make(map[int64]int, len(accounts))
	for start := 0; start < len(accounts); start += concurrencyAccountBatchSize {
		end := start + concurrencyAccountBatchSize
		if end > len(accounts) {
			end = len(accounts)
		}
		batch := accounts[start:end]
		loads := make([]AccountWithConcurrency, 0, len(batch))
		for i := range batch {
			loads = append(loads, AccountWithConcurrency{ID: batch[i].ID, MaxConcurrency: batch[i].EffectiveLoadFactor()})
		}
		loadMap, loadErr := s.concurrency.GetAccountsLoadBatch(ctx, loads)
		if loadErr != nil {
			return nil, loadErr
		}
		for i := range batch {
			account := &batch[i]
			current, waiting := 0, 0
			if load := loadMap[account.ID]; load != nil {
				current = load.CurrentConcurrency
				waiting = load.WaitingCount
			}
			rows = append(rows, ConcurrencyCurrentValue{
				Kind: ConcurrencySampleKindAccount, AccountID: account.ID,
				Current: current, Limit: account.EffectiveLoadFactor(), Waiting: waiting,
				SampledAt: sampledAt, Available: true,
			})
			accountWaits[account.ID] = waiting
		}
	}

	refs, err := s.cachedRouteReferences(ctx)
	if err != nil {
		return nil, err
	}
	accountLimits := make(map[int64]int, len(accounts))
	for i := range accounts {
		accountLimits[accounts[i].ID] = accounts[i].EffectiveLoadFactor()
	}
	routeRequests := make([]RouteLoadRequest, 0, len(refs))
	routeKeys := make([]string, 0, len(refs))
	for i := range refs {
		if refs[i].AccountConcurrency <= 0 {
			refs[i].AccountConcurrency = accountLimits[refs[i].AccountID]
		}
		ref := refs[i]
		key := routeConcurrencyKey(&ref.GroupID, ref.RouteAlias, ref.AccountID)
		routeKeys = append(routeKeys, key)
		routeRequests = append(routeRequests, RouteLoadRequest{
			Key: key, AccountID: ref.AccountID, MaxConcurrency: ref.MaxConcurrency,
			AccountMaxConcurrency: ref.AccountConcurrency,
		})
	}
	routeLoads, err := s.concurrency.GetRouteLoadsBatch(ctx, routeRequests)
	if err != nil {
		return nil, err
	}
	for i, ref := range refs {
		load := routeLoads[routeKeys[i]]
		value := ConcurrencyCurrentValue{
			Kind: ConcurrencySampleKindRoute, AccountID: ref.AccountID,
			GroupID: ref.GroupID, RouteAlias: ref.RouteAlias,
			Current: load.CurrentConcurrency, Limit: ref.AccountConcurrency,
			Waiting: accountWaits[ref.AccountID], SampledAt: sampledAt, Available: true,
		}
		if ref.MaxConcurrency != nil {
			value.Limit = *ref.MaxConcurrency
		}
		rows = append(rows, value)
	}
	return rows, nil
}

// CurrentBatch returns current values for explicit account and candidate keys.
// Missing Redis entries are represented as current=0 while retaining the
// requested effective limit, so callers never mistake absence for a zero-limit
// object.
func (s *ConcurrencyHistoryService) CurrentBatch(
	ctx context.Context,
	accounts []AccountWithConcurrency,
	routes []RouteLoadRequest,
	sampledAt time.Time,
) (map[int64]ConcurrencyCurrentValue, map[string]ConcurrencyCurrentValue, error) {
	accountResult := make(map[int64]ConcurrencyCurrentValue, len(accounts))
	routeResult := make(map[string]ConcurrencyCurrentValue, len(routes))
	if len(accounts) > 0 {
		loads, err := s.concurrency.GetAccountsLoadBatch(ctx, accounts)
		if err != nil {
			return nil, nil, err
		}
		for _, account := range accounts {
			current, waiting := 0, 0
			if load := loads[account.ID]; load != nil {
				current = load.CurrentConcurrency
				waiting = load.WaitingCount
			}
			accountResult[account.ID] = ConcurrencyCurrentValue{
				Kind: ConcurrencySampleKindAccount, AccountID: account.ID,
				Current: current, Limit: account.MaxConcurrency, Waiting: waiting,
				SampledAt: sampledAt, Available: true,
			}
		}
	}
	if len(routes) > 0 {
		loads, err := s.concurrency.GetRouteLoadsBatch(ctx, routes)
		if err != nil {
			return nil, nil, err
		}
		for _, route := range routes {
			limit := route.AccountMaxConcurrency
			if route.MaxConcurrency != nil {
				limit = *route.MaxConcurrency
			}
			current := 0
			if load, ok := loads[route.Key]; ok {
				current = load.CurrentConcurrency
			}
			routeResult[route.Key] = ConcurrencyCurrentValue{
				Kind: ConcurrencySampleKindRoute, AccountID: route.AccountID,
				Current: current, Limit: limit, SampledAt: sampledAt, Available: true,
			}
		}
	}
	return accountResult, routeResult, nil
}

func (s *ConcurrencyHistoryService) schedulableAccounts(ctx context.Context) ([]Account, error) {
	s.sourceMu.Lock()
	if time.Now().Before(s.accountsExpire) && s.accounts != nil {
		accounts := append([]Account(nil), s.accounts...)
		s.sourceMu.Unlock()
		return accounts, nil
	}
	s.sourceMu.Unlock()

	accounts, err := s.accountRepo.ListSchedulableByPlatforms(ctx, []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity})
	if err != nil {
		return nil, err
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	s.sourceMu.Lock()
	s.accounts = append([]Account(nil), accounts...)
	s.accountsExpire = time.Now().Add(concurrencySamplerSourceTTL)
	s.sourceMu.Unlock()
	return accounts, nil
}

func (s *ConcurrencyHistoryService) cachedRouteReferences(ctx context.Context) ([]ModelRouteReference, error) {
	s.sourceMu.Lock()
	if time.Now().Before(s.routeRefsExpire) && s.routeRefs != nil {
		refs := append([]ModelRouteReference(nil), s.routeRefs...)
		s.sourceMu.Unlock()
		return refs, nil
	}
	s.sourceMu.Unlock()

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]ModelRouteReference, 0)
	for i := range groups {
		if !groups[i].ModelRoutingEnabled {
			continue
		}
		for _, alias := range groups[i].ModelRoutingRuleNames() {
			for _, candidate := range groups[i].GetRoutingCandidates(alias) {
				for _, accountID := range candidate.AccountIDs {
					refs = append(refs, ModelRouteReference{
						GroupID: groups[i].ID, GroupName: groups[i].Name,
						RouteAlias: alias, AccountID: accountID,
					})
				}
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].GroupID != refs[j].GroupID {
			return refs[i].GroupID < refs[j].GroupID
		}
		if refs[i].RouteAlias != refs[j].RouteAlias {
			return refs[i].RouteAlias < refs[j].RouteAlias
		}
		return refs[i].AccountID < refs[j].AccountID
	})
	// Resolve allocation and per-account caps from the materialized route table.
	if reader, ok := s.groupRepo.(groupModelRouteGroupReader); ok {
		for i := range groups {
			value, readErr := reader.ListGroupModelRouteReferencesByGroup(ctx, groups[i].ID)
			if readErr != nil {
				return nil, readErr
			}
			if materialized, ok := value.([]ModelRouteReference); ok {
				refs = append(refs, materialized...)
			}
		}
		refs = dedupeRouteReferences(refs)
	}
	s.sourceMu.Lock()
	s.routeRefs = append([]ModelRouteReference(nil), refs...)
	s.routeRefsExpire = time.Now().Add(concurrencySamplerRouteTTL)
	s.sourceMu.Unlock()
	return refs, nil
}

func (s *ConcurrencyHistoryService) QueryHistory(ctx context.Context, filter ConcurrencyHistoryFilter) (ConcurrencyHistoryResult, error) {
	switch filter.WindowSeconds {
	case ObservabilityWindowMinute, ObservabilityWindowFiveMinute, ObservabilityWindowHour:
	default:
		return ConcurrencyHistoryResult{}, fmt.Errorf("unsupported concurrency window %d", filter.WindowSeconds)
	}
	rows, err := s.repo.QueryConcurrencyHistory(ctx, filter)
	if err != nil {
		return ConcurrencyHistoryResult{}, err
	}
	state, stateErr := s.repo.GetConcurrencySamplerState(ctx)
	status := "unavailable"
	if stateErr == nil {
		status = observability.ConcurrencySampleQuality(state.QualityStatus, state.LastSampleAt, time.Now().UTC(), 90*time.Second)
	}
	return ConcurrencyHistoryResult{Rows: rows, Available: len(rows) > 0, QualityStatus: status}, nil
}

// QueryConcurrencyHeatmapHistory delegates to repositories that can bound the
// heatmap to the busiest dimensions before loading their full time series.
func (s *ConcurrencyHistoryService) QueryConcurrencyHeatmapHistory(
	ctx context.Context,
	filter ConcurrencyHistoryFilter,
	groupBy string,
	limit int,
) ([]ConcurrencyHistoryRow, error) {
	if bounded, ok := s.repo.(interface {
		QueryConcurrencyHeatmapHistory(context.Context, ConcurrencyHistoryFilter, string, int) ([]ConcurrencyHistoryRow, error)
	}); ok {
		return bounded.QueryConcurrencyHeatmapHistory(ctx, filter, groupBy, limit)
	}
	return s.repo.QueryConcurrencyHistory(ctx, filter)
}

func (s *ConcurrencyHistoryService) SamplerState(ctx context.Context) (ConcurrencySamplerState, error) {
	return s.repo.GetConcurrencySamplerState(ctx)
}

func dedupeRouteReferences(refs []ModelRouteReference) []ModelRouteReference {
	seen := make(map[string]int, len(refs))
	out := make([]ModelRouteReference, 0, len(refs))
	for _, ref := range refs {
		key := routeConcurrencyKey(&ref.GroupID, ref.RouteAlias, ref.AccountID)
		if index, exists := seen[key]; exists {
			if ref.MaxConcurrency != nil {
				out[index].MaxConcurrency = ref.MaxConcurrency
			}
			if ref.AccountConcurrency > 0 {
				out[index].AccountConcurrency = ref.AccountConcurrency
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, ref)
	}
	return out
}

func newConcurrencySamplerID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(int64(os.Getpid()), 36)
}
