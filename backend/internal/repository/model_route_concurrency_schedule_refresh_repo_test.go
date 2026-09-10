package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestListModelRouteConcurrencyScheduleCandidatesJoinsLegacyDefaultAndGroupsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := newGroupRepositoryWithSQL(nil, db)
	created := time.Date(2026, 8, 18, 1, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT s.group_id, s.route_alias, s.account_id,")).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "route_alias", "account_id", "default_max", "id", "start_minute", "end_minute", "max_concurrency", "created_at", "updated_at",
		}).
			AddRow(int64(1), "test", int64(2), 20, int64(10), 0, 570, 10, created, updated).
			AddRow(int64(1), "test", int64(2), 20, int64(11), 570, 1440, nil, created, updated))

	items, err := repo.ListModelRouteConcurrencyScheduleCandidates(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(1), items[0].GroupID)
	require.Equal(t, "test", items[0].RouteAlias)
	require.Equal(t, int64(2), items[0].AccountID)
	require.NotNil(t, items[0].DefaultMaxConcurrency)
	require.Equal(t, 20, *items[0].DefaultMaxConcurrency)
	require.Len(t, items[0].Schedules, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}
