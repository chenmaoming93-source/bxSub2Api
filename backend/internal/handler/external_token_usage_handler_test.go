package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type externalTokenUsageQuerierStub struct {
	result service.ExternalTokenUsageResult
	err    error
}

func (s externalTokenUsageQuerierStub) QueryCurrentUsage(context.Context, service.ExternalTokenUsageInput) (service.ExternalTokenUsageResult, error) {
	return s.result, s.err
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
	require.Equal(t, http.StatusServiceUnavailable, performExternalTokenUsage(t, externalTokenUsageQuerierStub{err: service.ErrTokenUsageUnavailable}, `{"username":"u@example.com","group_name":"g","api_key":"k","route_alias":"r"}`).Code)
	require.False(t, errors.Is(service.ErrRouteAliasNotFound, service.ErrTokenUsageUnavailable))
}
