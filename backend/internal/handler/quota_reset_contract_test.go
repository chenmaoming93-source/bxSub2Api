package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type sharedResetRuntime struct{}

func (sharedResetRuntime) Enabled() bool { return true }

type sharedResetDiscovery struct{ calls int }

func (s *sharedResetDiscovery) Discover(context.Context, tokenstat.ResetIdentityDiscoveryRequest) (tokenstat.IdentityDiscoveryResult, error) {
	s.calls++
	return tokenstat.IdentityDiscoveryResult{Status: tokenstat.IdentityDiscoveryFound, MatchedQuotaCount: 2, Identities: []tokenstat.StatisticIdentity{{ProjectionID: 1}}}, nil
}

type sharedResetResolver struct{}

func (sharedResetResolver) ResolveQuotaResetDimensions(context.Context, service.ExternalTokenQuotaResetDimensions) (map[tokenstat.DimensionCode]tokenstat.DimensionValue, error) {
	return map[tokenstat.DimensionCode]tokenstat.DimensionValue{tokenstat.DimensionUserID: tokenstat.Int64Value(42)}, nil
}

type sharedResetWriter struct{ calls int }

func (s *sharedResetWriter) SnapshotBatch(context.Context, []tokenstat.StatisticIdentity) ([]tokenstat.QuotaResetEntryResult, error) {
	s.calls++
	return []tokenstat.QuotaResetEntryResult{{Status: tokenstat.QuotaResetEntryReset}}, nil
}

func TestAdminAndIntegrationQuotaResetUseSameServiceContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	discovery := &sharedResetDiscovery{}
	writer := &sharedResetWriter{}
	resetService := tokenstat.NewQuotaResetService(discovery, writer, sharedResetRuntime{}, 100)
	admin := adminhandler.NewDynamicTokenStatisticsHandler(nil, nil, resetService)
	external := handler.NewExternalTokenUsageHandlerWithServices(nil, resetService, sharedResetResolver{})
	adminBody := `{"dimension_values":{"user_id":{"type":"int64","int64":42}},"metric_code":"total_tokens","period_type":"D"}`
	externalBody := `{"dimension_values":{"username":"u@example.com"},"metric_code":"total_tokens","period_type":"D"}`

	invoke := func(path, body string, action gin.HandlerFunc) map[string]any {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		action(ctx)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		var envelope response.Response
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
		encoded, err := json.Marshal(envelope.Data)
		require.NoError(t, err)
		var data map[string]any
		require.NoError(t, json.Unmarshal(encoded, &data))
		return data
	}

	adminData := invoke("/api/v1/admin/token-statistics/quota-usage/reset", adminBody, admin.ResetQuotaUsage)
	externalData := invoke("/api/v1/integrations/token-usage/reset", externalBody, external.ResetQuotaUsage)
	for _, field := range []string{"status", "matched_quota_count", "matched_usage_count", "reset_count", "no_usage_count", "failed_count"} {
		require.Equal(t, adminData[field], externalData[field], field)
	}
	require.Contains(t, adminData, "matched_entries", "admin response includes troubleshooting details")
	require.NotContains(t, externalData, "matched_entries", "integration response keeps internal identities private")
	require.Equal(t, "RESET", adminData["status"])
	require.Equal(t, float64(1), adminData["reset_count"])
	require.Equal(t, 2, discovery.calls)
	require.Equal(t, 2, writer.calls)
}
