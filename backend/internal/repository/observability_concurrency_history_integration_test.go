//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestObservabilityConcurrencyHistoryRollupAndFilter(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { require.NoError(t, timezone.Init("UTC")) })
	ddl, err := os.ReadFile("../../sqlArchiving/178_create_observability_concurrency_rollup.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(ddl))
	require.NoError(t, err)
	capacityDDL, err := os.ReadFile("../../sqlArchiving/179_add_observability_capacity_metrics.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(capacityDDL))
	require.NoError(t, err)
	ratioDDL, err := os.ReadFile("../../sqlArchiving/180_add_concurrency_peak_ratio_sample.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(ratioDDL))
	require.NoError(t, err)

	minute := time.Date(2026, 11, 11, 2, 1, 0, 0, time.UTC)
	repo := NewObservabilityConcurrencyRepository(integrationDB)
	rows := []service.ConcurrencyHistoryRow{
		{WindowSeconds: 60, BucketStart: minute, SampleKind: "account", DimensionHash: "it-a1", AccountID: 1,
			CurrentPeak: 5, CurrentSum: 15, SampleCount: 3, EffectiveLimit: 10, WaitingPeak: 2,
			PeakRatioActual: 5, PeakRatioLimit: 10, PeakRatioSampleAt: minute.Add(40 * time.Second),
			WaitSumMS: 700, WaitSampleCount: 2,
			FirstSampledAt: minute.Add(10 * time.Second), LastSampledAt: minute.Add(50 * time.Second)},
		{WindowSeconds: 60, BucketStart: minute, SampleKind: "route", DimensionHash: "it-r1", AccountID: 1, GroupID: 7, RouteAlias: "coding",
			CurrentPeak: 3, CurrentSum: 6, SampleCount: 3, EffectiveLimit: 8, WaitingPeak: 1,
			FirstSampledAt: minute.Add(10 * time.Second), LastSampledAt: minute.Add(50 * time.Second)},
	}
	require.NoError(t, repo.FlushConcurrencyHistory(context.Background(), minute, rows))
	rows[0].CurrentPeak = 6
	rows[0].CurrentSum = 16
	require.NoError(t, repo.FlushConcurrencyHistory(context.Background(), minute, rows))

	history, err := repo.QueryConcurrencyHistory(context.Background(), service.ConcurrencyHistoryFilter{
		Start: minute, End: minute.Add(time.Minute), WindowSeconds: 60, GroupID: 7, RouteAlias: "coding",
	})
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, 3, history[0].CurrentPeak)
	require.Equal(t, "2026-11-11T10:01:00+08:00", history[0].BucketStart.In(timezone.Location()).Format(time.RFC3339))

	fiveMinute, err := repo.QueryConcurrencyHistory(context.Background(), service.ConcurrencyHistoryFilter{
		Start: minute, End: minute.Add(time.Minute), WindowSeconds: service.ObservabilityWindowFiveMinute, GroupID: 7,
	})
	require.NoError(t, err)
	require.Len(t, fiveMinute, 1)
	accountHistory, err := repo.QueryConcurrencyHistory(context.Background(), service.ConcurrencyHistoryFilter{
		Start: minute, End: minute.Add(time.Minute), WindowSeconds: 60, AccountID: 1,
	})
	require.NoError(t, err)
	require.Len(t, accountHistory, 1)
	require.Equal(t, int64(700), accountHistory[0].WaitSumMS)
	require.Equal(t, int64(2), accountHistory[0].WaitSampleCount)
	require.Equal(t, 5, accountHistory[0].PeakRatioActual)
	require.Equal(t, 10, accountHistory[0].PeakRatioLimit)
	require.Equal(t, minute.Add(40*time.Second), accountHistory[0].PeakRatioSampleAt)
	state, err := repo.GetConcurrencySamplerState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "healthy", state.QualityStatus)
}
