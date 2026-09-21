package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpsServiceGetModelLatencyPercentiles_ForwardsResolvedFilter(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	var captured *OpsDashboardFilter
	repo := &opsRepoMock{
		GetModelLatencyPercentilesFn: func(_ context.Context, filter *OpsDashboardFilter) (*OpsModelLatencyPercentilesResponse, error) {
			captured = filter
			return &OpsModelLatencyPercentilesResponse{Models: []*OpsModelLatencyPercentile{}}, nil
		},
	}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	filter := &OpsDashboardFilter{StartTime: start, EndTime: start.Add(time.Hour), Model: "gpt-5", QueryMode: OpsQueryModeRaw}

	response, err := svc.GetModelLatencyPercentiles(context.Background(), filter)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Same(t, filter, captured)
	require.Equal(t, OpsQueryModeRaw, captured.QueryMode)
}

func TestOpsServiceGetModelLatencyTrend_ValidatesBucket(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	svc := NewOpsService(&opsRepoMock{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetModelLatencyTrend(context.Background(), &OpsDashboardFilter{StartTime: start, EndTime: start.Add(time.Hour)}, 30)
	require.Error(t, err)
}

func TestOpsServiceGetModelLatencyPercentiles_ValidatesRange(t *testing.T) {
	svc := NewOpsService(&opsRepoMock{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	_, err := svc.GetModelLatencyPercentiles(context.Background(), &OpsDashboardFilter{StartTime: start, EndTime: start.Add(-time.Hour)})
	require.Error(t, err)
}
