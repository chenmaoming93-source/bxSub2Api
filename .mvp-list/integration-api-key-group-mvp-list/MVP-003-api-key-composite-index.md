# MVP-003：归档三元组查询组合索引

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `新增一个 Ent 索引声明、一份 SQL 归档及静态迁移测试，交付物小而完整。`
- Dependencies: `none`

## 预期成果

`api_keys` 具备适配三元组未软删除查询的组合索引定义和可人工部署的 SQL 归档，并有自动化测试约束其存在。

## 背景

新查询将稳定使用 `user_id`、`platform`、`group_id` 和 `deleted_at`。项目规约要求表结构变更 SQL 存放在 `backend/sqlArchiving/`，不得写入 `backend/migrations/`。

## 范围内

- 在 `backend/ent/schema/api_key.go` 声明组合普通索引。
- 在 `backend/sqlArchiving/` 使用下一个可用编号新增索引 SQL。
- 增加静态迁移/归档测试，检查索引字段顺序和 SQL 文件存在。
- 明确该索引不是 UNIQUE。

## 范围外

- 不执行数据库 SQL。
- 不修改既有 `backend/migrations/*.sql`。
- 不引入生成列或软删除唯一索引。
- 不修改业务查询代码。

## 实现说明

- 目标字段顺序：`(user_id, platform, group_id, deleted_at)`。
- 创建前再次检查 `backend/sqlArchiving/` 的最大编号，避免文件名冲突。
- SQL 归档仅作为部署、升级和人工执行材料，不假设运行时自动执行。

## 验收标准

- [x] Ent schema 包含四字段组合普通索引。
- [x] 新 SQL 文件位于 `backend/sqlArchiving/` 且编号不冲突。
- [x] 没有新增或修改 `backend/migrations/*.sql`。
- [x] SQL 明确创建 `(user_id, platform, group_id, deleted_at)` 索引。
- [x] 静态索引测试通过。

## 验证计划

- `cd backend && go test ./migrations -run 'Test.*APIKey.*Index|Test.*Platform.*Group'`
- `git diff -- backend/migrations backend/sqlArchiving backend/ent/schema/api_key.go`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Schema | `backend/ent/schema/api_key.go` | 声明 `(user_id, platform, group_id, deleted_at)` 四字段普通索引。 |
| SQL 归档 | `backend/sqlArchiving/161_api_key_platform_group_lookup_index.sql` | 使用重新扫描后的下一编号 161，创建非 UNIQUE 且带 `IF NOT EXISTS` 的组合索引。 |
| 静态测试 | `backend/migrations/api_key_platform_group_index_test.go` | 校验 SQL 路径、字段顺序、非唯一性和 Ent schema 声明。测试文件本身不改变数据库结构。 |
| 验证 | `cd backend && go test ./migrations -run 'Test.*APIKey.*Index|Test.*Platform.*Group'` | 通过：`ok github.com/Wei-Shaw/sub2api/migrations 1.660s`。 |
| 变更检查 | `git diff -- backend/migrations backend/sqlArchiving backend/ent/schema/api_key.go` | 未修改或新增 `backend/migrations/*.sql`；结构变更 SQL 仅新增到 `backend/sqlArchiving/`。 |

## 执行记录

2026-07-16：按项目 SQL 归档规约新增 161 号索引文件、Ent 索引声明及静态约束测试。
