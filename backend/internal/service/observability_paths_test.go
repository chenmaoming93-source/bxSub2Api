package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/stretchr/testify/require"
)

type observabilityPathStageLoader struct {
	aggregate ObservabilityPathAggregate
	stages    map[string]string
}

func (l *observabilityPathStageLoader) LoadPathAggregates(context.Context, ObservabilityPathQuery) ([]ObservabilityPathAggregate, int, error) {
	return []ObservabilityPathAggregate{l.aggregate}, 1, nil
}

func (l *observabilityPathStageLoader) LoadAPIKeyRelations(context.Context, observability.Filter, string) ([]APIKeyRelation, error) {
	return []APIKeyRelation{{UserID: l.aggregate.UserID, KeyID: l.aggregate.APIKeyID, KeyName: "primary", GroupID: l.aggregate.GroupID}}, nil
}

func (l *observabilityPathStageLoader) LoadRouteRelations(context.Context, observability.Filter) ([]RouteRelation, error) {
	return nil, nil
}

func (l *observabilityPathStageLoader) LoadPathErrorStages(_ context.Context, _, _ time.Time, refs []PathRef) (map[string]string, error) {
	return l.stages, nil
}

func TestObservabilityPathsPopulatesDominantErrorStage(t *testing.T) {
	start := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	aggregate := ObservabilityPathAggregate{
		UserID: 1, APIKeyID: 2, GroupID: 3, RouteAlias: "coding-fast", AccountID: 4,
		RequestCount: 10, SuccessCount: 9, ErrorCount: 1,
		DataAt: start.Add(5 * time.Minute),
	}
	ref := PathRef{
		UserID: aggregate.UserID, APIKeyID: aggregate.APIKeyID, GroupID: aggregate.GroupID,
		RouteAlias: aggregate.RouteAlias, AccountID: aggregate.AccountID,
	}
	loader := &observabilityPathStageLoader{
		aggregate: aggregate,
		stages:    map[string]string{pathRefKey(ref): "upstream"},
	}
	service := NewObservabilityPathService(loader, nil, nil, nil, nil)

	response, err := service.Paths(context.Background(), ObservabilityPathQuery{
		Start: start, End: start.Add(time.Hour), Window: observability.Window1h,
		Bucket: observability.Bucket5m, Page: 1, PageSize: 20,
	})

	require.NoError(t, err)
	require.Len(t, response.Rows, 1)
	require.Equal(t, "upstream", response.Rows[0].ErrorStage)
	require.NotContains(t, response.Meta.Warnings, "error_stage_unavailable")
}
