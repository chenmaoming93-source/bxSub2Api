package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestProvisioningAuthBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const token = "secret_0123456789abcdef0123456789abcdef"

	run := func(cfg config.ExternalAPIKeyProvisioningConfig, header, target string, body []byte) *httptest.ResponseRecorder {
		router := gin.New()
		router.POST("/ensure", ExternalProvisioningAuth(cfg), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		router.ServeHTTP(recorder, request)
		return recorder
	}

	t.Run("correct bearer passes", func(t *testing.T) {
		response := run(config.ExternalAPIKeyProvisioningConfig{Enabled: true, AccessToken: token}, "Bearer "+token, "/ensure", nil)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("disabled is hidden", func(t *testing.T) {
		response := run(config.ExternalAPIKeyProvisioningConfig{AccessToken: token}, "Bearer "+token, "/ensure", nil)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), token) {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	})

	for _, tc := range []struct {
		name, header, target string
		body                 []byte
	}{
		{name: "missing", target: "/ensure"},
		{name: "wrong", header: "Bearer wrong-token", target: "/ensure"},
		{name: "query rejected", target: "/ensure?access_token=" + token},
		{name: "body rejected", target: "/ensure", body: []byte(`{"access_token":"` + token + `"}`)},
		{name: "custom scheme rejected", header: "Token " + token, target: "/ensure"},
		{name: "extra field rejected", header: "Bearer " + token + " extra", target: "/ensure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := run(config.ExternalAPIKeyProvisioningConfig{Enabled: true, AccessToken: token}, tc.header, tc.target, tc.body)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), token) || strings.Contains(response.Body.String(), "secret_") {
				t.Fatalf("response leaked credential: %s", response.Body.String())
			}
		})
	}
}
