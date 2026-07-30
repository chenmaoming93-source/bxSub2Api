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
		`"/models", rbac.PermissionTokenUsageRead`,
		`"", rbac.PermissionTokenQuotaUpdate, h.Admin.ModelTokenQuota.Update`,
	} {
		if !strings.Contains(source, check) {
			t.Errorf("ops/usage/token route mapping missing: %s", check)
		}
	}
}
