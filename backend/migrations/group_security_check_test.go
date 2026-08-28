package migrations

import (
	"strings"
	"testing"
)

func TestGroupSecurityCheckMigrationAddsSafeDefaultsAndIndexes(t *testing.T) {
	sql, err := FS.ReadFile("159_group_security_check.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	content := string(sql)
	for _, want := range []string{
		"ADD COLUMN `security_check_config` JSON NULL",
		"UPDATE `groups`",
		`"exception_action":"allow"`,
		"MODIFY COLUMN `security_check_config` JSON NOT NULL",
		"CREATE TABLE IF NOT EXISTS `security_check_logs`",
		"UNIQUE KEY uk_security_check_logs_event_id",
		"idx_security_check_logs_created_at",
		"idx_security_check_logs_group_created",
		"idx_security_check_logs_decision_created",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("migration missing %q", want)
		}
	}
}
