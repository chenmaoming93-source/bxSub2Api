package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGetRouteConcurrencyLimitPrefersScheduleHashAndFallsBackToLegacy(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	ctx := context.Background()
	cache := NewConcurrencyCache(rdb, 15, 60)
	routeCache, ok := cache.(service.RouteConcurrencyCache)
	require.True(t, ok)

	require.NoError(t, rdb.Set(ctx, routeConfigKeyPrefix+"group:1|test|2", "20", 0).Err())
	limit, hit, err := routeCache.GetRouteConcurrencyLimit(ctx, "group:1|test|2")
	require.NoError(t, err)
	require.True(t, hit)
	require.NotNil(t, limit)
	require.Equal(t, 20, *limit)

	require.NoError(t, rdb.HSet(ctx, routeScheduleKeyPrefix+"group:1|test|2", "limit", "50", "updated_at", time.Now().Format(time.RFC3339Nano)).Err())
	limit, hit, err = routeCache.GetRouteConcurrencyLimit(ctx, "group:1|test|2")
	require.NoError(t, err)
	require.True(t, hit)
	require.NotNil(t, limit)
	require.Equal(t, 50, *limit)

	require.NoError(t, rdb.HSet(ctx, routeScheduleKeyPrefix+"group:1|test|2", "limit", "unlimited").Err())
	limit, hit, err = routeCache.GetRouteConcurrencyLimit(ctx, "group:1|test|2")
	require.NoError(t, err)
	require.True(t, hit)
	require.Nil(t, limit)

	require.NoError(t, rdb.HSet(ctx, routeScheduleKeyPrefix+"group:1|test|2", "other", "value").Err())
	// Removing only limit makes the current Hash unusable and restores legacy fallback.
	require.NoError(t, rdb.HDel(ctx, routeScheduleKeyPrefix+"group:1|test|2", "limit").Err())
	limit, hit, err = routeCache.GetRouteConcurrencyLimit(ctx, "group:1|test|2")
	require.NoError(t, err)
	require.True(t, hit)
	require.NotNil(t, limit)
	require.Equal(t, 20, *limit)
}

func TestGetRouteConcurrencyLimitMissingBothKeepsLegacyMiss(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := NewConcurrencyCache(rdb, 15, 60)
	routeCache, ok := cache.(service.RouteConcurrencyCache)
	require.True(t, ok)

	limit, hit, err := routeCache.GetRouteConcurrencyLimit(context.Background(), "group:missing")
	require.NoError(t, err)
	require.False(t, hit)
	require.Nil(t, limit)
	keys, err := rdb.Keys(context.Background(), routeScheduleKeyPrefix+"*").Result()
	require.NoError(t, err)
	require.Empty(t, keys, "request-side lookup must not create schedule keys")
}

func TestGetRouteConcurrencyLimitBatchPrefersScheduleAndPreservesMiss(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := NewConcurrencyCache(rdb, 15, 60)
	batchCache, ok := cache.(service.RouteConcurrencyLimitBatchCache)
	require.True(t, ok)
	ctx := context.Background()

	require.NoError(t, rdb.Set(ctx, routeConfigKeyPrefix+"group:1|legacy|1", "20", 0).Err())
	require.NoError(t, rdb.Set(ctx, routeConfigKeyPrefix+"group:1|missing|1", "30", 0).Err())
	require.NoError(t, rdb.HSet(ctx, routeScheduleKeyPrefix+"group:1|scheduled|1", "limit", "unlimited").Err())

	limits, err := batchCache.GetRouteConcurrencyLimitBatch(ctx, []string{
		"group:1|scheduled|1",
		"group:1|legacy|1",
		"group:1|missing|1",
	})
	require.NoError(t, err)
	require.True(t, limits["group:1|scheduled|1"].Hit)
	require.Nil(t, limits["group:1|scheduled|1"].Limit)
	require.True(t, limits["group:1|legacy|1"].Hit)
	require.Equal(t, 20, *limits["group:1|legacy|1"].Limit)
	require.True(t, limits["group:1|missing|1"].Hit)
	require.Equal(t, 30, *limits["group:1|missing|1"].Limit)

	limits, err = batchCache.GetRouteConcurrencyLimitBatch(ctx, []string{"group:1|not-found|1"})
	require.NoError(t, err)
	require.False(t, limits["group:1|not-found|1"].Hit)
	require.Nil(t, limits["group:1|not-found|1"].Limit)
}

func TestRouteScheduleCacheWritesAndDeletesHash(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := NewConcurrencyCache(rdb, 15, 60)
	scheduleCache, ok := cache.(service.RouteScheduleCache)
	require.True(t, ok)
	ctx := context.Background()
	updatedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	limit := 50
	require.NoError(t, scheduleCache.SetRouteScheduleConcurrencyLimit(ctx, "group:1|test|2", &limit, updatedAt))
	values, err := rdb.HGetAll(ctx, routeScheduleKeyPrefix+"group:1|test|2").Result()
	require.NoError(t, err)
	require.Equal(t, "50", values["limit"])
	require.Equal(t, updatedAt.Format(time.RFC3339Nano), values["updated_at"])
	require.NoError(t, scheduleCache.DeleteRouteScheduleConcurrencyLimit(ctx, "group:1|test|2"))
	exists, err := rdb.Exists(ctx, routeScheduleKeyPrefix+"group:1|test|2").Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}
