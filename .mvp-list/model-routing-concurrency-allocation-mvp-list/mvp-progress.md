# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../../.plans/model-routing-concurrency-allocation-implementation-plan.md`
- Target effort per MVP: 默认每个 MVP 约一个开发日
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-11T18:00:00+08:00`
- Overall: `5/5 (100%)`

## 状态规则

- `PENDING`：尚未验证完成
- `BLOCKED`：无法继续，且不计入完成
- `DONE`：已实现、测试通过并记录证据

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-allocation-calculation.md](./MVP-001-allocation-calculation.md) | DONE | none | 1 day | 2026-08-11 | [MVP-001-allocation-calculation.md](./MVP-001-allocation-calculation.md) |
| MVP-002 | [MVP-002-account-concurrency-sync.md](./MVP-002-account-concurrency-sync.md) | DONE | MVP-001 | 1 day | 2026-08-11 | [MVP-002-account-concurrency-sync.md](./MVP-002-account-concurrency-sync.md) |
| MVP-003 | [MVP-003-group-config-validation.md](./MVP-003-group-config-validation.md) | DONE | MVP-001 | 1 day | 2026-08-11 | [MVP-003-group-config-validation.md](./MVP-003-group-config-validation.md) |
| MVP-004 | [MVP-004-admin-allocation-display.md](./MVP-004-admin-allocation-display.md) | DONE | MVP-001, MVP-003 | 1 day | 2026-08-11 | [MVP-004-admin-allocation-display.md](./MVP-004-admin-allocation-display.md) |
| MVP-005 | [MVP-005-integration-verification.md](./MVP-005-integration-verification.md) | DONE | MVP-002, MVP-003, MVP-004 | 1 day | 2026-08-11 | [MVP-005-integration-verification.md](./MVP-005-integration-verification.md) |

## 依赖说明

先实现可复用的整数分配与比例计算，再接入账号更新同步和分组保存校验，随后完成管理页面展示，最后进行全链路验证。现有请求级并发执行逻辑不在本清单改造范围内。
