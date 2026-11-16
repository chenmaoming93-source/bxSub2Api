package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRBACAdminOpsReadWriteAndWebSocketPermissions(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatalf("read admin routes: %v", err)
	}
	source := string(data)
	for _, check := range []string{
		`"/qps", rbac.PermissionOpsRead`,
		`"/alert-rules", rbac.PermissionOpsUpdate`,
		`"/errors/:id/resolve", rbac.PermissionOpsLogsManage`,
		`"/system-logs/cleanup", rbac.PermissionOpsLogsManage`,
		`"/cleanup-tasks", rbac.PermissionUsageAdminManage`,
		`"/dimensions", rbac.PermissionTokenUsageRead`,
		`"/projections", rbac.PermissionTokenUsageManage`,
		`"/quotas", rbac.PermissionTokenQuotaRead`,
		`"/quotas/:id", rbac.PermissionTokenQuotaUpdate`,
		`"/quotas/:id/enable", rbac.PermissionTokenQuotaUpdate`,
		`"/quota-usage/reset", rbac.PermissionTokenQuotaUpdate`,
		`"/query", rbac.PermissionTokenUsageRead`,
		`"/status", rbac.PermissionTokenUsageRead`,
		`"/capabilities", rbac.PermissionObservabilityRead`,
		`"/snapshot", rbac.PermissionObservabilityRead`,
		`"/timeseries", rbac.PermissionObservabilityRead`,
		`"/latency-timeseries", rbac.PermissionObservabilityRead`,
		`"/heatmap", rbac.PermissionObservabilityRead`,
		`"/paths", rbac.PermissionObservabilityRead`,
		`"/relations", rbac.PermissionObservabilityRead`,
	} {
		if !strings.Contains(source, check) {
			t.Errorf("ops/usage/token route mapping missing: %s", check)
		}
	}
	if !strings.Contains(source, `adminDELETE(routes, stats, "/quotas/:id", rbac.PermissionTokenQuotaUpdate`) {
		t.Error("token quota delete route must use the quota update permission")
	}
}

func TestObservabilityRoutesAreReadOnlyAndUseDedicatedPermission(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatalf("read admin routes: %v", err)
	}
	source := string(data)
	if !strings.Contains(source, `admin.Group("/observability")`) {
		t.Fatal("observability route group is missing")
	}
	if !strings.Contains(source, `adminGET(routes, observability, "/capabilities", rbac.PermissionObservabilityRead`) {
		t.Fatal("capabilities route must use observability.read")
	}
	if !strings.Contains(source, `adminGET(routes, observability, "/snapshot", rbac.PermissionObservabilityRead`) {
		t.Fatal("snapshot route must use observability.read")
	}
	for _, endpoint := range []string{`"/timeseries"`, `"/latency-timeseries"`, `"/heatmap"`, `"/paths"`, `"/relations"`} {
		if !strings.Contains(source, `adminGET(routes, observability, `+endpoint+`, rbac.PermissionObservabilityRead`) {
			t.Fatalf("%s route must use observability.read", endpoint)
		}
	}
	for _, method := range []string{"adminPOST(routes, observability", "adminPUT(routes, observability", "adminDELETE(routes, observability"} {
		if strings.Contains(source, method) {
			t.Fatalf("observability must remain read-only, found %s", method)
		}
	}
}

type observabilityPermissionProvider struct {
	permissions []string
}

func (p observabilityPermissionProvider) GetEffectivePermissions(context.Context, int64) (rbac.EffectivePermissions, error) {
	return rbac.EffectivePermissions{Permissions: p.permissions}, nil
}

func TestObservabilityRouteRequiresReadPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		rbac.SetPrincipal(c, rbac.Principal{Type: rbac.PrincipalUser, UserID: 42, Status: "active"})
		c.Next()
	})
	registry := rbac.NewRegistry()
	provider := observabilityPermissionProvider{}
	registrar := rbac.NewRouteRegistrar(registry, middleware.RequirePermission(provider, middleware.RBACModeEnforce, nil))
	group := router.Group("/api/v1/admin/observability")
	adminGET(registrar, group, "/probe", rbac.PermissionObservabilityRead, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/v1/admin/observability/probe", nil))
	require.Equal(t, http.StatusForbidden, denied.Code)

	provider.permissions = []string{rbac.PermissionObservabilityRead}
	registrar = rbac.NewRouteRegistrar(rbac.NewRegistry(), middleware.RequirePermission(provider, middleware.RBACModeEnforce, nil))
	allowedRouter := gin.New()
	allowedRouter.Use(func(c *gin.Context) {
		rbac.SetPrincipal(c, rbac.Principal{Type: rbac.PrincipalUser, UserID: 42, Status: "active"})
		c.Next()
	})
	allowedGroup := allowedRouter.Group("/api/v1/admin/observability")
	adminGET(registrar, allowedGroup, "/probe", rbac.PermissionObservabilityRead, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	allowed := httptest.NewRecorder()
	allowedRouter.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/api/v1/admin/observability/probe", nil))
	require.Equal(t, http.StatusNoContent, allowed.Code)
}
