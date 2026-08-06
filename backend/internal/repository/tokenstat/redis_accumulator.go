package tokenstat

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

const (
	dynamicCountPrefix   = "sub2api:dynamic_token_stats:v1:"
	dynamicVersionPrefix = "sub2api:dynamic_token_stats_ver:v1:"
	dynamicDirtyKey      = "sub2api:dynamic_token_stats_dirty:v1:current"
	defaultMaxOperations = 4096
)

type RedisAccumulator struct {
	client        *redis.Client
	shardCount    int
	maxOperations int
	orphanTTL     time.Duration
}

func NewRedisAccumulator(client *redis.Client, shardCount, orphanTTLDays int) *RedisAccumulator {
	return &RedisAccumulator{
		client: client, shardCount: shardCount, maxOperations: defaultMaxOperations,
		orphanTTL: time.Duration(orphanTTLDays) * 24 * time.Hour,
	}
}

func (a *RedisAccumulator) Add(ctx context.Context, operations []domain.AccountingOperation) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("dynamic token statistics redis client is required")
	}
	if a.shardCount <= 0 {
		return fmt.Errorf("shard count must be positive")
	}
	if len(operations) == 0 || len(operations) > a.maxOperations {
		return fmt.Errorf("operation count must be between 1 and %d", a.maxOperations)
	}
	keys := make([]string, 0, len(operations)*2+1)
	args := make([]any, 0, len(operations)*5)
	for _, operation := range operations {
		if operation.ProjectionID <= 0 || operation.Delta < 0 || operation.MetricCode == "" {
			return fmt.Errorf("invalid redis accounting operation")
		}
		shard := int(binary.BigEndian.Uint16(operation.DimensionHash[:2])) % a.shardCount
		countKey := DynamicCountKey(operation.Period, operation.ProjectionID, shard)
		versionKey := fmt.Sprintf("%s%s:%s:%d:%d", dynamicVersionPrefix, operation.Period.Type, RedisPeriodStart(operation.Period), operation.ProjectionID, shard)
		field := hex.EncodeToString(operation.DimensionHash[:]) + ":" + string(operation.MetricCode)
		dirty, err := dirtyIdentityJSON(operation, shard, field)
		if err != nil {
			return err
		}
		expireAt := operation.Period.End.Add(a.orphanTTL).Unix()
		keys = append(keys, countKey, versionKey)
		args = append(args, field, operation.Delta, dirty, expireAt)
	}
	keys = append(keys, dynamicDirtyKey)
	if err := dynamicAddScript.Run(ctx, a.client, keys, args...).Err(); err != nil {
		return fmt.Errorf("atomic dynamic token statistics add: %w", err)
	}
	return nil
}

func dirtyIdentityJSON(operation domain.AccountingOperation, shard int, field string) (string, error) {
	payload := map[string]any{
		"period_type": operation.Period.Type, "period_start": operation.Period.Start,
		"period_end": operation.Period.End, "projection_id": operation.ProjectionID,
		"shard": shard, "field": field, "dimension_hash": hex.EncodeToString(operation.DimensionHash[:]),
		"dimension_values": operation.DimensionValues, "metric_code": operation.MetricCode,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal dirty identity: %w", err)
	}
	return string(encoded), nil
}

var dynamicAddScript = redis.NewScript(`
local operation_count = (#KEYS - 1) / 2
local dirty_key = KEYS[#KEYS]
for index = 1, operation_count do
    local key_index = (index - 1) * 2 + 1
    local arg_index = (index - 1) * 4 + 1
    local count_key = KEYS[key_index]
    local version_key = KEYS[key_index + 1]
    local field = ARGV[arg_index]
    local delta = tonumber(ARGV[arg_index + 1])
    local dirty_identity = ARGV[arg_index + 2]
    local expire_at = tonumber(ARGV[arg_index + 3])
    redis.call('HINCRBY', count_key, field, delta)
    redis.call('HINCRBY', version_key, field, 1)
    redis.call('EXPIREAT', count_key, expire_at)
    redis.call('EXPIREAT', version_key, expire_at)
    redis.call('SADD', dirty_key, dirty_identity)
end
return operation_count
`)

func RedisField(hash [16]byte, metric domain.MetricCode) string {
	return hex.EncodeToString(hash[:]) + ":" + string(metric)
}

func RedisShard(hash [16]byte, shardCount int) int {
	if shardCount <= 0 {
		return 0
	}
	return int(binary.BigEndian.Uint16(hash[:2])) % shardCount
}

func RedisPeriodStart(period domain.Period) string {
	return period.Start.Format("20060102T150405-0700")
}

func RedisProjectionID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// DynamicCountKey returns the Redis hash key shared by the dynamic statistics
// writer and exact current-period readers.
func DynamicCountKey(period domain.Period, projectionID int64, shard int) string {
	return fmt.Sprintf("%s%s:%s:%d:%d", dynamicCountPrefix, period.Type, RedisPeriodStart(period), projectionID, shard)
}
