package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestIncrementUsageBillingAPIKeyQuotaBindsAllPlaceholders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT quota_used, quota").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"quota_used", "quota"}).AddRow(0.25, 1.0))
	mock.ExpectExec("UPDATE api_keys").
		WithArgs(0.75, service.StatusAPIKeyActive, 0.75, service.StatusAPIKeyQuotaExhausted, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	exhausted, err := incrementUsageBillingAPIKeyQuota(context.Background(), tx, 7, 0.75)
	require.NoError(t, err)
	require.True(t, exhausted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementUsageBillingAPIKeyRateLimitBindsEveryWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE api_keys SET").
		WithArgs(1.25, 1.25, 1.25, 1.25, 1.25, 1.25, int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, incrementUsageBillingAPIKeyRateLimit(context.Background(), tx, 9, 1.25))
	require.NoError(t, mock.ExpectationsWereMet())
}
