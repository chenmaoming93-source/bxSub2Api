# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/model-routing-account-model-and-query-api-implementation-plan.md`
- Target effort per MVP: `40min`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-17T00:26:46+08:00`
- Overall: `9/9 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-account-model-api-client.md](./MVP-001-account-model-api-client.md) | DONE | none | 40min | 2026-07-17T00:03:27+08:00 | API 聚焦测试与前端类型检查通过，详见 MVP 文档。 |
| MVP-002 | [MVP-002-model-intersection-logic.md](./MVP-002-model-intersection-logic.md) | DONE | MVP-001 | 40min | 2026-07-17T00:04:37+08:00 | 模型交集纯函数的 5 个聚焦测试通过，详见 MVP 文档。 |
| MVP-003 | [MVP-003-account-first-model-selector.md](./MVP-003-account-first-model-selector.md) | DONE | MVP-001, MVP-002 | 40min | 2026-07-17T00:07:46+08:00 | 单账号模型下拉组件测试与类型检查通过，详见 MVP 文档。 |
| MVP-004 | [MVP-004-multi-account-model-state.md](./MVP-004-multi-account-model-state.md) | DONE | MVP-003 | 40min | 2026-07-17T00:11:20+08:00 | 多账号缓存、交集、失败和竞态组件测试通过，详见 MVP 文档。 |
| MVP-005 | [MVP-005-routing-editor-validation-i18n.md](./MVP-005-routing-editor-validation-i18n.md) | DONE | MVP-004 | 40min | 2026-07-17T00:15:16+08:00 | 前端 13 个聚焦测试与 production build 通过，详见 MVP 文档。 |
| MVP-006 | [MVP-006-group-route-query-service.md](./MVP-006-group-route-query-service.md) | DONE | none | 40min | 2026-07-17T00:18:09+08:00 | Service 路由投影聚焦测试通过且无结构/配置变更，详见 MVP 文档。 |
| MVP-007 | [MVP-007-group-route-query-handler.md](./MVP-007-group-route-query-handler.md) | DONE | MVP-006 | 40min | 2026-07-17T00:20:51+08:00 | Handler 成功/错误/脱敏契约聚焦测试通过，详见 MVP 文档。 |
| MVP-008 | [MVP-008-integration-route-auth-contract.md](./MVP-008-integration-route-auth-contract.md) | DONE | MVP-007 | 40min | 2026-07-17T00:23:31+08:00 | integrations 路由/鉴权/隐藏契约与旧接口回归通过，详见 MVP 文档。 |
| MVP-009 | [MVP-009-cross-feature-regression.md](./MVP-009-cross-feature-regression.md) | DONE | MVP-005, MVP-008 | 40min | 2026-07-17T00:26:46+08:00 | 前后端集成回归、构建、格式及结构审计通过，详见 MVP 文档。 |

## 依赖说明

- 前端关键路径：`MVP-001 → MVP-002 → MVP-003 → MVP-004 → MVP-005 → MVP-009`。
- 后端关键路径：`MVP-006 → MVP-007 → MVP-008 → MVP-009`。
- `MVP-001` 与 `MVP-006` 可并行启动；前后端分支在 `MVP-009` 汇合。
- 每个 MVP 验证完成后，必须先更新本文件状态和证据，再开始后续 MVP。

## 规划假设

- 每个 MVP 以一名熟悉本仓库的开发者约 40 分钟专注工作为目标，允许范围为约 20–60 分钟。
- 复用现有 `GET /api/v1/admin/accounts/:id/models`，不修改其后端契约。
- 新查询接口固定为 `POST /api/v1/integrations/model-routes/list`。
- 新接口复用现有 integrations 鉴权和加固配置，不新增配置字段。
- 不修改数据库表、Ent Schema 或现有 `model_routing` 持久化结构。
- 文档中的测试命令基于当前 `pnpm`、Vitest 和 Go 项目结构；执行时如测试名发生调整，应记录实际命令但不得降低验收范围。
