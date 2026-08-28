package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type externalTokenUsageQuerierStub struct {
	result      service.ExternalTokenUsageResult
	err         error
	dailyResult service.ExternalDailyUsageResult
	dailyErr    error
	filled      service.ExternalDailyUsageResult
	filledErr   error
}

func (s externalTokenUsageQuerierStub) QueryCurrentUsage(context.Context, service.ExternalTokenUsageInput) (service.ExternalTokenUsageResult, error) {
	return s.result, s.err
}

func (s externalTokenUsageQuerierStub) QueryDailyUsage(context.Context, service.ExternalDailyUsageInput) (service.ExternalDailyUsageResult, error) {
	return s.dailyResult, s.dailyErr
}

func (s externalTokenUsageQuerierStub) QueryDailyUsageFilled(context.Context, service.ExternalDailyUsageInput) (service.ExternalDailyUsageResult, error) {
	return s.filled, s.filledErr
}

func performExternalTokenUsage(t *testing.T, stub externalTokenUsageQuerierStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/token-usage/query", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	NewExternalTokenUsageHandlerWithQuerier(stub).Query(c)
	return recorder
}

func TestExternalTokenUsageHandlerContract(t *testing.T) {
	zero := int64(0)
	result := service.ExternalTokenUsageResult{Dimensions: service.ExternalTokenUsageDimensions{UserID: 1, GroupID: 2, APIKeyID: 3, RouteAlias: "gpt-main"}, Metric: "total_tokens", Timezone: "Asia/Shanghai", Day: service.ExternalTokenUsagePeriodResult{DimensionConfigured: true, TotalTokens: &zero}}
	response := performExternalTokenUsage(t, externalTokenUsageQuerierStub{result: result}, `{"username":" u@example.com ","group_name":" public ","api_key":" sk-test-key-1234567890 ","route_alias":" gpt-main "}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"username":"u@example.com"`)
	require.Contains(t, response.Body.String(), `"periods"`)
	require.Contains(t, response.Body.String(), `"enforced_limit":null`)
	require.Contains(t, response.Body.String(), `"user_id":1`)
	// 回显只含脱敏后的 API Key，不得泄露完整明文。
	require.Contains(t, response.Body.String(), `"api_key":"sk-t****7890"`)
	require.NotContains(t, response.Body.String(), "sk-test-key-1234567890")
}

func TestExternalTokenUsageHandlerErrors(t *testing.T) {
	for _, body := range []string{`{}`, `{"username":" ","group_name":"g","api_key":"k","route_alias":"r"}`} {
		require.Equal(t, http.StatusBadRequest, performExternalTokenUsage(t, externalTokenUsageQuerierStub{}, body).Code)
	}
	require.Equal(t, http.StatusNotFound, performExternalTokenUsage(t, externalTokenUsageQuerierStub{err: service.ErrRouteAliasNotFound}, `{"username":"u@example.com","group_name":"g","api_key":"k","route_alias":"r"}`).Code)
	mismatch := performExternalTokenUsage(t, externalTokenUsageQuerierStub{err: service.ErrAPIKeyMismatch}, `{"username":"u@example.com","group_name":"g","api_key":"sk-existing-key-1234567890","route_alias":"r"}`)
	require.Equal(t, http.StatusBadRequest, mismatch.Code)
	require.Contains(t, mismatch.Body.String(), "API_KEY_MISMATCH")
	require.Equal(t, http.StatusServiceUnavailable, performExternalTokenUsage(t, externalTokenUsageQuerierStub{err: service.ErrTokenUsageUnavailable}, `{"username":"u@example.com","group_name":"g","api_key":"k","route_alias":"r"}`).Code)
	require.False(t, errors.Is(service.ErrRouteAliasNotFound, service.ErrTokenUsageUnavailable))
}

func performExternalTokenUsageDaily(t *testing.T, stub externalTokenUsageQuerierStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/token-usage/query/group-api-key/daily", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	NewExternalTokenUsageHandlerWithQuerier(stub).DailyQuery(c)
	return recorder
}

func TestExternalTokenUsageDailyHandlerContract(t *testing.T) {
	enabledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	syncedAt := time.Date(2026, 7, 31, 15, 55, 0, 0, time.UTC)
	result := service.ExternalDailyUsageResult{
		GroupID: 2, APIKeyID: 4, ProjectionID: 7, Metric: "total_tokens", Timezone: "Asia/Shanghai",
		DimensionConfigured: true, ProjectionEnabledAt: &enabledAt, LastSyncedAt: &syncedAt, Complete: true,
		Days: []service.ExternalDailyUsageDay{{Date: "2026-07-01", TotalTokens: 12500}, {Date: "2026-07-03", TotalTokens: 9800}},
	}
	response := performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyResult: result}, `{"group_name":" public ","api_key":" sk-test-key-1234567890 ","start_date":"2026-07-01","end_date":"2026-07-31"}`)
	require.Equal(t, http.StatusOK, response.Code)
	body := response.Body.String()
	require.Contains(t, body, `"group_name":"public"`)
	require.Contains(t, body, `"group_id":2`)
	require.Contains(t, body, `"api_key_id":4`)
	require.Contains(t, body, `"projection_id":7`)
	require.Contains(t, body, `"dimension_configured":true`)
	require.Contains(t, body, `"date":"2026-07-01"`)
	require.Contains(t, body, `"total_tokens":12500`)
	require.Contains(t, body, `"complete":true`)
	// 回显只含脱敏后的 API Key，不得泄露完整明文。
	require.Contains(t, body, `"api_key":"sk-t****7890"`)
	require.NotContains(t, body, "sk-test-key-1234567890")
}

func TestExternalTokenUsageDailyHandlerThreeStates(t *testing.T) {
	unconfigured := performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyResult: service.ExternalDailyUsageResult{
		GroupID: 2, APIKeyID: 4, Metric: "total_tokens", Timezone: "Asia/Shanghai",
		DimensionConfigured: false, Message: "统计维度未配置", Days: []service.ExternalDailyUsageDay{},
	}}, `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-31"}`)
	require.Equal(t, http.StatusOK, unconfigured.Code)
	require.Contains(t, unconfigured.Body.String(), `"dimension_configured":false`)
	require.Contains(t, unconfigured.Body.String(), `统计维度未配置`)
	require.Contains(t, unconfigured.Body.String(), `"days":[]`)

	empty := performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyResult: service.ExternalDailyUsageResult{
		GroupID: 2, APIKeyID: 4, ProjectionID: 7, Metric: "total_tokens", Timezone: "Asia/Shanghai",
		DimensionConfigured: true, Days: []service.ExternalDailyUsageDay{},
	}}, `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-31"}`)
	require.Equal(t, http.StatusOK, empty.Code)
	require.Contains(t, empty.Body.String(), `"dimension_configured":true`)
	require.Contains(t, empty.Body.String(), `"days":[]`)
}

func TestExternalTokenUsageDailyHandlerErrors(t *testing.T) {
	valid := `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-31"}`
	for _, body := range []string{
		`{}`,
		`{"group_name":" ","api_key":"k","start_date":"2026-07-01","end_date":"2026-07-02"}`,
		`{"group_name":"g","api_key":"k","start_date":"not-a-date","end_date":"2026-07-02"}`,
		`{"group_name":"g","api_key":"k","start_date":"2026-07-02","end_date":"2026-07-01"}`,
		`{"group_name":"g","api_key":"k","start_date":"2026-01-01","end_date":"2027-01-03"}`,
	} {
		require.Equal(t, http.StatusBadRequest, performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{}, body).Code, "body=%s", body)
	}
	require.Equal(t, http.StatusNotFound, performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyErr: service.ErrGroupNotFound}, valid).Code)
	require.Equal(t, http.StatusNotFound, performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyErr: service.ErrAPIKeyNotFound}, valid).Code)
	mismatch := performExternalTokenUsageDaily(t, externalTokenUsageQuerierStub{dailyErr: service.ErrAPIKeyMismatch}, valid)
	require.Equal(t, http.StatusBadRequest, mismatch.Code)
	require.Contains(t, mismatch.Body.String(), "API_KEY_MISMATCH")
}

func performExternalTokenUsageDailyCSV(t *testing.T, stub externalTokenUsageQuerierStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/token-usage/query/group-api-key/daily/csv", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	NewExternalTokenUsageHandlerWithQuerier(stub).DailyQueryCSV(c)
	return recorder
}

func TestExternalTokenUsageDailyCSVHandlerContract(t *testing.T) {
	result := service.ExternalDailyUsageResult{
		GroupID: 2, APIKeyID: 4, ProjectionID: 7, Metric: "total_tokens", Timezone: "Asia/Shanghai",
		DimensionConfigured: true, Complete: true,
		Days: []service.ExternalDailyUsageDay{{Date: "2026-07-01", TotalTokens: 12500}, {Date: "2026-07-02", TotalTokens: 0}},
	}
	response := performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{filled: result}, `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-02"}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "text/csv; charset=utf-8", response.Header().Get("Content-Type"))
	require.Contains(t, response.Header().Get("Content-Disposition"), "attachment")
	body := response.Body.String()
	require.Contains(t, body, "date,total_tokens")
	require.Contains(t, body, "2026-07-01,12500")
	require.Contains(t, body, "2026-07-02,0")
	// CSV 不得泄露 API Key 明文。
	require.NotContains(t, body, "sk-test-key-1234567890")
}

func TestExternalTokenUsageDailyCSVHandlerUnconfigured(t *testing.T) {
	response := performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{filled: service.ExternalDailyUsageResult{
		GroupID: 2, APIKeyID: 4, Metric: "total_tokens", Timezone: "Asia/Shanghai",
		DimensionConfigured: false, Message: "统计维度未配置", Days: []service.ExternalDailyUsageDay{},
	}}, `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-31"}`)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), "STATISTICS_NOT_CONFIGURED")
}

func TestExternalTokenUsageDailyCSVHandlerErrors(t *testing.T) {
	valid := `{"group_name":"public","api_key":"sk-test-key-1234567890","start_date":"2026-07-01","end_date":"2026-07-31"}`
	for _, body := range []string{
		`{}`,
		`{"group_name":"g","api_key":"k","start_date":"not-a-date","end_date":"2026-07-02"}`,
		`{"group_name":"g","api_key":"k","start_date":"2026-07-02","end_date":"2026-07-01"}`,
	} {
		require.Equal(t, http.StatusBadRequest, performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{}, body).Code, "body=%s", body)
	}
	require.Equal(t, http.StatusNotFound, performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{filledErr: service.ErrGroupNotFound}, valid).Code)
	require.Equal(t, http.StatusNotFound, performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{filledErr: service.ErrAPIKeyNotFound}, valid).Code)
	require.Equal(t, http.StatusBadRequest, performExternalTokenUsageDailyCSV(t, externalTokenUsageQuerierStub{filledErr: service.ErrAPIKeyMismatch}, valid).Code)
}
