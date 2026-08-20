package repository

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestRescaleAccountModelRouteConcurrencySchedulesScalesNumericRowsOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	accountID := int64(7)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, max_concurrency")).
		WithArgs(accountID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "max_concurrency"}).
			AddRow(int64(11), 10).
			AddRow(int64(12), 50))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE group_model_route_account_concurrency_schedules")).
		WithArgs(20, int64(11), accountID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE group_model_route_account_concurrency_schedules")).
		WithArgs(100, int64(12), accountID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, rescaleAccountModelRouteConcurrencySchedules(context.Background(), db, accountID, 100, 200))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRescaleAccountModelRouteConcurrencySchedulesSkipsWhenRatioIsUndefined(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, rescaleAccountModelRouteConcurrencySchedules(context.Background(), db, 7, 0, 200))
	require.NoError(t, rescaleAccountModelRouteConcurrencySchedules(context.Background(), db, 7, 100, 0))
	require.NoError(t, mock.ExpectationsWereMet())
}
