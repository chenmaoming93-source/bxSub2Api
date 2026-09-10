# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/group-route-concurrency-view-implementation-plan.md`
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发日；按可独立实现和验证的垂直切片拆分
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-17T18:27:00+08:00`
- Overall: `4/4 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、已验证、已记录证据
- 每个 MVP 验证完成后，必须立即更新本进度文档，再开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-data-contract.md](./MVP-001-data-contract.md) | DONE | none | 1 天 | 2026-08-17 | [evidence](./MVP-001-data-contract.md) |
| MVP-002 | [MVP-002-concurrency-card.md](./MVP-002-concurrency-card.md) | DONE | MVP-001 | 1 天 | 2026-08-17 | [evidence](./MVP-002-concurrency-card.md) |
| MVP-003 | [MVP-003-refresh-lifecycle.md](./MVP-003-refresh-lifecycle.md) | DONE | MVP-002 | 1 天 | 2026-08-17 | [evidence](./MVP-003-refresh-lifecycle.md) |
| MVP-004 | [MVP-004-tests-and-regression.md](./MVP-004-tests-and-regression.md) | DONE | MVP-003 | 1 天 | 2026-08-17 | [evidence](./MVP-004-tests-and-regression.md) |

## 依赖说明

- MVP-001 先明确现有分组路由、候选上限和账号并发数据的可用接口与前端数据结构。
- MVP-002 基于 MVP-001 的数据结构实现分组列表入口和并发查看卡片。
- MVP-003 在卡片可展示的基础上接入自动刷新、手动刷新和资源清理。
- MVP-004 对完整功能进行组件、集成和回归验证。

## 规划假设

- 优先复用已有接口，不改变 `admin/accounts` 现有逻辑。
- 当前并发量继续使用现有 Redis 槽位统计链路。
- 未配置路由候选最大并发时展示 `∞`。
- MVP 执行内容和证据必须以实际仓库状态及测试结果为准。
