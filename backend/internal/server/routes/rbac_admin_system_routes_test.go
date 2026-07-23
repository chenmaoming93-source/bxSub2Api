package routes

import (
	"os"
	"strings"
	"testing"
)

func TestRBACAdminSystemCriticalPermissions(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatalf("read admin routes: %v", err)
	}
	source := string(data)
	for _, check := range []string{
		`"/admin-api-key", rbac.PermissionSettingsSecretsManage`,
		`"/admin-api-key/regenerate", rbac.PermissionSettingsSecretsManage`,
		`"/:id/restore", rbac.PermissionBackupsRestore`,
		`"/update", rbac.PermissionSystemOperate`,
		`"/rollback", rbac.PermissionSystemOperate`,
		`"/restart", rbac.PermissionSystemOperate`,
	} {
		if !strings.Contains(source, check) {
			t.Errorf("critical route mapping missing: %s", check)
		}
	}
}
