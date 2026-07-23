package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type permissionProviderStub struct {
	effective rbac.EffectivePermissions
	err       error
}

func (s permissionProviderStub) GetEffectivePermissions(context.Context, int64) (rbac.EffectivePermissions, error) {
	return s.effective, s.err
}

func TestRBACMiddlewarePermissionWildcardAndDenial(t *testing.T) {
	tests := []struct {
		name       string
		principal  *rbac.Principal
		effective  rbac.EffectivePermissions
		wantStatus int
		wantCalled bool
	}{
		{name: "missing principal", wantStatus: http.StatusUnauthorized},
		{name: "ordinary permission", principal: &rbac.Principal{UserID: 1, Status: "active"},
			effective:  rbac.EffectivePermissions{Permissions: []string{rbac.PermissionUsersRead}},
			wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "wildcard from role", principal: &rbac.Principal{UserID: 1, Status: "active"},
			effective:  rbac.EffectivePermissions{Permissions: []string{rbac.PermissionAll}, IsSuperAdmin: true},
			wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "admin api key principal", principal: &rbac.Principal{UserID: 1, Status: "active", SuperAdmin: true},
			wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "permission denied", principal: &rbac.Principal{UserID: 1, Status: "active"},
			wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			called := false
			router.GET("/trial", func(c *gin.Context) {
				if tt.principal != nil {
					rbac.SetPrincipal(c, *tt.principal)
				}
				c.Next()
			}, RequirePermission(permissionProviderStub{effective: tt.effective}, RBACModeEnforce, nil)(rbac.PermissionUsersRead),
				func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodGet, "/trial", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tt.wantStatus, response.Code)
			require.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestRBACShadowAuditsButAllows(t *testing.T) {
	router := gin.New()
	var decision AuthorizationDecision
	router.GET("/trial",
		func(c *gin.Context) {
			rbac.SetPrincipal(c, rbac.Principal{UserID: 2, Status: "active"})
		},
		RequirePermission(permissionProviderStub{}, RBACModeShadow, func(_ context.Context, got AuthorizationDecision) {
			decision = got
		})(rbac.PermissionUsersRead),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trial", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.True(t, decision.Shadow)
	require.Equal(t, rbac.PermissionUsersRead, decision.Permission)
}

func TestRBACEnforceAndShadowRollback(t *testing.T) {
	for _, tt := range []struct {
		mode RBACMode
		want int
	}{{RBACModeEnforce, http.StatusForbidden}, {RBACModeShadow, http.StatusNoContent}} {
		router := gin.New()
		router.GET("/trial",
			func(c *gin.Context) { rbac.SetPrincipal(c, rbac.Principal{UserID: 2, Status: "active"}) },
			RequirePermission(permissionProviderStub{}, tt.mode, nil)(rbac.PermissionUsersRead),
			func(c *gin.Context) { c.Status(http.StatusNoContent) },
		)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trial", nil))
		require.Equal(t, tt.want, response.Code)
	}
}

func TestParseRBACModeFailsClosed(t *testing.T) {
	require.Equal(t, RBACModeShadow, ParseRBACMode("shadow"))
	require.Equal(t, RBACModeEnforce, ParseRBACMode("enforce"))
	require.Equal(t, RBACModeEnforce, ParseRBACMode("unexpected"))
}

func TestLegacyAdminFallbackAllowsControlledRouteButProtectsUnregisteredRoute(t *testing.T) {
	registry := rbac.NewRegistry()
	require.NoError(t, registry.RegisterControlled(http.MethodGet, "/controlled", rbac.PermissionUsersRead))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		rbac.SetPrincipal(c, rbac.Principal{UserID: 8, Status: "active"})
		c.Set(string(ContextKeyUserRole), "user")
	})
	router.Use(RequireLegacyAdminForUnregistered(registry))
	router.GET("/controlled", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/legacy", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	controlled := httptest.NewRecorder()
	router.ServeHTTP(controlled, httptest.NewRequest(http.MethodGet, "/controlled", nil))
	require.Equal(t, http.StatusNoContent, controlled.Code)
	legacy := httptest.NewRecorder()
	router.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/legacy", nil))
	require.Equal(t, http.StatusForbidden, legacy.Code)
}
