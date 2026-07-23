package routes

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type integrationProvisioningServiceStub struct{}

func (integrationProvisioningServiceStub) EnsurePlatformKey(_ context.Context, input service.EnsurePlatformKeyInput) (*service.EnsurePlatformKeyResult, error) {
	groupID := int64(2)
	return &service.EnsurePlatformKeyResult{
		User:   &service.User{ID: 1, Email: input.User, Username: "test"},
		APIKey: &service.APIKey{Key: "sk-test", GroupID: &groupID},
		Group:  &service.Group{ID: groupID, Name: input.GroupName},
	}, nil
}

func (integrationProvisioningServiceStub) ListGroupModelRoutes(_ context.Context, _ service.ListGroupModelRoutesInput) ([]service.GroupModelRouteProjection, error) {
	return []service.GroupModelRouteProjection{}, nil
}

func integrationRouter(cfg config.ExternalAPIKeyProvisioningConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterIntegrationRoutes(
		v1,
		handler.NewExternalProvisioningHandler(integrationProvisioningServiceStub{}),
		middleware.ExternalProvisioningAuth(cfg),
		middleware.NewProvisioningHardening(nil, nil).Middleware(),
	)
	return router
}

func performIntegrationRequest(router http.Handler, path, token, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestIntegrationRoutes_GroupModelRoutesAuthContract(t *testing.T) {
	const token = "secret_0123456789abcdef0123456789abcdef"
	path := "/api/v1/integrations/model-routes/list"
	body := `{"group_name":"public"}`

	for _, auth := range []string{"", "Token " + token, "Bearer wrong"} {
		response := performIntegrationRequest(integrationRouter(config.ExternalAPIKeyProvisioningConfig{Enabled: true, AccessToken: token}), path, auth, body)
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "INVALID_ACCESS_TOKEN") {
			t.Fatalf("auth %q: status=%d body=%s", auth, response.Code, response.Body.String())
		}
	}

	hidden := performIntegrationRequest(integrationRouter(config.ExternalAPIKeyProvisioningConfig{Enabled: false, AccessToken: token}), path, "Bearer "+token, body)
	if hidden.Code != http.StatusNotFound || !strings.Contains(hidden.Body.String(), "NOT_FOUND") {
		t.Fatalf("disabled: status=%d body=%s", hidden.Code, hidden.Body.String())
	}

	success := performIntegrationRequest(integrationRouter(config.ExternalAPIKeyProvisioningConfig{Enabled: true, AccessToken: token}), path, "Bearer "+token, body)
	if success.Code != http.StatusOK || !strings.Contains(success.Body.String(), `"routes":[]`) {
		t.Fatalf("success: status=%d body=%s", success.Code, success.Body.String())
	}
}

func TestIntegrationRoutes_GetOrCreateAuthRegression(t *testing.T) {
	const token = "secret_0123456789abcdef0123456789abcdef"
	router := integrationRouter(config.ExternalAPIKeyProvisioningConfig{Enabled: true, AccessToken: token})
	body := `{"user":"u@example.com","platform":"openai","group_name":"public"}`

	unauthorized := performIntegrationRequest(router, "/api/v1/integrations/api-keys/getOrCreate", "", body)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	authorized := performIntegrationRequest(router, "/api/v1/integrations/api-keys/getOrCreate", "Bearer "+token, body)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status=%d body=%s", authorized.Code, authorized.Body.String())
	}
}
