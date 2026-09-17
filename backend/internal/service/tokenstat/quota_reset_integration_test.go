package tokenstat_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	repository "github.com/Wei-Shaw/sub2api/internal/repository/tokenstat"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

func TestQuotaCheckerUsesResetAwareRuntimeReaderAndFailsOpen(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	values := map[tokenstat.DimensionCode]tokenstat.DimensionValue{tokenstat.DimensionUserID: tokenstat.Int64Value(7)}
	identity, err := tokenstat.BuildDimensionIdentity([]tokenstat.DimensionCode{tokenstat.DimensionUserID}, values)
	require.NoError(t, err)
	period := tokenstat.NaturalPeriods(at, location)[0]
	shard := repository.RedisShard(identity.Hash, 16)
	field := repository.RedisField(identity.Hash, tokenstat.MetricTotalTokens)
	rawKey := repository.DynamicCountKey(period, 2, shard)
	resetKey := repository.DynamicQuotaResetKey(period, 2, shard)
	require.NoError(t, client.HSet(context.Background(), rawKey, field, 120_000).Err())
	require.NoError(t, client.HSet(context.Background(), resetKey, field, 100_000).Err())

	checker := tokenstat.NewQuotaChecker(repository.NewQuotaReader(client), 16)
	checker.ReplaceRules([]tokenstat.QuotaRule{{
		ID: 1, ProjectionID: 2, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID},
		DimensionValues: values, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay,
		LimitValue: 50_000, Mode: tokenstat.QuotaModeEnforce,
	}})
	decisions := checker.Check(context.Background(), at, values)
	require.Len(t, decisions, 1)
	require.Equal(t, int64(20_000), decisions[0].Used)
	require.False(t, decisions[0].Enforced)

	mini.Close()
	require.Empty(t, checker.Check(context.Background(), at, values), "Redis errors must retain fail-open behavior")
}
