package tokenstat

import (
	"context"
	"database/sql"
	"fmt"
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
	for id, department := range []string{"研发部", "产品部", "产品部"} {
		_, err = client.User.Create().SetEmail(fmt.Sprintf("user-%d@example.com", id+1)).SetPasswordHash("hash").SetDepartment(department).Save(ctx)
		require.NoError(t, err)
	}

	projection, err := service.Create(ctx, ProjectionInput{
		Name: "user model", DimensionCodes: []DimensionCode{DimensionUserID, DimensionUpstreamModel, DimensionDepartment},
		MetricCodes: []MetricCode{MetricTotalTokens},
	})
	require.NoError(t, err)
	enabledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.TokenStatProjection.UpdateOneID(projection.ID).
		SetStatus(ProjectionStatusActive).SetEnabledAt(enabledAt).Save(ctx)
	require.NoError(t, err)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i, item := range []struct {
		user       int64
		model      string
		department string
		value      int64
	}{
		{1, "model-a", "研发部", 100},
		{2, "model-a", "未设置", 200},
		{1, "model-b", "研发部", 50},
	} {
		hash := make([]byte, 16)
		hash[0] = byte(i + 1)
		_, err = client.TokenStatAggregate.Create().
			SetPeriodType(string(PeriodDay)).SetPeriodStart(start).SetPeriodEnd(start.AddDate(0, 0, 1)).
			SetProjectionID(projection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"user_id": item.user, "upstream_model": item.model, "department": item.department}).
			SetMetricCode(string(MetricTotalTokens)).SetMetricValue(item.value).SetSourceVersion(1).
			SetUserID(item.user).SetUpstreamModel(item.model).SetDepartment(item.department).SetLastSyncedAt(start.AddDate(0, 0, 2)).
			Save(ctx)
		require.NoError(t, err)
	}

	departmentProjection, err := service.Create(ctx, ProjectionInput{
		Name: "department", DimensionCodes: []DimensionCode{DimensionDepartment},
		MetricCodes: []MetricCode{MetricTotalTokens},
	})
	require.NoError(t, err)
	_, err = client.TokenStatProjection.UpdateOneID(departmentProjection.ID).
		SetStatus(ProjectionStatusActive).SetEnabledAt(enabledAt).Save(ctx)
	require.NoError(t, err)
	for i, item := range []struct {
		department string
		value      int64
	}{
		{"研发部", 150},
		{"未设置", 200},
	} {
		hash := make([]byte, 16)
		hash[0] = byte(10 + i)
		_, err = client.TokenStatAggregate.Create().
			SetPeriodType(string(PeriodDay)).SetPeriodStart(start).SetPeriodEnd(start.AddDate(0, 0, 1)).
			SetProjectionID(departmentProjection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"department": item.department}).
			SetMetricCode(string(MetricTotalTokens)).SetMetricValue(item.value).SetSourceVersion(1).
			SetDepartment(item.department).SetLastSyncedAt(start.AddDate(0, 0, 2)).Save(ctx)
		require.NoError(t, err)
	}

	userProjection, err := service.Create(ctx, ProjectionInput{
		Name: "user totals", DimensionCodes: []DimensionCode{DimensionUserID},
		MetricCodes: []MetricCode{MetricTotalTokens},
	})
	require.NoError(t, err)
	_, err = client.TokenStatProjection.UpdateOneID(userProjection.ID).
		SetStatus(ProjectionStatusActive).SetEnabledAt(enabledAt).Save(ctx)
	require.NoError(t, err)
	for i, item := range []struct {
		user  int64
		value int64
	}{
		{1, 150},
		{2, 200},
	} {
		hash := make([]byte, 16)
		hash[0] = byte(20 + i)
		_, err = client.TokenStatAggregate.Create().
			SetPeriodType(string(PeriodDay)).SetPeriodStart(start).SetPeriodEnd(start.AddDate(0, 0, 1)).
			SetProjectionID(userProjection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"user_id": item.user}).
			SetMetricCode(string(MetricTotalTokens)).SetMetricValue(item.value).SetSourceVersion(1).
			SetUserID(item.user).SetLastSyncedAt(start.AddDate(0, 0, 2)).Save(ctx)
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

	departmentResult, err := service.QueryUsage(ctx, UsageQueryInput{
		ProjectionID: projection.ID, MetricCode: MetricTotalTokens, PeriodType: PeriodDay,
		Start: start, End: start.AddDate(0, 0, 2),
		Filters: map[DimensionCode]DimensionValue{DimensionDepartment: StringValue("未设置")},
		GroupBy: []DimensionCode{DimensionDepartment}, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(200), departmentResult.Summary)
	require.Equal(t, "未设置", departmentResult.Rows[0].Dimensions[DimensionDepartment].String)

	departmentSummary, err := service.QueryDepartmentUsage(ctx, start, start.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Equal(t, int64(350), departmentSummary.Summary)
	require.Len(t, departmentSummary.Rows, 2)
	require.Equal(t, "产品部", departmentSummary.Rows[0].Department)
	require.Equal(t, int64(200), departmentSummary.Rows[0].TotalTokens)
	require.Equal(t, int64(2), departmentSummary.Rows[0].UserCount)
	require.Equal(t, float64(100), departmentSummary.Rows[0].AverageTokens)
	require.Equal(t, "研发部", departmentSummary.Rows[1].Department)
	require.Equal(t, int64(150), departmentSummary.Rows[1].TotalTokens)
	require.Equal(t, int64(1), departmentSummary.Rows[1].UserCount)
	require.Equal(t, float64(150), departmentSummary.Rows[1].AverageTokens)

	usersResult, err := service.QueryDepartmentUserUsage(ctx, start, start.AddDate(0, 0, 2), "产品部", 1, 50)
	require.NoError(t, err)
	require.Equal(t, int64(200), usersResult.DepartmentTotalTokens)
	require.Equal(t, 2, usersResult.Total)
	require.Len(t, usersResult.Rows, 2)
	require.Equal(t, int64(2), usersResult.Rows[0].UserID)
	require.Equal(t, int64(200), usersResult.Rows[0].TotalTokens)
	require.Equal(t, int64(3), usersResult.Rows[1].UserID)
	require.Equal(t, int64(0), usersResult.Rows[1].TotalTokens)

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
