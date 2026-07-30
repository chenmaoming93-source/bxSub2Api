package routes

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRBACAdminCoreRouteDeclarations(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatalf("read admin routes: %v", err)
	}
	declarations := regexp.MustCompile(`(?m)^\s+admin(GET|POST|PUT|DELETE)\(routes,`).FindAll(data, -1)
	if len(declarations) < 57 {
		t.Fatalf("admin RBAC declarations = %d, want at least core baseline 57", len(declarations))
	}
	source := string(data)
	for _, permission := range []string{
		"rbac.PermissionUsersBalanceAdjust",
		"rbac.PermissionUsersDelete",
		"rbac.PermissionUsersQuotaUpdate",
		"rbac.PermissionGroupsDelete",
		"rbac.PermissionDashboardBackfill",
	} {
		if !strings.Contains(source, permission) {
			t.Errorf("high-risk permission declaration missing: %s", permission)
		}
	}
}
