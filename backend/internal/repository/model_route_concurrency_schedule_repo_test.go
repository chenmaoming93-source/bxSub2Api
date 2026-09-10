package repository

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestReplaceModelRouteConcurrencySchedulesUsesOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := newGroupRepositoryWithSQL(nil, db)
	limit := 10
	schedules := []service.ModelRouteConcurrencySchedule{
		{StartMinute: 0, EndMinute: 570, MaxConcurrency: &limit},
		{StartMinute: 570, EndMinute: 1440},
	}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM group_model_route_account_concurrency_schedules WHERE group_id = ? AND route_alias = ? AND account_id = ?")).
		WithArgs(int64(1), "test", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_model_route_account_concurrency_schedules")).
		WithArgs(int64(1), "test", int64(2), 0, 570, &limit).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_model_route_account_concurrency_schedules")).
		WithArgs(int64(1), "test", int64(2), 570, 1440, nil).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.ReplaceModelRouteConcurrencySchedules(context.Background(), 1, "test", 2, schedules))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplaceModelRouteConcurrencySchedulesRollsBackOnInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := newGroupRepositoryWithSQL(nil, db)
	limit := 10
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM group_model_route_account_concurrency_schedules WHERE group_id = ? AND route_alias = ? AND account_id = ?")).
		WithArgs(int64(1), "test", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_model_route_account_concurrency_schedules")).
		WithArgs(int64(1), "test", int64(2), 0, 570, &limit).
		WillReturnError(context.Canceled)
	mock.ExpectRollback()

	err = repo.ReplaceModelRouteConcurrencySchedules(context.Background(), 1, "test", 2, []service.ModelRouteConcurrencySchedule{{StartMinute: 0, EndMinute: 570, MaxConcurrency: &limit}})
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, mock.ExpectationsWereMet())
}
