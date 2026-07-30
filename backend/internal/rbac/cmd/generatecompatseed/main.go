package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
)

func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: generatecompatseed <output.sql>")
	}
	catalog := rbac.Catalog()
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Code < catalog[j].Code })

	var out strings.Builder
	out.WriteString(`-- Compatibility seed for the RBAC0 rollout on MySQL 8 / GoldenDB.
-- Run 162_create_rbac_schema.sql first. This file is safe to execute repeatedly.
-- Any historical users.role value outside admin/user aborts the transaction.

SET NAMES utf8mb4;

START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS rbac_seed_role_guard;
CREATE TEMPORARY TABLE rbac_seed_role_guard (
    invalid_count BIGINT NOT NULL,
    CONSTRAINT chk_rbac_seed_known_roles CHECK (invalid_count = 0)
);
INSERT INTO rbac_seed_role_guard (invalid_count)
SELECT COUNT(*) FROM users WHERE role IS NULL OR role NOT IN ('admin', 'user');

INSERT INTO rbac_permissions
    (code, name, module, description, risk_level, is_system, status, created_at, updated_at, deleted_at)
VALUES
`)
	for i, permission := range catalog {
		suffix := ",\n"
		if i == len(catalog)-1 {
			suffix = "\n"
		}
		fmt.Fprintf(&out, "    (%s, %s, %s, %s, %s, TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)%s",
			sqlQuote(permission.Code), sqlQuote(permission.Name), sqlQuote(permission.Module),
			sqlQuote(permission.Description), sqlQuote(string(permission.Risk)), suffix)
	}
	out.WriteString(`ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    module = VALUES(module),
    description = VALUES(description),
    risk_level = VALUES(risk_level),
    is_system = TRUE,
    status = 'active',
    updated_at = CURRENT_TIMESTAMP(6),
    deleted_at = NULL;

INSERT INTO rbac_roles
    (code, name, description, is_system, status, created_at, updated_at, deleted_at)
VALUES
    ('admin', '超级管理员', '内置超级管理员角色，固定拥有通配权限 *', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('user', '普通用户', '内置普通用户角色，保持升级前全部个人能力', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    description = VALUES(description),
    is_system = TRUE,
    status = 'active',
    updated_at = CURRENT_TIMESTAMP(6),
    deleted_at = NULL;

INSERT INTO rbac_role_permissions (role_id, permission_id, created_at, updated_at)
SELECT role_row.id, permission_row.id, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6)
FROM rbac_roles AS role_row
JOIN rbac_permissions AS permission_row ON permission_row.code = '*'
WHERE role_row.code = 'admin'
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(6);

INSERT INTO rbac_role_permissions (role_id, permission_id, created_at, updated_at)
SELECT role_row.id, permission_row.id, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6)
FROM rbac_roles AS role_row
JOIN rbac_permissions AS permission_row ON permission_row.module = 'self' AND permission_row.deleted_at IS NULL
WHERE role_row.code = 'user'
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(6);

INSERT INTO rbac_user_roles (user_id, role_id, assigned_by, created_at, updated_at)
SELECT user_row.id, role_row.id, NULL, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6)
FROM users AS user_row
JOIN rbac_roles AS role_row ON role_row.code = user_row.role
WHERE user_row.role IN ('admin', 'user')
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(6);

INSERT INTO rbac_user_versions (user_id, authz_version, updated_at)
SELECT id, 1, CURRENT_TIMESTAMP(6) FROM users
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

DROP TEMPORARY TABLE rbac_seed_role_guard;
COMMIT;

-- Post-migration verification queries. Each result must be zero.
SELECT COUNT(*) AS unknown_role_count
FROM users WHERE role IS NULL OR role NOT IN ('admin', 'user');

SELECT COUNT(*) AS users_without_compat_role
FROM users AS user_row
LEFT JOIN rbac_user_roles AS user_role ON user_role.user_id = user_row.id
LEFT JOIN rbac_roles AS role_row ON role_row.id = user_role.role_id AND role_row.code = user_row.role
WHERE user_row.role IN ('admin', 'user') AND role_row.id IS NULL;

SELECT COUNT(*) AS admins_without_wildcard
FROM users AS user_row
LEFT JOIN rbac_user_roles AS user_role ON user_role.user_id = user_row.id
LEFT JOIN rbac_roles AS role_row ON role_row.id = user_role.role_id AND role_row.code = 'admin'
LEFT JOIN rbac_role_permissions AS role_permission ON role_permission.role_id = role_row.id
LEFT JOIN rbac_permissions AS permission_row ON permission_row.id = role_permission.permission_id AND permission_row.code = '*'
WHERE user_row.role = 'admin' AND permission_row.id IS NULL;
`)
	if err := os.WriteFile(os.Args[1], []byte(out.String()), 0o644); err != nil {
		panic(err)
	}
}
