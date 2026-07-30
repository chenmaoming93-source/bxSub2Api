package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestRBACRepositoryLoadsGrantsInOneQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_row.code, permission_row.code")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"role_code", "permission_code"}).
			AddRow("operator", "users.read").
			AddRow("viewer", nil))

	grants, err := NewRBACRepository(db).LoadActiveGrants(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, grants, 2)
	require.Equal(t, "users.read", grants[0].PermissionCode)
	require.False(t, grants[1].PermissionActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRepositoryReadsVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(regexp.QuoteMeta("SELECT authz_version FROM rbac_user_versions WHERE user_id = ?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"authz_version"}).AddRow(3))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT policy_version FROM rbac_policy_state WHERE id = 1")).
		WillReturnRows(sqlmock.NewRows([]string{"policy_version"}).AddRow(9))

	repo := NewRBACRepository(db)
	userVersion, err := repo.GetUserVersion(context.Background(), 7)
	require.NoError(t, err)
	policyVersion, err := repo.GetPolicyVersion(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 3, userVersion)
	require.EqualValues(t, 9, policyVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}
