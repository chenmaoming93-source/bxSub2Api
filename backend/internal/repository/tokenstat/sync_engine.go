package tokenstat

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

const dynamicSyncLockKey = "sub2api:dynamic_token_stats_sync:v1:lock"

type AggregateSink interface {
	UpsertAggregate(ctx context.Context, aggregate Aggregate) error
}

type SyncStats struct {
	SyncedRows    uint64
	Failures      uint64
	LastSuccessAt time.Time
}

type SyncEngine struct {
	redis     *redis.Client
	sink      AggregateSink
	lockTTL   time.Duration
	batchSize int
	stats     SyncStats
}

type syncResult struct {
	rows       int
	lockHeld   bool
	dirtyEmpty bool
}

func NewSyncEngine(client *redis.Client, sink AggregateSink, batchSize int) *SyncEngine {
	return &SyncEngine{redis: client, sink: sink, lockTTL: 2 * time.Minute, batchSize: batchSize}
}

func (e *SyncEngine) Sync(ctx context.Context) error {
	_, err := e.sync(ctx)
	return err
}

func (e *SyncEngine) sync(ctx context.Context) (syncResult, error) {
	token := strconv.FormatInt(time.Now().UnixNano(), 36)
	locked, err := e.redis.SetNX(ctx, dynamicSyncLockKey, token, e.lockTTL).Result()
	if err != nil {
		e.stats.Failures++
		domain.RecordSync(0, true)
		return syncResult{}, err
	}
	if !locked {
		return syncResult{lockHeld: true}, nil
	}
	defer releaseSyncLock.Run(context.Background(), e.redis, []string{dynamicSyncLockKey}, token)

	processing := "sub2api:dynamic_token_stats_dirty:v1:processing:" + token
	rotated, err := rotateDirtySet.Run(ctx, e.redis, []string{dynamicDirtyKey, processing}).Int()
	if err != nil {
		return syncResult{}, err
	}
	if rotated == 0 {
		e.stats.LastSuccessAt = time.Now()
		domain.RecordSync(0, false)
		return syncResult{dirtyEmpty: true}, nil
	}
	members, err := e.redis.SMembers(ctx, processing).Result()
	if err != nil {
		return syncResult{}, e.requeue(ctx, processing, err)
	}
	for start := 0; start < len(members); start += e.batchSize {
		end := min(start+e.batchSize, len(members))
		for _, member := range members[start:end] {
			aggregate, readErr := e.readAggregate(ctx, member)
			if readErr != nil {
				return syncResult{}, e.requeue(ctx, processing, readErr)
			}
			if err := e.sink.UpsertAggregate(ctx, aggregate); err != nil {
				return syncResult{}, e.requeue(ctx, processing, err)
			}
			e.stats.SyncedRows++
		}
	}
	if err := e.redis.Del(ctx, processing).Err(); err != nil {
		return syncResult{}, err
	}
	e.stats.LastSuccessAt = time.Now()
	domain.RecordSync(uint64(len(members)), false)
	return syncResult{rows: len(members)}, nil
}

func (e *SyncEngine) Stats() SyncStats { return e.stats }

func (e *SyncEngine) Start(ctx context.Context, interval time.Duration) context.CancelFunc {
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		e.runScheduledSync(runCtx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				e.runScheduledSync(runCtx)
			}
		}
	}()
	return cancel
}

func (e *SyncEngine) runScheduledSync(ctx context.Context) {
	result, err := e.sync(ctx)
	if err != nil {
		slog.Error("dynamic token statistics sync failed", "error", err)
		return
	}
	if result.lockHeld {
		slog.Info("dynamic token statistics sync skipped", "reason", "another sync holds the lock")
		return
	}
	if result.dirtyEmpty {
		slog.Info("dynamic token statistics sync completed", "synced_rows", 0, "reason", "no dirty statistics")
		return
	}
	slog.Info("dynamic token statistics sync completed", "synced_rows", result.rows)
}

func (e *SyncEngine) requeue(ctx context.Context, processing string, cause error) error {
	e.stats.Failures++
	domain.RecordSync(0, true)
	_, _ = requeueDirtySet.Run(ctx, e.redis, []string{processing, dynamicDirtyKey}).Result()
	return cause
}

type dirtyIdentity struct {
	PeriodType      domain.PeriodType `json:"period_type"`
	PeriodStart     time.Time         `json:"period_start"`
	PeriodEnd       time.Time         `json:"period_end"`
	ProjectionID    int64             `json:"projection_id"`
	Shard           int               `json:"shard"`
	Field           string            `json:"field"`
	DimensionHash   string            `json:"dimension_hash"`
	DimensionValues map[string]any    `json:"dimension_values"`
	MetricCode      domain.MetricCode `json:"metric_code"`
}

func (e *SyncEngine) readAggregate(ctx context.Context, member string) (Aggregate, error) {
	var identity dirtyIdentity
	if err := json.Unmarshal([]byte(member), &identity); err != nil {
		return Aggregate{}, err
	}
	periodStart := identity.PeriodStart.Format("20060102T150405-0700")
	suffix := fmt.Sprintf("%s:%s:%d:%d", identity.PeriodType, periodStart, identity.ProjectionID, identity.Shard)
	values, err := e.redis.HMGet(ctx, dynamicCountPrefix+suffix, identity.Field).Result()
	if err != nil || len(values) != 1 || values[0] == nil {
		return Aggregate{}, fmt.Errorf("read dynamic token statistic count: %w", err)
	}
	versions, err := e.redis.HMGet(ctx, dynamicVersionPrefix+suffix, identity.Field).Result()
	if err != nil || len(versions) != 1 || versions[0] == nil {
		return Aggregate{}, fmt.Errorf("read dynamic token statistic version: %w", err)
	}
	value, err := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
	if err != nil {
		return Aggregate{}, err
	}
	version, err := strconv.ParseInt(fmt.Sprint(versions[0]), 10, 64)
	if err != nil {
		return Aggregate{}, err
	}
	hashBytes, err := hex.DecodeString(identity.DimensionHash)
	if err != nil || len(hashBytes) != 16 {
		return Aggregate{}, fmt.Errorf("invalid dimension hash")
	}
	var hash [16]byte
	copy(hash[:], hashBytes)
	aggregate := Aggregate{
		PeriodType: string(identity.PeriodType), PeriodStart: identity.PeriodStart, PeriodEnd: identity.PeriodEnd,
		ProjectionID: identity.ProjectionID, DimensionHash: hash, DimensionValues: identity.DimensionValues,
		MetricCode: string(identity.MetricCode), MetricValue: value, SourceVersion: version, LastSyncedAt: time.Now(),
	}
	applyRedundantDimensions(&aggregate)
	return aggregate, nil
}

func applyRedundantDimensions(aggregate *Aggregate) {
	aggregate.UserID = jsonInt64(aggregate.DimensionValues["user_id"])
	aggregate.APIKeyID = jsonInt64(aggregate.DimensionValues["api_key_id"])
	aggregate.GroupID = jsonInt64(aggregate.DimensionValues["group_id"])
	aggregate.AccountID = jsonInt64(aggregate.DimensionValues["account_id"])
	aggregate.RouteAlias = jsonString(aggregate.DimensionValues["route_alias"])
	aggregate.UpstreamModel = jsonString(aggregate.DimensionValues["upstream_model"])
	aggregate.Department = jsonString(aggregate.DimensionValues["department"])
}

func jsonInt64(value any) *int64 {
	number, ok := value.(float64)
	if !ok {
		return nil
	}
	result := int64(number)
	return &result
}

func jsonString(value any) *string {
	text, ok := value.(string)
	if !ok || text == "" {
		return nil
	}
	return &text
}

var rotateDirtySet = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
redis.call('RENAME', KEYS[1], KEYS[2])
return 1`)

var requeueDirtySet = redis.NewScript(`
local members = redis.call('SMEMBERS', KEYS[1])
for _, member in ipairs(members) do redis.call('SADD', KEYS[2], member) end
redis.call('DEL', KEYS[1])
return #members`)

var releaseSyncLock = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end
return 0`)
