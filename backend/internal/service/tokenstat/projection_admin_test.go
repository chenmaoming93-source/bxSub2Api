package tokenstat

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	modernsqlite "modernc.org/sqlite"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
)

func TestProjectionLifecycleAndConfigPublication(t *testing.T) {
	sql.Register("sqlite3-tokenstat", &modernsqlite.Driver{})
	db, err := sql.Open("sqlite3-tokenstat", "file:tokenstat_projection?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	service := NewProjectionAdminService(client, redisClient)
	ctx := context.Background()
	input := ProjectionInput{
		Name:           "by user and model",
		DimensionCodes: []DimensionCode{DimensionUpstreamModel, DimensionUserID},
		MetricCodes:    []MetricCode{MetricTotalTokens},
	}

	projection, err := service.Create(ctx, input)
	require.NoError(t, err)
	require.Equal(t, []string{"user_id", "upstream_model"}, projection.DimensionCodes)

	input.DimensionCodes = []DimensionCode{DimensionUserID, DimensionUpstreamModel}
	_, err = service.Create(ctx, input)
	require.Error(t, err, "canonical signature must reject a reordered duplicate")

	_, err = service.Activate(ctx, projection.ID)
	require.ErrorIs(t, err, ErrInvalidProjectionTransition)
	_, err = service.Publish(ctx, projection.ID)
	require.NoError(t, err)
	active, err := service.Activate(ctx, projection.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectionStatusActive, active.Status)
	version, err := mini.Get(configVersionKey)
	require.NoError(t, err)
	require.Equal(t, "2", version)
	require.True(t, mini.Exists(configActiveKey))

	quota, err := service.CreateQuota(ctx, QuotaInput{
		Name: "user daily observation", DimensionCodes: []DimensionCode{DimensionUserID},
		DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42)},
		MetricCode:      MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 1000,
		Mode: QuotaModeObserve,
	})
	require.NoError(t, err)
	require.Equal(t, QuotaStatusPending, quota.Status)
	require.False(t, quota.EffectiveFrom.After(time.Now()), "new quotas must apply to the current natural period immediately")

	autoProjection, err := service.Get(ctx, quota.ProjectionID)
	require.NoError(t, err)
	require.Equal(t, ProjectionStatusDraft, autoProjection.Status)
	_, err = service.Publish(ctx, autoProjection.ID)
	require.NoError(t, err)
	_, err = service.Activate(ctx, autoProjection.ID)
	require.NoError(t, err)
	quota, err = service.client.TokenStatQuotaRule.Get(ctx, quota.ID)
	require.NoError(t, err)
	require.Equal(t, QuotaStatusEnabled, quota.Status)

	_, err = service.Disable(ctx, autoProjection.ID)
	require.Error(t, err, "an enabled quota must protect its projection")
	_, err = service.SetQuotaStatus(ctx, quota.ID, false)
	require.NoError(t, err)
	_, err = service.Disable(ctx, autoProjection.ID)
	require.NoError(t, err)
	reactivated, err := service.Activate(ctx, autoProjection.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectionStatusActive, reactivated.Status)
	require.Nil(t, reactivated.DisabledAt)
	require.NotNil(t, reactivated.EnabledAt)
}
