package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type refreshRepoStub struct {
	candidates []service.ModelRouteConcurrencyScheduleCandidate
}

func (r refreshRepoStub) ListModelRouteConcurrencyScheduleCandidates(context.Context) ([]service.ModelRouteConcurrencyScheduleCandidate, error) {
	return r.candidates, nil
}

func TestModelRouteConcurrencyScheduleRefreshMaterializesCurrentMinuteAndCleansStaleKeys(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := repository.NewConcurrencyCache(rdb, 15, 60)
	concurrency := service.NewConcurrencyService(cache)

	limit10 := 10
	limit50 := 50
	defaultLimit := 20
	refresher := service.NewModelRouteConcurrencyScheduleRefresher(
		refreshRepoStub{candidates: []service.ModelRouteConcurrencyScheduleCandidate{
			{
				GroupID: 1, RouteAlias: "DeepSeekV4-zlx", AccountID: 2,
				DefaultMaxConcurrency: &defaultLimit,
				Schedules: []service.ModelRouteConcurrencySchedule{
					{StartMinute: 0, EndMinute: 570, MaxConcurrency: &limit10},
					{StartMinute: 570, EndMinute: 1230, MaxConcurrency: &limit50},
				},
			},
			{
				GroupID: 1, RouteAlias: "unlimited-default", AccountID: 3,
				DefaultMaxConcurrency: nil,
				Schedules:             []service.ModelRouteConcurrencySchedule{{StartMinute: 0, EndMinute: 30, MaxConcurrency: &limit10}},
			},
		}},
		concurrency,
		&config.Config{Timezone: "Asia/Shanghai", Gateway: config.GatewayConfig{ModelRouteSchedule: config.ModelRouteScheduleConfig{RefreshLockTTLSeconds: 300, RefreshLockRenewIntervalSeconds: 30}}},
	)
	refresher.SetNowForTest(func() time.Time { return time.Date(2026, 8, 18, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)) })

	ctx := context.Background()
	require.NoError(t, rdb.Set(ctx, "concurrency:route-config:group:1|DeepSeekV4-zlx|2", "99", 0).Err())
	require.NoError(t, rdb.HSet(ctx, "concurrency:route-schedule:stale", "limit", "1").Err())

	result, err := refresher.Refresh(ctx, "scheduled")
	require.NoError(t, err)
	require.False(t, result.Skipped)
	require.Equal(t, 2, result.CandidateCount)
	require.Equal(t, 2, result.UpdatedCount)
	require.Equal(t, 1, result.DeletedCount)

	values, err := rdb.HGetAll(ctx, "concurrency:route-schedule:group:1|DeepSeekV4-zlx|2").Result()
	require.NoError(t, err)
	require.Equal(t, "50", values["limit"])
	values, err = rdb.HGetAll(ctx, "concurrency:route-schedule:group:1|unlimited-default|3").Result()
	require.NoError(t, err)
	require.Equal(t, "unlimited", values["limit"])
	require.NoError(t, rdb.Get(ctx, "concurrency:route-config:group:1|DeepSeekV4-zlx|2").Err())
	exists, err := rdb.Exists(ctx, "concurrency:route-schedule:stale").Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

func TestModelRouteConcurrencyScheduleRefreshLockIsExclusiveAndTokenSafe(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := repository.NewConcurrencyCache(rdb, 15, 60)
	concurrency := service.NewConcurrencyService(cache)
	ctx := context.Background()

	first, err := concurrency.TryAcquireRouteScheduleRefreshLock(ctx, "token-1", time.Minute)
	require.NoError(t, err)
	require.True(t, first)
	second, err := concurrency.TryAcquireRouteScheduleRefreshLock(ctx, "token-2", time.Minute)
	require.NoError(t, err)
	require.False(t, second)

	renewed, err := concurrency.RenewRouteScheduleRefreshLock(ctx, "token-2", time.Minute)
	require.NoError(t, err)
	require.False(t, renewed)
	renewed, err = concurrency.RenewRouteScheduleRefreshLock(ctx, "token-1", time.Minute)
	require.NoError(t, err)
	require.True(t, renewed)

	require.NoError(t, concurrency.ReleaseRouteScheduleRefreshLock(ctx, "token-2"))
	stillHeld, err := concurrency.TryAcquireRouteScheduleRefreshLock(ctx, "token-2", time.Minute)
	require.NoError(t, err)
	require.False(t, stillHeld)
	require.NoError(t, concurrency.ReleaseRouteScheduleRefreshLock(ctx, "token-1"))
	acquired, err := concurrency.TryAcquireRouteScheduleRefreshLock(ctx, "token-2", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestModelRouteConcurrencyScheduleRefreshConflictIsReportedWithoutWrites(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := repository.NewConcurrencyCache(rdb, 15, 60)
	concurrency := service.NewConcurrencyService(cache)
	limit := 10
	refresher := service.NewModelRouteConcurrencyScheduleRefresher(
		refreshRepoStub{candidates: []service.ModelRouteConcurrencyScheduleCandidate{{GroupID: 1, RouteAlias: "test", AccountID: 2, Schedules: []service.ModelRouteConcurrencySchedule{{StartMinute: 0, EndMinute: 1440, MaxConcurrency: &limit}}}}},
		concurrency,
		&config.Config{Timezone: "Asia/Shanghai"},
	)
	refresher.SetNowForTest(func() time.Time { return time.Date(2026, 8, 18, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)) })
	require.True(t, mustAcquire(t, concurrency, "held"))

	result, err := refresher.Refresh(context.Background(), "scheduled")
	require.ErrorIs(t, err, service.ErrModelRouteConcurrencyScheduleRefreshInProgress)
	require.True(t, result.Skipped)
	exists, err := rdb.Exists(context.Background(), "concurrency:route-schedule:group:1|test|2").Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

func mustAcquire(t *testing.T, concurrency *service.ConcurrencyService, token string) bool {
	t.Helper()
	ok, err := concurrency.TryAcquireRouteScheduleRefreshLock(context.Background(), token, time.Minute)
	require.NoError(t, err)
	return ok
}
