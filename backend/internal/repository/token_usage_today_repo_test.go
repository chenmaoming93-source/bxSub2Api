package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestTodayTokenUsageCollectionsUseOneQueryPerDimension(t *testing.T) {
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		columns []string
		values  []driver.Value
		call    func(*tokenUsageReportRepository) (int, error)
	}{
		{"model", []string{"model", "used_tokens", "daily_limit_tokens"}, []driver.Value{"gpt", int64(3), int64(10)}, func(r *tokenUsageReportRepository) (int, error) {
			x, e := r.ListTodayModelTokenUsage(context.Background(), service.TokenUsageReportQuery{Model: "gpt"}, day)
			return len(x), e
		}},
		{"route", []string{"group_id", "group_name", "route_alias", "upstream_model", "used_tokens", "daily_limit_tokens", "priority"}, []driver.Value{int64(1), "g", "fast", "gpt", int64(4), int64(20), int64(2)}, func(r *tokenUsageReportRepository) (int, error) {
			x, e := r.ListTodayRouteTokenUsage(context.Background(), service.RouteTokenUsageReportQuery{GroupID: 1, RouteAlias: "fast", UpstreamModel: "gpt"}, day)
			return len(x), e
		}},
		{"user", []string{"user_id", "email", "username", "deleted", "model", "used_tokens", "daily_limit_tokens"}, []driver.Value{int64(2), "a@b", "alice", false, "gpt", int64(5), int64(30)}, func(r *tokenUsageReportRepository) (int, error) {
			x, e := r.ListTodayUserTokenUsage(context.Background(), service.UserTokenUsageReportQuery{UserID: 2, TokenUsageReportQuery: service.TokenUsageReportQuery{Model: "gpt"}}, day)
			return len(x), e
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery("WITH targets AS").WillReturnRows(sqlmock.NewRows(tt.columns).AddRow(tt.values...))
			count, err := tt.call(&tokenUsageReportRepository{db: db})
			if err != nil || count != 1 {
				t.Fatalf("count=%d err=%v", count, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTodayTokenUsageNonexistentFilterReturnsEmptyWithoutPointLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("WITH targets AS").WillReturnRows(sqlmock.NewRows([]string{"model", "used_tokens", "daily_limit_tokens"}))
	rows, err := (&tokenUsageReportRepository{db: db}).ListTodayModelTokenUsage(context.Background(), service.TokenUsageReportQuery{Model: "not-exists"}, day)
	if err != nil || len(rows) != 0 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
