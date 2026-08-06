package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDynamicTokenStatisticsRegistryEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDynamicTokenStatisticsHandler(nil, nil)
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
