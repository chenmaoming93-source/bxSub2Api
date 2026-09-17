package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDynamicTokenStatisticsRegistryEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDynamicTokenStatisticsHandler(nil, nil, nil)
	router.GET("/dimensions", handler.Dimensions)
	router.GET("/metrics", handler.Metrics)

	for path, expectedCount := range map[string]int{"/dimensions": 6, "/metrics": 1} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		var envelope struct {
			Data map[string][]json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
		for _, items := range envelope.Data {
			require.Len(t, items, expectedCount)
		}
	}
}

type adminQuotaResetterStub struct {
	result tokenstat.QuotaResetResult
	err    error
	input  tokenstat.ResetIdentityDiscoveryRequest
}

func (s *adminQuotaResetterStub) ResetQuotaUsage(_ context.Context, input tokenstat.ResetIdentityDiscoveryRequest) (tokenstat.QuotaResetResult, error) {
	s.input = input
	return s.result, s.err
}

func performAdminQuotaReset(stub *adminQuotaResetterStub, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/quota-usage/reset", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	newDynamicTokenStatisticsHandlerWithResetter(stub).ResetQuotaUsage(context)
	return recorder
}

func TestAdminQuotaResetHandlerContract(t *testing.T) {
	for _, status := range []tokenstat.QuotaResetStatus{tokenstat.QuotaResetStatusReset, tokenstat.QuotaResetStatusPartialReset, tokenstat.QuotaResetStatusNoQuota, tokenstat.QuotaResetStatusNoUsage} {
		stub := &adminQuotaResetterStub{result: tokenstat.QuotaResetResult{Status: status, MatchedQuotaCount: 2, MatchedUsageCount: 3, ResetCount: 1, NoUsageCount: 1, FailedCount: 1}}
		response := performAdminQuotaReset(stub, `{"dimension_values":{"user_id":{"type":"int64","int64":42},"group_id":{"type":"int64","int64":7}},"metric_code":"total_tokens","period_type":"D"}`)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), `"status":"`+string(status)+`"`)
		require.Contains(t, response.Body.String(), `"matched_usage_count":3`)
		require.Len(t, stub.input.DimensionValues, 2)
		require.True(t, stub.input.IncludeDetails)
	}
}

func TestAdminQuotaResetHandlerRejectsUnknownFieldsAndMapsErrors(t *testing.T) {
	unknown := performAdminQuotaReset(&adminQuotaResetterStub{}, `{"dimension_values":{},"metric_code":"total_tokens","period_type":"D","projection_id":1}`)
	require.Equal(t, http.StatusBadRequest, unknown.Code)

	invalid := performAdminQuotaReset(&adminQuotaResetterStub{err: fmt.Errorf("%w: wildcard is not allowed", tokenstat.ErrInvalidQuotaResetRequest)}, `{"dimension_values":{"user_id":{"type":"wildcard"}},"metric_code":"total_tokens","period_type":"D"}`)
	require.Equal(t, http.StatusBadRequest, invalid.Code)

	unavailable := performAdminQuotaReset(&adminQuotaResetterStub{err: tokenstat.ErrTokenQuotaResetUnavailable}, `{"dimension_values":{"user_id":{"type":"int64","int64":1}},"metric_code":"total_tokens","period_type":"D"}`)
	require.Equal(t, http.StatusServiceUnavailable, unavailable.Code)
	require.Contains(t, unavailable.Body.String(), "TOKEN_QUOTA_RESET_UNAVAILABLE")
}

func TestQuotaUpdateRequestDoesNotRequireImmutableCreateFields(t *testing.T) {
	body := `{"name":"edited quota","limit_value":2000,"mode":"ENFORCE"}`

	updateContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	updateContext.Request = httptest.NewRequest(http.MethodPut, "/quotas/1", strings.NewReader(body))
	updateContext.Request.Header.Set("Content-Type", "application/json")
	var update quotaUpdateRequest
	require.NoError(t, updateContext.ShouldBindJSON(&update))

	createContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	createContext.Request = httptest.NewRequest(http.MethodPost, "/quotas", strings.NewReader(body))
	createContext.Request.Header.Set("Content-Type", "application/json")
	var create quotaRequest
	require.Error(t, createContext.ShouldBindJSON(&create))
}
