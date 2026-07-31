package tokenstat

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

func TestCurrentUsageReaderMatchesAccumulatorForThreePeriods(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	periods := domain.NaturalPeriods(time.Date(2026, 7, 31, 12, 0, 0, 0, location), location)
	identity, err := domain.BuildDimensionIdentity(
		[]domain.DimensionCode{domain.DimensionUserID, domain.DimensionAPIKeyID, domain.DimensionGroupID, domain.DimensionRouteAlias},
		map[domain.DimensionCode]domain.DimensionValue{
			domain.DimensionUserID: domain.Int64Value(1), domain.DimensionAPIKeyID: domain.Int64Value(2),
			domain.DimensionGroupID: domain.Int64Value(3), domain.DimensionRouteAlias: domain.StringValue("gpt-main"),
		},
	)
	require.NoError(t, err)
	operations := make([]domain.AccountingOperation, 0, len(periods))
	for _, period := range periods {
		operations = append(operations, domain.AccountingOperation{Period: period, ProjectionID: 9, DimensionHash: identity.Hash, DimensionValues: map[string]any{"user_id": 1}, MetricCode: domain.MetricTotalTokens, Delta: 17})
	}
	require.NoError(t, NewRedisAccumulator(client, 16, 7).Add(context.Background(), operations))
	reader := NewCurrentUsageReader(client, 16)
	for _, period := range periods {
		got, readErr := reader.Read(context.Background(), period, 9, identity.Hash, domain.MetricTotalTokens)
		require.NoError(t, readErr)
		require.True(t, got.Exists)
		require.Equal(t, int64(17), got.Value)
	}
}

func TestCurrentUsageReaderDistinguishesMissingZeroAndInvalid(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	period := domain.Period{Type: domain.PeriodDay, Start: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	hash := [16]byte{1, 2}
	reader := NewCurrentUsageReader(client, 4)

	missing, err := reader.Read(context.Background(), period, 5, hash, domain.MetricTotalTokens)
	require.NoError(t, err)
	require.False(t, missing.Exists)
	require.Zero(t, missing.Value)

	key := DynamicCountKey(period, 5, RedisShard(hash, 4))
	field := RedisField(hash, domain.MetricTotalTokens)
	mini.HSet(key, field, "0")
	zero, err := reader.Read(context.Background(), period, 5, hash, domain.MetricTotalTokens)
	require.NoError(t, err)
	require.True(t, zero.Exists)
	require.Zero(t, zero.Value)

	for _, invalid := range []string{"bad", "-1"} {
		mini.HSet(key, field, invalid)
		_, err = reader.Read(context.Background(), period, 5, hash, domain.MetricTotalTokens)
		require.Error(t, err)
	}
}

func TestCurrentUsageReaderRejectsInvalidConfiguration(t *testing.T) {
	_, err := NewCurrentUsageReader(nil, 0).Read(context.Background(), domain.Period{}, 0, [16]byte{}, domain.MetricTotalTokens)
	require.Error(t, err)
}
