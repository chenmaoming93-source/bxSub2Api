package tokenstat

import (
	"context"
	"fmt"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

const dynamicQuotaResetPrefix = "sub2api:dynamic_token_quota_reset:v1:"

type QuotaResetSnapshotStatus string

const (
	QuotaResetSnapshotReset   QuotaResetSnapshotStatus = "RESET"
	QuotaResetSnapshotNoUsage QuotaResetSnapshotStatus = "NO_USAGE"
)

// QuotaResetIdentity identifies one concrete dynamic-token statistic field.
type QuotaResetIdentity struct {
	Period        domain.Period
	ProjectionID  int64
	DimensionHash [16]byte
	MetricCode    domain.MetricCode
}

// RedisQuotaResetStore writes quota reset baselines without changing the raw
// statistics hash, version hash, dirty set, or MySQL aggregates.
type RedisQuotaResetStore struct {
	client     *redis.Client
	shardCount int
	orphanTTL  time.Duration
}

func NewRedisQuotaResetStore(client *redis.Client, shardCount, orphanTTLDays int) *RedisQuotaResetStore {
	return &RedisQuotaResetStore{
		client: client, shardCount: shardCount,
		orphanTTL: time.Duration(orphanTTLDays) * 24 * time.Hour,
	}
}

// DynamicQuotaResetKey returns the reset hash key corresponding to a dynamic
// statistics hash. Its suffix deliberately uses the existing period,
// projection, and shard semantics.
func DynamicQuotaResetKey(period domain.Period, projectionID int64, shard int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", dynamicQuotaResetPrefix, period.Type, RedisPeriodStart(period), projectionID, shard)
}

// ReadBaselines reads reset snapshots in one Redis pipeline. Missing fields are
// valid and mean that the matching statistic identity has never been reset.
func (s *RedisQuotaResetStore) ReadBaselines(ctx context.Context, identities []domain.StatisticIdentity) ([]int64, error) {
	result := make([]int64, len(identities))
	if len(identities) == 0 {
		return result, nil
	}
	if s == nil || s.client == nil || s.shardCount <= 0 {
		return nil, fmt.Errorf("dynamic token quota reset redis client is required")
	}
	pipeline := s.client.Pipeline()
	commands := make([]*redis.StringCmd, len(identities))
	for index, identity := range identities {
		shard := RedisShard(identity.DimensionHash, s.shardCount)
		commands[index] = pipeline.HGet(ctx,
			DynamicQuotaResetKey(identity.Period, identity.ProjectionID, shard),
			RedisField(identity.DimensionHash, identity.MetricCode),
		)
	}
	if _, err := pipeline.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("read dynamic token quota reset baselines: %w", err)
	}
	for index, command := range commands {
		value, err := command.Int64()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read dynamic token quota reset baseline: %w", err)
		}
		if value < 0 {
			return nil, fmt.Errorf("dynamic token quota reset baseline must not be negative")
		}
		result[index] = value
	}
	return result, nil
}

// Snapshot atomically copies the current raw usage into the reset namespace.
// A missing raw field does not create a baseline and returns NO_USAGE.
func (s *RedisQuotaResetStore) Snapshot(ctx context.Context, identity QuotaResetIdentity) (QuotaResetSnapshotStatus, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("dynamic token quota reset redis client is required")
	}
	if s.shardCount <= 0 {
		return "", fmt.Errorf("shard count must be positive")
	}
	if identity.ProjectionID <= 0 || identity.MetricCode == "" || identity.Period.End.IsZero() {
		return "", fmt.Errorf("invalid quota reset identity")
	}

	shard := RedisShard(identity.DimensionHash, s.shardCount)
	rawKey := DynamicCountKey(identity.Period, identity.ProjectionID, shard)
	resetKey := DynamicQuotaResetKey(identity.Period, identity.ProjectionID, shard)
	field := RedisField(identity.DimensionHash, identity.MetricCode)
	result, err := quotaResetSnapshotScript.Run(ctx, s.client, []string{rawKey, resetKey}, field, identity.Period.End.Add(s.orphanTTL).Unix()).Int64()
	if err != nil {
		return "", fmt.Errorf("snapshot dynamic token quota usage: %w", err)
	}
	if result == 0 {
		return QuotaResetSnapshotNoUsage, nil
	}
	return QuotaResetSnapshotReset, nil
}

// SnapshotBatch pipelines independent per-entry Lua commands. The pipeline is
// intentionally non-transactional: each script is atomic, while one entry's
// error does not roll back or hide successful siblings.
func (s *RedisQuotaResetStore) SnapshotBatch(ctx context.Context, identities []domain.StatisticIdentity) ([]domain.QuotaResetEntryResult, error) {
	results := make([]domain.QuotaResetEntryResult, len(identities))
	if len(identities) == 0 {
		return results, nil
	}
	if s == nil || s.client == nil || s.shardCount <= 0 {
		for i := range results {
			results[i] = domain.QuotaResetEntryResult{Status: domain.QuotaResetEntryFailed, Err: fmt.Errorf("dynamic token quota reset redis client is required")}
		}
		return results, results[0].Err
	}

	pipeline := s.client.Pipeline()
	commands := make([]*redis.Cmd, len(identities))
	for i, identity := range identities {
		if identity.ProjectionID <= 0 || identity.MetricCode == "" || identity.Period.End.IsZero() {
			results[i] = domain.QuotaResetEntryResult{Status: domain.QuotaResetEntryFailed, Err: fmt.Errorf("invalid quota reset identity")}
			continue
		}
		shard := RedisShard(identity.DimensionHash, s.shardCount)
		commands[i] = pipeline.Eval(ctx, quotaResetSnapshotLua,
			[]string{DynamicCountKey(identity.Period, identity.ProjectionID, shard), DynamicQuotaResetKey(identity.Period, identity.ProjectionID, shard)},
			RedisField(identity.DimensionHash, identity.MetricCode), identity.Period.End.Add(s.orphanTTL).Unix())
	}
	_, execErr := pipeline.Exec(ctx)
	for i, command := range commands {
		if command == nil {
			continue
		}
		value, err := command.Int64()
		if err != nil {
			results[i] = domain.QuotaResetEntryResult{Status: domain.QuotaResetEntryFailed, Err: err}
		} else if value == 0 {
			results[i] = domain.QuotaResetEntryResult{Status: domain.QuotaResetEntryNoUsage}
		} else {
			results[i] = domain.QuotaResetEntryResult{Status: domain.QuotaResetEntryReset}
		}
	}
	return results, execErr
}

const quotaResetSnapshotLua = `
local raw = redis.call('HGET', KEYS[1], ARGV[1])
if not raw then
    return 0
end
redis.call('HSET', KEYS[2], ARGV[1], raw)
redis.call('EXPIREAT', KEYS[2], tonumber(ARGV[2]))
return 1
`

var quotaResetSnapshotScript = redis.NewScript(quotaResetSnapshotLua)
