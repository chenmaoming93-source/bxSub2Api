# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/rbac0-implementation-plan.md`
- Target effort per MVP: `假设每个 MVP 为一个专注开发日`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-23T18:06:01+08:00`
- Overall: `21/21 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-permission-catalog.md](./MVP-001-permission-catalog.md) | DONE | none | 1 个专注开发日 | 2026-07-23T15:35:23+08:00 | [完成证据](./MVP-001-permission-catalog.md#完成证据) |
| MVP-002 | [MVP-002-rbac-schema-sql.md](./MVP-002-rbac-schema-sql.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-23T16:08:56+08:00 | [完成证据](./MVP-002-rbac-schema-sql.md#完成证据) |
| MVP-003 | [MVP-003-compatibility-seed-sql.md](./MVP-003-compatibility-seed-sql.md) | DONE | MVP-001, MVP-002 | 1 个专注开发日 | 2026-07-23T16:13:21+08:00 | [完成证据](./MVP-003-compatibility-seed-sql.md#完成证据) |
| MVP-004 | [MVP-004-repository-evaluator.md](./MVP-004-repository-evaluator.md) | DONE | MVP-002 | 1 个专注开发日 | 2026-07-23T16:20:22+08:00 | [完成证据](./MVP-004-repository-evaluator.md#完成证据) |
| MVP-005 | [MVP-005-distributed-permission-cache.md](./MVP-005-distributed-permission-cache.md) | DONE | MVP-004 | 1 个专注开发日 | 2026-07-23T16:22:52+08:00 | [完成证据](./MVP-005-distributed-permission-cache.md#完成证据) |
| MVP-006 | [MVP-006-principal-registry-middleware.md](./MVP-006-principal-registry-middleware.md) | DONE | MVP-001, MVP-004, MVP-005 | 1 个专注开发日 | 2026-07-23T16:26:16+08:00 | [完成证据](./MVP-006-principal-registry-middleware.md#完成证据) |
| MVP-007 | [MVP-007-user-route-migration.md](./MVP-007-user-route-migration.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T16:35:06+08:00 | [完成证据](./MVP-007-user-route-migration.md#完成证据) |
| MVP-008 | [MVP-008-admin-user-group-dashboard-routes.md](./MVP-008-admin-user-group-dashboard-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T16:41:30+08:00 | [完成证据](./MVP-008-admin-user-group-dashboard-routes.md#完成证据) |
| MVP-009 | [MVP-009-admin-account-proxy-oauth-routes.md](./MVP-009-admin-account-proxy-oauth-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T16:45:10+08:00 | [完成证据](./MVP-009-admin-account-proxy-oauth-routes.md#完成证据) |
| MVP-010 | [MVP-010-admin-settings-system-data-routes.md](./MVP-010-admin-settings-system-data-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T16:52:00+08:00 | [完成证据](./MVP-010-admin-settings-system-data-routes.md#完成证据) |
| MVP-011 | [MVP-011-admin-ops-usage-token-routes.md](./MVP-011-admin-ops-usage-token-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T16:57:00+08:00 | [完成证据](./MVP-011-admin-ops-usage-token-routes.md#完成证据) |
| MVP-012 | [MVP-012-admin-commercial-routes.md](./MVP-012-admin-commercial-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T17:03:00+08:00 | [完成证据](./MVP-012-admin-commercial-routes.md#完成证据) |
| MVP-013 | [MVP-013-admin-channel-risk-remaining-routes.md](./MVP-013-admin-channel-risk-remaining-routes.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-23T17:08:00+08:00 | [完成证据](./MVP-013-admin-channel-risk-remaining-routes.md#完成证据) |
| MVP-014 | [MVP-014-route-coverage-gate.md](./MVP-014-route-coverage-gate.md) | DONE | MVP-007, MVP-008, MVP-009, MVP-010, MVP-011, MVP-012, MVP-013 | 1 个专注开发日 | 2026-07-23T17:15:00+08:00 | [完成证据](./MVP-014-route-coverage-gate.md#完成证据) |
| MVP-015 | [MVP-015-audit-and-system-role-guards.md](./MVP-015-audit-and-system-role-guards.md) | DONE | MVP-004, MVP-005 | 1 个专注开发日 | 2026-07-23T17:24:00+08:00 | [完成证据](./MVP-015-audit-and-system-role-guards.md#完成证据) |
| MVP-016 | [MVP-016-role-permission-management-api.md](./MVP-016-role-permission-management-api.md) | DONE | MVP-006, MVP-015 | 1 个专注开发日 | 2026-07-23T17:39:00+08:00 | [完成证据](./MVP-016-role-permission-management-api.md#完成证据) |
| MVP-017 | [MVP-017-user-role-provisioning-auth-response.md](./MVP-017-user-role-provisioning-auth-response.md) | DONE | MVP-003, MVP-006, MVP-015 | 1 个专注开发日 | 2026-07-23T17:51:00+08:00 | [完成证据](./MVP-017-user-role-provisioning-auth-response.md#完成证据) |
| MVP-018 | [MVP-018-frontend-permission-foundation.md](./MVP-018-frontend-permission-foundation.md) | DONE | MVP-017 | 1 个专注开发日 | 2026-07-23T18:01:00+08:00 | [完成证据](./MVP-018-frontend-permission-foundation.md#完成证据) |
| MVP-019 | [MVP-019-frontend-route-menu-action-migration.md](./MVP-019-frontend-route-menu-action-migration.md) | DONE | MVP-014, MVP-018 | 1 个专注开发日 | 2026-07-23T18:12:00+08:00 | [完成证据](./MVP-019-frontend-route-menu-action-migration.md#完成证据) |
| MVP-020 | [MVP-020-role-and-user-role-ui.md](./MVP-020-role-and-user-role-ui.md) | DONE | MVP-016, MVP-018 | 1 个专注开发日 | 2026-07-23T18:26:00+08:00 | [完成证据](./MVP-020-role-and-user-role-ui.md#完成证据) |
| MVP-021 | [MVP-021-shadow-enforce-rollout.md](./MVP-021-shadow-enforce-rollout.md) | DONE | MVP-003, MVP-014, MVP-017, MVP-019, MVP-020 | 1 个专注开发日 | 2026-07-23T18:06:01+08:00 | [完成证据](./MVP-021-shadow-enforce-rollout.md#完成证据) |

## 依赖说明

- 关键路径：`MVP-001 → MVP-002 → MVP-004 → MVP-005 → MVP-006 → MVP-007～013 → MVP-014 → MVP-019 → MVP-021`。
- `MVP-003` 可在 Schema 与权限目录稳定后推进；`MVP-015` 可与路由迁移并行。
- `MVP-007～013` 可由不同执行者并行，但必须共同通过 `MVP-014` 的路由闭合检查。
- 管理 API 与前端基础可在后端路由迁移期间并行，最终在 `MVP-021` 汇合。

## 规划假设

- 未指定单个 MVP 工作量，采用一个专注开发日作为拆分目标。
- `RBAC_ENDPOINT_INVENTORY.md` 是当前接口基线；实现时实际 Gin 路由是最终校验来源。
- 后端测试以 `cd backend && go test ./...` 为总体验证命令，MVP 中优先运行更小的相关包。
- 前端测试使用 `pnpm --dir frontend run test:run`，并按 MVP 指定相关 Vitest 文件。
- SQL 文件编号在执行 MVP-002、MVP-003 时重新扫描 `backend/migrations` 与 `backend/sqlArchiving` 后确定。
