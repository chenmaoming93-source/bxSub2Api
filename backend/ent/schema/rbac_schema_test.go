package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRBACDDLContainsAllEntTablesAndSafetyGuards(t *testing.T) {
	ddlPath := filepath.Join("..", "..", "sqlArchiving", "162_create_rbac_schema.sql")
	data, err := os.ReadFile(ddlPath)
	if err != nil {
		t.Fatalf("read RBAC DDL: %v", err)
	}
	ddl := string(data)
	tables := []string{
		"rbac_roles", "rbac_permissions", "rbac_user_roles",
		"rbac_role_permissions", "rbac_user_versions",
		"rbac_policy_state", "rbac_audit_logs",
	}
	for _, table := range tables {
		if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("DDL does not create %s idempotently", table)
		}
	}
	if !strings.Contains(ddl, "ON DUPLICATE KEY UPDATE id = id") {
		t.Error("policy singleton seed lacks repeat-execution guard")
	}
	if !strings.Contains(ddl, "REFERENCES users(id)") {
		t.Error("DDL lacks user foreign keys")
	}
}
