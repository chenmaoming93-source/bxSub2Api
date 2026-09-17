# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/dynamic-token-quota-reset-implementation-plan.md`
- Target effort per MVP: `假设每个 MVP 为一个专注开发者日；允许约 0.5～1.5 个开发者日的合理波动`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-09-10T09:19:09+08:00`
- Overall: `7/7 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-reset-snapshot-primitives.md](./MVP-001-reset-snapshot-primitives.md) | DONE | none | 1 个开发者日 | 2026-09-09T22:30:56+08:00 | [实现与测试证据](./MVP-001-reset-snapshot-primitives.md#完成证据) |
| MVP-002 | [MVP-002-reset-identity-discovery.md](./MVP-002-reset-identity-discovery.md) | DONE | none | 1.5 个开发者日 | 2026-09-09T22:44:01+08:00 | [实现与测试证据](./MVP-002-reset-identity-discovery.md#完成证据) |
| MVP-003 | [MVP-003-reset-orchestration.md](./MVP-003-reset-orchestration.md) | DONE | MVP-001, MVP-002 | 1.5 个开发者日 | 2026-09-09T22:55:28+08:00 | [实现与测试证据](./MVP-003-reset-orchestration.md#完成证据) |
| MVP-004 | [MVP-004-admin-reset-api.md](./MVP-004-admin-reset-api.md) | DONE | MVP-003 | 1 个开发者日 | 2026-09-09T23:05:08+08:00 | [实现与测试证据](./MVP-004-admin-reset-api.md#完成证据) |
| MVP-005 | [MVP-005-integration-reset-api.md](./MVP-005-integration-reset-api.md) | DONE | MVP-003 | 1 个开发者日 | 2026-09-09T23:12:05+08:00 | [实现与测试证据](./MVP-005-integration-reset-api.md#完成证据) |
| MVP-006 | [MVP-006-admin-reset-ui.md](./MVP-006-admin-reset-ui.md) | DONE | MVP-004 | 1.5 个开发者日 | 2026-09-09T23:29:52+08:00 | [实现与测试证据](./MVP-006-admin-reset-ui.md#完成证据) |
| MVP-007 | [MVP-007-release-verification.md](./MVP-007-release-verification.md) | DONE | MVP-004, MVP-005, MVP-006 | 1 个开发者日 | 2026-09-10T09:19:09+08:00 | [发布验证证据](./MVP-007-release-verification.md#完成证据) |

## 依赖说明

- `MVP-001` 与 `MVP-002` 可并行：前者建立 Redis 快照与限额差值读取，后者建立部分维度统计身份发现。
- `MVP-003` 汇合两项基础能力，形成可被不同入口复用的完整重置服务。
- `MVP-004` 与 `MVP-005` 在 `MVP-003` 后可并行，分别交付管理员和 integrations 后端入口。
- `MVP-006` 依赖管理员 API；`MVP-007` 在所有入口及页面完成后执行发布级回归。
- 关键依赖链：`MVP-001/MVP-002 → MVP-003 → MVP-004 → MVP-006 → MVP-007`。

## 规划假设

- 用户未指定单个 MVP 的目标工时，因此采用一个专注开发者日作为拆分基准。
- 本期不新增数据库表或生产 DDL，重置快照只保存在 Redis。
- 当前仓库继续使用 Go、Gin、go-redis、MySQL、Vue、Vitest 和 pnpm 的既有工具链。
- dirty identity 只读取一次，不加同步锁、不重复检查，也不扫描 processing set；遗漏可通过再次重置处理。
- 多个快照不要求整体原子，单条失败不回滚成功项；匹配数量多时分页、SSCAN 和 pipeline 继续处理，不因数量直接拒绝。
- 外部接口复用现有 `/api/v1/integrations` Bearer Token 与 hardening，不增加业务幂等和复杂权限模型。
