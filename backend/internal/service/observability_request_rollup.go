package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/google/uuid"
)

const (
	ObservabilityWindowMinute     = 60
	ObservabilityWindowFiveMinute = 300
	ObservabilityWindowHour       = 3600

	observabilityRollupInterval       = 45 * time.Second
	observabilityRollupLookback       = 2 * time.Minute
	observabilityRollupBatch          = 15 * time.Minute
	observabilityRollupTimeout        = 2 * time.Minute
	observabilityRollupLeaderLockKey  = "observability:request-rollup:leader"
	observabilityRollupLeaderLockTTL  = 3 * time.Minute
	observabilityMinuteRetention      = 48 * time.Hour
	observabilityFiveMinuteRetention  = 7 * 24 * time.Hour
	observabilityHourlyRetention      = 90 * 24 * time.Hour
	observabilityInitialBackfillRange = 7 * 24 * time.Hour
)

type ObservabilityHistogram = observability.Histogram

type ObservabilityRequestBucket struct {
	WindowSeconds        int
	BucketStart          time.Time
	DimensionHash        string
	UserID               int64
	APIKeyID             int64
	GroupID              int64
	RouteAlias           string
	AccountID            int64
	RequestCount         int64
	SuccessCount         int64
	ErrorCount           int64
	SLAErrorCount        int64
	BusinessLimitedCount int64
	RateLimitedCount     int64
	LegacyInferredCount  int64
	InputTokens          int64
	OutputTokens         int64
	CacheCreationTokens  int64
	CacheReadTokens      int64
	TotalTokens          int64
	Duration             ObservabilityHistogram
	TTFT                 ObservabilityHistogram
	LastFactAt           time.Time
}

type ObservabilityBucketTotal struct {
	BucketStart          time.Time
	RequestCount         int64
	SuccessCount         int64
	ErrorCount           int64
	SLAErrorCount        int64
	BusinessLimitedCount int64
	RateLimitedCount     int64
	LegacyInferredCount  int64
	InputTokens          int64
	OutputTokens         int64
	CacheCreationTokens  int64
	CacheReadTokens      int64
	TotalTokens          int64
	Duration             ObservabilityHistogram
	TTFT                 ObservabilityHistogram
}

type ObservabilityRollupState struct {
	LastSourceAt      *time.Time
	LastSuccessAt     *time.Time
	LastSuccessBucket *time.Time
	SkippedCount      int64
	DelayedCount      int64
	QualityStatus     string
}

type ObservabilityRequestRollupRepository interface {
	AggregateRequestRange(ctx context.Context, start, end time.Time) error
	GetRequestRollupState(ctx context.Context) (ObservabilityRollupState, error)
	RecordRequestRollupFailure(ctx context.Context, delayed bool) error
	QueryRequestBucketTotals(ctx context.Context, start, end time.Time, windowSeconds int) ([]ObservabilityBucketTotal, error)
	CleanupRequestRollups(ctx context.Context, now time.Time) error
}

type ObservabilityRequestRollupService struct {
	repo        ObservabilityRequestRollupRepository
	timingWheel *TimingWheelService
	lockCache   LeaderLockCache
	db          *sql.DB
	instanceID  string

	running int32
	stopped int32
}

func NewObservabilityRequestRollupService(repo ObservabilityRequestRollupRepository, timingWheel *TimingWheelService, lockCache LeaderLockCache, db *sql.DB) *ObservabilityRequestRollupService {
	return &ObservabilityRequestRollupService{
		repo:        repo,
		timingWheel: timingWheel,
		lockCache:   lockCache,
		db:          db,
		instanceID:  uuid.NewString(),
	}
}

func (s *ObservabilityRequestRollupService) Start() {
	if s == nil || s.repo == nil || s.timingWheel == nil {
		return
	}
	s.timingWheel.Schedule("observability:request-rollup:initial", 2*time.Second, s.runScheduled)
	s.timingWheel.ScheduleRecurring("observability:request-rollup", observabilityRollupInterval, s.runScheduled)
}

func (s *ObservabilityRequestRollupService) Stop() {
	if s == nil {
		return
	}
	atomic.StoreInt32(&s.stopped, 1)
	if s.timingWheel != nil {
		s.timingWheel.Cancel("observability:request-rollup")
		s.timingWheel.Cancel("observability:request-rollup:initial")
	}
}

func (s *ObservabilityRequestRollupService) runScheduled() {
	if s == nil || atomic.LoadInt32(&s.stopped) != 0 || !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	ctx, cancel := context.WithTimeout(context.Background(), observabilityRollupTimeout)
	defer cancel()
	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, observabilityRollupLeaderLockKey, s.instanceID, observabilityRollupLeaderLockTTL)
	if !ok {
		return
	}
	defer release()

	now := time.Now().UTC()
	end := now.Truncate(time.Minute)
	state, err := s.repo.GetRequestRollupState(ctx)
	if err != nil {
		slog.Warn("observability_request_rollup_state_failed", "error", err)
		_ = s.repo.RecordRequestRollupFailure(ctx, true)
		return
	}

	start := now.Add(-observabilityInitialBackfillRange)
	if state.LastSuccessBucket != nil {
		start = state.LastSuccessBucket.UTC().Add(-observabilityRollupLookback)
	}
	if start.After(end) {
		start = end.Add(-observabilityRollupLookback)
	}

	for cursor := start.UTC(); cursor.Before(end); {
		batchEnd := cursor.Add(observabilityRollupBatch)
		if batchEnd.After(end) {
			batchEnd = end
		}
		if err := s.repo.AggregateRequestRange(ctx, cursor, batchEnd); err != nil {
			slog.Warn("observability_request_rollup_failed",
				"start", cursor.Format(time.RFC3339),
				"end", batchEnd.Format(time.RFC3339),
				"error", err,
			)
			_ = s.repo.RecordRequestRollupFailure(ctx, true)
			return
		}
		cursor = batchEnd
	}
	if err := s.repo.CleanupRequestRollups(ctx, now); err != nil {
		slog.Warn("observability_request_rollup_cleanup_failed", "error", err)
	}
}

func HistogramQuantiles(hist ObservabilityHistogram, quantiles ...float64) map[float64]int64 {
	return hist.Quantiles(quantiles...)
}

func (s *ObservabilityRequestRollupService) QueryTotals(ctx context.Context, start, end time.Time, windowSeconds int) ([]ObservabilityBucketTotal, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("observability request rollup service is unavailable")
	}
	return s.repo.QueryRequestBucketTotals(ctx, start, end, windowSeconds)
}
