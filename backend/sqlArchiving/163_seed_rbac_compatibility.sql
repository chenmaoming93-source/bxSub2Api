-- Compatibility seed for the RBAC0 rollout on MySQL 8 / GoldenDB.
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
    ('*', '全部权限', 'system', '内置 admin 与管理员 API Key 专用通配权限', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.create', '创建账号', 'accounts', '创建账号', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.credentials.read', '查看账号凭据', 'accounts', '查看敏感上游凭据', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.credentials.update', '修改账号凭据', 'accounts', '导入、交换或批量修改凭据', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.delete', '删除账号', 'accounts', '删除账号', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.operate', '操作上游账号', 'accounts', '测试、刷新、恢复或同步账号', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.read', '查看账号', 'accounts', '查看账号信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('accounts.update', '修改账号', 'accounts', '修改账号', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('affiliate.self.read', '查看个人返利', 'self', '查看当前用户返利信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('affiliate.self.transfer', '转移个人返利', 'self', '转移当前用户返利额度', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('affiliates.manage', '管理返利', 'billing', '修改专属返利或批量费率', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('affiliates.read', '查看返利管理', 'billing', '查看邀请、返利和转账', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('announcements.create', '创建公告', 'announcements', '创建公告', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('announcements.delete', '删除公告', 'announcements', '删除公告', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('announcements.read', '查看公告', 'announcements', '查看公告信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('announcements.self.read', '查看公告', 'self', '查看并标记当前用户公告', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('announcements.update', '修改公告', 'announcements', '修改公告', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('api_keys.self.create', '创建个人 API Key', 'self', '创建当前用户 API Key', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('api_keys.self.delete', '删除个人 API Key', 'self', '删除当前用户 API Key', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('api_keys.self.read', '查看个人 API Key', 'self', '查看当前用户拥有的 API Key', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('api_keys.self.update', '修改个人 API Key', 'self', '修改当前用户 API Key', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('backups.create', '创建备份', 'backups', '创建或配置备份', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('backups.read', '查看备份', 'backups', '查看和下载备份', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('backups.restore', '恢复备份', 'backups', '从备份恢复系统', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('billing.orders.manage', '管理支付订单', 'billing', '取消、重试或退款', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('billing.plans.manage', '管理支付套餐', 'billing', '创建、修改或删除套餐', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('billing.providers.manage', '管理支付服务商', 'billing', '修改支付服务商和密钥', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('billing.read', '查看支付管理', 'billing', '查看支付仪表盘、订单和配置', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('channels.create', '创建渠道', 'channels', '创建渠道', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('channels.delete', '删除渠道', 'channels', '删除渠道', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('channels.read', '查看渠道', 'channels', '查看渠道信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('channels.self.read', '查看可用渠道', 'self', '查看当前用户可用渠道', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('channels.update', '修改渠道', 'channels', '修改渠道', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('dashboard.backfill', '回填仪表盘聚合', 'dashboard', '执行聚合数据回填', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('dashboard.read', '查看管理仪表盘', 'dashboard', '查看管理统计和趋势', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('data_management.read', '查看数据管理', 'data_management', '查看数据源和任务', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('data_management.update', '修改数据管理', 'data_management', '修改数据源或执行任务', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('groups.create', '创建分组', 'groups', '创建分组', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('groups.delete', '删除分组', 'groups', '删除分组', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('groups.read', '查看分组', 'groups', '查看分组信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('groups.self.read', '查看个人分组', 'self', '查看当前用户可用分组与倍率', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('groups.update', '修改分组', 'groups', '修改分组', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('monitors.read', '查看渠道监控', 'monitors', '查看监控与历史', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('monitors.run', '运行渠道监控', 'monitors', '手动运行监控', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('monitors.self.read', '查看渠道状态', 'self', '查看用户可见渠道监控', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('monitors.update', '修改渠道监控', 'monitors', '修改监控和模板', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('ops.logs.manage', '管理运维日志', 'ops', '解决错误或清理系统日志', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('ops.read', '查看运维信息', 'ops', '查看指标、错误和日志', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('ops.update', '修改运维配置', 'ops', '修改告警、通知和运行时设置', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('pages.self.read', '查看自定义页面', 'self', '查看用户可见自定义页面', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('payments.self.create', '创建个人订单', 'self', '创建当前用户支付订单', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('payments.self.read', '查看个人支付', 'self', '查看当前用户订单和支付配置', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('payments.self.update', '操作个人订单', 'self', '验证、取消或申请退款', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.create', '创建业务权限', 'roles', '创建可分配给角色的业务权限编码', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.delete', '删除业务权限', 'roles', '删除非系统权限及其角色绑定', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.read', '查看权限目录', 'roles', '查看系统权限定义', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('permissions.update', '修改业务权限', 'roles', '修改或停用非系统权限', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('profile.self.read', '查看个人资料', 'self', '查看当前用户资料', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('profile.self.security', '管理个人安全', 'self', '修改密码、身份绑定、通知邮箱和 TOTP', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('profile.self.update', '修改个人资料', 'self', '修改当前用户资料', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('promo_codes.manage', '管理优惠码', 'billing', '创建、修改或删除优惠码', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('promo_codes.read', '查看优惠码', 'billing', '查看优惠码和使用记录', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('proxies.create', '创建代理', 'proxies', '创建代理', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('proxies.delete', '删除代理', 'proxies', '删除代理', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('proxies.operate', '操作代理', 'proxies', '测试、质检或批量操作代理', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('proxies.read', '查看代理', 'proxies', '查看代理信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('proxies.update', '修改代理', 'proxies', '修改代理', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('redeem.self.read', '查看兑换记录', 'self', '查看当前用户兑换历史', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('redeem.self.use', '使用兑换码', 'self', '当前用户兑换额度或订阅', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('redeem_codes.manage', '管理兑换码', 'billing', '生成、修改或删除兑换码', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('redeem_codes.read', '查看兑换码', 'billing', '查看兑换码和统计', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('risk.operate', '执行风控操作', 'risk', '解封用户或清理风险哈希', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('risk.read', '查看风控', 'risk', '查看风控配置、状态和日志', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('risk.update', '修改风控', 'risk', '修改风控配置', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('roles.create', '创建角色', 'roles', '创建角色', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('roles.delete', '删除角色', 'roles', '删除角色', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('roles.permissions.assign', '配置角色权限', 'roles', '全量替换角色权限', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('roles.read', '查看角色', 'roles', '查看角色信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('roles.update', '修改角色', 'roles', '修改角色', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('settings.read', '查看系统设置', 'settings', '查看系统设置', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('settings.secrets.manage', '管理系统密钥', 'settings', '管理管理员 API Key 等敏感配置', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('settings.update', '修改系统设置', 'settings', '修改系统运行设置', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('subscriptions.manage', '管理订阅', 'billing', '分配、延期、重置或删除订阅', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('subscriptions.read', '查看订阅管理', 'billing', '查看用户订阅', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('subscriptions.self.read', '查看个人订阅', 'self', '查看当前用户订阅', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('system.operate', '执行系统操作', 'system', '更新、回滚或重启系统', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('system.read', '查看系统状态', 'system', '查看版本和更新状态', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('token_quota.read', '查看 Token 配额', 'usage', '查看模型 Token 配额', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('token_quota.update', '修改 Token 配额', 'usage', '修改全局或用户模型 Token 配额', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('token_usage.read', '查看 Token 统计', 'usage', '查看模型、路由和用户 Token 统计', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('usage.admin.manage', '管理用量数据', 'usage', '创建或取消用量清理任务', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('usage.admin.read', '查看全局用量', 'usage', '查看用户和 API Key 用量', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('usage.self.read', '查看个人用量', 'self', '查看当前用户用量与错误详情', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.api_keys.read', '查看用户 API Key', 'users', '查看指定用户 API Key', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.balance.adjust', '调整用户余额', 'users', '增加或扣减用户余额', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.create', '创建用户', 'users', '创建用户', 'medium', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.delete', '删除用户', 'users', '删除用户', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.quota.read', '查看用户配额', 'users', '查看用户平台和模型配额', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.quota.update', '修改用户配额', 'users', '修改或重置用户配额', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.read', '查看用户', 'users', '查看用户信息', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.roles.assign', '分配用户角色', 'users', '替换用户角色并可能授予管理能力', 'critical', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.roles.read', '查看用户角色', 'users', '查看用户角色分配', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.update', '修改用户', 'users', '修改用户', 'high', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL),
    ('users.usage.read', '查看用户用量', 'users', '查看指定用户用量', 'low', TRUE, 'active', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), NULL)
ON DUPLICATE KEY UPDATE
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
