package tokenstat

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	modernsqlite "modernc.org/sqlite"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
)

func TestDynamicTokenStatQueryValidationAggregationAndPagination(t *testing.T) {
	sql.Register("sqlite3-tokenstat-query", &modernsqlite.Driver{})
	db, err := sql.Open("sqlite3-tokenstat-query", "file:tokenstat_query?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))
	service := NewProjectionAdminService(client, nil)
	ctx := context.Background()

	projection, err := service.Create(ctx, ProjectionInput{
		Name: "user model", DimensionCodes: []DimensionCode{DimensionUserID, DimensionUpstreamModel},
		MetricCodes: []MetricCode{MetricTotalTokens},
	})
	require.NoError(t, err)
	enabledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.TokenStatProjection.UpdateOneID(projection.ID).
		SetStatus(ProjectionStatusActive).SetEnabledAt(enabledAt).Save(ctx)
	require.NoError(t, err)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i, item := range []struct {
		user  int64
		model string
		value int64
	}{
		{1, "model-a", 100},
		{2, "model-a", 200},
		{1, "model-b", 50},
	} {
		hash := make([]byte, 16)
		hash[0] = byte(i + 1)
		_, err = client.TokenStatAggregate.Create().
			SetPeriodType(string(PeriodDay)).SetPeriodStart(start).SetPeriodEnd(start.AddDate(0, 0, 1)).
			SetProjectionID(projection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"user_id": item.user, "upstream_model": item.model}).
			SetMetricCode(string(MetricTotalTokens)).SetMetricValue(item.value).SetSourceVersion(1).
			SetUserID(item.user).SetUpstreamModel(item.model).SetLastSyncedAt(start.AddDate(0, 0, 2)).
			Save(ctx)
		require.NoError(t, err)
	}

	result, err := service.QueryUsage(ctx, UsageQueryInput{
		ProjectionID: projection.ID, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		Start: start, End: start.AddDate(0, 0, 2),
		Filters: map[DimensionCode]DimensionValue{DimensionUpstreamModel: StringValue("model-a")},
		GroupBy: []DimensionCode{DimensionUserID}, Sort: "value_desc", Page: 1, PageSize: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(300), result.Summary)
	require.Equal(t, 2, result.Total)
	require.Len(t, result.Rows, 1)
	require.Equal(t, int64(200), result.Rows[0].Value)
	require.Equal(t, int64(2), result.Rows[0].Dimensions[DimensionUserID].Int64)
	require.Equal(t, "mysql_eventual", result.Consistency)
	require.NotNil(t, result.LastSyncedAt)

	_, err = service.QueryUsage(ctx, UsageQueryInput{
		ProjectionID: projection.ID, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		Start: start, End: start.AddDate(0, 0, 2), GroupBy: []DimensionCode{DimensionAccountID},
	})
	require.ErrorContains(t, err, "not in projection")

	_, err = service.QueryUsage(ctx, UsageQueryInput{
		ProjectionID: projection.ID, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		Start: start, End: start.AddDate(0, 0, 2), Sort: "metric_value; DROP TABLE users",
	})
	require.ErrorContains(t, err, "invalid sort")

	_, err = service.QueryUsage(ctx, UsageQueryInput{
		ProjectionID: projection.ID, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		Start: start, End: start.AddDate(0, 0, 367),
	})
	require.ErrorContains(t, err, "exceeds")
}
