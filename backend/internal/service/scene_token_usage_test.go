package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojectionmetric"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/stretchr/testify/require"
	modernsqlite "modernc.org/sqlite"
)

func TestSceneAccountDailyUsageServiceAggregatesByGroupAndAccountModel(t *testing.T) {
	driverName := "sqlite3-scene-usage-" + t.Name()
	sql.Register(driverName, &modernsqlite.Driver{})
	db, err := sql.Open(driverName, "file:scene_usage?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))

	groupA, err := client.Group.Create().SetName("technical-a").SetSceneName("same scene").SetPlatform(PlatformOpenAI).SetRateMultiplier(1).Save(ctx)
	require.NoError(t, err)
	groupB, err := client.Group.Create().SetName("technical-b").SetSceneName("same scene").SetPlatform(PlatformOpenAI).SetRateMultiplier(1).Save(ctx)
	require.NoError(t, err)
	accountA, err := client.Account.Create().SetName("account-a").SetPlatform(PlatformOpenAI).SetType("apikey").Save(ctx)
	require.NoError(t, err)
	accountB, err := client.Account.Create().SetName("account-b").SetPlatform(PlatformOpenAI).SetType("apikey").Save(ctx)
	require.NoError(t, err)

	projectionService := tokenstat.NewProjectionAdminService(client, nil)
	projection, err := projectionService.Create(ctx, tokenstat.ProjectionInput{
		Name:           "scene daily",
		DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionGroupID, tokenstat.DimensionAccountID, tokenstat.DimensionUpstreamModel},
		MetricCodes:    []tokenstat.MetricCode{tokenstat.MetricTotalTokens},
	})
	require.NoError(t, err)
	enabledAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.TokenStatProjection.UpdateOneID(projection.ID).SetStatus(tokenstat.ProjectionStatusActive).SetEnabledAt(enabledAt).Save(ctx)
	require.NoError(t, err)
	_, err = client.TokenStatProjectionMetric.Update().Where(tokenstatprojectionmetric.ProjectionIDEQ(projection.ID)).SetStatus(tokenstat.ProjectionStatusActive).Save(ctx)
	require.NoError(t, err)

	day := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	addAggregate := func(groupID, accountID int64, model string, value int64) {
		hash := []byte{byte(groupID), byte(accountID), byte(value)}
		_, saveErr := client.TokenStatAggregate.Create().
			SetPeriodType(string(tokenstat.PeriodDay)).SetPeriodStart(day).SetPeriodEnd(day.AddDate(0, 0, 1)).
			SetProjectionID(projection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"group_id": groupID, "account_id": accountID, "upstream_model": model}).
			SetMetricCode(string(tokenstat.MetricTotalTokens)).SetMetricValue(value).SetSourceVersion(1).
			SetGroupID(groupID).SetAccountID(accountID).SetUpstreamModel(model).SetLastSyncedAt(day.Add(time.Hour)).Save(ctx)
		require.NoError(t, saveErr)
	}
	addAggregate(groupA.ID, accountA.ID, "model-a", 100)
	addAggregate(groupA.ID, accountA.ID, "model-b", 50)
	addAggregate(groupB.ID, accountB.ID, "model-a", 200)

	svc := NewSceneAccountDailyUsageService(client, projectionService, nil, time.UTC)
	result, err := svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2026-08-02"})
	require.NoError(t, err)
	require.Equal(t, int64(projection.ID), result.ProjectionID)
	require.False(t, result.Complete)
	require.Len(t, result.Days, 1)
	require.Len(t, result.Days[0].Scenes, 2)
	require.Equal(t, int64(150), result.Days[0].Scenes[0].TotalTokens)
	require.Equal(t, "same scene", result.Days[0].Scenes[0].SceneName)
	require.Len(t, result.Days[0].Scenes[0].Accounts, 2)
	require.Equal(t, "account-a", result.Days[0].Scenes[0].Accounts[0].AccountName)
	require.Equal(t, int64(200), result.Days[0].Scenes[1].TotalTokens)
	require.Equal(t, "technical-b", result.Days[0].Scenes[1].GroupName)

	filtered, err := svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2026-08-02", GroupName: "technical-b"})
	require.NoError(t, err)
	require.Len(t, filtered.Days, 1)
	require.Len(t, filtered.Days[0].Scenes, 1)
	require.Equal(t, "technical-b", filtered.Days[0].Scenes[0].GroupName)

	emptyResult, err := svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-03", EndDate: "2026-08-03"})
	require.NoError(t, err)
	require.Empty(t, emptyResult.Days)
}

func TestSceneAccountDailyUsageServiceRejectsInvalidConfigurationAndDate(t *testing.T) {
	driverName := "sqlite3-scene-usage-errors-" + t.Name()
	sql.Register(driverName, &modernsqlite.Driver{})
	db, err := sql.Open(driverName, "file:scene_usage_errors?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))
	projectionService := tokenstat.NewProjectionAdminService(client, nil)
	svc := NewSceneAccountDailyUsageService(client, projectionService, nil, time.UTC)

	_, err = svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2027-08-03"})
	require.ErrorContains(t, err, "SCENE_USAGE_INVALID_DATE_RANGE")
	_, err = svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2026-08-02"})
	require.ErrorContains(t, err, "SCENE_USAGE_STATISTICS_NOT_CONFIGURED")

	_, err = projectionService.Create(ctx, tokenstat.ProjectionInput{
		Name:           "inactive scene daily",
		DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionGroupID, tokenstat.DimensionAccountID, tokenstat.DimensionUpstreamModel},
		MetricCodes:    []tokenstat.MetricCode{tokenstat.MetricTotalTokens},
	})
	require.NoError(t, err)
	_, err = svc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2026-08-02"})
	require.ErrorContains(t, err, "SCENE_USAGE_STATISTICS_NOT_ACTIVE")

	disabledRuntime := tokenstat.NewRuntimeController(nil, &config.Config{Gateway: config.GatewayConfig{DynamicTokenStatistics: config.GatewayDynamicTokenStatisticsConfig{Enabled: false}}})
	disabledSvc := NewSceneAccountDailyUsageService(client, projectionService, disabledRuntime, time.UTC)
	_, err = disabledSvc.QuerySceneAccountDailyUsage(ctx, SceneAccountDailyUsageInput{StartDate: "2026-08-02", EndDate: "2026-08-02"})
	require.ErrorContains(t, err, "TOKEN_STATISTICS_DISABLED")
	// Restore the package-level compatibility controller for other service tests.
	tokenstat.NewRuntimeController(nil, &config.Config{Gateway: config.GatewayConfig{DynamicTokenStatistics: config.GatewayDynamicTokenStatisticsConfig{Enabled: true}}})
}
