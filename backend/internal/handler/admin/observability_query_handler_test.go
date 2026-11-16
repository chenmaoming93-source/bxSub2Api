package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type queryRollupLoaderStub struct {
	now         time.Time
	rollupCalls int
	seriesCalls int
}

type timeoutQueryStub struct{}

func (timeoutQueryStub) Timeseries(context.Context, service.ObservabilityQueryRequest) (observability.TimeSeriesResponse, error) {
	return observability.TimeSeriesResponse{}, context.DeadlineExceeded
}
func (timeoutQueryStub) LatencyTimeseries(context.Context, service.ObservabilityQueryRequest) (observability.TimeSeriesResponse, error) {
	return observability.TimeSeriesResponse{}, context.DeadlineExceeded
}
func (timeoutQueryStub) Heatmap(context.Context, service.ObservabilityQueryRequest) (observability.HeatmapResponse, error) {
	return observability.HeatmapResponse{}, context.DeadlineExceeded
}

type concurrencyHeatmapStub struct{}

func (concurrencyHeatmapStub) QueryHistory(_ context.Context, filter service.ConcurrencyHistoryFilter) (service.ConcurrencyHistoryResult, error) {
	bucket := filter.End.Add(-time.Second).UTC().Truncate(5 * time.Minute)
	return service.ConcurrencyHistoryResult{
		Available: true, QualityStatus: "healthy",
		Rows: []service.ConcurrencyHistoryRow{{
			BucketStart: bucket, SampleKind: service.ConcurrencySampleKindRoute,
			AccountID: 101, GroupID: 7, RouteAlias: "coding", CurrentPeak: 5, EffectiveLimit: 10,
			PeakRatioActual: 8, PeakRatioLimit: 10, PeakRatioSampleAt: bucket.Add(42 * time.Second),
		}},
	}, nil
}

func (s *queryRollupLoaderStub) LoadSnapshotRollup(_ context.Context, _, _ time.Time, _ observability.Bucket, filter observability.Filter) (service.ObservabilitySnapshotRollup, error) {
	s.rollupCalls++
	return service.ObservabilitySnapshotRollup{
		Users:       []service.SnapshotDimensionRow{{ID: 1, RequestCount: 10, TotalTokens: 100}},
		Groups:      []service.SnapshotDimensionRow{{ID: 7, RequestCount: 10, TotalTokens: 100}},
		Aliases:     []service.SnapshotDimensionRow{{RouteAlias: "coding", RequestCount: 10, TotalTokens: 100}},
		Accounts:    []service.SnapshotDimensionRow{{ID: 101, RequestCount: 10, TotalTokens: 100}},
		SourceMaxAt: &s.now,
	}, nil
}

func (s *queryRollupLoaderStub) LoadSnapshotSeries(_ context.Context, _, end time.Time, _ observability.Bucket, _ observability.Filter, dimension string, _ []int64, _ []string) ([]service.SnapshotSeriesBucket, error) {
	s.seriesCalls++
	bucketStart := end.Add(-time.Second).UTC().Truncate(5 * time.Minute)
	totals := service.SnapshotRollupTotals{RequestCount: 10, SuccessCount: 9, ErrorCount: 1, TotalTokens: 100}
	if dimension == "" {
		return []service.SnapshotSeriesBucket{{BucketStart: bucketStart, Totals: totals}}, nil
	}
	first := totals
	first.RequestCount = 6
	first.TotalTokens = 70
	first.ErrorCount = 1
	second := totals
	second.RequestCount = 4
	second.TotalTokens = 30
	second.ErrorCount = 0
	return []service.SnapshotSeriesBucket{
		{BucketStart: bucketStart, DimensionID: 7, Totals: first},
		{BucketStart: bucketStart, DimensionID: 8, Totals: second},
	}, nil
}

func TestObservabilityQueryGroupByKeepsCanonicalSummary(t *testing.T) {
	loader := &queryRollupLoaderStub{now: time.Now().UTC()}
	query := service.NewObservabilityQueryService(loader, nil)
	base := service.ObservabilityQueryRequest{Window: observability.Window1h, Bucket: observability.Bucket5m}

	plain, err := query.Timeseries(context.Background(), base)
	require.NoError(t, err)
	groupedRequest := base
	groupedRequest.GroupBy = "group_id"
	grouped, err := query.Timeseries(context.Background(), groupedRequest)
	require.NoError(t, err)
	require.Equal(t, seriesSummaryValue(plain, "summary_total_requests"), seriesSummaryValue(grouped, "summary_total_requests"))
	require.Equal(t, seriesSummaryValue(plain, "summary_total_tokens"), seriesSummaryValue(grouped, "summary_total_tokens"))
	require.Equal(t, seriesSummaryValue(grouped, "summary_total_requests"), sumDimensionSeries(grouped, "requests:"))
	require.Equal(t, seriesSummaryValue(grouped, "summary_total_tokens"), sumDimensionSeries(grouped, "tokens:"))

	heatmapRequest := base
	heatmapRequest.GroupBy = "group_id"
	heatmapRequest.ValueKind = "error_rate"
	heatmap, err := query.Heatmap(context.Background(), heatmapRequest)
	require.NoError(t, err)
	require.True(t, heatmap.Available)
	require.NotEmpty(t, heatmap.Cells)
	foundErrorRate := false
	for _, cell := range heatmap.Cells {
		if cell.Row == "7" && cell.Value == float64(1)/6 {
			foundErrorRate = true
			break
		}
	}
	require.True(t, foundErrorRate, "cells=%+v series=%+v", heatmap.Cells, grouped.Series)

	beforeSeriesCalls := loader.seriesCalls
	_, err = query.Timeseries(context.Background(), groupedRequest)
	require.NoError(t, err)
	require.Equal(t, beforeSeriesCalls, loader.seriesCalls, "5s cache must avoid a second series query")
}

func TestObservabilityQueryUnavailableStatesAndDSTLabels(t *testing.T) {
	loader := &queryRollupLoaderStub{now: time.Now().UTC()}
	query := service.NewObservabilityQueryService(loader, nil)
	heatmap, err := query.Heatmap(context.Background(), service.ObservabilityQueryRequest{
		Window: observability.Window6h, Bucket: observability.Bucket15m, ValueKind: "sampled_concurrency",
	})
	require.NoError(t, err)
	require.False(t, heatmap.Available)
	require.Contains(t, heatmap.Meta.Warnings, "concurrency_samples_unavailable")

	latency, err := query.LatencyTimeseries(context.Background(), service.ObservabilityQueryRequest{
		Window: observability.Window1h, Bucket: observability.Bucket5m,
	})
	require.NoError(t, err)
	require.Len(t, latency.Labels, 12)
	require.Contains(t, latency.Meta.Warnings, "partial_latency_histogram")
}

func TestObservabilityQuerySampledConcurrencyHeatmapUsesPersistedPeak(t *testing.T) {
	loader := &queryRollupLoaderStub{now: time.Now().UTC()}
	query := service.NewObservabilityQueryService(loader, concurrencyHeatmapStub{})
	response, err := query.Heatmap(context.Background(), service.ObservabilityQueryRequest{
		Window: observability.Window1h, Bucket: observability.Bucket5m,
		GroupBy: "group_id", ValueKind: "sampled_concurrency",
	})
	require.NoError(t, err)
	require.True(t, response.Available)
	require.Equal(t, []string{"7"}, response.Rows)
	found := false
	for _, cell := range response.Cells {
		if cell.ActualConcurrency == 8 && cell.ConcurrencyLimit == 10 && cell.Value == 0.8 {
			require.NotNil(t, cell.SampleAt)
			found = true
		}
	}
	require.True(t, found)
}

func TestObservabilityQueryHandlerValidatesGroupByAndValueKind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := &queryRollupLoaderStub{now: time.Now().UTC()}
	h := NewOpsHandler(nil)
	h.observabilityQuery = service.NewObservabilityQueryService(loader, nil)
	router := gin.New()
	router.GET("/timeseries", h.GetObservabilityTimeseries)
	router.GET("/heatmap", h.GetObservabilityHeatmap)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeseries?window=7d&bucket=5m", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/heatmap?window=1h&value_kind=unknown", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestObservabilityQueryHandlerTimeoutIsStable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOpsHandler(nil)
	h.observabilityQuery = timeoutQueryStub{}
	router := gin.New()
	router.GET("/timeseries", h.GetObservabilityTimeseries)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeseries?window=1h", nil))
	require.Equal(t, http.StatusGatewayTimeout, recorder.Code)
	require.Contains(t, recorder.Body.String(), "OBSERVABILITY_QUERY_TIMEOUT")
}

func TestObservabilityQueryHandlerAcceptsCustomRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := &queryRollupLoaderStub{now: time.Now().UTC()}
	handler := NewOpsHandler(nil)
	handler.observabilityQuery = service.NewObservabilityQueryService(loader, nil)
	router := gin.New()
	router.GET("/timeseries", handler.GetObservabilityTimeseries)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/timeseries?window=custom&bucket=15m&start_at=2026-11-12T01:00:00Z&end_at=2026-11-12T02:00:00Z",
		nil,
	))
	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Data observability.TimeSeriesResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC), envelope.Data.Meta.StartAt)
	require.Equal(t, time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC), envelope.Data.Meta.EndAt)
	require.Len(t, envelope.Data.Labels, 4)
}

func seriesSummaryValue(response observability.TimeSeriesResponse, key string) float64 {
	for _, item := range response.Summary {
		if item.Key == key && len(item.Points) > 0 {
			return item.Points[0]
		}
	}
	return -1
}

func sumDimensionSeries(response observability.TimeSeriesResponse, prefix string) float64 {
	var total float64
	for _, item := range response.Series {
		if len(item.Key) > len(prefix) && item.Key[:len(prefix)] == prefix && !hasSuffix(item.Key, ":other") {
			for _, point := range item.Points {
				total += point
			}
		}
	}
	return total
}

func hasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
