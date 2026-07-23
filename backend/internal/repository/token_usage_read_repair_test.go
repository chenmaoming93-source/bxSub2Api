package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTokenUsageReadRepairIsAtomicIdempotentAndKeepsTTL(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repairer := NewRedisTokenUsageReadRepairer(client, 2, 2)
	day := time.Now().In(tokenStatisticsLocation)
	rows := []service.ModelTokenUsageRow{{UsageDate: day, Model: "missing", UsedTokens: 10}, {UsageDate: day, Model: "concurrent", UsedTokens: 20}}
	key, _ := TokenStatisticsKey(TokenStatisticsModel, day)
	concurrentField, _ := EncodeModelTokenStatisticsField("concurrent")
	server.HSet(key, concurrentField, "99")

	first, err := repairer.RepairModelUsage(context.Background(), day, rows)
	if err != nil || first.Repaired != 1 || first.Skipped != 1 {
		t.Fatalf("first repair=%+v err=%v", first, err)
	}
	missingField, _ := EncodeModelTokenStatisticsField("missing")
	if got := server.HGet(key, missingField); got != "10" {
		t.Fatalf("absolute repair value=%q", got)
	}
	if got := server.HGet(key, concurrentField); got != "99" {
		t.Fatalf("existing value overwritten=%q", got)
	}
	if ttl := server.TTL(key); ttl <= 0 {
		t.Fatalf("missing TTL: %v", ttl)
	}

	second, err := repairer.RepairModelUsage(context.Background(), day, rows)
	if err != nil || second.Repaired != 0 || second.Skipped != 2 {
		t.Fatalf("repeat repair=%+v err=%v", second, err)
	}
	if got := server.HGet(key, missingField); got != "10" {
		t.Fatalf("repeat accumulated value=%q", got)
	}
}

func TestTokenUsageReadRepairSupportsAllDimensionsAndPartialValidationFailure(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repairer := NewRedisTokenUsageReadRepairer(client, 2, 10)
	day := time.Now().In(tokenStatisticsLocation)

	routes, err := repairer.RepairRouteUsage(context.Background(), day, []service.RouteTokenUsageRow{{GroupID: 1, RouteAlias: "fast", UpstreamModel: "gpt", UsedTokens: 3}})
	if err != nil || routes.Repaired != 1 {
		t.Fatalf("route repair=%+v err=%v", routes, err)
	}
	users, err := repairer.RepairUserModelUsage(context.Background(), day, []service.UserTokenUsageRow{{UserID: 2, Model: "gpt", UsedTokens: 4}})
	if err != nil || users.Repaired != 1 {
		t.Fatalf("user repair=%+v err=%v", users, err)
	}
	partial, err := repairer.RepairModelUsage(context.Background(), day, []service.ModelTokenUsageRow{{Model: "valid", UsedTokens: 1}, {Model: "", UsedTokens: 2}})
	if err == nil || partial.Repaired != 1 || partial.Failed != 1 {
		t.Fatalf("partial repair=%+v err=%v", partial, err)
	}
}
