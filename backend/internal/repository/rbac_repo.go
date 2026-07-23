package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
)

type RBACRepository struct {
	db rbacDB
}

type rbacDB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func NewRBACRepository(db *sql.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

func NewRBACRepositoryTx(tx *sql.Tx) *RBACRepository {
	return &RBACRepository{db: tx}
}

func (r *RBACRepository) LoadActiveGrants(ctx context.Context, userID int64) ([]rbac.Grant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT role_row.code, permission_row.code
		FROM rbac_user_roles AS user_role
		JOIN rbac_roles AS role_row
		  ON role_row.id = user_role.role_id
		 AND role_row.status = 'active'
		 AND role_row.deleted_at IS NULL
		LEFT JOIN rbac_role_permissions AS role_permission
		  ON role_permission.role_id = role_row.id
		LEFT JOIN rbac_permissions AS permission_row
		  ON permission_row.id = role_permission.permission_id
		 AND permission_row.status = 'active'
		 AND permission_row.deleted_at IS NULL
		WHERE user_role.user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query active RBAC grants: %w", err)
	}
	defer rows.Close()

	grants := make([]rbac.Grant, 0)
	for rows.Next() {
		var roleCode string
		var permissionCode sql.NullString
		if err := rows.Scan(&roleCode, &permissionCode); err != nil {
			return nil, fmt.Errorf("scan active RBAC grant: %w", err)
		}
		grants = append(grants, rbac.Grant{
			RoleCode: roleCode, RoleActive: true,
			PermissionCode: permissionCode.String, PermissionActive: permissionCode.Valid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active RBAC grants: %w", err)
	}
	return grants, nil
}

func (r *RBACRepository) GetUserVersion(ctx context.Context, userID int64) (int64, error) {
	var version int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT authz_version FROM rbac_user_versions WHERE user_id = ?`, userID,
	).Scan(&version); err != nil {
		return 0, fmt.Errorf("get RBAC user version: %w", err)
	}
	return version, nil
}

func (r *RBACRepository) GetPolicyVersion(ctx context.Context) (int64, error) {
	var version int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT policy_version FROM rbac_policy_state WHERE id = 1`,
	).Scan(&version); err != nil {
		return 0, fmt.Errorf("get RBAC policy version: %w", err)
	}
	return version, nil
}

func (r *RBACRepository) IncrementUserVersion(ctx context.Context, userID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE rbac_user_versions
		SET authz_version = authz_version + 1, updated_at = CURRENT_TIMESTAMP(6)
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("increment RBAC user version: %w", err)
	}
	return requireOneAffected(result, "RBAC user version")
}

func (r *RBACRepository) IncrementPolicyVersion(ctx context.Context) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE rbac_policy_state
		SET policy_version = policy_version + 1, updated_at = CURRENT_TIMESTAMP(6)
		WHERE id = 1
	`)
	if err != nil {
		return fmt.Errorf("increment RBAC policy version: %w", err)
	}
	return requireOneAffected(result, "RBAC policy version")
}

func requireOneAffected(result sql.Result, subject string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read %s affected rows: %w", subject, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s update affected %d rows, want 1", subject, affected)
	}
	return nil
}

var _ rbac.AuthorizationRepository = (*RBACRepository)(nil)
var _ rbac.VersionMutationRepository = (*RBACRepository)(nil)
