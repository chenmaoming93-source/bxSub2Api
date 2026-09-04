package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	modernsqlite "modernc.org/sqlite"
)

func TestUsageHandlerDepartmentStatsUsesInclusiveEndDateAndReturnsSortedRows(t *testing.T) {
	driverName := "sqlite3-department-stats-handler"
	sql.Register(driverName, &modernsqlite.Driver{})
	db, err := sql.Open(driverName, "file:department_stats_handler?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))
	stats := tokenstat.NewProjectionAdminService(client, nil)
	ctx := context.Background()
	user1, err := client.User.Create().SetEmail("department-user-1@example.com").SetPasswordHash("hash").SetDepartment("研发部").Save(ctx)
	require.NoError(t, err)
	user2, err := client.User.Create().SetEmail("department-user-2@example.com").SetPasswordHash("hash").SetDepartment("未设置").Save(ctx)
	require.NoError(t, err)
	projection, err := stats.Create(ctx, tokenstat.ProjectionInput{
		Name: "users", DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID},
		MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens},
	})
	require.NoError(t, err)
	enabledAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.TokenStatProjection.UpdateOneID(projection.ID).
		SetStatus(tokenstat.ProjectionStatusActive).SetEnabledAt(enabledAt).Save(context.Background())
	require.NoError(t, err)
	for i, item := range []struct {
		userID int64
		value  int64
	}{
		{int64(user1.ID), 100},
		{int64(user2.ID), 200},
	} {
		hash := make([]byte, 16)
		hash[0] = byte(i + 1)
		_, err = client.TokenStatAggregate.Create().
			SetPeriodType(string(tokenstat.PeriodDay)).
			SetPeriodStart(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)).
			SetPeriodEnd(time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)).
			SetProjectionID(projection.ID).SetDimensionHash(hash).
			SetDimensionValues(map[string]any{"user_id": item.userID}).
			SetMetricCode(string(tokenstat.MetricTotalTokens)).SetMetricValue(item.value).
			SetUserID(item.userID).SetLastSyncedAt(enabledAt).Save(ctx)
		require.NoError(t, err)
	}

	gin.SetMode(gin.TestMode)
	handler := NewUsageHandlerWithTokenStats(nil, nil, nil, nil, stats)
	router := gin.New()
	router.GET("/api/v1/admin/usage/department-stats", handler.DepartmentStats)
	router.GET("/api/v1/admin/usage/department-stats/users", handler.DepartmentUsers)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/usage/department-stats?start_date=2026-08-01&end_date=2026-08-01&timezone=UTC", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var envelope struct {
		Data tokenstat.DepartmentUsageResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, int64(300), envelope.Data.Summary)
	require.Len(t, envelope.Data.Rows, 2)
	require.Equal(t, "未设置", envelope.Data.Rows[0].Department)
	require.Equal(t, int64(200), envelope.Data.Rows[0].TotalTokens)
	require.Equal(t, int64(1), envelope.Data.Rows[0].UserCount)
	require.Equal(t, float64(200), envelope.Data.Rows[0].AverageTokens)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/usage/department-stats/users?department=%E6%9C%AA%E8%AE%BE%E7%BD%AE&start_date=2026-08-01&end_date=2026-08-01&timezone=UTC&page=1&page_size=10", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var userEnvelope struct {
		Data tokenstat.DepartmentUserUsageResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &userEnvelope))
	require.Equal(t, "未设置", userEnvelope.Data.Department)
	require.Equal(t, int64(200), userEnvelope.Data.DepartmentTotalTokens)
	require.Len(t, userEnvelope.Data.Rows, 1)
	require.Equal(t, user2.ID, userEnvelope.Data.Rows[0].UserID)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/v1/admin/usage/department-stats?start_date=2026-08-02&end_date=2026-08-01&timezone=UTC", nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
