package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type boundedHeatmapRepository struct {
	boundedCalled   bool
	unboundedCalled bool
}

func (r *boundedHeatmapRepository) FlushConcurrencyHistory(context.Context, time.Time, []ConcurrencyHistoryRow) error {
	return nil
}

func (r *boundedHeatmapRepository) GetConcurrencySamplerState(context.Context) (ConcurrencySamplerState, error) {
	return ConcurrencySamplerState{}, nil
}

func (r *boundedHeatmapRepository) RecordConcurrencySample(context.Context, time.Time, int64, bool) error {
	return nil
}

func (r *boundedHeatmapRepository) QueryConcurrencyHistory(context.Context, ConcurrencyHistoryFilter) ([]ConcurrencyHistoryRow, error) {
	r.unboundedCalled = true
	return nil, nil
}

func (r *boundedHeatmapRepository) CleanupConcurrencyHistory(context.Context, time.Time) error {
	return nil
}

func (r *boundedHeatmapRepository) QueryConcurrencyHeatmapHistory(
	_ context.Context,
	_ ConcurrencyHistoryFilter,
	_ string,
	_ int,
) ([]ConcurrencyHistoryRow, error) {
	r.boundedCalled = true
	return []ConcurrencyHistoryRow{{DimensionName: "coding"}}, nil
}

func TestConcurrencyHistoryServiceUsesBoundedHeatmapQuery(t *testing.T) {
	repo := &boundedHeatmapRepository{}
	service := NewConcurrencyHistoryService(repo, nil, nil, nil, nil, nil, nil)

	rows, err := service.QueryConcurrencyHeatmapHistory(
		context.Background(),
		ConcurrencyHistoryFilter{WindowSeconds: ObservabilityWindowFiveMinute},
		"group_id",
		20,
	)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, repo.boundedCalled)
	require.False(t, repo.unboundedCalled)
}
