package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type externalProvisioningServiceStub struct {
	result      *service.EnsurePlatformKeyResult
	err         error
	input       service.EnsurePlatformKeyInput
	routes      []service.GroupModelRouteProjection
	routesErr   error
	routesInput service.ListGroupModelRoutesInput
}

func (s *externalProvisioningServiceStub) ListGroupModelRoutes(_ context.Context, input service.ListGroupModelRoutesInput) ([]service.GroupModelRouteProjection, error) {
	s.routesInput = input
	return s.routes, s.routesErr
}

func (s *externalProvisioningServiceStub) EnsurePlatformKey(_ context.Context, input service.EnsurePlatformKeyInput) (*service.EnsurePlatformKeyResult, error) {
	s.input = input
	return s.result, s.err
}

func performEnsureAPIKey(t *testing.T, svc *externalProvisioningServiceStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/api-keys/getOrCreate", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	NewExternalProvisioningHandler(svc).EnsureAPIKey(ctx)
	return recorder
}

func TestExternalProvisioningHandlerRequiresGroupName(t *testing.T) {
	for _, body := range []string{
		`{"user":"u@example.com","platform":"openai"}`,
		`{"user":"u@example.com","platform":"openai","group_name":"   "}`,
	} {
		svc := &externalProvisioningServiceStub{}
		res := performEnsureAPIKey(t, svc, body)
		if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "INVALID_REQUEST") {
			t.Fatalf("expected INVALID_REQUEST, status=%d body=%s", res.Code, res.Body.String())
		}
	}
}

func TestExternalProvisioningHandlerRejectsInvalidRequestFields(t *testing.T) {
	for _, body := range []string{
		`{"user":"   ","platform":"openai","group_name":"target"}`,
		`{"user":"u@example.com","platform":"   ","group_name":"target"}`,
		`{"user":"u@example.com","platform":"open-ai","group_name":"target"}`,
	} {
		svc := &externalProvisioningServiceStub{}
		res := performEnsureAPIKey(t, svc, body)
		if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "INVALID_REQUEST") {
			t.Fatalf("expected INVALID_REQUEST, status=%d body=%s", res.Code, res.Body.String())
		}
	}
}

func TestExternalProvisioningHandlerSuccessContract(t *testing.T) {
	for _, tc := range []struct {
		name       string
		keyCreated bool
		wantStatus int
	}{
		{name: "created", keyCreated: true, wantStatus: http.StatusCreated},
		{name: "existing", keyCreated: false, wantStatus: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &externalProvisioningServiceStub{result: &service.EnsurePlatformKeyResult{
				User:   &service.User{ID: 7, Email: "u@example.com", Username: "user"},
				APIKey: &service.APIKey{Key: "sk-secret"}, Group: &service.Group{ID: 9, Name: "target"}, KeyCreated: tc.keyCreated,
			}}
			res := performEnsureAPIKey(t, svc, `{"user":"u@example.com","platform":"openai","group_name":" target "}`)
			if res.Code != tc.wantStatus || res.Header().Get("Cache-Control") != "no-store" || res.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("unexpected response status=%d headers=%v", res.Code, res.Header())
			}
			var envelope struct {
				Data EnsureAPIKeyResponse `json:"data"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Data.GroupID != 9 || envelope.Data.GroupName != "target" || svc.input.GroupName != "target" {
				t.Fatalf("unexpected group contract: response=%+v input=%+v", envelope.Data, svc.input)
			}
		})
	}
}

func TestExternalProvisioningHandlerAllowsUppercasePlatform(t *testing.T) {
	svc := &externalProvisioningServiceStub{result: &service.EnsurePlatformKeyResult{
		User:   &service.User{ID: 7, Email: "u@example.com", Username: "user"},
		APIKey: &service.APIKey{Key: "sk-secret"}, Group: &service.Group{ID: 9, Name: "target"},
	}}
	res := performEnsureAPIKey(t, svc, `{"user":"u@example.com","platform":"Open_AI","group_name":"target"}`)
	if res.Code != http.StatusOK || svc.input.Platform != "Open_AI" {
		t.Fatalf("unexpected response status=%d input=%+v body=%s", res.Code, svc.input, res.Body.String())
	}
}

func TestExternalProvisioningHandlerMapsGroupErrors(t *testing.T) {
	tests := []struct {
		err    error
		status int
		reason string
	}{
		{service.ErrGroupNotFound, http.StatusNotFound, "GROUP_NOT_FOUND"},
		{service.ErrProvisioningGroupInactive, http.StatusConflict, "GROUP_INACTIVE"},
		{service.ErrProvisioningSubscriptionGroup, http.StatusBadRequest, "SUBSCRIPTION_GROUP_NOT_SUPPORTED"},
		{service.ErrProvisioningGroupNotAllowed, http.StatusForbidden, "GROUP_NOT_ALLOWED"},
	}
	for _, tc := range tests {
		svc := &externalProvisioningServiceStub{err: errors.Join(tc.err)}
		res := performEnsureAPIKey(t, svc, `{"user":"u@example.com","platform":"openai","group_name":"target"}`)
		if res.Code != tc.status || !strings.Contains(res.Body.String(), tc.reason) {
			t.Fatalf("error %v: status=%d body=%s", tc.err, res.Code, res.Body.String())
		}
	}
}

func performListGroupModelRoutes(t *testing.T, svc *externalProvisioningServiceStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/model-routes/list", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	NewExternalProvisioningHandler(svc).ListGroupModelRoutes(ctx)
	return recorder
}

func TestExternalProvisioningHandler_ListGroupModelRoutesSuccess(t *testing.T) {
	svc := &externalProvisioningServiceStub{routes: []service.GroupModelRouteProjection{
		{RouteAlias: "coding", UpstreamModels: []string{"model-a", "model-b"}},
	}}
	res := performListGroupModelRoutes(t, svc, `{"group_name":" target "}`)
	if res.Code != http.StatusOK || svc.routesInput.GroupName != "target" {
		t.Fatalf("status=%d input=%+v body=%s", res.Code, svc.routesInput, res.Body.String())
	}
	var envelope struct {
		Code int                          `json:"code"`
		Data ListGroupModelRoutesResponse `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.GroupName != "target" || len(envelope.Data.Routes) != 1 || envelope.Data.Routes[0].RouteAlias != "coding" {
		t.Fatalf("unexpected response: %+v", envelope)
	}
	body := res.Body.String()
	for _, forbidden := range []string{"account_ids", "priority", "daily_token_limit", "token"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
}

func TestExternalProvisioningHandler_ListGroupModelRoutesEmpty(t *testing.T) {
	res := performListGroupModelRoutes(t, &externalProvisioningServiceStub{routes: []service.GroupModelRouteProjection{}}, `{"group_name":"target"}`)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"routes":[]`) {
		t.Fatalf("expected empty routes array, status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestExternalProvisioningHandler_ListGroupModelRoutesErrors(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"group_name":"   "}`} {
		res := performListGroupModelRoutes(t, &externalProvisioningServiceStub{}, body)
		if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "INVALID_REQUEST") {
			t.Fatalf("body %q: status=%d response=%s", body, res.Code, res.Body.String())
		}
	}

	notFound := performListGroupModelRoutes(t, &externalProvisioningServiceStub{routesErr: service.ErrGroupNotFound}, `{"group_name":"missing"}`)
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), "GROUP_NOT_FOUND") {
		t.Fatalf("not found: status=%d body=%s", notFound.Code, notFound.Body.String())
	}
	internal := performListGroupModelRoutes(t, &externalProvisioningServiceStub{routesErr: errors.New("database details")}, `{"group_name":"target"}`)
	if internal.Code != http.StatusInternalServerError || strings.Contains(internal.Body.String(), "database details") {
		t.Fatalf("internal: status=%d body=%s", internal.Code, internal.Body.String())
	}
}
