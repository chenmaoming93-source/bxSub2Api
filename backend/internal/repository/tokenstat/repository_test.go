package tokenstat

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUpsertAggregateUsesMonotonicSourceVersionSQL(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(`INSERT INTO token_stat_aggregates[\s\S]+new\.source_version > token_stat_aggregates\.source_version[\s\S]+GREATEST\(token_stat_aggregates\.source_version, new\.source_version\)`).
		WithArgs(
			"D", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(7), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"total_tokens", int64(123), int64(4),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	err = NewRepository(db).UpsertAggregate(context.Background(), Aggregate{
		PeriodType:      "D",
		PeriodStart:     now,
		PeriodEnd:       now.Add(24 * time.Hour),
		ProjectionID:    7,
		DimensionValues: map[string]any{"user_id": 42},
		MetricCode:      "total_tokens",
		MetricValue:     123,
		SourceVersion:   4,
		LastSyncedAt:    now,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
