package service

import (
	"context"
	"testing"
	"time"
)

func TestTokenUsageOptionsIncludeRedisOnlyAndRespectParents(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{options: []TokenUsageOption{{Label: "mysql", Model: "mysql"}}}
	reader := &hybridReaderStub{models: []ModelTokenUsageRow{{UsageDate: today, Model: "redis-model", UsedTokens: 9}}, routes: []RouteTokenUsageRow{{UsageDate: today, GroupID: 7, RouteAlias: "fast", UpstreamModel: "redis-up", UsedTokens: 8}}, users: []UserTokenUsageRow{{UsageDate: today, UserID: 9, Model: "redis-user-model", UsedTokens: 7}}}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	models, err := svc.SearchOptions(context.Background(), "models", 0, "", 20)
	if err != nil || len(models) != 2 {
		t.Fatalf("models=%+v err=%v", models, err)
	}
	repo.options = nil
	routes, err := svc.SearchOptions(context.Background(), "routes", 7, "fa", 20)
	if err != nil || len(routes) != 1 || routes[0].RouteAlias != "fast" {
		t.Fatalf("routes=%+v err=%v", routes, err)
	}
	routeModels, err := svc.SearchOptions(context.Background(), "route_models", 7, "fast\x00redis", 20)
	if err != nil || len(routeModels) != 1 || routeModels[0].Model != "redis-up" {
		t.Fatalf("route_models=%+v err=%v", routeModels, err)
	}
	userModels, err := svc.SearchOptions(context.Background(), "user_models", 9, "redis", 20)
	if err != nil || len(userModels) != 1 || userModels[0].Model != "redis-user-model" {
		t.Fatalf("user_models=%+v err=%v", userModels, err)
	}
}
func TestDefaultTargetTodayUsesMergedHighestAndHistoricalSkipsRedis(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{today: []ModelTokenUsageRow{{UsageDate: today, Model: "mysql", UsedTokens: 20}}, defaultTarget: &TokenUsageOption{Label: "historical", Model: "historical"}}
	reader := &hybridReaderStub{models: []ModelTokenUsageRow{{UsageDate: today, Model: "redis", UsedTokens: 30}}}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, &hybridRepairStub{})
	svc.SetNowForTest(func() time.Time { return today })
	target, err := svc.DefaultTarget(context.Background(), "model", today)
	if err != nil || target == nil || target.Model != "redis" || repo.defaultCalls != 0 {
		t.Fatalf("target=%+v calls=%d err=%v", target, repo.defaultCalls, err)
	}
	calls := reader.calls
	historical, err := svc.DefaultTarget(context.Background(), "model", today.AddDate(0, 0, -1))
	if err != nil || historical.Model != "historical" || reader.calls != calls || repo.defaultCalls != 1 {
		t.Fatalf("historical=%+v reader=%d/%d repo=%d err=%v", historical, reader.calls, calls, repo.defaultCalls, err)
	}
}
