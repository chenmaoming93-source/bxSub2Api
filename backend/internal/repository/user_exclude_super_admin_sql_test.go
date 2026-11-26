package repository

import (
	"context"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestExcludeSuperAdminUsersPredicateSQLIsValidShape(t *testing.T) {
	var actualSQL string
	matcher := sqlmock.QueryMatcherFunc(func(expectedSQL, actual string) error {
		actualSQL = actual
		return nil
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.MySQL, db)))
	t.Cleanup(func() { _ = client.Close() })

	_, _ = client.User.Query().
		Where(excludeSuperAdminUsersPredicate()).
		Limit(1).
		All(context.Background())

	require.NotEmpty(t, actualSQL)
	t.Log(actualSQL)
	require.Equal(t, 2, strings.Count(actualSQL, "NOT EXISTS ("))
	require.Contains(
		t,
		normalizeSQLWhitespace(actualSQL),
		"WHERE wildcard_user_role.user_id = `users`.`id`) AND `users`.`deleted_at` IS NULL",
	)
}
