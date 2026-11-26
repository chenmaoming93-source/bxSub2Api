# MVP Progress

- Protocol: `mvp-list/v1`
- Source plan: `docs/GROWTH_PLAN_SUB2API_DEV.md` + `docs/prototype/growth_pages_preview.html`，并以 `docs/GROWTH_PLAN_AIWORK_API.md` v4.0、`docs/GROWTH_PLAN_DESIGN.md` 和 `docs/GROWTH_PLAN_PAGES_DESIGN.md` 为补充约束
- Target effort per MVP: 1 focused developer-day
- Update batch size: 2 completed MVPs
- Last updated: `2026-11-18T10:45:00+08:00`
- Overall: `18/19 (95%)`

## Status Rules

- `PENDING`: not yet recorded as verified complete
- `BLOCKED`: cannot proceed and is not counted as complete
- `DONE`: implemented, acceptance criteria checked, tests run, and evidence recorded
- Progress may lag verified work between batch updates; it must never lead verified work.

## MVP List

| ID | MVP document | Status | Dependencies | Estimate | Completed at | Evidence |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-contract-and-crypto-readiness.md](./MVP-001-contract-and-crypto-readiness.md) | DONE | none | 0.5 day | 2026-11-17 | [evidence](./MVP-001-contract-and-crypto-readiness.md#completion-evidence) |
| MVP-002 | [MVP-002-growth-data-foundation.md](./MVP-002-growth-data-foundation.md) | DONE | MVP-001 | 1.5 days | 2026-11-17 | [evidence](./MVP-002-growth-data-foundation.md#completion-evidence) |
| MVP-003 | [MVP-003-budget-and-quota-control.md](./MVP-003-budget-and-quota-control.md) | DONE | MVP-002 | 1.5 days | 2026-11-17 | [evidence](./MVP-003-budget-and-quota-control.md#completion-evidence) |
| MVP-004 | [MVP-004-task-rules-lifecycle.md](./MVP-004-task-rules-lifecycle.md) | DONE | MVP-002, MVP-003 | 1.5 days | 2026-11-17 | [evidence](./MVP-004-task-rules-lifecycle.md#completion-evidence) |
| MVP-005 | [MVP-005-product-version-inventory.md](./MVP-005-product-version-inventory.md) | DONE | MVP-002 | 1.5 days | 2026-11-17 | [evidence](./MVP-005-product-version-inventory.md#completion-evidence) |
| MVP-006 | [MVP-006-reward-transfer-engine.md](./MVP-006-reward-transfer-engine.md) | DONE | MVP-002, MVP-003, MVP-004 | 1.5 days | 2026-11-17 | [evidence](./MVP-006-reward-transfer-engine.md#completion-evidence) |
| MVP-007 | [MVP-007-sm2-envelope-gateway.md](./MVP-007-sm2-envelope-gateway.md) | DONE | MVP-001, MVP-002 | 1.5 days | 2026-11-17 | [evidence](./MVP-007-sm2-envelope-gateway.md#completion-evidence) |
| MVP-008 | [MVP-008-u-w-read-apis.md](./MVP-008-u-w-read-apis.md) | DONE | MVP-004, MVP-005, MVP-006, MVP-007 | 1.5 days | 2026-11-17 | [evidence](./MVP-008-u-w-read-apis.md#completion-evidence) |
| MVP-009 | [MVP-009-checkin-award.md](./MVP-009-checkin-award.md) | DONE | MVP-006, MVP-008 | 1.5 days | 2026-11-17 | [evidence](./MVP-009-checkin-award.md#completion-evidence) |
| MVP-010 | [MVP-010-task-completion-award.md](./MVP-010-task-completion-award.md) | DONE | MVP-006, MVP-008 | 1.5 days | 2026-11-17 | [evidence](./MVP-010-task-completion-award.md#completion-evidence) |
| MVP-011 | [MVP-011-atomic-exchange-order.md](./MVP-011-atomic-exchange-order.md) | DONE | MVP-005, MVP-006, MVP-008 | 1.5 days | 2026-11-17 | [evidence](./MVP-011-atomic-exchange-order.md#completion-evidence) |
| MVP-012 | [MVP-012-outbox-fulfillment-and-recovery.md](./MVP-012-outbox-fulfillment-and-recovery.md) | DONE | MVP-005, MVP-011 | 1.5 days | 2026-11-17 | [evidence](./MVP-012-outbox-fulfillment-and-recovery.md#completion-evidence) |
| MVP-013 | [MVP-013-adjustment-audit-reconciliation.md](./MVP-013-adjustment-audit-reconciliation.md) | DONE | MVP-003, MVP-006, MVP-012 | 1.5 days | 2026-11-18 | [evidence](./MVP-013-adjustment-audit-reconciliation.md#completion-evidence) |
| MVP-014 | [MVP-014-admin-shell-budget-ui.md](./MVP-014-admin-shell-budget-ui.md) | DONE | MVP-003 | 1.5 days | 2026-11-18 | [evidence](./MVP-014-admin-shell-budget-ui.md#completion-evidence) |
| MVP-015 | [MVP-015-admin-task-ui.md](./MVP-015-admin-task-ui.md) | DONE | MVP-004, MVP-014 | 1.5 days | 2026-11-18 | [evidence](./MVP-015-admin-task-ui.md#completion-evidence) |
| MVP-016 | [MVP-016-admin-product-ui.md](./MVP-016-admin-product-ui.md) | DONE | MVP-005, MVP-014 | 1 day | 2026-11-18 | [evidence](./MVP-016-admin-product-ui.md#completion-evidence) |
| MVP-017 | [MVP-017-admin-order-user-ui.md](./MVP-017-admin-order-user-ui.md) | DONE | MVP-012, MVP-013, MVP-014, MVP-016 | 1.5 days | 2026-11-18 | [evidence](./MVP-017-admin-order-user-ui.md#completion-evidence) |
| MVP-018 | [MVP-018-user-growth-web.md](./MVP-018-user-growth-web.md) | DONE | MVP-008, MVP-009, MVP-011, MVP-012, MVP-014 | 1.5 days | 2026-11-18 | [evidence](./MVP-018-user-growth-web.md#completion-evidence) |
| MVP-019 | [MVP-019-growth-interop-e2e-release.md](./MVP-019-growth-interop-e2e-release.md) | BLOCKED | MVP-010, MVP-017, MVP-018 | 1.5 days |  |  |

## Dependency Notes

- 关键路径：`MVP-001 → MVP-002 → MVP-003 → MVP-004 → MVP-006 → MVP-009/MVP-010 → MVP-011 → MVP-012 → MVP-017 → MVP-019`。
- 数据内核完成后，`MVP-003` 总预算与 `MVP-005` 商品目录可并行；`MVP-007` SM2 网关也可与任务/商品管理开发并行。
- 管理端从 `MVP-014` 起可并行推进：任务页依赖 `MVP-004`，商品页依赖 `MVP-005`，订单/用户页依赖 `MVP-012` 与 `MVP-013`。
- 用户端 `MVP-018` 可在读接口和兑换链路稳定后单独推进；最终 `MVP-019` 统一验证跨三平面共享账务、SM2 互操作与完整原型。

## Planning Assumptions

- 估算按 1 名熟悉 Go、Gin、Ent、MySQL 8/GoldenDB、Vue 3、TypeScript 的开发者计算，包含实现、单测、必要集成测试和文档，不含等待外部评审或发布窗口。
- 1 个 focused developer-day 约等于 6-7 小时有效开发；1.5 天表示存在事务并发、跨层联调或视觉验收。19 个 MVP 合计 27 个开发者工作日。
- 交付范围严格限定在 sub2api：管理端 `/admin/growth`、Web 用户端 `/growth`、M/W 面、供 AIWork 调用的 U 面、共享账务服务及测试。AIWork 的 Main、Renderer、本地任务判定、签到判定和前端页面均由其他项目组实现，不在本计划内。
- sub2api 负责保证 U 面符合冻结契约 v4.0、提供固定测试向量和接口文档，并以 Mock/契约测试证明兼容；与 AIWork 真实版本的项目组联合联调属于交付后的外部协调验收，不是 sub2api 内部开发任务。
- 现有 `mvp-list/` 是已完成的观测台计划，共 27 项且全部为 `DONE`；本计划按设计文档要求独立放在 `mvp-list/growth-plan-mvp-list/`，不改动旧包编号与证据。
- 仓库已归档 SQL 当前最大编号为 `181`，且 `PROJECT_CONVENTIONS.md` 明确禁止新增 `backend/migrations/` DDL。因此本计划固定使用 `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql`，不采纳内部设计早期文字中的 `161` 和自动迁移路径。
- 原型包含 Web 用户端，故计划纳入同源 `W` 面 `/api/v1/user/growth/*`：使用现有 JWT 会话，不重复使用 AIWork 的 SM2 信封；Web 不提供签到或任务完成接口。
- 页面设计只列了 `growth.self.read`，但兑换是写操作。为避免读权限同时授予兑换能力，本计划假定新增 `growth.self.exchange`，仅用于 W 面兑换下单；列表与详情仍使用 `growth.self.read`。
- 管理端使用 `growth.read`、`growth.manage`、`growth.adjust`；调账不做双人复核，由高风险权限、必填原因、审计和每日对账共同约束。
- 任务永不删除，不增加 `deleted_at`；`task_code` 创建后不可修改或复用，只有 `DRAFT`、`ACTIVE`、`OFFLINE`，下线后可通过重新划拨额度恢复为 `ACTIVE`。
- 商品默认支持 `TOKEN_QUOTA`、`SUBSCRIPTION`、`PHYSICAL` 三种展示类型；自动履约优先实现 Token 额度，订阅履约复用现有订阅服务，实物走人工完成/退款，不新增跨系统 HTTP 长事务。
- 管理端 CSV 导出基于当前筛选结果在浏览器生成，不新增后端导出接口；若产品要求导出全量数据，应另立 MVP 设计服务端任务和审计。
- 首版不做积分趋势图、年度/季度预算自动重置、管理员代用户兑换、Web 签到和任务完成证据回查。
- SM2 生产密钥不进入文档、配置或客户端；计划只提交测试专用固定向量，生产密钥由 sub2api 生成后按受控发布流程分别写入服务端和 AIWork Main。
