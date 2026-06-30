# MVP Progress

- Protocol: `mvp-list/v1`
- Source plan: `../PLAN.md`
- Target effort per MVP: `20min以内`
- Update batch size: `1 completed MVP`
- Last updated: `2026-06-30T14:31:06+08:00`
- Overall: `10/23 (43%)`

## Status Rules

- `PENDING`: not yet recorded as verified complete
- `BLOCKED`: cannot proceed and is not counted as complete
- `DONE`: implemented, acceptance criteria checked, tests run, and evidence recorded
- Progress may lag verified work between batch updates; it must never lead verified work.

## MVP List

| ID | MVP document | Status | Dependencies | Estimate | Completed at | Evidence |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-quota-design-decision.md](./MVP-001-quota-design-decision.md) | DONE | none | 20min | 2026-06-30T11:46:46+08:00 | [决策、仓库核对与人工验证](./MVP-001-quota-design-decision.md#completion-evidence) |
| MVP-002 | [MVP-002-routing-config-domain.md](./MVP-002-routing-config-domain.md) | DONE | MVP-001 | 20min | 2026-06-30T11:48:55+08:00 | [领域解析实现与单元测试](./MVP-002-routing-config-domain.md#completion-evidence) |
| MVP-003 | [MVP-003-group-routing-persistence.md](./MVP-003-group-routing-persistence.md) | DONE | MVP-002 | 20min | 2026-06-30T12:01:24+08:00 | [兼容持久化、DTO 隔离与测试](./MVP-003-group-routing-persistence.md#completion-evidence) |
| MVP-004 | [MVP-004-global-model-quota-schema.md](./MVP-004-global-model-quota-schema.md) | DONE | MVP-001 | 20min | 2026-06-30T12:05:28+08:00 | [全局模型日配额 schema、migration 与测试](./MVP-004-global-model-quota-schema.md#completion-evidence) |
| MVP-005 | [MVP-005-user-model-quota-schema.md](./MVP-005-user-model-quota-schema.md) | DONE | MVP-001 | 20min | 2026-06-30T12:08:25+08:00 | [用户模型日配额 schema、外键与测试](./MVP-005-user-model-quota-schema.md#completion-evidence) |
| MVP-006 | [MVP-006-group-candidate-quota-schema.md](./MVP-006-group-candidate-quota-schema.md) | DONE | MVP-001 | 20min | 2026-06-30T12:12:13+08:00 | [候选日用量身份、级联与测试](./MVP-006-group-candidate-quota-schema.md#completion-evidence) |
| MVP-007 | [MVP-007-quota-read-port.md](./MVP-007-quota-read-port.md) | DONE | MVP-004, MVP-005, MVP-006 | 20min | 2026-06-30T12:17:36+08:00 | [统一读取端口、耗尽错误与边界测试](./MVP-007-quota-read-port.md#completion-evidence) |
| MVP-008 | [MVP-008-quota-atomic-increment.md](./MVP-008-quota-atomic-increment.md) | DONE | MVP-007 | 20min | 2026-06-30T13:57:58+08:00 | [原子累加、跨日与回滚测试](./MVP-008-quota-atomic-increment.md#completion-evidence) |
| MVP-009 | [MVP-009-quota-cache.md](./MVP-009-quota-cache.md) | DONE | MVP-007, MVP-008 | 20min | 2026-06-30T14:11:44+08:00 | [Redis 快速判断、TTL 与隔离测试](./MVP-009-quota-cache.md#completion-evidence) |
| MVP-010 | [MVP-010-global-quota-admin-api.md](./MVP-010-global-quota-admin-api.md) | DONE | MVP-004, MVP-007, MVP-009 | 20min | 2026-06-30T14:31:06+08:00 | [全局模型配额管理 API、缓存失效与测试](./MVP-010-global-quota-admin-api.md#completion-evidence) |
| MVP-011 | [MVP-011-user-quota-admin-api.md](./MVP-011-user-quota-admin-api.md) | PENDING | MVP-005, MVP-007, MVP-009 | 20min |  |  |
| MVP-012 | [MVP-012-candidate-routing.md](./MVP-012-candidate-routing.md) | PENDING | MVP-002, MVP-003 | 20min |  |  |
| MVP-013 | [MVP-013-quota-aware-routing.md](./MVP-013-quota-aware-routing.md) | PENDING | MVP-007, MVP-009, MVP-012 | 20min |  |  |
| MVP-014 | [MVP-014-candidate-failover.md](./MVP-014-candidate-failover.md) | PENDING | MVP-013 | 20min |  |  |
| MVP-015 | [MVP-015-model-identity.md](./MVP-015-model-identity.md) | PENDING | MVP-012 | 20min |  |  |
| MVP-016 | [MVP-016-usage-quota-accounting.md](./MVP-016-usage-quota-accounting.md) | PENDING | MVP-008, MVP-009, MVP-014, MVP-015 | 20min |  |  |
| MVP-017 | [MVP-017-frontend-routing-normalizer.md](./MVP-017-frontend-routing-normalizer.md) | PENDING | MVP-002 | 15min |  |  |
| MVP-018 | [MVP-018-group-routing-editor.md](./MVP-018-group-routing-editor.md) | PENDING | MVP-003, MVP-017 | 20min |  |  |
| MVP-019 | [MVP-019-frontend-quota-api.md](./MVP-019-frontend-quota-api.md) | PENDING | MVP-010, MVP-011 | 15min |  |  |
| MVP-020 | [MVP-020-user-quota-modal.md](./MVP-020-user-quota-modal.md) | PENDING | MVP-019 | 20min |  |  |
| MVP-021 | [MVP-021-global-quota-modal.md](./MVP-021-global-quota-modal.md) | PENDING | MVP-019 | 20min |  |  |
| MVP-022 | [MVP-022-i18n-ui-regression.md](./MVP-022-i18n-ui-regression.md) | PENDING | MVP-018, MVP-020, MVP-021 | 15min |  |  |
| MVP-023 | [MVP-023-backend-integration-regression.md](./MVP-023-backend-integration-regression.md) | PENDING | MVP-016 | 20min |  |  |

## Dependency Notes

- 后端关键链：MVP-001 → MVP-004/005/006 → MVP-007 → MVP-008/009 → MVP-013 → MVP-014/015 → MVP-016 → MVP-023。
- 路由配置链：MVP-001 → MVP-002 → MVP-003/012；MVP-012 与配额链在 MVP-013 汇合。
- 前端可在 API 就绪前并行完成 MVP-017；随后 MVP-018 与 MVP-019 并行，MVP-020/021 并行，最终汇合至 MVP-022。
- MVP-001 是阻断性决策：未记录候选用量键、跨日额度继承及 429/503 语义前，不启动依赖它的实现任务。

## Planning Assumptions

- 每个 MVP 由一名熟悉仓库的开发者连续执行，估时包含聚焦测试与证据记录，不包含首次下载依赖或修复无关环境故障。
- 额度值 `0`/`null` 的最终统一语义、候选用量的持久化键以及额度跨日继承方式由 MVP-001 固化；当前计划不擅自替 PLAN.md 补设计。
- 每日边界继续使用项目全局 `timezone.StartOfDay`；`CountTokens` 不计入每日限额。
- Token 计量使用 `UsageLog.TotalTokens()`，模型维度使用实际上游模型名；旧 model_routing 保持读取兼容，新 UI 保存新格式。
- 若某个实现 MVP 实测无法在 20min 内连同验证完成，执行者应在开始编码前把它拆成新的稳定 ID，而不是扩大估时或省略测试。
