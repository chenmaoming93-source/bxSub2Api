package service

import (
	"context"
	"testing"
	"time"
)

func TestUserTokenUsageHistoricalDoesNotReadRedis(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{userHistory: []UserTokenUsageRow{{UsageDate: today.AddDate(0, 0, -1), UserID: 1, Model: "m"}}}
	reader := &hybridReaderStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	_, err := svc.GetUserReport(context.Background(), UserTokenUsageReportQuery{TokenUsageReportQuery: TokenUsageReportQuery{StartDate: today.AddDate(0, 0, -1), EndDate: today.AddDate(0, 0, -1), Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"}})
	if err != nil || reader.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, reader.calls)
	}
}
func TestUserTokenUsageHybridRealtimeDeletedAndSummary(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	limit := int64(10)
	repo := &hybridModelRepoStub{userToday: []UserTokenUsageRow{{UsageDate: today, UserID: 1, Email: "a@b", Model: "gpt", UsedTokens: 2, DailyLimitTokens: &limit}, {UsageDate: today, UserID: 2, Email: "deleted@b", Model: "gpt", UserDeleted: true, UsedTokens: 4}}}
	reader := &hybridReaderStub{users: []UserTokenUsageRow{{UsageDate: today, UserID: 1, Model: "gpt", UsedTokens: 9}, {UsageDate: today, UserID: 2, Model: "gpt", UsedTokens: 8}, {UsageDate: today, UserID: 3, Model: "redis", UsedTokens: 7}}}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	q := UserTokenUsageReportQuery{TokenUsageReportQuery: TokenUsageReportQuery{StartDate: today, EndDate: today, Page: 1, PageSize: 20, SortBy: "used_tokens", SortOrder: "desc"}}
	report, err := svc.GetUserReport(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.UsedTokens != 16 || report.Items[0].UserID != 1 || report.Items[0].Status != "warning" {
		t.Fatalf("report=%+v", report)
	}
	q.IncludeDeleted = true
	included, err := svc.GetUserReport(context.Background(), q)
	if err != nil || included.Total != 3 || included.UsedTokens != 24 {
		t.Fatalf("included=%+v err=%v", included, err)
	}
}
func TestUserTokenUsageHybridNonexistentCombination(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{}
	reader := &hybridReaderStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	r, e := svc.GetUserReport(context.Background(), UserTokenUsageReportQuery{UserID: 999, TokenUsageReportQuery: TokenUsageReportQuery{Model: "none", StartDate: today, EndDate: today, Page: 1, PageSize: 20, SortBy: "model", SortOrder: "asc"}})
	if e != nil || r.Total != 0 {
		t.Fatalf("report=%+v err=%v", r, e)
	}
}
