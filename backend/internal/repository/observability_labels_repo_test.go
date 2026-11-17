package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestLoadObservabilityDimensionNames(t *testing.T) {
	tests := []struct {
		name      string
		dimension string
		query     string
		column    string
		want      string
	}{
		{name: "group", dimension: "group_id", query: "SELECT id, name FROM `groups` WHERE id IN \\(\\?\\)", column: "name", want: "coding"},
		{name: "account", dimension: "account_id", query: "SELECT id, name FROM `accounts` WHERE id IN \\(\\?\\)", column: "name", want: "claude-main"},
		{name: "user", dimension: "user_id", query: "SELECT id, email FROM `users` WHERE id IN \\(\\?\\)", column: "email", want: "user@example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectQuery(test.query).
				WithArgs(7).
				WillReturnRows(sqlmock.NewRows([]string{"id", test.column}).AddRow(7, test.want))

			names, err := loadObservabilityDimensionNames(context.Background(), db, test.dimension, []int64{7, 7, 0})
			require.NoError(t, err)
			require.Equal(t, map[int64]string{7: test.want}, names)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
