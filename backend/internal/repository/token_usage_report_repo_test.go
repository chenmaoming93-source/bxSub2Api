package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestTokenUsageReportRepositoryModelZeroFillPaginationAndSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 6)
	baseArgs := []driver.Value{start, end, "gpt", start, end, "gpt"}
	mock.ExpectQuery("WITH RECURSIVE dates").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(7, 42))
	mock.ExpectQuery("COALESCE\\(u.used_tokens, 0\\).*ORDER BY r.used_tokens DESC, r.model ASC").WithArgs(append(baseArgs, 2, 2)...).WillReturnRows(
		sqlmock.NewRows([]string{"usage_date", "model", "used_tokens", "daily_limit_tokens"}).
			AddRow(start.AddDate(0, 0, 2), "gpt", 0, 0).
			AddRow(start.AddDate(0, 0, 3), "gpt", 0, 0),
	)
	repo := NewTokenUsageReportRepository(db)
	items, total, used, err := repo.ListModelTokenUsage(context.Background(), service.TokenUsageReportQuery{Model: "gpt", StartDate: start, EndDate: end, Page: 2, PageSize: 2, SortBy: "used_tokens", SortOrder: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || total != 7 || used != 42 || items[0].UsedTokens != 0 || items[0].DailyLimitTokens == nil || *items[0].DailyLimitTokens != 0 {
		t.Fatalf("unexpected result: %+v %d %d", items, total, used)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryModelKeepsUsageWithoutConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	baseArgs := []driver.Value{day, day, "legacy", day, day, "legacy"}
	mock.ExpectQuery("WITH RECURSIVE dates").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(1, 11))
	mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "model", "used_tokens", "daily_limit_tokens"}).AddRow(day, "legacy", 11, nil))
	repo := NewTokenUsageReportRepository(db)
	items, total, used, err := repo.ListModelTokenUsage(context.Background(), service.TokenUsageReportQuery{Model: "legacy", StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"})
	if err != nil || len(items) != 1 || total != 1 || used != 11 || items[0].DailyLimitTokens != nil {
		t.Fatalf("items=%+v total=%d used=%d err=%v", items, total, used, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryRouteKeepsHistoricalCandidate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	baseArgs := []driver.Value{day, day, int64(1), "fast", "gpt", day, day, int64(1), "fast", "gpt"}
	mock.ExpectQuery("WITH RECURSIVE dates").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(1, 7))
	mock.ExpectQuery("LEFT JOIN JSON_TABLE").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "group_id", "group_name", "route_alias", "upstream_model", "used_tokens", "daily_limit_tokens", "priority"}).AddRow(day, 1, "Group", "fast", "gpt", 7, nil, nil))
	repo := NewTokenUsageReportRepository(db)
	rows, total, used, err := repo.ListRouteTokenUsage(context.Background(), service.RouteTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}, GroupID: 1, RouteAlias: "fast", UpstreamModel: "gpt"})
	if err != nil || len(rows) != 1 || total != 1 || used != 7 || rows[0].Priority != nil || rows[0].DailyLimitTokens != nil {
		t.Fatalf("%+v %d %d %v", rows, total, used, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryRouteZeroFillsEveryConfiguredCandidate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 1)
	baseArgs := []driver.Value{start, end, int64(3), "fast", start, end, int64(3), "fast"}
	mock.ExpectQuery("targets\\(group_id, route_alias, upstream_model\\)").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(4, 5))
	result := sqlmock.NewRows([]string{"usage_date", "group_id", "group_name", "route_alias", "upstream_model", "used_tokens", "limit", "priority"}).
		AddRow(start, 3, "Group", "fast", "claude", 5, 0, 1).
		AddRow(end, 3, "Group", "fast", "claude", 0, 0, 1).
		AddRow(start, 3, "Group", "fast", "gpt", 0, 100, 2).
		AddRow(end, 3, "Group", "fast", "gpt", 0, 100, 2)
	mock.ExpectQuery("CROSS JOIN dates.*ORDER BY r.used_tokens DESC, r.group_id ASC, r.route_alias ASC, r.upstream_model ASC").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(result)
	repo := NewTokenUsageReportRepository(db)
	rows, total, used, err := repo.ListRouteTokenUsage(context.Background(), service.RouteTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: start, EndDate: end, Page: 1, PageSize: 20, SortBy: "used_tokens", SortOrder: "desc"}, GroupID: 3, RouteAlias: "fast"})
	if err != nil || len(rows) != 4 || total != 4 || used != 5 {
		t.Fatalf("rows=%+v total=%d used=%d err=%v", rows, total, used, err)
	}
	if rows[1].UsedTokens != 0 || rows[1].DailyLimitTokens == nil || *rows[1].DailyLimitTokens != 0 || rows[1].Priority == nil {
		t.Fatalf("unexpected zero-filled row: %+v", rows[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryRouteFiltersConfiguredCandidate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	baseArgs := []driver.Value{day, day, int64(3), "fast", "gpt", day, day, int64(3), "fast", "gpt"}
	mock.ExpectQuery("daily_limit_configs WHERE 1=1 AND group_id = \\? AND route_alias = \\? AND upstream_model = \\?").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(1, 0))
	mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "group_id", "group_name", "route_alias", "upstream_model", "used_tokens", "limit", "priority"}).AddRow(day, 3, "Group", "fast", "gpt", 0, 100, nil))
	repo := NewTokenUsageReportRepository(db)
	rows, _, _, err := repo.ListRouteTokenUsage(context.Background(), service.RouteTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"}, GroupID: 3, RouteAlias: "fast", UpstreamModel: "gpt"})
	if err != nil || len(rows) != 1 || rows[0].UpstreamModel != "gpt" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryUserIncludesSoftDeleted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	baseArgs := []driver.Value{day, day, int64(2), day, day, int64(2)}
	mock.ExpectQuery("WITH RECURSIVE dates").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(1, 9))
	mock.ExpectQuery("LEFT JOIN users").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "user_id", "email", "username", "deleted", "model", "used_tokens", "limit"}).AddRow(day, 2, "old@example.com", "old", true, "gpt", 9, nil))
	repo := NewTokenUsageReportRepository(db)
	rows, _, _, err := repo.ListUserTokenUsage(context.Background(), service.UserTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}, UserID: 2, IncludeDeleted: true})
	if err != nil || len(rows) != 1 || !rows[0].UserDeleted || rows[0].DailyLimitTokens != nil {
		t.Fatalf("%+v %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryUserZeroFillsConfiguredModelsAndFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 2)
	baseArgs := []driver.Value{start, end, int64(7), "gpt", start, end, int64(7), "gpt"}
	mock.ExpectQuery("user_model_token_daily_limit_configs WHERE 1=1.*user_id = \\?.*model = \\?").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(3, 0))
	mock.ExpectQuery("CROSS JOIN dates.*ORDER BY r.usage_date ASC, r.user_id ASC, r.model ASC").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(
		sqlmock.NewRows([]string{"usage_date", "user_id", "email", "username", "deleted", "model", "used_tokens", "limit"}).
			AddRow(start, 7, "user@example.com", "user", false, "gpt", 0, 100).
			AddRow(start.AddDate(0, 0, 1), 7, "user@example.com", "user", false, "gpt", 0, 100).
			AddRow(end, 7, "user@example.com", "user", false, "gpt", 0, 100),
	)
	repo := NewTokenUsageReportRepository(db)
	rows, total, used, err := repo.ListUserTokenUsage(context.Background(), service.UserTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{Model: "gpt", StartDate: start, EndDate: end, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"}, UserID: 7})
	if err != nil || len(rows) != 3 || total != 3 || used != 0 {
		t.Fatalf("rows=%+v total=%d used=%d err=%v", rows, total, used, err)
	}
	for _, row := range rows {
		if row.Model != "gpt" || row.UsedTokens != 0 || row.DailyLimitTokens == nil || *row.DailyLimitTokens != 100 {
			t.Fatalf("unexpected row: %+v", row)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryUserZeroFillsEveryConfiguredModel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 1)
	baseArgs := []driver.Value{start, end, int64(7), start, end, int64(7)}
	mock.ExpectQuery("targets\\(user_id, model\\)").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(4, 0))
	result := sqlmock.NewRows([]string{"usage_date", "user_id", "email", "username", "deleted", "model", "used_tokens", "limit"})
	for _, model := range []string{"claude", "gpt"} {
		result.AddRow(start, 7, "user@example.com", "user", false, model, 0, 100)
		result.AddRow(end, 7, "user@example.com", "user", false, model, 0, 100)
	}
	mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(result)
	repo := NewTokenUsageReportRepository(db)
	rows, total, _, err := repo.ListUserTokenUsage(context.Background(), service.UserTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: start, EndDate: end, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"}, UserID: 7})
	if err != nil || len(rows) != 4 || total != 4 {
		t.Fatalf("rows=%+v total=%d err=%v", rows, total, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenUsageReportRepositoryAllowsEmptyFilters(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	baseArgs := []driver.Value{day, day, day, day}

	t.Run("model", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery("model_token_daily_limit_configs WHERE 1=1").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(0, 0))
		mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "model", "used_tokens", "limit"}))
		_, _, _, err = NewTokenUsageReportRepository(db).ListModelTokenUsage(context.Background(), service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("route", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery("candidate_token_daily_limit_configs WHERE 1=1").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(0, 0))
		mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "group_id", "group_name", "route_alias", "upstream_model", "used_tokens", "limit", "priority"}))
		_, _, _, err = NewTokenUsageReportRepository(db).ListRouteTokenUsage(context.Background(), service.RouteTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("user", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery("user_model_token_daily_limit_configs WHERE 1=1.*deleted_at IS NULL").WithArgs(baseArgs...).WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(0, 0))
		mock.ExpectQuery("FROM report r").WithArgs(append(baseArgs, 20, 0)...).WillReturnRows(sqlmock.NewRows([]string{"usage_date", "user_id", "email", "username", "deleted", "model", "used_tokens", "limit"}))
		_, _, _, err = NewTokenUsageReportRepository(db).ListUserTokenUsage(context.Background(), service.UserTokenUsageReportQuery{TokenUsageReportQuery: service.TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestTokenUsageOptionSearches(t *testing.T) {
	tests := []struct {
		name, kind, query string
		parent            int64
		args              []driver.Value
		id                int64
		label             string
	}{
		{"models", "models", "SELECT 0,model", 0, []driver.Value{"%gpt%", "%gpt%", 20}, 0, "gpt-5"},
		{"routes include usage", "routes", "UNION SELECT 0,route_alias FROM group_candidate_token_daily_usages", 3, []driver.Value{int64(3), "%fast%", int64(3), "%fast%", 20}, 0, "fast"},
		{"route models", "route_models", "UNION SELECT 0,upstream_model FROM group_candidate_token_daily_usages", 3, []driver.Value{int64(3), "fast", "%gpt%", int64(3), "fast", "%gpt%", 20}, 0, "gpt-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(tt.query).WithArgs(tt.args...).WillReturnRows(sqlmock.NewRows([]string{"id", "label"}).AddRow(tt.id, tt.label))
			repo := NewTokenUsageReportRepository(db)
			q := "gpt"
			if tt.kind == "routes" {
				q = "fast"
			}
			if tt.kind == "route_models" {
				q = "fast\x00gpt"
			}
			items, err := repo.SearchTokenUsageOptions(context.Background(), tt.kind, tt.parent, q, 20)
			if err != nil || len(items) != 1 || items[0].Label != tt.label {
				t.Fatalf("items=%+v err=%v", items, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
