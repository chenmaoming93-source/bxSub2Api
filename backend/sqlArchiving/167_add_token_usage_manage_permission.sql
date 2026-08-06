-- Adds the authorization code for managing configurable Token statistics projections.
-- MySQL 8 / GoldenDB. Safe to execute repeatedly after 162.
SET NAMES utf8mb4;

INSERT INTO rbac_permissions
    (code, name, module, description, risk_level, is_system, status, created_at, updated_at, deleted_at)
VALUES
    ('token_usage.manage', '管理 Token 统计', 'usage', '创建、发布和停用可配置 Token 统计投影', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)
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
