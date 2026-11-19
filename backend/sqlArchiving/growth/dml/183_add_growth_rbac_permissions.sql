-- MySQL manual migration: add growth-plan RBAC permissions.
-- Safe to execute repeatedly after the RBAC compatibility seed has been applied.

INSERT INTO rbac_permissions (
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
  ('growth.read', '查看成长计划', 'growth', '查看总预算、任务、商品、订单和用户积分', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
  ('growth.manage', '管理成长计划', 'growth', '管理预算额度、任务、商品和订单履约', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
  ('growth.adjust', '调整用户积分', 'growth', '对用户成长积分执行受审计调账', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
  ('growth.self.read', '查看我的积分', 'growth', '查看当前用户成长积分、任务、商品和流水', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
  ('growth.self.exchange', '兑换成长商品', 'growth', '使用当前用户积分兑换商品', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  module = VALUES(module),
  description = VALUES(description),
  risk_level = VALUES(risk_level),
  is_system = TRUE,
  status = 'active',
  updated_at = CURRENT_TIMESTAMP(6),
  deleted_at = NULL;

INSERT INTO rbac_role_permissions (role_id, permission_id, created_at, updated_at)
SELECT role_row.id, permission_row.id, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6)
FROM rbac_roles AS role_row
JOIN rbac_permissions AS permission_row
  ON permission_row.code IN ('growth.self.read', 'growth.self.exchange')
 AND permission_row.deleted_at IS NULL
WHERE role_row.code = 'user'
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(6);
