# MVP-003：用 SQL 初始化权限并无损回填 admin/user

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `Seed、历史回填和兼容验证构成一个可独立执行的迁移成果。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

一份可独立执行的兼容初始化 SQL，使现有 admin/user 在 RBAC 数据中获得与升级前一致的能力。

## 背景

admin 必须绑定 `*`；user 必须绑定全部现有个人权限；未知历史 role 不得静默处理。

## 范围内

- 插入完整系统权限目录。
- 插入内置 admin/user 角色。
- admin 绑定 `*`，user 绑定个人权限集合。
- 回填 `rbac_user_roles` 和 `rbac_user_versions`。
- 未知 role 预检查、幂等性和迁移后核对查询。

## 范围外

- 新用户运行时自动绑定。
- 切换 RBAC 强制执行。

## 实现说明

- SQL 存放于下一个编号的 `backend/sqlArchiving/NNN_seed_rbac_compatibility.sql`。
- 权限编码必须与 `permissions.go` 完全一致。
- 执行前发现非 admin/user role 时应明确阻断。

## 验收标准

- [x] SQL 可在包含历史用户的数据库独立执行。
- [x] 重复执行不产生重复角色、权限或关联。
- [x] 每个现有 admin 绑定 admin，每个现有 user 绑定 user。
- [x] admin 仅通过系统角色获得 `*`，user 获得完整个人权限。

## 验证计划

- 在匿名化历史数据快照上执行两次 Seed SQL。
- `cd backend && go test ./internal/repository/... -run RBACSeed`
- 对比 SQL 权限 code 与 `backend/internal/rbac/permissions.go`。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Seed SQL | `backend/sqlArchiving/163_seed_rbac_compatibility.sql` | 由代码权限目录生成 101 个权限，初始化 admin/user 系统角色、角色权限、历史用户角色和用户版本。 |
| 目录一致性 | `cd backend && go test ./internal/rbac/...` | 通过；Seed 中权限编码与 `permissions.go` 完全一致。 |
| 历史快照模拟 | `cd backend; RBAC_VERIFY_DDL=1 go test ./ent/schema -run TestRBACDDLOnEmptyDatabase -count=1 -v` | 通过；MySQL/GoldenDB 隔离库内放入 admin/user 历史用户，Seed 连续执行两次无重复，角色、版本、通配权限和全部个人权限核对通过。 |
| 未知角色阻断 | 同一集成测试插入 `legacy_role` 后再次执行 Seed | 按预期被临时表 `CHECK` 约束阻断，事务回滚，未静默降级。 |

## 执行记录

匿名化历史数据模拟规模为 2 名用户（1 admin、1 user）。两次执行后保持 101 个权限、2 个系统角色、2 条用户角色和 2 条用户版本；admin 角色仅有 `*`，user 角色覆盖全部 `module = 'self'` 权限。SQL 生成器位于 `backend/internal/rbac/cmd/generatecompatseed`，用于降低权限目录变更时漏 Seed 的风险。
