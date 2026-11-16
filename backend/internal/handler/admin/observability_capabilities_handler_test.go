package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestObservabilityCapabilitiesContractAndETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOpsHandler(nil)
	router := gin.New()
	router.GET("/api/v1/admin/observability/capabilities", h.GetObservabilityCapabilities)

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/observability/capabilities", nil)
	router.ServeHTTP(first, req)
	require.Equal(t, http.StatusOK, first.Code)
	require.NotEmpty(t, first.Header().Get("ETag"))

	var envelope struct {
		Code int                              `json:"code"`
		Data observability.ContractDescriptor `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, observability.Contract(), envelope.Data)

	second := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/observability/capabilities", nil)
	req.Header.Set("If-None-Match", first.Header().Get("ETag"))
	router.ServeHTTP(second, req)
	require.Equal(t, http.StatusNotModified, second.Code)
	require.Empty(t, second.Body.String())
}
