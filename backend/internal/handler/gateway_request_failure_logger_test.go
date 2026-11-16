package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func captureGatewayFailureEvents(t *testing.T) *[]logger.RequestFailureEvent {
	t.Helper()
	events := make([]logger.RequestFailureEvent, 0, 4)
	previous := writeGatewayFailureRequest
	writeGatewayFailureRequest = func(event logger.RequestFailureEvent) {
		events = append(events, event)
	}
	t.Cleanup(func() { writeGatewayFailureRequest = previous })
	return &events
}

func serveGatewayFailureTestRequest(t *testing.T, method, path, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OpsErrorLoggerMiddleware(nil))
	router.Handle(method, path, func(c *gin.Context) {
		_, _ = io.ReadAll(c.Request.Body)
		c.Status(status)
	})

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestOpsErrorLoggerLogsOnlyNon429FailuresAcrossProtocols(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	tests := []struct {
		path     string
		endpoint string
	}{
		{path: "/v1/messages", endpoint: "/v1/messages"},
		{path: "/v1/chat/completions", endpoint: "/v1/chat/completions"},
		{path: "/v1/responses", endpoint: "/v1/responses"},
		{path: "/v1beta/models/gemini-2.5-pro:generateContent", endpoint: "/v1beta/models"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			before := len(*events)
			response := serveGatewayFailureTestRequest(t, http.MethodPost, tt.path, `{"model":"test","messages":[]}`, http.StatusBadRequest)
			require.Equal(t, http.StatusBadRequest, response.Code)
			require.Len(t, *events, before+1)
			event := (*events)[len(*events)-1]
			require.Equal(t, tt.endpoint, event.Endpoint)
			require.Equal(t, http.StatusBadRequest, event.Status)
			require.Contains(t, event.RequestBody, `"model":"test"`)
		})
	}
}

func TestOpsErrorLoggerExcludesSuccessAnd429(t *testing.T) {
	events := captureGatewayFailureEvents(t)

	serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/messages", `{"model":"ok"}`, http.StatusOK)
	serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/messages", `{"model":"limited"}`, http.StatusTooManyRequests)
	require.Empty(t, *events)
}

func TestOpsErrorLoggerCoversClientAndServerFailureStatuses(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		529,
	}
	for _, status := range statuses {
		serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/messages", `{"model":"status"}`, status)
	}
	require.Len(t, *events, len(statuses))
	for i, status := range statuses {
		require.Equal(t, status, (*events)[i].Status)
	}
}

func TestOpsErrorLoggerCapturesBodyBeforeAuthRejectsWithoutReading(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OpsErrorLoggerMiddleware(nil))
	router.POST("/v1/messages", func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"auth-test"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Len(t, *events, 1)
	require.Contains(t, (*events)[0].RequestBody, `"model":"auth-test"`)
}

func TestGatewayFailureBodyRecursivelyRedactsSecretsAndPreservesCounts(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	body := `{
		"model":"test",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":256,
		"api_key":"sk-live",
		"nested":{"access_token":"access","password":"pw","client_secret":"cs"},
		"authorization":"Bearer abc"
	}`
	serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/responses", body, http.StatusInternalServerError)
	require.Len(t, *events, 1)

	var logged map[string]any
	require.NoError(t, json.Unmarshal([]byte((*events)[0].RequestBody), &logged))
	require.Equal(t, float64(256), logged["max_tokens"])
	require.Equal(t, "***REDACTED***", logged["api_key"])
	require.Equal(t, "***REDACTED***", logged["authorization"])
	nested := logged["nested"].(map[string]any)
	require.Equal(t, "***REDACTED***", nested["access_token"])
	require.Equal(t, "***REDACTED***", nested["password"])
	require.Equal(t, "***REDACTED***", nested["client_secret"])
}

func TestGatewayFailureBodyTruncatesAt64KiBAndReportsOriginalSize(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	largeText := strings.Repeat("a", 70*1024)
	body := `{"model":"test","messages":"` + largeText + `"}`
	serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/chat/completions", body, http.StatusServiceUnavailable)

	require.Len(t, *events, 1)
	event := (*events)[0]
	require.True(t, event.Truncated)
	require.Equal(t, int64(len(body)), event.RequestBodySize)
	require.LessOrEqual(t, len(event.RequestBody), gatewayRequestBodyCaptureLimit+32)
	require.Equal(t, "truncated_json_redacted_prefix", event.BodyFormat)
}

func TestGatewayFailureTruncatedSecretIsNotLeaked(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	secret := strings.Repeat("sensitive", 9000)
	body := `{"api_key":"` + secret
	serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/messages", body, http.StatusUnauthorized)

	require.Len(t, *events, 1)
	require.NotContains(t, (*events)[0].RequestBody, "sensitive")
	require.Contains(t, (*events)[0].RequestBody, "***REDACTED***")
}

func TestWebSocketFailureLogsMetadataWithoutInventingBody(t *testing.T) {
	events := captureGatewayFailureEvents(t)
	response := serveGatewayFailureTestRequest(t, http.MethodGet, "/v1/responses", "", http.StatusForbidden)
	require.Equal(t, http.StatusForbidden, response.Code)
	require.Len(t, *events, 1)
	require.Empty(t, (*events)[0].RequestBody)
	require.Equal(t, "request_body_unavailable", (*events)[0].BodyOmitted)
}

func TestGatewayFailureLoggingPanicDoesNotChangeResponse(t *testing.T) {
	previous := writeGatewayFailureRequest
	writeGatewayFailureRequest = func(logger.RequestFailureEvent) {
		panic("log sink unavailable")
	}
	t.Cleanup(func() { writeGatewayFailureRequest = previous })

	response := serveGatewayFailureTestRequest(t, http.MethodPost, "/v1/messages", `{"model":"test"}`, http.StatusBadGateway)
	require.Equal(t, http.StatusBadGateway, response.Code)
}
