package routes

import (
	"os"
	"strings"
	"testing"
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
		`"/query", rbac.PermissionTokenUsageRead`,
		`"/status", rbac.PermissionTokenUsageRead`,
	} {
		if !strings.Contains(source, check) {
			t.Errorf("ops/usage/token route mapping missing: %s", check)
		}
	}
	if !strings.Contains(source, `adminDELETE(routes, stats, "/quotas/:id", rbac.PermissionTokenQuotaUpdate`) {
		t.Error("token quota delete route must use the quota update permission")
	}
}
