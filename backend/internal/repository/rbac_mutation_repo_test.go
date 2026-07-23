package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/stretchr/testify/require"
)

func TestRBACAuditFailureRollsBackPermissionMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRBACRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT code, is_system FROM rbac_roles WHERE id = ? AND deleted_at IS NULL FOR UPDATE`)).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"code", "is_system"}).AddRow("operator", false))
	mock.ExpectQuery(`SELECT p.code FROM rbac_role_permissions`).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow(rbac.PermissionUsersRead))
	mock.ExpectExec(`DELETE FROM rbac_role_permissions`).WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO rbac_role_permissions`).WithArgs(int64(9), rbac.PermissionUsersUpdate).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE rbac_policy_state`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO rbac_audit_logs`).WillReturnError(errors.New("audit unavailable"))
	mock.ExpectRollback()

	err = repo.ReplaceRolePermissions(context.Background(), rbac.AuditActor{}, 9, []string{rbac.PermissionUsersUpdate})
	require.EqualError(t, err, "audit unavailable")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACLastAdminRemovalIsLockedAndRejected(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewRBACRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r.code FROM rbac_user_roles`).WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("admin"))
	mock.ExpectQuery(`SELECT ur.user_id FROM rbac_user_roles`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(7)))
	mock.ExpectRollback()

	err = repo.ReplaceUserRoles(context.Background(), rbac.AuditActor{}, 7, []string{"user"})
	require.ErrorIs(t, err, rbac.ErrLastSuperAdmin)
	require.NoError(t, mock.ExpectationsWereMet())
}
