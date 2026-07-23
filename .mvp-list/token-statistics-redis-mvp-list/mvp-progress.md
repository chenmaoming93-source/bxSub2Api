# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/token-statistics-redis-implementation-plan.md`
- Target effort per MVP: `40min`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-14T12:23:01+08:00`
- Overall: `13/13 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-token-statistics-config.md](./MVP-001-token-statistics-config.md) | DONE | none | 40min | 2026-07-14T10:57:50+08:00 | [配置、示例与测试证据](./MVP-001-token-statistics-config.md#完成证据) |
| MVP-002 | [MVP-002-redis-key-codec-and-ttl.md](./MVP-002-redis-key-codec-and-ttl.md) | DONE | MVP-001 | 40min | 2026-07-14T11:02:39+08:00 | [Key、Field 与 TTL 证据](./MVP-002-redis-key-codec-and-ttl.md#完成证据) |
| MVP-003 | [MVP-003-redis-token-accumulator.md](./MVP-003-redis-token-accumulator.md) | DONE | MVP-002 | 40min | 2026-07-14T11:09:25+08:00 | [Pipeline、并发与错误证据](./MVP-003-redis-token-accumulator.md#完成证据) |
| MVP-004 | [MVP-004-mysql-absolute-batch-upsert.md](./MVP-004-mysql-absolute-batch-upsert.md) | DONE | none | 40min | 2026-07-14T11:16:31+08:00 | [三表绝对值 upsert 证据](./MVP-004-mysql-absolute-batch-upsert.md#完成证据) |
| MVP-005 | [MVP-005-hscan-sync-engine.md](./MVP-005-hscan-sync-engine.md) | DONE | MVP-002, MVP-004 | 40min | 2026-07-14T11:18:35+08:00 | [HSCAN、分批与重试证据](./MVP-005-hscan-sync-engine.md#完成证据) |
| MVP-006 | [MVP-006-sync-scheduler-and-lock.md](./MVP-006-sync-scheduler-and-lock.md) | DONE | MVP-001, MVP-005 | 40min | 2026-07-14T11:21:49+08:00 | [调度、锁与生命周期证据](./MVP-006-sync-scheduler-and-lock.md#完成证据) |
| MVP-007 | [MVP-007-redis-first-quota-reads.md](./MVP-007-redis-first-quota-reads.md) | DONE | MVP-003 | 40min | 2026-07-14T11:23:49+08:00 | [Redis 优先与回退证据](./MVP-007-redis-first-quota-reads.md#完成证据) |
| MVP-008 | [MVP-008-disable-monetary-eligibility.md](./MVP-008-disable-monetary-eligibility.md) | DONE | none | 40min | 2026-07-14T11:29:29+08:00 | [金额旁路与非金额回归证据](./MVP-008-disable-monetary-eligibility.md#完成证据) |
| MVP-009 | [MVP-009-gateway-record-usage-switch.md](./MVP-009-gateway-record-usage-switch.md) | DONE | MVP-003, MVP-008 | 40min | 2026-07-14T11:32:07+08:00 | [Gateway 用量路径切换证据](./MVP-009-gateway-record-usage-switch.md#完成证据) |
| MVP-010 | [MVP-010-openai-record-usage-switch.md](./MVP-010-openai-record-usage-switch.md) | DONE | MVP-003, MVP-008 | 40min | 2026-07-14T11:48:05+08:00 | [OpenAI 路径切换与回滚证据](./MVP-010-openai-record-usage-switch.md#完成证据) |
| MVP-011 | [MVP-011-runtime-wiring-and-path-coverage.md](./MVP-011-runtime-wiring-and-path-coverage.md) | DONE | MVP-006, MVP-007, MVP-009, MVP-010 | 40min | 2026-07-14T11:57:32+08:00 | [Wiring、生命周期与路径证据](./MVP-011-runtime-wiring-and-path-coverage.md#完成证据) |
| MVP-012 | [MVP-012-diagnostic-logging-and-fault-tests.md](./MVP-012-diagnostic-logging-and-fault-tests.md) | DONE | MVP-003, MVP-005, MVP-006, MVP-007 | 40min | 2026-07-14T12:02:23+08:00 | [事件名与故障注入证据](./MVP-012-diagnostic-logging-and-fault-tests.md#完成证据) |
| MVP-013 | [MVP-013-integration-and-hotspot-regression.md](./MVP-013-integration-and-hotspot-regression.md) | DONE | MVP-011, MVP-012 | 40min | 2026-07-14T12:23:01+08:00 | [全量集成与热点回归证据](./MVP-013-integration-and-hotspot-regression.md#完成证据) |

## 依赖说明

- 关键链路：`MVP-001 → MVP-002 → MVP-003 → MVP-009/MVP-010 → MVP-011 → MVP-013`。
- MySQL 同步链路：`MVP-004 → MVP-005 → MVP-006 → MVP-011 → MVP-013`。
- `MVP-008` 可与 Redis/MySQL 基础设施并行实施；`MVP-009` 与 `MVP-010` 可并行实施。
- `MVP-012` 在核心 Redis、同步和配额读取组件具备后进行，可与请求路径切换并行。

## 规划假设

- 每个 MVP 目标工作量为 40 分钟，允许在约 20–60 分钟范围内浮动。
- 测试命令默认在 `backend` 目录运行，并假设本机 Go 环境可用。
- 不新增 MySQL 表、不修改 `usage_logs` Schema、不实现恢复或对账。
- 原金额计费实现和原逐请求 Token 增量实现保留，只停用正常调用路径。
- Redis 与 MySQL 异常导致的计数损失属于已接受边界。
