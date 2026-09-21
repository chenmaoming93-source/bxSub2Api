package service

import (
	"context"
	"errors"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *OpsService) GetModelLatencyPercentiles(ctx context.Context, filter *OpsDashboardFilter) (*OpsModelLatencyPercentilesResponse, error) {
	if err := s.validateModelLatencyFilter(ctx, filter); err != nil {
		return nil, err
	}

	result, err := s.opsRepo.GetModelLatencyPercentiles(ctx, filter)
	if err != nil && shouldFallbackOpsPreagg(filter, err) {
		result, err = s.opsRepo.GetModelLatencyPercentiles(ctx, cloneOpsFilterWithMode(filter, OpsQueryModeRaw))
	}
	if errors.Is(err, ErrOpsPreaggregatedNotPopulated) {
		return nil, infraerrors.Conflict("OPS_PREAGG_NOT_READY", "Pre-aggregated ops metrics are not populated yet")
	}
	return result, err
}

func (s *OpsService) GetModelLatencyTrend(ctx context.Context, filter *OpsDashboardFilter, bucketSeconds int) (*OpsModelLatencyTrendResponse, error) {
	if err := s.validateModelLatencyFilter(ctx, filter); err != nil {
		return nil, err
	}
	if bucketSeconds != 60 && bucketSeconds != 300 && bucketSeconds != 3600 {
		return nil, infraerrors.BadRequest("OPS_BUCKET_INVALID", "bucket must be 1m, 5m, or 1h")
	}

	result, err := s.opsRepo.GetModelLatencyTrend(ctx, filter, bucketSeconds)
	if err != nil && shouldFallbackOpsPreagg(filter, err) {
		result, err = s.opsRepo.GetModelLatencyTrend(ctx, cloneOpsFilterWithMode(filter, OpsQueryModeRaw), bucketSeconds)
	}
	if errors.Is(err, ErrOpsPreaggregatedNotPopulated) {
		return nil, infraerrors.Conflict("OPS_PREAGG_NOT_READY", "Pre-aggregated ops metrics are not populated yet")
	}
	return result, err
}

func (s *OpsService) validateModelLatencyFilter(ctx context.Context, filter *OpsDashboardFilter) error {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return err
	}
	if s.opsRepo == nil {
		return infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if filter == nil {
		return infraerrors.BadRequest("OPS_FILTER_REQUIRED", "filter is required")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return infraerrors.BadRequest("OPS_TIME_RANGE_REQUIRED", "start_time/end_time are required")
	}
	if filter.StartTime.After(filter.EndTime) {
		return infraerrors.BadRequest("OPS_TIME_RANGE_INVALID", "start_time must be <= end_time")
	}
	filter.QueryMode = s.resolveOpsQueryMode(ctx, filter.QueryMode)
	return nil
}
