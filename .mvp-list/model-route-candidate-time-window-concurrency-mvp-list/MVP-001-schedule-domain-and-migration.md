# MVP-001：分时段并发配置领域模型与独立数据表

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 新增一张表、迁移、领域校验和候选级 CRUD，范围集中且可用仓储测试独立验证。
- Dependencies: `none`

## 预期成果

系统拥有一套不触碰旧表的分时段并发配置持久化能力：新表初始为空，能够按分组、路由别名和账号查询并整体替换某个候选的每日时间段配置。

## 背景

现有候选默认并发值位于 `group_model_route_accounts.max_concurrency`，该字段必须继续保留原语义。本 MVP 只新增 `group_model_route_account_concurrency_schedules`，时间使用每日循环的分钟数，`start_minute` 为 `0..1439`，`end_minute` 为 `1..1440`，采用 `[start, end)`；`max_concurrency IS NULL` 表示 `unlimited`。

## 范围内

- 新增数据库迁移和与现有迁移体系一致的索引/约束。
- 新增 Go 领域结构、分钟值转换和 `[start, end)` 重叠校验。
- 实现按候选读取、按候选整体替换、删除和清理所需的仓储方法。
- 整体替换在一个事务中完成，避免页面保存期间出现半套配置。
- 覆盖空配置、未覆盖全天、边界时间、数值和 `NULL` 的后端测试。

## 范围外

- 不修改 `group_model_route_accounts` 表或其 `max_concurrency` 逻辑。
- 不实现管理 API、前端页面、Redis 当前值刷新和请求路径读取。
- 不做候选账号之间的并发总和合理性校验。

## 实现说明

- 表至少包含 `id`、`group_id`、`route_alias`、`account_id`、`start_minute`、`end_minute`、`max_concurrency`、`created_at`、`updated_at`。
- 数据库约束负责基础范围；同一候选的重叠检查在应用层完成，并在整体替换事务中再次保证。
- API 层后续可将 `NULL` 映射为 `unlimited`，本层保留现有数据库空值语义。

## 验收标准

- [x] 新迁移可以创建分时段表，且未对 `group_model_route_accounts` 产生结构或数据变更。
- [x] 同一候选的相邻区间允许保存，重叠区间和 `start >= end` 被拒绝。
- [x] `00:00`、`24:00`、未覆盖时间段、数值并发和 `NULL/unlimited` 均可被正确保存和读取。
- [x] 整体替换事务失败时不会留下部分新配置；空列表能删除该候选全部分时段配置。
- [x] 相关 Go 单元/仓储测试通过。

## 验证计划

- `go test ./internal/repository/... ./internal/service/...`（在 `backend` 目录执行，按实际包调整）。
- 检查迁移 SQL、旧表 schema/diff 和针对边界/重叠的定向测试结果。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 迁移 | `backend/sqlArchiving/170_create_group_model_route_account_concurrency_schedules.sql` | 已新增独立 MySQL/GoldenDB 兼容 DDL，包含候选索引、窗口和并发值约束；未修改旧表文件 |
| 领域模型 | `backend/internal/service/model_route_concurrency_schedule.go` | 已实现分钟边界、`HH:mm`/`24:00` 转换、相邻区间和重叠校验，`nil` 表示 unlimited |
| 仓储 | `backend/internal/repository/model_route_concurrency_schedule_repo.go` | 已实现按候选查询、全量查询和事务整体替换；空列表可删除配置，插入失败回滚 |
| 测试 | `go test ./internal/service ./internal/repository`（`backend`） | 通过：service 51.878s，repository 6.896s |
| 兼容检查 | `git diff -- backend/sqlArchiving/169_create_group_model_route_accounts.sql backend/internal/repository/group_model_route_account_repo.go` | 无输出，旧表 DDL 与既有候选并发仓储逻辑未被修改 |

## 执行记录

- 2026-08-18：按 Plan 新增独立分时段表；未扩展旧 `GroupRepository` 接口，使用独立 schedule repository 接口，避免影响既有调用方。
- 2026-08-18：仓储测试使用 sqlmock 验证整体替换的提交与失败回滚；未连接真实 MySQL 测试库，DDL 通过仓库方言和结构静态核对。
