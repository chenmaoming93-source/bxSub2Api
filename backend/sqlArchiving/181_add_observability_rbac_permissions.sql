-- MySQL manual migration: add the two permissions introduced by this release.
-- This is a simplified replacement for running the full RBAC compatibility seed.

INSERT IGNORE INTO rbac_permissions (
  code,
  name,
  module,
  description,
  risk_level,
  is_system,
  status,
  created_at,
  updated_at,
  deleted_at
) VALUES
  (
    'observability.read',
    '查看模型观测台',
    'ops',
    '查看只读模型使用、压力和质量观测',
    'low',
    TRUE,
    'active',
    CURRENT_TIMESTAMP(6),
    CURRENT_TIMESTAMP(6),
    NULL
  ),
  (
    'token_usage.manage',
    '管理 Token 统计',
    'usage',
    '创建、发布和停用可配置 Token 统计投影',
    'high',
    TRUE,
    'active',
    CURRENT_TIMESTAMP(6),
    CURRENT_TIMESTAMP(6),
    NULL
  );

UPDATE rbac_permissions
SET status = 'active',
    deleted_at = NULL,
    updated_at = CURRENT_TIMESTAMP(6)
WHERE code IN ('observability.read', 'token_usage.manage');

-- Verification
SELECT code, name, module, status, deleted_at
FROM rbac_permissions
WHERE code IN ('observability.read', 'token_usage.manage');
