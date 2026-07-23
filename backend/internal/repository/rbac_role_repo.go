package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *RBACRepository) ListRoles(ctx context.Context, page, pageSize int, status, search string) ([]rbac.Role, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	where := ` WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		where += ` AND status = ?`
		args = append(args, status)
	}
	if search != "" {
		where += ` AND (code LIKE ? OR name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rbac_roles`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `SELECT id, code, name, description, is_system, status FROM rbac_roles`+where+` ORDER BY is_system DESC, id LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var roles []rbac.Role
	for rows.Next() {
		var role rbac.Role
		if err = rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.Status); err != nil {
			return nil, 0, err
		}
		roles = append(roles, role)
	}
	return roles, total, rows.Err()
}

func (r *RBACRepository) CreateRole(ctx context.Context, actor rbac.AuditActor, code, name, description string) (*rbac.Role, error) {
	return r.mutateRole(ctx, actor, 0, code, name, description, "active", "role.create")
}

func (r *RBACRepository) UpdateRole(ctx context.Context, actor rbac.AuditActor, id int64, name, description, status string) (*rbac.Role, error) {
	return r.mutateRole(ctx, actor, id, "", name, description, status, "role.update")
}

func (r *RBACRepository) mutateRole(ctx context.Context, actor rbac.AuditActor, id int64, code, name, description, status, action string) (*rbac.Role, error) {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("RBAC role mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var before any = map[string]any{}
	if id == 0 {
		result, execErr := tx.ExecContext(ctx, `INSERT INTO rbac_roles (code,name,description,is_system,status,created_at,updated_at) VALUES (?,?,?,FALSE,'active',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`, code, name, description)
		if execErr != nil {
			return nil, execErr
		}
		id, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
	} else {
		var old rbac.Role
		if err = tx.QueryRowContext(ctx, `SELECT id,code,name,description,is_system,status FROM rbac_roles WHERE id=? AND deleted_at IS NULL FOR UPDATE`, id).
			Scan(&old.ID, &old.Code, &old.Name, &old.Description, &old.IsSystem, &old.Status); err != nil {
			return nil, err
		}
		before = old
		if old.IsSystem && status != "" && status != "active" {
			return nil, rbac.ErrSystemRoleProtected
		}
		if name == "" {
			name = old.Name
		}
		if status == "" {
			status = old.Status
		}
		if _, err = tx.ExecContext(ctx, `UPDATE rbac_roles SET name=?,description=?,status=?,updated_at=CURRENT_TIMESTAMP(6) WHERE id=?`, name, description, status, id); err != nil {
			return nil, err
		}
		code = old.Code
	}
	after := rbac.Role{ID: id, Code: code, Name: name, Description: description, IsSystem: false, Status: status}
	if action == "role.create" {
		after.Status = "active"
	}
	if err = insertRBACAudit(ctx, tx, actor, action, "role", fmt.Sprint(id), before, after); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_policy_state SET policy_version=policy_version+1,updated_at=CURRENT_TIMESTAMP(6) WHERE id=1`); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &after, nil
}

func (r *RBACRepository) DeleteRole(ctx context.Context, actor rbac.AuditActor, id int64) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("RBAC role mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var code, name, status string
	var system bool
	if err = tx.QueryRowContext(ctx, `SELECT code,name,status,is_system FROM rbac_roles WHERE id=? AND deleted_at IS NULL FOR UPDATE`, id).Scan(&code, &name, &status, &system); err != nil {
		return err
	}
	if system {
		return rbac.ErrSystemRoleProtected
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_roles SET status='disabled',deleted_at=CURRENT_TIMESTAMP(6),updated_at=CURRENT_TIMESTAMP(6) WHERE id=?`, id); err != nil {
		return err
	}
	if err = insertRBACAudit(ctx, tx, actor, "role.delete", "role", fmt.Sprint(id), map[string]any{"code": code, "name": name, "status": status}, map[string]any{"deleted": true}); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_policy_state SET policy_version=policy_version+1,updated_at=CURRENT_TIMESTAMP(6) WHERE id=1`); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *RBACRepository) GetRolePermissions(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.code FROM rbac_role_permissions rp JOIN rbac_permissions p ON p.id=rp.permission_id WHERE rp.role_id=? AND p.deleted_at IS NULL ORDER BY p.module,p.code`, roleID)
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

var _ service.RBACRoleRepository = (*RBACRepository)(nil)
