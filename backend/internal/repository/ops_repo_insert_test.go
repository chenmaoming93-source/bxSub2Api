package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryInsertErrorLogPersistsClientDisconnectShape(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(insertOpsErrorLogSQL)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := &opsRepository{db: db}
	entry := &service.OpsInsertErrorLogInput{
		ErrorPhase:        "network",
		ErrorType:         service.OpsErrorTypeClientDisconnected,
		Severity:          "P3",
		StatusCode:        499,
		IsBusinessLimited: false,
		ErrorSource:       "client_request",
		ErrorOwner:        "client",
	}
	_, err = repo.InsertErrorLog(context.Background(), entry)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
