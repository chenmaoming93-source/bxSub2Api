package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestTokenUsageQueryContract(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	valid := TokenUsageQueryContract{StartDate: day, EndDate: day, Page: TokenUsagePage{Page: 1, PageSize: 20}, Sort: TokenUsageSort{By: "usage_date", Order: "desc"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
	checks := []TokenUsageQueryContract{
		{StartDate: day, EndDate: day.AddDate(0, 0, -1), Page: valid.Page, Sort: valid.Sort},
		{StartDate: day, EndDate: day, Page: TokenUsagePage{Page: 0, PageSize: 20}, Sort: valid.Sort},
		{StartDate: day, EndDate: day, Page: TokenUsagePage{Page: 1, PageSize: 101}, Sort: valid.Sort},
		{StartDate: day, EndDate: day, Page: valid.Page, Sort: TokenUsageSort{By: "model; DROP TABLE users", Order: "asc"}},
	}
	for i, check := range checks {
		if check.Validate() == nil {
			t.Errorf("invalid contract %d accepted", i)
		}
	}
}

func TestTokenUsageSQLIsBoundedAndDeterministic(t *testing.T) {
	sort := TokenUsageSort{By: "used_tokens", Order: "desc"}
	queries := []string{modelTokenUsageListSQL(sort), routeTokenUsageListSQL(sort, false), routeTokenUsageListSQL(sort, true), userModelTokenUsageListSQL(sort, false), userModelTokenUsageListSQL(sort, true)}
	for _, query := range queries {
		for _, required := range []string{"BETWEEN ? AND ?", "ORDER BY u.used_tokens DESC, u.id ASC", "LIMIT ? OFFSET ?"} {
			if !strings.Contains(query, required) {
				t.Errorf("query missing %q:\n%s", required, query)
			}
		}
	}
}

func TestTokenUsageDateSeriesAndReportOrderAreSafe(t *testing.T) {
	if !strings.Contains(tokenUsageDateSeriesSQL, "SELECT DATE(?)") || !strings.Contains(tokenUsageDateSeriesSQL, "usage_date < DATE(?)") {
		t.Fatalf("date series must use bound inclusive endpoints: %s", tokenUsageDateSeriesSQL)
	}
	order := tokenUsageReportOrderBy(TokenUsageSort{By: "used_tokens", Order: "desc"}, reportSortColumns(map[string]string{"model": "r.model"}), "r.model ASC")
	if order != "r.used_tokens DESC, r.model ASC" {
		t.Fatalf("unexpected report ordering: %s", order)
	}
}

func TestTokenUsageReportOrderSupportsOrderedRules(t *testing.T) {
	columns := reportSortColumns(map[string]string{"model": "r.model"})
	order := tokenUsageReportOrderBy(TokenUsageSort{By: "model,used_tokens,usage_date", Order: "asc,desc,asc"}, columns, "r.model ASC")
	if order != "r.model ASC, r.used_tokens DESC, r.usage_date ASC, r.model ASC" {
		t.Fatalf("unexpected multi-field ordering: %s", order)
	}
	if validSortList("model,model", "asc,desc", columnsToAllowed(columns)) {
		t.Fatal("duplicate sort field accepted")
	}
}

func columnsToAllowed(columns map[string]string) map[string]bool {
	allowed := make(map[string]bool, len(columns))
	for field := range columns {
		allowed[field] = true
	}
	return allowed
}

func TestTokenUsageRepresentativeQueriesHaveIndexedPlans(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ddl := []string{
		`CREATE TABLE model_token_daily_usages (id INTEGER PRIMARY KEY, model TEXT, usage_date DATE, used_tokens INTEGER, UNIQUE(model, usage_date))`,
		`CREATE INDEX idx_model_usage_date_tokens ON model_token_daily_usages(usage_date, used_tokens)`,
		`CREATE TABLE group_candidate_token_daily_usages (id INTEGER PRIMARY KEY, group_id INTEGER, route_alias TEXT, upstream_model TEXT, usage_date DATE, used_tokens INTEGER, UNIQUE(group_id, route_alias, upstream_model, usage_date))`,
		`CREATE INDEX idx_route_report ON group_candidate_token_daily_usages(group_id, route_alias, usage_date, upstream_model)`,
		`CREATE INDEX idx_route_default ON group_candidate_token_daily_usages(usage_date, group_id, route_alias)`,
		`CREATE TABLE user_model_token_daily_usages (id INTEGER PRIMARY KEY, user_id INTEGER, model TEXT, usage_date DATE, used_tokens INTEGER, UNIQUE(user_id, model, usage_date))`,
		`CREATE INDEX idx_user_report ON user_model_token_daily_usages(user_id, usage_date, model)`,
		`CREATE INDEX idx_user_default ON user_model_token_daily_usages(usage_date, user_id, model)`,
	}
	for _, statement := range ddl {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	queries := []string{
		`SELECT * FROM model_token_daily_usages WHERE model='m' AND usage_date BETWEEN '2026-07-01' AND '2026-07-06'`,
		`SELECT * FROM group_candidate_token_daily_usages WHERE group_id=1 AND route_alias='r' AND usage_date BETWEEN '2026-07-01' AND '2026-07-06'`,
		`SELECT * FROM user_model_token_daily_usages WHERE user_id=1 AND usage_date BETWEEN '2026-07-01' AND '2026-07-06'`,
		`SELECT model FROM model_token_daily_usages WHERE usage_date='2026-07-06' ORDER BY used_tokens DESC LIMIT 1`,
		`SELECT group_id, route_alias FROM group_candidate_token_daily_usages WHERE usage_date='2026-07-06' LIMIT 1`,
		`SELECT user_id, model FROM user_model_token_daily_usages WHERE usage_date='2026-07-06' LIMIT 1`,
	}
	for _, query := range queries {
		rows, err := db.Query("EXPLAIN QUERY PLAN " + query)
		if err != nil {
			t.Fatalf("explain %q: %v", query, err)
		}
		indexed := false
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(detail, "INDEX") {
				indexed = true
			}
		}
		rows.Close()
		if !indexed {
			t.Errorf("representative query did not use an index: %s", query)
		}
	}
}
