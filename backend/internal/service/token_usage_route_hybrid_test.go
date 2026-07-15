package service

import (
	"context"
	"testing"
	"time"
)

func TestRouteTokenUsageHistoricalDoesNotReadRedis(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{routeHistory: []RouteTokenUsageRow{{UsageDate: today.AddDate(0, 0, -1), GroupID: 1, GroupName: "g", RouteAlias: "a", UpstreamModel: "m"}}}
	reader := &hybridReaderStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	_, err := svc.GetRouteReport(context.Background(), RouteTokenUsageReportQuery{TokenUsageReportQuery: TokenUsageReportQuery{StartDate: today.AddDate(0, 0, -1), EndDate: today.AddDate(0, 0, -1), Page: 1, PageSize: 20, SortBy: "usage_date", SortOrder: "asc"}})
	if err != nil || reader.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, reader.calls)
	}
}

func TestRouteTokenUsageHybridMetadataCompositeKeySortingAndOrphanFilter(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	limit := int64(100)
	priority := 2
	repo := &hybridModelRepoStub{routeToday: []RouteTokenUsageRow{{UsageDate: today, GroupID: 1, GroupName: "group", RouteAlias: "a|b", UpstreamModel: "c", UsedTokens: 10, DailyLimitTokens: &limit, Priority: &priority}, {UsageDate: today, GroupID: 1, GroupName: "group", RouteAlias: "mysql", UpstreamModel: "m", UsedTokens: 20}}}
	reader := &hybridReaderStub{routes: []RouteTokenUsageRow{{UsageDate: today, GroupID: 1, RouteAlias: "a", UpstreamModel: "b|c", UsedTokens: 99}, {UsageDate: today, GroupID: 1, RouteAlias: "a|b", UpstreamModel: "c", UsedTokens: 30}}}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	report, err := svc.GetRouteReport(context.Background(), RouteTokenUsageReportQuery{TokenUsageReportQuery: TokenUsageReportQuery{StartDate: today, EndDate: today, Page: 1, PageSize: 20, SortBy: "used_tokens,route_alias", SortOrder: "desc,asc"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.UsedTokens != 50 || report.Items[0].RouteAlias != "a|b" || report.Items[0].UsedTokens != 30 || report.Items[0].GroupName != "group" || report.Items[0].Priority == nil || *report.Items[0].Priority != 2 {
		t.Fatalf("report=%+v", report)
	}
}
