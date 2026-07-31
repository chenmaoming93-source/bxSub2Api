# MVP-002：建立动态统计持久化模型

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `五个新表及其 Repository 契约可以作为独立可迁移、可测试的持久化基础交付。`
- Dependencies: `MVP-001`

## 预期成果

项目拥有独立的 `token_stat_*` Ent Schema、数据库约束和基础 Repository，且不读写任何旧统计限额表。

## 背景

Plan 要求投影、投影指标、聚合数据、通用限额和周期状态拥有独立表。新投影不回填旧数据。

## 范围内

- `token_stat_projections`。
- `token_stat_projection_metrics`。
- `token_stat_aggregates`。
- `token_stat_quota_rules`。
- `token_stat_period_states`。
- 唯一约束、常用索引、JSON字段和关系。
- 基础 CRUD/UPSERT Repository 及测试。

## 范围外

- Redis同步任务。
- 管理 API。
- 删除旧表。

## 实现说明

- 聚合 UPSERT 以 `source_version` 单调更新。
- 高频通用查询索引优先，避免预建所有维度组合索引。
- 不修改或关联旧日统计和旧日限额表。

## 验收标准

- [x] 五个新 Schema 可成功生成。
- [x] 表名、唯一约束和字段满足 Plan。
- [x] 同版本重复 UPSERT 幂等，旧版本不能覆盖新版本。
- [x] 停用投影和指标仍保留历史解释能力。
- [x] Repository 集成或数据库契约测试通过。
- [x] 旧表和旧 Repository 未被新代码引用。

## 验证计划

- `cd backend && go generate ./ent`
- `cd backend && go test ./ent/schema/... ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Schema | `backend/ent/schema/token_stat_*.go`、`backend/ent/tokenstat*` | 五张独立 Schema 及 Ent 生成代码已生成 |
| DDL | `backend/sqlArchiving/165_create_dynamic_token_statistics_tables.sql` | 已按 MySQL 8 / GoldenDB 语法和归档编号规约编写 |
| Repository | `backend/internal/repository/tokenstat` | 单调 `source_version` UPSERT 契约及 sqlmock 测试已实现 |
| 测试 | `cd backend && go test ./ent/... ./internal/repository/tokenstat ./internal/service/tokenstat ./internal/config` | 通过 |
| 隔离 | 对新 Schema、Repository 和归档 SQL 扫描旧固定类型与表名 | 无匹配 |
| SQL 交付边界 | 用户于 2026-07-30 明确由其自行执行 SQL | 本次以 MySQL 8 语法审查和数据库契约测试验收，不代为执行归档 SQL |

## 执行记录

2026-07-30：代码、Schema、归档 SQL 和契约测试均完成。用户明确归档 SQL 由其自行执行，本 MVP 按约定完成验证。
