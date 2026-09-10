package tokenstat

import (
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojection"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojectionmetric"
)

var (
	ErrUsageProjectionNotConfigured = errors.New("usage projection is not configured")
	ErrUsageProjectionNotActive     = errors.New("usage projection is not active")
)

// ResolveUsageProjection finds the exact dynamic-statistics projection required
// by a usage report and verifies that both the projection and metric are active.
// The daily period is intentionally validated here so callers cannot silently
// switch this report to a different aggregation period.
func (s *ProjectionAdminService) ResolveUsageProjection(ctx context.Context, dimensions []DimensionCode, metric MetricCode, period PeriodType) (*ent.TokenStatProjection, error) {
	if period != PeriodDay {
		return nil, fmt.Errorf("scene usage requires daily period")
	}
	signature, err := DimensionSignature(dimensions)
	if err != nil {
		return nil, err
	}
	projections, err := s.client.TokenStatProjection.Query().
		Where(tokenstatprojection.DimensionSignatureEQ(signature)).
		Order(ent.Asc(tokenstatprojection.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(projections) == 0 {
		return nil, ErrUsageProjectionNotConfigured
	}
	for _, projection := range projections {
		if projection.Status != ProjectionStatusActive {
			continue
		}
		activeMetric, err := s.client.TokenStatProjectionMetric.Query().
			Where(
				tokenstatprojectionmetric.ProjectionIDEQ(projection.ID),
				tokenstatprojectionmetric.MetricCodeEQ(string(metric)),
				tokenstatprojectionmetric.StatusEQ(ProjectionStatusActive),
			).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if activeMetric {
			return projection, nil
		}
	}
	return nil, ErrUsageProjectionNotActive
}
