package tokenstat

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDynamicTokenRuntimeControllerHotTogglePersistsState(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cfg := &config.Config{}
	cfg.Gateway.DynamicTokenStatistics.Enabled = true
	controller := NewRuntimeController(client, cfg)
	t.Cleanup(func() { defaultRuntimeController.Store(nil) })

	require.Equal(t, RuntimeState{Available: true, Enabled: true}, controller.State())
	require.NoError(t, controller.SetEnabled(context.Background(), false))
	require.False(t, controller.Enabled())
	raw, err := mini.Get(runtimeEnabledKey)
	require.NoError(t, err)
	require.Equal(t, "false", raw)
	require.NoError(t, controller.SetEnabled(context.Background(), true))
	require.True(t, controller.Enabled())
}

type blockingQuotaReader struct{}

func (blockingQuotaReader) Read(ctx context.Context, _, _ string) (int64, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}

func TestDefaultQuotaCheckTimesOutAndFailsOpen(t *testing.T) {
	checker := NewQuotaChecker(blockingQuotaReader{}, 16)
	values := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(7)}
	checker.ReplaceRules([]QuotaRule{{
		ID: 1, ProjectionID: 1, DimensionCodes: []DimensionCode{DimensionUserID},
		DimensionValues: values, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		LimitValue: 1, Mode: QuotaModeEnforce,
	}})
	SetDefaultQuotaChecker(checker)
	SetDefaultQuotaTimeout(10 * time.Millisecond)
	defaultRuntimeController.Store(nil)
	t.Cleanup(func() {
		SetDefaultQuotaChecker(nil)
		SetDefaultQuotaTimeout(0)
		defaultRuntimeController.Store(nil)
	})

	started := time.Now()
	decisions := CheckDefaultQuota(context.Background(), time.Now(), values)
	require.Empty(t, decisions)
	require.Less(t, time.Since(started), 100*time.Millisecond)
}
