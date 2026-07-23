package service

import (
	"context"
	"testing"
	"time"
)

type hybridModelRepoStub struct {
	history       []ModelTokenUsageRow
	today         []ModelTokenUsageRow
	routeHistory  []RouteTokenUsageRow
	routeToday    []RouteTokenUsageRow
	userHistory   []UserTokenUsageRow
	userToday     []UserTokenUsageRow
	historyCalls  int
	options       []TokenUsageOption
	defaultTarget *TokenUsageOption
	defaultCalls  int
}

func (r *hybridModelRepoStub) ListModelTokenUsage(context.Context, TokenUsageReportQuery) ([]ModelTokenUsageRow, int64, int64, error) {
	r.historyCalls++
	var used int64
	for _, x := range r.history {
		used += x.UsedTokens
	}
	return r.history, int64(len(r.history)), used, nil
}
func (r *hybridModelRepoStub) ListRouteTokenUsage(context.Context, RouteTokenUsageReportQuery) ([]RouteTokenUsageRow, int64, int64, error) {
	var used int64
	for _, x := range r.routeHistory {
		used += x.UsedTokens
	}
	return r.routeHistory, int64(len(r.routeHistory)), used, nil
}
func (r *hybridModelRepoStub) ListUserTokenUsage(context.Context, UserTokenUsageReportQuery) ([]UserTokenUsageRow, int64, int64, error) {
	var used int64
	for _, x := range r.userHistory {
		used += x.UsedTokens
	}
	return r.userHistory, int64(len(r.userHistory)), used, nil
}
func (r *hybridModelRepoStub) SearchTokenUsageOptions(context.Context, string, int64, string, int) ([]TokenUsageOption, error) {
	return append([]TokenUsageOption(nil), r.options...), nil
}
func (r *hybridModelRepoStub) FindDefaultTokenUsageTarget(context.Context, string, time.Time) (*TokenUsageOption, error) {
	r.defaultCalls++
	return r.defaultTarget, nil
}
func (r *hybridModelRepoStub) ListTodayModelTokenUsage(context.Context, TokenUsageReportQuery, time.Time) ([]ModelTokenUsageRow, error) {
	return r.today, nil
}
func (r *hybridModelRepoStub) ListTodayRouteTokenUsage(context.Context, RouteTokenUsageReportQuery, time.Time) ([]RouteTokenUsageRow, error) {
	return r.routeToday, nil
}
func (r *hybridModelRepoStub) ListTodayUserTokenUsage(context.Context, UserTokenUsageReportQuery, time.Time) ([]UserTokenUsageRow, error) {
	return r.userToday, nil
}

type hybridReaderStub struct {
	models []ModelTokenUsageRow
	routes []RouteTokenUsageRow
	users  []UserTokenUsageRow
	calls  int
	err    error
}

func (r *hybridReaderStub) ReadModelUsage(context.Context, time.Time, []string) (CurrentTokenUsageReadResult[ModelTokenUsageRow], error) {
	r.calls++
	if r.err != nil {
		return CurrentTokenUsageReadResult[ModelTokenUsageRow]{}, r.err
	}
	return CurrentTokenUsageReadResult[ModelTokenUsageRow]{Rows: r.models}, nil
}
func (r *hybridReaderStub) ReadRouteUsage(context.Context, time.Time, []RouteTokenUsageRow) (CurrentTokenUsageReadResult[RouteTokenUsageRow], error) {
	r.calls++
	return CurrentTokenUsageReadResult[RouteTokenUsageRow]{Rows: r.routes}, nil
}
func (r *hybridReaderStub) ReadUserModelUsage(context.Context, time.Time, []UserTokenUsageRow) (CurrentTokenUsageReadResult[UserTokenUsageRow], error) {
	r.calls++
	return CurrentTokenUsageReadResult[UserTokenUsageRow]{Rows: r.users}, nil
}

type hybridRepairStub struct {
	rows     []ModelTokenUsageRow
	userRows []UserTokenUsageRow
}

func (r *hybridRepairStub) RepairModelUsage(_ context.Context, _ time.Time, rows []ModelTokenUsageRow) error {
	r.rows = append(r.rows, rows...)
	return nil
}
func (*hybridRepairStub) RepairRouteUsage(context.Context, time.Time, []RouteTokenUsageRow) error {
	return nil
}
func (r *hybridRepairStub) RepairUserModelUsage(_ context.Context, _ time.Time, rows []UserTokenUsageRow) error {
	r.userRows = append(r.userRows, rows...)
	return nil
}

func TestModelTokenUsageHistoricalDoesNotReadRedis(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{history: []ModelTokenUsageRow{{UsageDate: today.AddDate(0, 0, -1), Model: "old", UsedTokens: 3}}}
	reader := &hybridReaderStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	_, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{StartDate: today.AddDate(0, 0, -1), EndDate: today.AddDate(0, 0, -1), Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"})
	if err != nil || reader.calls != 0 {
		t.Fatalf("err=%v redis_calls=%d", err, reader.calls)
	}
}

func TestModelTokenUsageHybridUsesRedisSortsPagesAndRepairs(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	limit := int64(100)
	repo := &hybridModelRepoStub{today: []ModelTokenUsageRow{{UsageDate: today, Model: "both", UsedTokens: 10, DailyLimitTokens: &limit}, {UsageDate: today, Model: "mysql", UsedTokens: 20}}}
	reader := &hybridReaderStub{models: []ModelTokenUsageRow{{UsageDate: today, Model: "both", UsedTokens: 30}, {UsageDate: today, Model: "redis", UsedTokens: 40}}}
	repair := &hybridRepairStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, repair)
	svc.SetNowForTest(func() time.Time { return today })
	report, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{StartDate: today, EndDate: today, Page: 1, PageSize: 2, SortBy: "used_tokens", SortOrder: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 3 || report.UsedTokens != 90 || len(report.Items) != 2 || report.Items[0].Model != "redis" || report.Items[1].Model != "both" {
		t.Fatalf("report=%+v", report)
	}
	if len(repair.rows) != 1 || repair.rows[0].Model != "mysql" {
		t.Fatalf("repair=%+v", repair.rows)
	}
}

func TestModelTokenUsageHybridNonexistentFilterIsEmpty(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{}
	reader := &hybridReaderStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	report, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{Model: "missing", StartDate: today, EndDate: today, Page: 1, PageSize: 20, SortBy: "model", SortOrder: "asc"})
	if err != nil || report.Total != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}
