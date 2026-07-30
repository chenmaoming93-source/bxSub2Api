-- Adds authorization codes for managing custom business permissions.
-- MySQL 8 / GoldenDB. Safe to execute repeatedly after 162.
SET NAMES utf8mb4;

INSERT INTO rbac_permissions
    (code, name, module, description, risk_level, is_system, status, created_at, updated_at, deleted_at)
VALUES
    ('permissions.create', '创建业务权限', 'roles', '创建可分配给角色的业务权限编码', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.update', '修改业务权限', 'roles', '修改或停用非系统权限', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.delete', '删除业务权限', 'roles', '删除非系统权限及其角色绑定', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    module = VALUES(module),
    description = VALUES(description),
    risk_level = VALUES(risk_level),
    is_system = TRUE,
    status = 'active',
    updated_at = CURRENT_TIMESTAMP(6),
    deleted_at = NULL;

UPDATE rbac_policy_state
SET policy_version = policy_version + 1,
    updated_at = CURRENT_TIMESTAMP(6)
WHERE id = 1;
