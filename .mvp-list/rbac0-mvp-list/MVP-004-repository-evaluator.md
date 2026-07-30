# MVP-004：实现多角色权限 Repository 与计算器

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `聚焦数据库访问和纯权限计算，不包含缓存、HTTP 或管理界面。`
- Dependencies: `MVP-002`

## 预期成果

后端可以从数据库读取用户有效角色和权限，并正确计算多角色并集及超级管理员通配结果。

## 背景

授权事实来源为 RBAC 表；停用角色和停用权限不得参与计算。

## 范围内

- Role、Permission、UserRole、RolePermission Repository。
- 用户授权版本和全局策略版本 Repository。
- EffectivePermissions 领域模型与 evaluator。
- 多角色并集、停用过滤和 `*` 处理。

## 范围外

- Redis 缓存。
- Gin 中间件和管理 API。

## 实现说明

- 查询应避免逐角色 N+1。
- evaluator 保持纯逻辑，便于单元测试。
- 空角色用户默认没有权限，不静默授予 user。

## 验收标准

- [x] 多角色权限正确去重合并。
- [x] 停用角色和权限被排除。
- [x] admin 的 `*` 返回 `is_super_admin=true`。
- [x] Repository 查询和事务接口可被 Service 复用。

## 验证计划

- `cd backend && go test ./internal/rbac/...`
- `cd backend && go test ./internal/repository/... -run RBAC`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 纯逻辑测试 | `cd backend && go test ./internal/rbac/...` | 通过；覆盖多角色并集、去重、停用过滤、通配权限和空角色默认拒绝。 |
| Repository 测试 | `cd backend && go test ./internal/repository/... -run RBAC` | 通过；单次 JOIN 查询读取全部有效角色和权限，版本读取契约通过。 |
| 数据库兼容回归 | `cd backend; RBAC_VERIFY_DDL=1 go test ./ent/schema -run TestRBACDDLOnEmptyDatabase -count=1` | 通过；补齐 `status`、`is_system`、`authz_version`、`policy_version` 后 MySQL/GoldenDB DDL 与 Seed 双次执行仍通过。 |
| 事务复用 | `NewRBACRepositoryTx(*sql.Tx)` | Service 可在既有事务中复用同一查询和版本递增实现。 |

## 执行记录

权限读取使用一条 `user_roles → roles → role_permissions → permissions` JOIN，避免逐角色 N+1；角色和权限按 `status = 'active'` 及 `deleted_at IS NULL` 过滤，对应状态/软删除索引已纳入 DDL。空角色用户返回空权限，不自动补 user 角色。
