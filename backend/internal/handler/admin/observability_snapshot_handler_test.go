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

type snapshotHandlerStub struct {
	result   service.SnapshotResult
	err      error
	requests []service.SnapshotRequest
}

type snapshotRollupLoaderStub struct {
	calls int
	now   time.Time
}

type snapshotConcurrencyStub struct{}

func (snapshotConcurrencyStub) CollectCurrent(_ context.Context, sampledAt time.Time) ([]service.ConcurrencyCurrentValue, error) {
	return []service.ConcurrencyCurrentValue{
		{Kind: service.ConcurrencySampleKindAccount, AccountID: 101, Current: 38, Limit: 40, Waiting: 21, SampledAt: sampledAt, Available: true},
		{Kind: service.ConcurrencySampleKindAccount, AccountID: 102, Current: 32, Limit: 40, Waiting: 8, SampledAt: sampledAt, Available: true},
		{Kind: service.ConcurrencySampleKindRoute, AccountID: 101, GroupID: 7, RouteAlias: "coding", Current: 34, Limit: 40, Waiting: 21, SampledAt: sampledAt, Available: true},
		{Kind: service.ConcurrencySampleKindRoute, AccountID: 102, GroupID: 8, RouteAlias: "other", Current: 20, Limit: 40, Waiting: 8, SampledAt: sampledAt, Available: true},
	}, nil
}

func (snapshotConcurrencyStub) QueryHistory(context.Context, service.ConcurrencyHistoryFilter) (service.ConcurrencyHistoryResult, error) {
	return service.ConcurrencyHistoryResult{Available: true, Rows: []service.ConcurrencyHistoryRow{
		{SampleKind: service.ConcurrencySampleKindAccount, AccountID: 101, WaitSumMS: 420000, WaitSampleCount: 1000},
		{SampleKind: service.ConcurrencySampleKindRoute, AccountID: 101, GroupID: 7, WaitSumMS: 420000, WaitSampleCount: 1000},
	}}, nil
}

func (s *snapshotRollupLoaderStub) LoadSnapshotRollup(_ context.Context, _, _ time.Time, _ observability.Bucket, _ observability.Filter) (service.ObservabilitySnapshotRollup, error) {
	s.calls++
	return service.ObservabilitySnapshotRollup{
		Totals: service.SnapshotRollupTotals{
			RequestCount: 8, SuccessCount: 7, ErrorCount: 1, TotalTokens: 800,
			RateLimitedCount: 126,
			Duration:         observability.Histogram{SampleCount: 8, MinMS: 100, MaxMS: 900, Buckets: [8]int64{1, 2, 2, 2, 1}},
			TTFT:             observability.Histogram{SampleCount: 4, MinMS: 100, MaxMS: 900, Buckets: [8]int64{1, 1, 1, 1}},
		},
		SourceMaxAt: &s.now,
	}, nil
}

func TestObservabilityCapacitySummaryUsesRealWaitAndStableDailyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := &snapshotRollupLoaderStub{now: time.Now().UTC()}
	snapshotService := service.NewObservabilitySnapshotService(loader, nil, nil, nil, snapshotConcurrencyStub{}, nil)
	handler := NewOpsHandler(nil)
	handler.observabilitySnapshot = snapshotService
	router := gin.New()
	router.GET("/snapshot", handler.GetObservabilitySnapshot)

	for _, target := range []string{
		"/snapshot?window=1h",
		"/snapshot?window=7d",
		"/snapshot?window=custom&bucket=15m&start_at=2026-11-12T01:00:00Z&end_at=2026-11-12T02:00:00Z",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		require.Equal(t, http.StatusOK, recorder.Code, target)
		var envelope struct {
			Data observability.SnapshotResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
		capacity := envelope.Data.Capacity
		require.Equal(t, 70, capacity.Global.Current)
		require.Equal(t, 80, capacity.Global.Limit)
		require.Equal(t, 29, capacity.QueueCurrent)
		require.NotNil(t, capacity.AverageWaitMS)
		require.InDelta(t, 420, *capacity.AverageWaitMS, 0.01)
		require.Equal(t, int64(1000), capacity.WaitSamples)
		require.Equal(t, int64(126), capacity.TodayRateLimited)
		require.True(t, capacity.RateLimitAvailable)
	}
}

func (s *snapshotRollupLoaderStub) LoadSnapshotSeries(_ context.Context, _, _ time.Time, _ observability.Bucket, _ observability.Filter, _ string, _ []int64, _ []string) ([]service.SnapshotSeriesBucket, error) {
	return nil, nil
}

func (s *snapshotHandlerStub) Snapshot(_ context.Context, request service.SnapshotRequest) (service.SnapshotResult, error) {
	s.requests = append(s.requests, request)
	return s.result, s.err
}

func TestObservabilitySnapshotHandlerContractAndETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &snapshotHandlerStub{result: service.SnapshotResult{
		ETag: `"snapshot-etag"`,
		Response: observability.SnapshotResponse{
			Meta: observability.Meta{
				AsOf:   time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC),
				Window: observability.Window1h, Bucket: observability.Bucket5m,
				Timezone: "Asia/Shanghai", Warnings: []string{"request_rollup_lagging"},
			},
			Metrics:  []observability.MetricValue{{Code: "client_failure_rate", Value: 0.125, Unit: "ratio"}},
			TopUsers: []observability.DimensionRow{{Key: "7", Name: "user@example.com · 研发部", Value: 42}},
			FilterOptions: observability.SnapshotFilterOptions{
				Users:    []observability.UserFilterOption{{UserID: 7, Email: "user@example.com", Department: "研发部"}},
				Accounts: []observability.AccountFilterOption{{AccountID: 9, Name: "claude-main", BoundModel: "claude-sonnet-4-5"}},
			},
		},
	}}
	handler := NewOpsHandler(nil)
	handler.observabilitySnapshot = stub
	router := gin.New()
	router.GET("/snapshot", handler.GetObservabilitySnapshot)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/snapshot?window=1h&user_id=7&search=user", nil))
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, `"snapshot-etag"`, first.Header().Get("ETag"))
	require.Contains(t, first.Body.String(), `"email":"user@example.com"`)
	require.Contains(t, first.Body.String(), `"department":"研发部"`)
	require.Contains(t, first.Body.String(), `"bound_model":"claude-sonnet-4-5"`)
	require.NotContains(t, first.Body.String(), "api_key")
	require.NotContains(t, first.Body.String(), "credentials")

	var envelope struct {
		Data observability.SnapshotResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &envelope))
	require.Equal(t, "request_rollup_lagging", envelope.Data.Meta.Warnings[0])
	require.Len(t, stub.requests, 1)
	require.Equal(t, []int64{7}, stub.requests[0].Filter.UserIDs)

	custom := httptest.NewRecorder()
	router.ServeHTTP(custom, httptest.NewRequest(
		http.MethodGet,
		"/snapshot?window=custom&bucket=15m&start_at=2026-11-12T01:00:00Z&end_at=2026-11-12T02:00:00Z",
		nil,
	))
	require.Equal(t, http.StatusOK, custom.Code)
	require.Len(t, stub.requests, 2)
	require.Equal(t, observability.WindowCustom, stub.requests[1].Window)
	require.Equal(t, time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC), stub.requests[1].Start)
	require.Equal(t, time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC), stub.requests[1].End)

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/snapshot?window=1h", nil)
	request.Header.Set("If-None-Match", `"snapshot-etag"`)
	router.ServeHTTP(second, request)
	require.Equal(t, http.StatusNotModified, second.Code)
	require.Empty(t, second.Body.String())
}

func TestObservabilitySnapshotHandlerRejectsInvalidParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOpsHandler(nil)
	handler.observabilitySnapshot = &snapshotHandlerStub{}
	router := gin.New()
	router.GET("/snapshot", handler.GetObservabilitySnapshot)

	for _, target := range []string{
		"/snapshot?window=7d&bucket=5m",
		"/snapshot?window=1h&limit=0",
		"/snapshot?window=1h&user_id=-1",
		"/snapshot?window=custom&start_at=2026-11-12T01:00:00Z",
		"/snapshot?window=custom&start_at=2026-11-12T02:00:00Z&end_at=2026-11-12T01:00:00Z",
		"/snapshot?window=custom&start_at=2026-11-05T01:00:00Z&end_at=2026-11-13T01:00:00Z",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code, target)
	}
}

func TestObservabilitySnapshotServiceFiveSecondCacheAndETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := &snapshotRollupLoaderStub{now: time.Now().UTC()}
	snapshotService := service.NewObservabilitySnapshotService(loader, nil, nil, nil, nil, nil)
	handler := NewOpsHandler(nil)
	handler.observabilitySnapshot = snapshotService
	router := gin.New()
	router.GET("/snapshot", handler.GetObservabilitySnapshot)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/snapshot?window=1h", nil))
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, 2, loader.calls, "snapshot window and local-day rate-limit rollup may each load once")
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/snapshot?window=1h", nil)
	request.Header.Set("If-None-Match", etag)
	started := time.Now()
	router.ServeHTTP(second, request)
	elapsed := time.Since(started)
	require.Equal(t, http.StatusNotModified, second.Code)
	require.Equal(t, 2, loader.calls, "cache hit must not reload rollups")
	require.Less(t, elapsed, 200*time.Millisecond)
}

func TestObservabilitySnapshotCacheSeparatesAbsoluteRanges(t *testing.T) {
	loader := &snapshotRollupLoaderStub{now: time.Now().UTC()}
	snapshotService := service.NewObservabilitySnapshotService(loader, nil, nil, nil, nil, nil)
	now := time.Now().UTC().Truncate(time.Second)
	base := service.SnapshotRequest{
		Window: observability.WindowCustom, Bucket: observability.Bucket15m,
		Filter: observability.Filter{}, OptionLimit: 20,
	}

	first := base
	first.Start, first.End = now.Add(-2*time.Hour), now.Add(-time.Hour)
	firstResult, err := snapshotService.Snapshot(context.Background(), first)
	require.NoError(t, err)
	require.Equal(t, 2, loader.calls)

	second := base
	second.Start, second.End = now.Add(-3*time.Hour), now.Add(-time.Hour)
	secondResult, err := snapshotService.Snapshot(context.Background(), second)
	require.NoError(t, err)
	require.Equal(t, 4, loader.calls, "different absolute ranges must not share a cache entry")
	require.NotEqual(t, firstResult.ETag, secondResult.ETag)
}
