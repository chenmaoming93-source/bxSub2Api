# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/configurable-token-statistics-and-quota-implementation-plan.md`
- Target effort per MVP: `假设每个 MVP 为一个专注开发日`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-30T23:00:00+08:00`
- Overall: `12/12 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-registry-contracts.md](./MVP-001-registry-contracts.md) | DONE | none | 1 开发日 | 2026-07-30 | `go test ./internal/service/... ./internal/config/...` 通过；隔离扫描无匹配 |
| MVP-002 | [MVP-002-persistence-schema.md](./MVP-002-persistence-schema.md) | DONE | MVP-001 | 1 开发日 | 2026-07-30 | Ent/Repository 测试通过；DDL 归档于 `backend/sqlArchiving/165_create_dynamic_token_statistics_tables.sql` |
| MVP-003 | [MVP-003-projection-admin-api.md](./MVP-003-projection-admin-api.md) | DONE | MVP-001, MVP-002 | 1 开发日 | 2026-07-30 | 投影生命周期、配置发布、RBAC/API 测试通过 |
| MVP-004 | [MVP-004-redis-atomic-accounting.md](./MVP-004-redis-atomic-accounting.md) | DONE | MVP-001, MVP-002 | 1 开发日 | 2026-07-30 | Redis Lua 三周期/并发/版本/脏集合测试通过 |
| MVP-005 | [MVP-005-async-usage-ingestion.md](./MVP-005-async-usage-ingestion.md) | DONE | MVP-003, MVP-004 | 1 开发日 | 2026-07-30 | 非阻塞队列、网关事件接入和 fail-open 测试通过 |
| MVP-006 | [MVP-006-mysql-sync.md](./MVP-006-mysql-sync.md) | DONE | MVP-002, MVP-004 | 1 开发日 | 2026-07-30 | 脏集合轮换、版本快照、失败重试和后台同步测试通过 |
| MVP-007 | [MVP-007-period-finalization.md](./MVP-007-period-finalization.md) | DONE | MVP-005, MVP-006 | 1 开发日 | 2026-07-30 | 周期状态、最终同步、版本门禁和 UNLINK 测试通过 |
| MVP-008 | [MVP-008-dynamic-quota.md](./MVP-008-dynamic-quota.md) | DONE | MVP-003, MVP-004, MVP-005 | 1.5 开发日 | 2026-07-30 | 动态限额 CRUD、投影依赖、两阶段 Redis 检查、候选排除、fail-open 与 RBAC 测试通过 |
| MVP-009 | [MVP-009-generic-query-api.md](./MVP-009-generic-query-api.md) | DONE | MVP-002, MVP-003, MVP-006 | 1 开发日 | 2026-07-30 | 通用 MySQL 查询、白名单校验、汇总/趋势/排行/分页、CSV、完整性元数据及 RBAC 测试通过 |
| MVP-010 | [MVP-010-admin-configuration-ui.md](./MVP-010-admin-configuration-ui.md) | DONE | MVP-003, MVP-008 | 1.5 开发日 | 2026-07-30 | 新 API 模块、投影/限额/同步状态页签、菜单/RBAC、定向测试与类型检查通过 |
| MVP-011 | [MVP-011-admin-query-ui.md](./MVP-011-admin-query-ui.md) | DONE | MVP-009, MVP-010 | 1 开发日 | 2026-07-30 | 同页动态查询、汇总/趋势/排行/明细、CSV、完整性提示、定向测试、类型检查与构建通过 |
| MVP-012 | [MVP-012-docs-observability-validation.md](./MVP-012-docs-observability-validation.md) | DONE | MVP-007, MVP-008, MVP-011 | 1.5 开发日 | 2026-07-30 | 两份正式指南、观测指标/告警、性能证据、全量后端回归、前端构建与隔离验收完成 |

## 依赖说明

- 核心路径：MVP-001 → MVP-002 → MVP-003/004 → MVP-005/006 → MVP-007/008/009 → MVP-010/011 → MVP-012。
- MVP-003 与 MVP-004 在持久化基础完成后可并行。
- MVP-005 与 MVP-006 分别处理实时写入和持久化同步，可在 MVP-004 后并行推进。
- 前端配置页和查询页分别依赖对应后端 API。
- MVP-012 是完整文档、观测性、新旧隔离和系统级验收门禁。

## 规划假设

- 每个 MVP 默认以一个专注开发日为目标；涉及两阶段限额和完整前端配置面的任务允许达到 1.5 开发日。
- 新体系使用独立的 `DynamicTokenStat*` 代码命名、`sub2api:dynamic_token_stats:*` Redis 命名空间和 `token_stat_*` MySQL 表。
- 不迁移、不双写、不调用旧三套固定统计限额逻辑。
- 使用现有 Go、Ent、Gin、Redis、Vue、Vitest 和 RBAC 基础设施。
- 测试命令以 `go test`、`pnpm test:run`、`pnpm typecheck` 为主；集成环境不可用时不得把依赖集成环境的 MVP 标记为 DONE。
