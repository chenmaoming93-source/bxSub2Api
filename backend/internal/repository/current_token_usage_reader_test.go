package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestCurrentTokenUsageReadAllTypes(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	reader := NewRedisCurrentTokenUsageReader(client, 2)
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)

	modelField, _ := EncodeModelTokenStatisticsField("gpt")
	routeField, _ := EncodeGroupCandidateTokenStatisticsField(7, "fast", "gpt-upstream")
	userField, _ := EncodeUserModelTokenStatisticsField(9, "gpt")
	for kind, values := range map[TokenStatisticsType]map[string]string{
		TokenStatisticsModel: {modelField: "11"}, TokenStatisticsGroupCandidate: {routeField: "22"}, TokenStatisticsUserModel: {userField: "33"},
	} {
		key, _ := TokenStatisticsKey(kind, day)
		for field, value := range values {
			server.HSet(key, field, value)
		}
	}

	models, err := reader.ReadModelUsage(context.Background(), day, nil)
	if err != nil || len(models.Rows) != 1 || models.Rows[0].Model != "gpt" || models.Rows[0].UsedTokens != 11 {
		t.Fatalf("model read: result=%+v err=%v", models, err)
	}
	routes, err := reader.ReadRouteUsage(context.Background(), day, nil)
	if err != nil || len(routes.Rows) != 1 || routes.Rows[0].GroupID != 7 || routes.Rows[0].RouteAlias != "fast" || routes.Rows[0].UsedTokens != 22 {
		t.Fatalf("route read: result=%+v err=%v", routes, err)
	}
	users, err := reader.ReadUserModelUsage(context.Background(), day, nil)
	if err != nil || len(users.Rows) != 1 || users.Rows[0].UserID != 9 || users.Rows[0].UsedTokens != 33 {
		t.Fatalf("user read: result=%+v err=%v", users, err)
	}
}

func TestCurrentTokenUsageReadEmptyAndFiltered(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	reader := NewRedisCurrentTokenUsageReader(client, 10)
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)

	empty, err := reader.ReadModelUsage(context.Background(), day, nil)
	if err != nil || len(empty.Rows) != 0 || empty.InvalidEntries != 0 {
		t.Fatalf("empty hash was not a successful empty result: %+v err=%v", empty, err)
	}
	field, _ := EncodeModelTokenStatisticsField("wanted")
	other, _ := EncodeModelTokenStatisticsField("other")
	key, _ := TokenStatisticsKey(TokenStatisticsModel, day)
	server.HSet(key, field, "5")
	server.HSet(key, other, "7")
	filtered, err := reader.ReadModelUsage(context.Background(), day, []string{"wanted"})
	if err != nil || len(filtered.Rows) != 1 || filtered.Rows[0].Model != "wanted" {
		t.Fatalf("filtered HMGET read: %+v err=%v", filtered, err)
	}
}

func TestCurrentTokenUsageReadSkipsInvalidEntries(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	reader := NewRedisCurrentTokenUsageReader(client, 10)
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	valid, _ := EncodeModelTokenStatisticsField("valid")
	negative, _ := EncodeModelTokenStatisticsField("negative")
	key, _ := TokenStatisticsKey(TokenStatisticsModel, day)
	server.HSet(key, valid, "8")
	server.HSet(key, negative, "-1")
	server.HSet(key, "not-a-field", "9")

	result, err := reader.ReadModelUsage(context.Background(), day, nil)
	if err != nil || len(result.Rows) != 1 || result.Rows[0].Model != "valid" || result.InvalidEntries != 2 {
		t.Fatalf("invalid entries: result=%+v err=%v", result, err)
	}
}

func TestCurrentTokenUsageReadReturnsRedisFailure(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	reader := NewRedisCurrentTokenUsageReader(client, 10)
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	result, err := reader.ReadRouteUsage(context.Background(), day, []service.RouteTokenUsageRow{{GroupID: 1, RouteAlias: "a", UpstreamModel: "m"}})
	if err == nil || len(result.Rows) != 0 {
		t.Fatalf("Redis failure was not distinct: result=%+v err=%v", result, err)
	}
}
