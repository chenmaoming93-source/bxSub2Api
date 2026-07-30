package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *RBACRepository) ListPermissions(ctx context.Context) ([]service.RBACPermission, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,code,name,module,description,risk_level,is_system,status
		FROM rbac_permissions WHERE deleted_at IS NULL ORDER BY module,is_system DESC,code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.RBACPermission
	for rows.Next() {
		var item service.RBACPermission
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Module, &item.Description, &item.Risk, &item.IsSystem, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *RBACRepository) PermissionsExist(ctx context.Context, codes []string) (bool, error) {
	if len(codes) == 0 {
		return true, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(codes)), ",")
	args := make([]any, len(codes))
	for i := range codes {
		args[i] = codes[i]
	}
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rbac_permissions WHERE deleted_at IS NULL AND status='active' AND code IN (`+placeholders+`)`, args...).Scan(&count)
	return count == len(codes), err
}

func (r *RBACRepository) CreatePermission(ctx context.Context, actor rbac.AuditActor, code, name, module, description string, risk rbac.RiskLevel) (*service.RBACPermission, error) {
	return r.mutatePermission(ctx, actor, 0, code, name, module, description, risk, "active", "permission.create")
}

func (r *RBACRepository) UpdatePermission(ctx context.Context, actor rbac.AuditActor, id int64, name, module, description string, risk rbac.RiskLevel, status string) (*service.RBACPermission, error) {
	return r.mutatePermission(ctx, actor, id, "", name, module, description, risk, status, "permission.update")
}

func (r *RBACRepository) mutatePermission(ctx context.Context, actor rbac.AuditActor, id int64, code, name, module, description string, risk rbac.RiskLevel, status, action string) (*service.RBACPermission, error) {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("RBAC permission mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var before any = map[string]any{}
	if id == 0 {
		res, execErr := tx.ExecContext(ctx, `INSERT INTO rbac_permissions
			(code,name,module,description,risk_level,is_system,status,created_at,updated_at)
			VALUES (?,?,?,?,?,FALSE,'active',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`,
			code, name, module, description, risk)
		if execErr != nil {
			return nil, execErr
		}
		id, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	} else {
		var old service.RBACPermission
		if err = tx.QueryRowContext(ctx, `SELECT id,code,name,module,description,risk_level,is_system,status
			FROM rbac_permissions WHERE id=? AND deleted_at IS NULL FOR UPDATE`, id).
			Scan(&old.ID, &old.Code, &old.Name, &old.Module, &old.Description, &old.Risk, &old.IsSystem, &old.Status); err != nil {
			return nil, err
		}
		if old.IsSystem {
			return nil, rbac.ErrSystemPermissionProtected
		}
		before, code = old, old.Code
		if _, err = tx.ExecContext(ctx, `UPDATE rbac_permissions SET name=?,module=?,description=?,risk_level=?,status=?,updated_at=CURRENT_TIMESTAMP(6) WHERE id=?`,
			name, module, description, risk, status, id); err != nil {
			return nil, err
		}
		if status == "disabled" {
			if _, err = tx.ExecContext(ctx, `DELETE FROM rbac_role_permissions WHERE permission_id=?`, id); err != nil {
				return nil, err
			}
		}
	}
	after := service.RBACPermission{ID: id, Code: code, Name: name, Module: module, Description: description, Risk: risk, Status: status}
	if err = insertRBACAudit(ctx, tx, actor, action, "permission", fmt.Sprint(id), before, after); err != nil {
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

func (r *RBACRepository) DeletePermission(ctx context.Context, actor rbac.AuditActor, id int64) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("RBAC permission mutation requires root database transaction")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var code string
	var system bool
	if err = tx.QueryRowContext(ctx, `SELECT code,is_system FROM rbac_permissions WHERE id=? AND deleted_at IS NULL FOR UPDATE`, id).Scan(&code, &system); err != nil {
		return err
	}
	if system {
		return rbac.ErrSystemPermissionProtected
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rbac_role_permissions WHERE permission_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_permissions SET status='disabled',deleted_at=CURRENT_TIMESTAMP(6),updated_at=CURRENT_TIMESTAMP(6) WHERE id=?`, id); err != nil {
		return err
	}
	if err = insertRBACAudit(ctx, tx, actor, "permission.delete", "permission", fmt.Sprint(id), map[string]any{"code": code}, map[string]any{"deleted": true}); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE rbac_policy_state SET policy_version=policy_version+1,updated_at=CURRENT_TIMESTAMP(6) WHERE id=1`); err != nil {
		return err
	}
	return tx.Commit()
}
