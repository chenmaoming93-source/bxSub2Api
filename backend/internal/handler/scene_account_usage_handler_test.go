package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type sceneAccountUsageQuerierStub struct {
	input  service.SceneAccountDailyUsageInput
	result *service.SceneAccountDailyUsageResult
	err    error
}

func (s *sceneAccountUsageQuerierStub) QuerySceneAccountDailyUsage(_ context.Context, input service.SceneAccountDailyUsageInput) (*service.SceneAccountDailyUsageResult, error) {
	s.input = input
	return s.result, s.err
}

func TestExternalSceneAccountDailyUsageHandlerBindsJSONAndReturnsSharedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &sceneAccountUsageQuerierStub{result: &service.SceneAccountDailyUsageResult{
		StartDate: "2026-08-01", EndDate: "2026-08-01", Timezone: "Asia/Shanghai", Days: []service.SceneAccountDailyUsageDay{},
	}}
	router := gin.New()
	router.POST("/api/v1/integrations/token-usage/query/scene-account/daily", NewExternalSceneAccountDailyUsageHandlerWithQuerier(stub).QueryDaily)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/token-usage/query/scene-account/daily", strings.NewReader(`{"start_date":"2026-08-01","end_date":"2026-08-02","group_name":"technical-a"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SceneAccountDailyUsageInput{StartDate: "2026-08-01", EndDate: "2026-08-02", GroupName: "technical-a"}, stub.input)
	require.Contains(t, recorder.Body.String(), `"timezone":"Asia/Shanghai"`)
}

func TestAdminSceneAccountDailyUsageHandlerBindsQueryAndMapsConfigError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &sceneAccountUsageQuerierStub{err: service.ErrSceneUsageStatisticsNotConfigured}
	router := gin.New()
	router.GET("/api/v1/admin/usage/scene-account/daily", adminhandler.NewSceneAccountDailyUsageHandlerWithQuerier(stub).QueryDaily)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/scene-account/daily?start_date=2026-08-01&end_date=2026-08-02&group_name=technical-a", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, service.SceneAccountDailyUsageInput{StartDate: "2026-08-01", EndDate: "2026-08-02", GroupName: "technical-a"}, stub.input)
	require.Contains(t, recorder.Body.String(), "SCENE_USAGE_STATISTICS_NOT_CONFIGURED")
}

func TestSceneAccountDailyUsageHandlersRejectMalformedExternalJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &sceneAccountUsageQuerierStub{err: errors.New("should not be called")}
	router := gin.New()
	router.POST("/query", NewExternalSceneAccountDailyUsageHandlerWithQuerier(stub).QueryDaily)

	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(`{"start_date":`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, stub.input.StartDate)
}
