package migrations

import (
	"strings"
	"testing"
)

func TestTokenUsageReportIndexesArePurposeBuilt(t *testing.T) {
	content, err := FS.ReadFile("159_token_usage_report_indexes.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(content)
	for _, expected := range []string{
		"(usage_date, used_tokens)",
		"(group_id, route_alias, usage_date, upstream_model)",
		"(usage_date, group_id, route_alias)",
		"(user_id, usage_date, model)",
		"(usage_date, user_id, model)",
	} {
		if !strings.Contains(sql, expected) {
			t.Errorf("missing index columns %s", expected)
		}
	}
}
