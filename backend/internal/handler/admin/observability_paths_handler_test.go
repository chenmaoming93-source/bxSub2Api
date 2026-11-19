package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type pathLoaderStub struct {
	lastQuery service.ObservabilityPathQuery
}

func (s *pathLoaderStub) LoadPathAggregates(_ context.Context, query service.ObservabilityPathQuery) ([]service.ObservabilityPathAggregate, int, error) {
	s.lastQuery = query
	return []service.ObservabilityPathAggregate{{
		UserID: 1, APIKeyID: 11, GroupID: 7, RouteAlias: "gpt-5", AccountID: 101,
		RequestCount: 5, SuccessCount: 5, TotalTokens: 500,
		TTFT: observability.Histogram{SampleCount: 5, MinMS: 100, MaxMS: 500, Buckets: [8]int64{1, 1, 1, 1, 1}},
	}}, 1, nil
}

func (s *pathLoaderStub) LoadAPIKeyRelations(context.Context, observability.Filter, string) ([]service.APIKeyRelation, error) {
	return []service.APIKeyRelation{
		{UserID: 1, KeyID: 11, KeyName: "active-key", GroupID: 7},
		{UserID: 2, KeyID: 22, KeyName: "idle-key", GroupID: 8},
	}, nil
}

func (s *pathLoaderStub) LoadRouteRelations(context.Context, observability.Filter) ([]service.RouteRelation, error) {
	return []service.RouteRelation{
		{GroupID: 7, RouteAlias: "gpt-*", AccountID: 101},
		{GroupID: 8, RouteAlias: "idle-route", AccountID: 202},
	}, nil
}

func TestObservabilityPathsAndRelationsIdleAndWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := &pathLoaderStub{}
	pathService := service.NewObservabilityPathService(loader, nil, nil, nil, nil)
	h := NewOpsHandler(nil)
	h.observabilityPath = pathService
	router := gin.New()
	router.GET("/paths", h.GetObservabilityPaths)
	router.GET("/relations", h.GetObservabilityRelations)

	paths := httptest.NewRecorder()
	router.ServeHTTP(paths, httptest.NewRequest(http.MethodGet,
		"/paths?window=1h&search=gpt&status=healthy&sort=requests&page=2&page_size=10", nil))
	require.Equal(t, http.StatusOK, paths.Code)
	require.Equal(t, 2, loader.lastQuery.Page)
	require.Equal(t, "requests", loader.lastQuery.Sort)
	require.Contains(t, paths.Body.String(), `"route_alias":"gpt-5"`)

	relations := httptest.NewRecorder()
	router.ServeHTTP(relations, httptest.NewRequest(http.MethodGet, "/relations?window=1h&include_idle=false", nil))
	require.Equal(t, http.StatusOK, relations.Code)
	require.Contains(t, relations.Body.String(), `"route_alias":"gpt-*"`)
	require.Contains(t, relations.Body.String(), `"idle":false`)
	require.NotContains(t, relations.Body.String(), `"idle-key"`)
}

func TestObservabilityPathsSevenDayCoverageIsTopN(t *testing.T) {
	loader := &pathLoaderStub{}
	pathService := service.NewObservabilityPathService(loader, nil, nil, nil, nil)
	response, err := pathService.Paths(context.Background(), service.ObservabilityPathQuery{
		Window: observability.Window7d, Bucket: observability.Bucket6h,
		Page: 9, PageSize: 100,
	})
	require.NoError(t, err)
	require.Equal(t, 1, loader.lastQuery.Page)
	require.Equal(t, 100, loader.lastQuery.PageSize)
	require.Equal(t, "top_n", response.Meta.Coverage)
	require.Contains(t, response.Meta.Warnings, "coverage_top_n")
}
