# MVP-002：交付 RBAC 表结构与可执行 DDL

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `一个内聚的 Schema 交付，包含 Ent 与独立 SQL，可单独在空库验证。`
- Dependencies: `MVP-001`

## 预期成果

提供 RBAC 角色、权限、关联、版本和审计表的 Ent Schema 与独立可执行 DDL。

## 背景

必须遵循 `PROJECT_CONVENTIONS.md`，DDL 只能归档到 `backend/sqlArchiving/`。

## 范围内

- `rbac_roles`、`rbac_permissions`、`rbac_user_roles`、`rbac_role_permissions`。
- `rbac_user_versions`、`rbac_policy_state`、`rbac_audit_logs`。
- 唯一约束、外键、索引、软删除和单行策略版本初始化。
- Ent 生成代码及 Schema 测试。

## 范围外

- 权限、角色和历史用户 Seed。
- Repository 与业务 API。

## 实现说明

- 实施前扫描两个 SQL 目录取得下一个编号。
- SQL 必须独立执行、语句完整并具备可行的重复执行保护。
- 不修改 `backend/migrations` 已有文件。

## 验收标准

- [x] 新 SQL 位于 `backend/sqlArchiving/NNN_create_rbac_schema.sql`。
- [x] 空库可执行并创建全部表、索引和约束。
- [x] 重复执行行为符合文件注释和项目方言能力。
- [x] Ent Schema 与 SQL 字段、约束一致。

## 验证计划

- `cd backend && go generate ./ent`
- `cd backend && go test ./ent/schema/...`
- 在项目支持的测试数据库执行 DDL 并核对表结构。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| DDL | `backend/sqlArchiving/162_create_rbac_schema.sql` | 使用 MySQL 8 / GoldenDB 方言，包含 7 张表、显式外键、唯一约束、内联索引、软删除字段和单行策略版本初始化。 |
| Ent 生成 | `cd backend && go generate ./ent` | 在隔离副本成功生成后同步回主目录；规避 Windows 文件映射占用导致的半写问题。 |
| Schema 测试 | `cd backend && go test ./ent/schema/...` | 通过。 |
| 空库双次执行 | `cd backend; RBAC_VERIFY_DDL=1 go test ./ent/schema -run TestRBACDDLOnEmptyDatabase -count=1 -v` | 通过；在配置的 MySQL/GoldenDB 服务上创建隔离临时库，DDL 连续执行两次，确认 7 张表后自动删除临时库。 |

## 执行记录

最终编号为 `162`。项目方言确认是 MySQL 8 / GoldenDB；索引内联在 `CREATE TABLE IF NOT EXISTS` 中，单行策略记录使用 `ON DUPLICATE KEY UPDATE`，因此整份文件可重复执行。实现过程中曾误用 PostgreSQL 方言，已全部纠正，并将方言核验和测试库双次执行写入 `PROJECT_CONVENTIONS.md`。
