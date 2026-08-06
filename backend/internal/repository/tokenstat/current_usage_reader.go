package tokenstat

import (
	"context"
	"fmt"
	"strconv"

	service "github.com/Wei-Shaw/sub2api/internal/service"
	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

type CurrentUsageReader struct {
	redis      *redis.Client
	shardCount int
}

func NewCurrentUsageReader(client *redis.Client, shardCount int) *CurrentUsageReader {
	return &CurrentUsageReader{redis: client, shardCount: shardCount}
}

func (r *CurrentUsageReader) Read(ctx context.Context, period domain.Period, projectionID int64, hash [16]byte, metric domain.MetricCode) (service.ExternalTokenUsageCurrentValue, error) {
	if r == nil || r.redis == nil {
		return service.ExternalTokenUsageCurrentValue{}, fmt.Errorf("dynamic token statistics redis client is required")
	}
	if r.shardCount <= 0 || projectionID <= 0 || metric == "" {
		return service.ExternalTokenUsageCurrentValue{}, fmt.Errorf("invalid current usage lookup")
	}
	key := DynamicCountKey(period, projectionID, RedisShard(hash, r.shardCount))
	value, err := r.redis.HGet(ctx, key, RedisField(hash, metric)).Result()
	if err == redis.Nil {
		return service.ExternalTokenUsageCurrentValue{Exists: false, Value: 0}, nil
	}
	if err != nil {
		return service.ExternalTokenUsageCurrentValue{}, fmt.Errorf("read current dynamic token usage: %w", err)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return service.ExternalTokenUsageCurrentValue{}, fmt.Errorf("invalid current dynamic token usage value %q", value)
	}
	return service.ExternalTokenUsageCurrentValue{Exists: true, Value: parsed}, nil
}
