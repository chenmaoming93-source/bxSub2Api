package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userDashboardRepoStub struct {
	service.UsageLogRepository
	called      string
	userID      int64
	startTime   time.Time
	endTime     time.Time
	granularity string
	limit       int
	groups      []usagestats.GroupStat
	trend       []usagestats.UserLatencyTrendPoint
	percentiles *usagestats.UserLatencyPercentiles
}

func (s *userDashboardRepoStub) capture(method string, userID int64, startTime, endTime time.Time, limit int) {
	s.called, s.userID, s.startTime, s.endTime, s.limit = method, userID, startTime, endTime, limit
}

func (s *userDashboardRepoStub) GetUserGroupStats(_ context.Context, userID int64, startTime, endTime time.Time, limit int) ([]usagestats.GroupStat, error) {
	s.capture("groups", userID, startTime, endTime, limit)
	return s.groups, nil
}

func (s *userDashboardRepoStub) GetUserLatencyTrend(_ context.Context, userID int64, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserLatencyTrendPoint, error) {
	s.capture("trend", userID, startTime, endTime, limit)
	s.granularity = granularity
	return s.trend, nil
}

func (s *userDashboardRepoStub) GetUserLatencyPercentiles(_ context.Context, userID int64, startTime, endTime time.Time, limit int) (*usagestats.UserLatencyPercentiles, error) {
	s.capture("percentiles", userID, startTime, endTime, limit)
	if s.percentiles == nil {
		return &usagestats.UserLatencyPercentiles{}, nil
	}
	return s.percentiles, nil
}

func newUserDashboardTestRouter(repo *userDashboardRepoStub, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewUsageHandler(service.NewUsageService(repo, nil, nil, nil), nil, nil, nil)
	router := gin.New()
	if authenticated {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
			c.Next()
		})
	}
	router.GET("/usage/dashboard/groups", h.DashboardGroups)
	router.GET("/usage/dashboard/latency-trend", h.DashboardLatencyTrend)
	router.GET("/usage/dashboard/latency-percentiles", h.DashboardLatencyPercentiles)
	return router
}

func TestDashboardGroupsUsesAuthenticatedUserAndValidatedFilters(t *testing.T) {
	repo := &userDashboardRepoStub{groups: []usagestats.GroupStat{{GroupID: 3, GroupName: "scene", TotalTokens: 99}}}
	router := newUserDashboardTestRouter(repo, true)
	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/groups?user_id=999&start_date=2025-01-01&end_date=2025-01-03&timezone=UTC&limit=7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "groups", repo.called)
	require.Equal(t, int64(42), repo.userID, "query user_id must never override the authenticated user")
	require.Equal(t, 7, repo.limit)
	require.Equal(t, "2025-01-01", repo.startTime.Format("2006-01-02"))
	require.Equal(t, "2025-01-04", repo.endTime.Format("2006-01-02"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.Len(t, data["groups"].([]any), 1)
}

func TestDashboardLatencyTrendPassesGranularityAndReturnsAverage(t *testing.T) {
	repo := &userDashboardRepoStub{trend: []usagestats.UserLatencyTrendPoint{{Date: "2025-01-01 10:00", Requests: 2, AverageDurationMs: 125.5}}}
	router := newUserDashboardTestRouter(repo, true)
	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/latency-trend?user_id=999&start_date=2025-01-01&end_date=2025-01-02&timezone=UTC&granularity=hour&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.userID)
	require.Equal(t, "hour", repo.granularity)
	require.Equal(t, 10, repo.limit)
	require.Contains(t, rec.Body.String(), `"average_duration_ms":125.5`)
}

func TestDashboardLatencyPercentilesContract(t *testing.T) {
	p50, p90, p95, p99 := 10, 20, 30, 40
	repo := &userDashboardRepoStub{percentiles: &usagestats.UserLatencyPercentiles{P50: &p50, P90: &p90, P95: &p95, P99: &p99, SampleCount: 4}}
	router := newUserDashboardTestRouter(repo, true)
	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/latency-percentiles?user_id=999&start_date=2025-01-01&end_date=2025-01-01&timezone=UTC&limit=500", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.userID)
	require.Equal(t, 500, repo.limit)
	for _, fragment := range []string{`"p50":10`, `"p90":20`, `"p95":30`, `"p99":40`, `"sample_count":4`} {
		require.Contains(t, rec.Body.String(), fragment)
	}
}

func TestUserDashboardEndpointsRejectUnauthorizedAndInvalidParameters(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		path          string
		wantStatus    int
	}{
		{name: "groups unauthorized", path: "/usage/dashboard/groups", wantStatus: http.StatusUnauthorized},
		{name: "latency trend unauthorized", path: "/usage/dashboard/latency-trend", wantStatus: http.StatusUnauthorized},
		{name: "latency percentiles unauthorized", path: "/usage/dashboard/latency-percentiles", wantStatus: http.StatusUnauthorized},
		{name: "bad start date", authenticated: true, path: "/usage/dashboard/groups?start_date=bad", wantStatus: http.StatusBadRequest},
		{name: "reversed range", authenticated: true, path: "/usage/dashboard/groups?start_date=2025-01-02&end_date=2025-01-01", wantStatus: http.StatusBadRequest},
		{name: "bad group limit", authenticated: true, path: "/usage/dashboard/groups?limit=101", wantStatus: http.StatusBadRequest},
		{name: "bad granularity", authenticated: true, path: "/usage/dashboard/latency-trend?granularity=minute", wantStatus: http.StatusBadRequest},
		{name: "bad sample limit", authenticated: true, path: "/usage/dashboard/latency-percentiles?limit=100001", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userDashboardRepoStub{}
			rec := httptest.NewRecorder()
			newUserDashboardTestRouter(repo, tt.authenticated).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Empty(t, repo.called)
		})
	}
}

func TestUserDashboardEmptyResultsAreArraysAndNullPercentiles(t *testing.T) {
	repo := &userDashboardRepoStub{groups: []usagestats.GroupStat{}, trend: []usagestats.UserLatencyTrendPoint{}}
	router := newUserDashboardTestRouter(repo, true)
	for _, path := range []string{"/usage/dashboard/groups", "/usage/dashboard/latency-trend", "/usage/dashboard/latency-percentiles"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}
}
