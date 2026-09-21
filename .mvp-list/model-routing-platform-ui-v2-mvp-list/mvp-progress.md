# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../../.plans/model-routing-platform-ui-v2-page-rework-implementation-plan.md`
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发日；超过范围的工作已拆分为独立 MVP
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-09-16T11:49:27.4107799Z`
- Overall: `17/19 (89%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-baseline-and-contract.md](./MVP-001-baseline-and-contract.md) | DONE | none | 1 个开发日 | 2026-09-16T06:25:04.4390006Z | `.plans/model-routing-platform-ui-v2-page-rework-implementation-plan.md`; `MVP-001-baseline-and-contract.md` |
| MVP-002 | [MVP-002-ui-v2-tokens-primitives.md](./MVP-002-ui-v2-tokens-primitives.md) | DONE | MVP-001 | 1 个开发日 | 2026-09-16T07:03:08.3612940Z | `MVP-002-ui-v2-tokens-primitives.md`; `frontend/src/components/ui-v2/` |
| MVP-003 | [MVP-003-ui-v2-dialog-data.md](./MVP-003-ui-v2-dialog-data.md) | DONE | MVP-002 | 1 个开发日 | 2026-09-16T07:11:02.7939824Z | `MVP-003-ui-v2-dialog-data.md`; `frontend/src/components/ui-v2/feedback/`; `frontend/src/components/ui-v2/data/` |
| MVP-004 | [MVP-004-ui-v2-shell-and-charts.md](./MVP-004-ui-v2-shell-and-charts.md) | DONE | MVP-002 | 1 个开发日 | 2026-09-16T07:17:44.3287351Z | `MVP-004-ui-v2-shell-and-charts.md`; `frontend/src/components/ui-v2/layout/`; `frontend/src/components/ui-v2/charts/` |
| MVP-005 | [MVP-005-admin-dashboard-data-coordinator.md](./MVP-005-admin-dashboard-data-coordinator.md) | DONE | MVP-001 | 1 个开发日 | 2026-09-16T07:22:54.6431729Z | `MVP-005-admin-dashboard-data-coordinator.md`; `frontend/src/features/admin-dashboard/adminDashboardData.ts` |
| MVP-006 | [MVP-006-admin-dashboard-layout.md](./MVP-006-admin-dashboard-layout.md) | DONE | MVP-003, MVP-004, MVP-005 | 1 个开发日 | 2026-09-16T07:31:05.8240679Z | `MVP-006-admin-dashboard-layout.md`; `/admin/dashboard` → `AdminDashboardV2View.vue` |
| MVP-007 | [MVP-007-scene-detail-dialog.md](./MVP-007-scene-detail-dialog.md) | DONE | MVP-003, MVP-005, MVP-006 | 1 个开发日 | 2026-09-16T07:42:24.2979154Z | `MVP-007-scene-detail-dialog.md`; `SceneAccountTokenDialog.vue` |
| MVP-008 | [MVP-008-department-detail-dialog.md](./MVP-008-department-detail-dialog.md) | DONE | MVP-003, MVP-005, MVP-006 | 1 个开发日 | 2026-09-16T07:48:09.4191507Z | `MVP-008-department-detail-dialog.md`; `DepartmentUserTokenDialog.vue` |
| MVP-009 | [MVP-009-admin-dashboard-checkpoint.md](./MVP-009-admin-dashboard-checkpoint.md) | PENDING | MVP-006, MVP-007, MVP-008 | 1 个开发日 |  |  |
| MVP-010 | [MVP-010-admin-usage-records.md](./MVP-010-admin-usage-records.md) | DONE | MVP-003, MVP-004 | 1 个开发日 | 2026-09-16T10:50:57.2563661Z | `MVP-010-admin-usage-records.md`; `frontend/src/views/admin/AdminUsageV2View.vue` |
| MVP-011 | [MVP-011-personal-dashboard-existing-data.md](./MVP-011-personal-dashboard-existing-data.md) | DONE | MVP-004 | 1 个开发日 | 2026-09-16T10:57:31.0396053Z | `MVP-011-personal-dashboard-existing-data.md`; `frontend/src/views/user/UserDashboardV2View.vue` |
| MVP-012 | [MVP-012-personal-dashboard-new-apis.md](./MVP-012-personal-dashboard-new-apis.md) | DONE | MVP-001 | 1 个开发日 | 2026-09-16T11:08:24.1806808Z | `MVP-012-personal-dashboard-new-apis.md`; `frontend/src/api/usage.ts`; `backend/internal/handler/usage_handler.go` |
| MVP-013 | [MVP-013-personal-usage-records.md](./MVP-013-personal-usage-records.md) | DONE | MVP-003, MVP-004 | 1 个开发日 | 2026-09-16T11:15:50.5952298Z | `MVP-013-personal-usage-records.md`; `frontend/src/views/user/UserUsageV2View.vue` |
| MVP-014 | [MVP-014-ops-existing-ui-v2.md](./MVP-014-ops-existing-ui-v2.md) | DONE | MVP-003, MVP-004 | 1 个开发日 | 2026-09-16T11:23:01.6734119Z | `MVP-014-ops-existing-ui-v2.md`; `frontend/src/views/admin/ops/AdminOpsV2View.vue` |
| MVP-015 | [MVP-015-ops-model-latency-api.md](./MVP-015-ops-model-latency-api.md) | DONE | MVP-001 | 1 个开发日 | 2026-09-16T11:30:36.7182985Z | `MVP-015-ops-model-latency-api.md`; `frontend/src/api/admin/ops.ts`; `backend/internal/service/ops_model_latency.go` |
| MVP-016 | [MVP-016-ops-model-latency-ui.md](./MVP-016-ops-model-latency-ui.md) | DONE | MVP-004, MVP-015 | 1 个开发日 | 2026-09-16T11:33:42.4360082Z | `MVP-016-ops-model-latency-ui.md`; `frontend/src/views/admin/ops/components/OpsModelLatencyPanel.vue` |
| MVP-017 | [MVP-017-keep-pages-ui-v2-migration.md](./MVP-017-keep-pages-ui-v2-migration.md) | DONE | MVP-002, MVP-003, MVP-004 | 1 个开发日 | 2026-09-16T11:46:33.5057288Z | `MVP-017-keep-pages-ui-v2-migration.md`; `frontend/src/components/ui-v2/layout/UiLegacyPageFrame.vue` |
| MVP-018 | [MVP-018-disable-legacy-routes-and-menu.md](./MVP-018-disable-legacy-routes-and-menu.md) | DONE | MVP-006, MVP-010, MVP-013 | 1 个开发日 | 2026-09-16T11:49:27.4107799Z | `MVP-018-disable-legacy-routes-and-menu.md`; `frontend/src/router/index.ts`; `frontend/src/components/layout/AppSidebar.vue` |
| MVP-019 | [MVP-019-full-regression-and-visual-acceptance.md](./MVP-019-full-regression-and-visual-acceptance.md) | PENDING | MVP-009, MVP-010, MVP-013, MVP-014, MVP-016, MVP-017, MVP-018 | 1 个开发日 |  |  |

## 依赖说明

- 关键路径：MVP-001 → MVP-002 → MVP-003/MVP-004 → MVP-005 → MVP-006 → MVP-007/MVP-008 → MVP-009（强制暂停）。
- MVP-010、MVP-011、MVP-012、MVP-013、MVP-014、MVP-015 可在前置基础完成后按依赖并行推进，但 MVP-009 前不得开始后续页面开发。
- MVP-019 是全量回归收口，不计入 Part 1 暂停前交付。

## 规划假设

- 未指定单个 MVP 工作量，按一个聚焦开发日拆分。
- 本 MVP 列表仅覆盖已审核 Plan，不删除任何旧页面和旧组件。
- 首个管理员仪表盘完成后，执行者必须在 MVP-009 完成并更新证据后暂停，等待用户检查。
- 当前前端可使用 `pnpm --dir frontend typecheck`、`pnpm --dir frontend test:run`、`pnpm --dir frontend build` 进行验证；若环境命令差异，以项目实际脚本为准。
