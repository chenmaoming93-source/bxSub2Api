# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/token-usage-redis-read-repair-implementation-plan.md`
- Target effort per MVP: `约 40min`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-15T11:02:47+08:00`
- Overall: `13/13 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-shared-merge-contract.md](./MVP-001-shared-merge-contract.md) | DONE | none | 40min | 2026-07-14T20:07:39+08:00 | [实现与测试证据](./MVP-001-shared-merge-contract.md#完成证据) |
| MVP-002 | [MVP-002-redis-bulk-reader.md](./MVP-002-redis-bulk-reader.md) | DONE | MVP-001 | 40min | 2026-07-14T20:10:37+08:00 | [实现与测试证据](./MVP-002-redis-bulk-reader.md#完成证据) |
| MVP-003 | [MVP-003-atomic-read-repair.md](./MVP-003-atomic-read-repair.md) | DONE | MVP-001, MVP-002 | 40min | 2026-07-15T09:34:51+08:00 | [实现与测试证据](./MVP-003-atomic-read-repair.md#完成证据) |
| MVP-004 | [MVP-004-mysql-current-day-collections.md](./MVP-004-mysql-current-day-collections.md) | DONE | MVP-001 | 40min | 2026-07-15T09:38:26+08:00 | [实现与测试证据](./MVP-004-mysql-current-day-collections.md#完成证据) |
| MVP-005 | [MVP-005-model-hybrid-report.md](./MVP-005-model-hybrid-report.md) | DONE | MVP-001, MVP-002, MVP-003, MVP-004 | 40min | 2026-07-15T10:26:56+08:00 | [实现与测试证据](./MVP-005-model-hybrid-report.md#完成证据) |
| MVP-006 | [MVP-006-route-hybrid-report.md](./MVP-006-route-hybrid-report.md) | DONE | MVP-001, MVP-002, MVP-003, MVP-004 | 40min | 2026-07-15T10:30:37+08:00 | [实现与测试证据](./MVP-006-route-hybrid-report.md#完成证据) |
| MVP-007 | [MVP-007-user-model-hybrid-report.md](./MVP-007-user-model-hybrid-report.md) | DONE | MVP-001, MVP-002, MVP-003, MVP-004 | 40min | 2026-07-15T10:34:23+08:00 | [实现与测试证据](./MVP-007-user-model-hybrid-report.md#完成证据) |
| MVP-008 | [MVP-008-quota-read-repair.md](./MVP-008-quota-read-repair.md) | DONE | MVP-002, MVP-003 | 40min | 2026-07-15T10:37:22+08:00 | [实现与测试证据](./MVP-008-quota-read-repair.md#完成证据) |
| MVP-009 | [MVP-009-global-quota-admin-realtime.md](./MVP-009-global-quota-admin-realtime.md) | DONE | MVP-001, MVP-002, MVP-003, MVP-004 | 40min | 2026-07-15T10:42:03+08:00 | [实现与测试证据](./MVP-009-global-quota-admin-realtime.md#完成证据) |
| MVP-010 | [MVP-010-user-quota-admin-realtime.md](./MVP-010-user-quota-admin-realtime.md) | DONE | MVP-001, MVP-002, MVP-003, MVP-004 | 40min | 2026-07-15T10:45:39+08:00 | [实现与测试证据](./MVP-010-user-quota-admin-realtime.md#完成证据) |
| MVP-011 | [MVP-011-today-options-default-target.md](./MVP-011-today-options-default-target.md) | DONE | MVP-005, MVP-006, MVP-007 | 40min | 2026-07-15T10:50:43+08:00 | [实现与测试证据](./MVP-011-today-options-default-target.md#完成证据) |
| MVP-012 | [MVP-012-fault-observability.md](./MVP-012-fault-observability.md) | DONE | MVP-003, MVP-005, MVP-006, MVP-007, MVP-008 | 40min | 2026-07-15T10:55:36+08:00 | [实现与测试证据](./MVP-012-fault-observability.md#完成证据) |
| MVP-013 | [MVP-013-contract-performance-regression.md](./MVP-013-contract-performance-regression.md) | DONE | MVP-009, MVP-010, MVP-011, MVP-012 | 40min | 2026-07-15T11:02:47+08:00 | [实现与测试证据](./MVP-013-contract-performance-regression.md#完成证据) |

## 依赖说明

- 基础关键路径：MVP-001 → MVP-002 → MVP-003。
- MySQL 集合查询 MVP-004 可在 MVP-002、MVP-003 开发期间并行。
- 三类报表 MVP-005、MVP-006、MVP-007 在基础能力完成后可并行。
- 配置链路 MVP-009、MVP-010 可在三类报表开发期间并行。
- 当天候选 MVP-011 依赖三类报表语义稳定。
- 最终收口路径：MVP-011、MVP-012 → MVP-013。

## 规划假设

- 每个 MVP 的目标工作量为约 40 分钟，可接受范围约 20～60 分钟。
- 执行者熟悉 Go、Gin、go-redis、Vue/Vitest 及当前仓库结构。
- 本次不新增数据库 migration，不改变 API DTO，不改变 Redis field codec。
- 测试命令基于当前 Makefile、Go package 与 frontend/package.json；若测试名在实现中调整，应运行等价的精确包测试并记录命令。
- 所有配置变更必须更新 `backend/config/config.yaml` 并添加含义与单位注释；时间间隔配置使用分钟。
