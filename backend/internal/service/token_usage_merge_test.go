package service

import (
	"fmt"
	"testing"
	"time"
)

func TestModelTokenUsageMerge(t *testing.T) {
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	limit := int64(100)
	mysql := []ModelTokenUsageRow{{UsageDate: day, Model: "both", UsedTokens: 10, DailyLimitTokens: &limit}, {UsageDate: day, Model: "mysql", UsedTokens: 20}}
	redis := []ModelTokenUsageRow{{UsageDate: day, Model: "both", UsedTokens: 30}, {UsageDate: day, Model: "redis", UsedTokens: 40}}

	got, repair := MergeModelTokenUsage(mysql, redis)
	if len(got) != 3 || len(repair) != 1 {
		t.Fatalf("got %d final rows and %d repair rows", len(got), len(repair))
	}
	byModel := indexRows(got, func(row ModelTokenUsageRow) string { return row.Model })
	if byModel["both"].UsedTokens != 30 || byModel["both"].DailyLimitTokens != &limit {
		t.Fatalf("overlap was not Redis-authoritative with MySQL metadata: %+v", byModel["both"])
	}
	if byModel["redis"].UsedTokens != 40 || repair[0].Model != "mysql" || repair[0].UsedTokens != 20 {
		t.Fatalf("Redis-only or MySQL-only semantics are incorrect: final=%+v repair=%+v", got, repair)
	}
	if empty, repairs := MergeModelTokenUsage(nil, nil); len(empty) != 0 || len(repairs) != 0 {
		t.Fatalf("empty merge returned data: final=%+v repair=%+v", empty, repairs)
	}
}

func TestRouteTokenUsageMerge(t *testing.T) {
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	limit, priority := int64(100), 2
	mysql := []RouteTokenUsageRow{{UsageDate: day, GroupID: 1, GroupName: "group", RouteAlias: "a|b", UpstreamModel: "c", UsedTokens: 10, DailyLimitTokens: &limit, Priority: &priority}}
	redis := []RouteTokenUsageRow{{UsageDate: day, GroupID: 1, RouteAlias: "a", UpstreamModel: "b|c", UsedTokens: 20}, {UsageDate: day, GroupID: 1, RouteAlias: "a|b", UpstreamModel: "c", UsedTokens: 30}}

	got, repair := MergeRouteTokenUsage(mysql, redis)
	if len(got) != 2 || len(repair) != 0 {
		t.Fatalf("structured route keys collided: final=%+v repair=%+v", got, repair)
	}
	if got[1].UsedTokens != 30 || got[1].GroupName != "group" || got[1].Priority != &priority || got[1].DailyLimitTokens != &limit {
		t.Fatalf("route overlap merge is incorrect: %+v", got[1])
	}
}

func TestUserTokenUsageMerge(t *testing.T) {
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	limit := int64(100)
	mysql := []UserTokenUsageRow{{UsageDate: day, UserID: 7, Model: "gpt", Email: "a@example.com", Username: "alice", UserDeleted: true, UsedTokens: 10, DailyLimitTokens: &limit}}
	redis := []UserTokenUsageRow{{UsageDate: day, UserID: 7, Model: "gpt", UsedTokens: 25}}

	got, repair := MergeUserTokenUsage(mysql, redis)
	if len(got) != 1 || len(repair) != 0 {
		t.Fatalf("unexpected user merge sizes: final=%+v repair=%+v", got, repair)
	}
	if got[0].UsedTokens != 25 || got[0].Username != "alice" || got[0].Email != "a@example.com" || !got[0].UserDeleted || got[0].DailyLimitTokens != &limit {
		t.Fatalf("user overlap merge is incorrect: %+v", got[0])
	}
}

func BenchmarkModelTokenUsageMerge10000(b *testing.B) {
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	mysql := make([]ModelTokenUsageRow, 10000)
	redis := make([]ModelTokenUsageRow, 10000)
	for i := range mysql {
		model := fmt.Sprintf("model-%05d", i)
		mysql[i] = ModelTokenUsageRow{UsageDate: day, Model: model, UsedTokens: int64(i)}
		redis[i] = ModelTokenUsageRow{UsageDate: day, Model: model, UsedTokens: int64(i + 1)}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MergeModelTokenUsage(mysql, redis)
	}
}
