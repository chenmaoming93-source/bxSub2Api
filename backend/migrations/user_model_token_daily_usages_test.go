package migrations

import (
	"strings"
	"testing"
)

func TestUserModelTokenDailyUsagesMigrationConstraints(t *testing.T) {
	content, err := FS.ReadFile("155_user_model_token_daily_usages.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	checks := []string{
		"CREATE TABLE IF NOT EXISTS user_model_token_daily_usages",
		"FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
		"UNIQUE (user_id, model, usage_date)",
		"INDEX user_model_token_daily_usages_user_idx (user_id)",
		"used_tokens        BIGINT NOT NULL DEFAULT 0",
		"daily_limit_tokens BIGINT",
	}
	for _, check := range checks {
		if !strings.Contains(sql, check) {
			t.Errorf("migration missing %q", check)
		}
	}
}
