# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../../.plans/model-route-candidate-time-window-concurrency-implementation-plan.md`
- Target effort per MVP: 默认每个 MVP 约一个开发日；包含调度器与并发控制的 MVP 允许约 1.5 个开发日
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-18T22:05:00+08:00`
- Overall: `7/7 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-schedule-domain-and-migration.md](./MVP-001-schedule-domain-and-migration.md) | DONE | none | 1 day | 2026-08-18 | [MVP-001-schedule-domain-and-migration.md](./MVP-001-schedule-domain-and-migration.md) |
| MVP-002 | [MVP-002-schedule-management-api.md](./MVP-002-schedule-management-api.md) | DONE | MVP-001 | 1 day | 2026-08-18 | [MVP-002-schedule-management-api.md](./MVP-002-schedule-management-api.md) |
| MVP-003 | [MVP-003-candidate-schedule-ui.md](./MVP-003-candidate-schedule-ui.md) | DONE | MVP-002 | 1 day | 2026-08-18 | [MVP-003-candidate-schedule-ui.md](./MVP-003-candidate-schedule-ui.md) |
| MVP-004 | [MVP-004-redis-effective-limit-fallback.md](./MVP-004-redis-effective-limit-fallback.md) | DONE | MVP-001 | 1 day | 2026-08-18 | [MVP-004-redis-effective-limit-fallback.md](./MVP-004-redis-effective-limit-fallback.md) |
| MVP-005 | [MVP-005-full-refresh-scheduler-and-lock.md](./MVP-005-full-refresh-scheduler-and-lock.md) | DONE | MVP-001, MVP-004 | 1.5 days | 2026-08-18 | [MVP-005-full-refresh-scheduler-and-lock.md](./MVP-005-full-refresh-scheduler-and-lock.md) |
| MVP-006 | [MVP-006-immediate-refresh-entry.md](./MVP-006-immediate-refresh-entry.md) | DONE | MVP-003, MVP-005 | 1 day | 2026-08-18 | [MVP-006-immediate-refresh-entry.md](./MVP-006-immediate-refresh-entry.md) |
| MVP-007 | [MVP-007-end-to-end-compatibility-verification.md](./MVP-007-end-to-end-compatibility-verification.md) | DONE | MVP-002, MVP-003, MVP-004, MVP-005, MVP-006 | 1 day | 2026-08-18 | [MVP-007-end-to-end-compatibility-verification.md](./MVP-007-end-to-end-compatibility-verification.md) |

## 依赖说明

- MVP-001 是数据模型和持久化基础；MVP-002 与 MVP-004 在其完成后可以并行开发。
- MVP-003 依赖管理 API；MVP-005 依赖新表读取能力和请求侧 Redis key 约定。
- MVP-006 将立即刷新入口接入已有页面，并复用 MVP-005 的同一把全局 Redis 刷新锁。
- MVP-007 负责跨实例、长任务、兼容回退、删除清理和最终前后端联调验证。

## 规划假设

- 只新增 `group_model_route_account_concurrency_schedules`，不修改 `group_model_route_accounts` 的结构、数据或既有逻辑。
- 时间段按每日循环、前闭后开处理，内部分钟范围为 `0..1440`；未覆盖区间使用旧字段默认值。
- 分时段并发值允许正整数或 `NULL`（表示现有语义的 `unlimited`）；不做候选账号之间的总和合理性校验。
- 请求路径只读 Redis，不计算当前时间、不查询分时段表；新 key 缺失时回退旧 route-config key。
- 所有网关实例在每个整分钟尝试抢同一把锁，锁持有者负责全量刷新；定时任务不补历史轮次。
- 仓库现有 Go、前端和 Redis/MySQL 测试设施可复用；若环境缺少依赖，执行时必须在对应 MVP 中记录阻塞和实际证据。
