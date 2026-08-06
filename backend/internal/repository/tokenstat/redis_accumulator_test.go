package tokenstat

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

func TestDynamicTokenRedisAccumulatorThreePeriodsAndConcurrency(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	accumulator := NewRedisAccumulator(client, 16, 7)
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	periods := domain.NaturalPeriods(time.Date(2026, 7, 30, 12, 0, 0, 0, location), location)
	var hash [16]byte
	hash[0], hash[1] = 1, 2
	operations := make([]domain.AccountingOperation, 0, 3)
	for _, period := range periods {
		operations = append(operations, domain.AccountingOperation{
			Period: period, ProjectionID: 9, DimensionHash: hash,
			DimensionValues: map[string]any{"user_id": 42}, MetricCode: domain.MetricTotalTokens, Delta: 5,
		})
	}

	var wait sync.WaitGroup
	for i := 0; i < 10; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			require.NoError(t, accumulator.Add(context.Background(), operations))
		}()
	}
	wait.Wait()

	field := RedisField(hash, domain.MetricTotalTokens)
	for _, period := range periods {
		keySuffix := string(period.Type) + ":" + RedisPeriodStart(period) + ":9:" + RedisProjectionID(int64(RedisShard(hash, 16)))
		require.Equal(t, "50", mini.HGet(dynamicCountPrefix+keySuffix, field))
		require.Equal(t, "10", mini.HGet(dynamicVersionPrefix+keySuffix, field))
		require.True(t, mini.TTL(dynamicCountPrefix+keySuffix) > 0)
	}
	dirty, err := client.SMembers(context.Background(), dynamicDirtyKey).Result()
	require.NoError(t, err)
	require.Len(t, dirty, 3)
	require.Contains(t, dirty[0], `"dimension_values":{"user_id":42}`)
	for _, key := range mini.Keys() {
		require.False(t, strings.HasPrefix(key, "sub2api:token_stats:"))
	}
}
