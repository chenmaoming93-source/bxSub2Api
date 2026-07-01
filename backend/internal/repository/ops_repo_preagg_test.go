package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUpsertHourlyMetricsValueCountMatchesColumns(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	start := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	usageColumns := []string{"created_at", "platform", "group_id", "duration_ms", "first_token_ms", "tokens"}
	mock.ExpectQuery("(?s)FROM usage_logs ul.*WHERE ul.created_at").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows(usageColumns).AddRow(start, "openai", int64(3), int64(120), int64(30), int64(42)))
	mock.ExpectQuery("(?s)FROM ops_error_logs.*WHERE created_at").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "platform", "group_id", "is_business_limited", "error_owner", "status_code", "effective_status_code"}))
	mock.ExpectBegin()
	// One overall, one platform, and one group bucket are written.
	args := make([]driver.Value, 24)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	for i := 0; i < 3; i++ {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ops_metrics_hourly")).
			WithArgs(args...).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	repo := &opsRepository{db: db}
	require.NoError(t, repo.UpsertHourlyMetrics(context.Background(), start, end))
	require.NoError(t, mock.ExpectationsWereMet())
}
