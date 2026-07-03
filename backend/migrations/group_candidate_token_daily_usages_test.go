package migrations

import (
	"strings"
	"testing"
)

func TestGroupCandidateTokenDailyUsagesMigrationIdentity(t *testing.T) {
	content, err := FS.ReadFile("156_group_candidate_token_daily_usages.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, check := range []string{
		"CREATE TABLE IF NOT EXISTS group_candidate_token_daily_usages",
		"FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE",
		"UNIQUE (group_id, route_alias, upstream_model, usage_date)",
		"INDEX group_candidate_token_daily_usages_group_idx (group_id)",
	} {
		if !strings.Contains(sql, check) {
			t.Errorf("migration missing %q", check)
		}
	}
}
