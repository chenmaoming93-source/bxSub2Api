package migrations

import (
	"strings"
	"testing"
)

func TestModelTokenDailyUsagesMigrationIsIdempotentAndComplete(t *testing.T) {
	content, err := FS.ReadFile("154_model_token_daily_usages.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	checks := []string{
		"CREATE TABLE IF NOT EXISTS model_token_daily_usages",
		"model              VARCHAR(255) NOT NULL",
		"usage_date         DATE NOT NULL",
		"used_tokens        BIGINT NOT NULL DEFAULT 0",
		"daily_limit_tokens BIGINT",
		"UNIQUE (model, usage_date)",
	}
	for _, check := range checks {
		if !strings.Contains(sql, check) {
			t.Errorf("migration missing %q", check)
		}
	}
	if strings.Contains(sql, "CREATE UNIQUE INDEX") {
		t.Fatal("unique index must be inline so repeated CREATE TABLE IF NOT EXISTS remains idempotent")
	}
}
