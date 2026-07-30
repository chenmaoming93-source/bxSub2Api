package routes

import (
	"os"
	"strings"
	"testing"
)

func TestRBACAdminAccountSensitivePermissions(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatalf("read admin routes: %v", err)
	}
	source := string(data)
	checks := []string{
		`"/:id/credentials", rbac.PermissionAccountsCredentialsRead`,
		`"/import/codex-session", rbac.PermissionAccountsCredentialsUpdate`,
		`"/batch-update-credentials", rbac.PermissionAccountsCredentialsUpdate`,
		`"/:id/test", rbac.PermissionAccountsOperate`,
		`"/exchange-code", rbac.PermissionAccountsCredentialsUpdate`,
		`"/:id/quality-check", rbac.PermissionProxiesOperate`,
	}
	for _, check := range checks {
		if !strings.Contains(source, check) {
			t.Errorf("sensitive route mapping missing: %s", check)
		}
	}
}
