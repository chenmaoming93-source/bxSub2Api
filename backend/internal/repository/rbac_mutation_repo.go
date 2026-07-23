package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
)

func (r *RBACRepository) ReplaceRolePermissions(ctx context.Context, actor rbac.AuditActor, roleID int64, permissionCodes []string) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("RBAC permission mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var roleCode string
	var isSystem bool
	if err := tx.QueryRowContext(ctx,
		`SELECT code, is_system FROM rbac_roles WHERE id = ? AND deleted_at IS NULL FOR UPDATE`, roleID,
	).Scan(&roleCode, &isSystem); err != nil {
		return err
	}
	codes := rbac.SortedUnique(permissionCodes)
	if err := rbac.ValidateRolePermissionReplacement(roleCode, isSystem, codes); err != nil {
		return err
	}
	before, err := rolePermissionCodes(ctx, tx, roleID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rbac_role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, code := range codes {
		result, execErr := tx.ExecContext(ctx, `
			INSERT INTO rbac_role_permissions (role_id, permission_id, created_at)
			SELECT ?, id, CURRENT_TIMESTAMP(6) FROM rbac_permissions
			WHERE code = ? AND status = 'active' AND deleted_at IS NULL`, roleID, code)
		if execErr != nil {
			return execErr
		}
		if err = requireOneAffected(result, "role permission insert"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_policy_state SET policy_version = policy_version + 1, updated_at = CURRENT_TIMESTAMP(6) WHERE id = 1`); err != nil {
		return err
	}
	action := "role.permissions.replace"
	if roleCode == "user" {
		action = "system_user.permissions.replace_high_impact"
	}
	if err = insertRBACAudit(ctx, tx, actor, action, "role", fmt.Sprint(roleID), before, codes); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RBACRepository) ReplaceUserRoles(ctx context.Context, actor rbac.AuditActor, userID int64, roleCodes []string) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("RBAC user-role mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	before, err := userRoleCodes(ctx, tx, userID, true)
	if err != nil {
		return err
	}
	after := rbac.SortedUnique(roleCodes)
	if containsRBACValue(before, "admin") && !containsRBACValue(after, "admin") {
		rows, queryErr := tx.QueryContext(ctx, `
			SELECT ur.user_id FROM rbac_user_roles ur
			JOIN rbac_roles r ON r.id = ur.role_id
			WHERE r.code = 'admin' AND r.status = 'active' AND r.deleted_at IS NULL FOR UPDATE`)
		if queryErr != nil {
			return queryErr
		}
		admins := map[int64]struct{}{}
		for rows.Next() {
			var id int64
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			admins[id] = struct{}{}
		}
		rows.Close()
		if len(admins) <= 1 {
			return rbac.ErrLastSuperAdmin
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rbac_user_roles WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, code := range after {
		result, execErr := tx.ExecContext(ctx, `
			INSERT INTO rbac_user_roles (user_id, role_id, created_at)
			SELECT ?, id, CURRENT_TIMESTAMP(6) FROM rbac_roles
			WHERE code = ? AND status = 'active' AND deleted_at IS NULL`, userID, code)
		if execErr != nil {
			return execErr
		}
		if err = requireOneAffected(result, "user role insert"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO rbac_user_versions (user_id, authz_version, updated_at)
		VALUES (?, 1, CURRENT_TIMESTAMP(6))
		ON DUPLICATE KEY UPDATE authz_version = authz_version + 1, updated_at = CURRENT_TIMESTAMP(6)`, userID); err != nil {
		return err
	}
	legacyRole := "user"
	if containsRBACValue(after, "admin") {
		legacyRole = "admin"
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET role = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?`, legacyRole, userID); err != nil {
		return err
	}
	if err = insertRBACAudit(ctx, tx, actor, "user.roles.replace", "user", fmt.Sprint(userID), before, after); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RBACRepository) GetUserRoles(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT role_row.code FROM rbac_user_roles user_role
		JOIN rbac_roles role_row ON role_row.id = user_role.role_id
		WHERE user_role.user_id = ? AND role_row.deleted_at IS NULL
		ORDER BY role_row.code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func rolePermissionCodes(ctx context.Context, tx *sql.Tx, roleID int64) ([]string, error) {
	return queryStrings(ctx, tx, `SELECT p.code FROM rbac_role_permissions rp JOIN rbac_permissions p ON p.id = rp.permission_id WHERE rp.role_id = ? ORDER BY p.code`, roleID)
}

func userRoleCodes(ctx context.Context, tx *sql.Tx, userID int64, lock bool) ([]string, error) {
	query := `SELECT r.code FROM rbac_user_roles ur JOIN rbac_roles r ON r.id = ur.role_id WHERE ur.user_id = ? ORDER BY r.code`
	if lock {
		query += ` FOR UPDATE`
	}
	return queryStrings(ctx, tx, query, userID)
}

func queryStrings(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func insertRBACAudit(ctx context.Context, tx *sql.Tx, actor rbac.AuditActor, action, targetType, targetID string, before, after any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO rbac_audit_logs
		(actor_user_id, action, target_type, target_id, before_data, after_data, request_id, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP(6))`,
		actor.UserID, action, targetType, targetID, beforeJSON, afterJSON, actor.RequestID, actor.IPAddress)
	return err
}

func containsRBACValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
