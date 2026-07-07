package service

import (
	"context"
	"testing"
	"time"
)

type modelTokenUsageRepoStub struct {
	rows        []ModelTokenUsageRow
	routeRows   []RouteTokenUsageRow
	userRows    []UserTokenUsageRow
	total, used int64
	lastLimit   *int
}

func (s modelTokenUsageRepoStub) ListModelTokenUsage(context.Context, TokenUsageReportQuery) ([]ModelTokenUsageRow, int64, int64, error) {
	return s.rows, s.total, s.used, nil
}
func (s modelTokenUsageRepoStub) ListRouteTokenUsage(context.Context, RouteTokenUsageReportQuery) ([]RouteTokenUsageRow, int64, int64, error) {
	return s.routeRows, s.total, s.used, nil
}
func (s modelTokenUsageRepoStub) ListUserTokenUsage(context.Context, UserTokenUsageReportQuery) ([]UserTokenUsageRow, int64, int64, error) {
	return s.userRows, s.total, s.used, nil
}
func (s modelTokenUsageRepoStub) SearchTokenUsageOptions(_ context.Context, _ string, _ int64, _ string, limit int) ([]TokenUsageOption, error) {
	if s.lastLimit != nil {
		*s.lastLimit = limit
	}
	return nil, nil
}

func TestTokenUsageOptionsAreBoundedAndDefaultMayBeEmpty(t *testing.T) {
	seen := 0
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{lastLimit: &seen})
	items, err := svc.SearchOptions(context.Background(), "models", 0, "gpt", 99)
	if err != nil || len(items) != 0 || seen != 20 {
		t.Fatalf("items=%v limit=%d err=%v", items, seen, err)
	}
	target, err := svc.DefaultTarget(context.Background(), "model", time.Now())
	if err != nil || target != nil {
		t.Fatalf("target=%v err=%v", target, err)
	}
	if _, err := svc.SearchOptions(context.Background(), "routes", 0, "", 20); err == nil {
		t.Fatal("expected parent validation")
	}
}
func (s modelTokenUsageRepoStub) FindDefaultTokenUsageTarget(context.Context, string, time.Time) (*TokenUsageOption, error) {
	return nil, nil
}

func TestTokenUsageReportServiceModelStatuses(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	limit := int64(100)
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{rows: []ModelTokenUsageRow{{UsageDate: day, Model: "gpt", UsedTokens: 80, DailyLimitTokens: &limit}, {UsageDate: day, Model: "free", UsedTokens: 10}}, total: 2, used: 90})
	report, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{Model: "gpt", StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Items[0].Status != "warning" || report.Items[0].UsageRate == nil || report.Items[1].Status != "unlimited" || report.Items[1].UsageRate != nil {
		t.Fatalf("unexpected statuses: %+v", report.Items)
	}
}

func TestTokenUsageReportServiceRouteAndDeletedUser(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{routeRows: []RouteTokenUsageRow{{UsageDate: day, GroupID: 1, RouteAlias: "fast", UpstreamModel: "gpt", UsedTokens: 3}}, userRows: []UserTokenUsageRow{{UsageDate: day, UserID: 2, Model: "gpt", UserDeleted: true, UsedTokens: 4}}})
	base := TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}
	route, err := svc.GetRouteReport(context.Background(), RouteTokenUsageReportQuery{TokenUsageReportQuery: base, GroupID: 1, RouteAlias: "fast"})
	if err != nil || len(route.Items) != 1 || route.Items[0].Priority != nil {
		t.Fatalf("route: %+v %v", route, err)
	}
	user, err := svc.GetUserReport(context.Background(), UserTokenUsageReportQuery{TokenUsageReportQuery: base, UserID: 2})
	if err != nil || !user.Items[0].UserDeleted {
		t.Fatalf("user: %+v %v", user, err)
	}
}

func TestTokenUsageReportServiceAllowsEmptyNonDateFilters(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	base := TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "desc"}
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{})
	if _, err := svc.GetModelReport(context.Background(), base); err != nil {
		t.Fatalf("empty model filter rejected: %v", err)
	}
	if _, err := svc.GetRouteReport(context.Background(), RouteTokenUsageReportQuery{TokenUsageReportQuery: base}); err != nil {
		t.Fatalf("empty route filters rejected: %v", err)
	}
	if _, err := svc.GetUserReport(context.Background(), UserTokenUsageReportQuery{TokenUsageReportQuery: base}); err != nil {
		t.Fatalf("empty user filters rejected: %v", err)
	}
}

func TestTokenUsageReportServiceAllowsDimensionSpecificSorts(t *testing.T) {
	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{})
	query := func(sortBy string) TokenUsageReportQuery {
		return TokenUsageReportQuery{StartDate: day, EndDate: day, Page: 1, PageSize: 20, SortBy: sortBy, SortOrder: "asc"}
	}
	for _, field := range []string{"model", "daily_limit_tokens", "usage_rate", "status"} {
		if _, err := svc.GetModelReport(context.Background(), query(field)); err != nil {
			t.Fatalf("model sort %q rejected: %v", field, err)
		}
	}
	for _, field := range []string{"group", "route_alias", "upstream_model", "priority"} {
		if _, err := svc.GetRouteReport(context.Background(), RouteTokenUsageReportQuery{TokenUsageReportQuery: query(field)}); err != nil {
			t.Fatalf("route sort %q rejected: %v", field, err)
		}
	}
	for _, field := range []string{"user", "user_deleted", "model"} {
		if _, err := svc.GetUserReport(context.Background(), UserTokenUsageReportQuery{TokenUsageReportQuery: query(field)}); err != nil {
			t.Fatalf("user sort %q rejected: %v", field, err)
		}
	}
}

func TestTokenUsageReportServiceRejectsUnboundedQuery(t *testing.T) {
	svc := NewTokenUsageReportService(modelTokenUsageRepoStub{})
	_, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{Page: 1, PageSize: 101})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
