# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/scene-token-usage-implementation-plan.md`
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发日；按垂直成果拆分，单个 MVP 控制在约 0.5～1.5 个开发日内。
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-26T22:49:06.0809887+08:00`
- Overall: `4/4 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-group-scene-name.md](./MVP-001-group-scene-name.md) | DONE | none | 1 个开发日 | 2026-08-26T22:12:16.5193518+08:00 | [MVP-001-group-scene-name.md](./MVP-001-group-scene-name.md)：验收项全选，后端聚焦测试与前端 typecheck 通过；全量测试的既有异步错误已记录 |
| MVP-002 | [MVP-002-shared-scene-usage-query.md](./MVP-002-shared-scene-usage-query.md) | DONE | MVP-001 | 1 个开发日 | 2026-08-26T22:27:18.6691651+08:00 | [MVP-002-shared-scene-usage-query.md](./MVP-002-shared-scene-usage-query.md)：验收项全选，场景用量聚焦测试和动态统计回归通过 |
| MVP-003 | [MVP-003-usage-api-adapters.md](./MVP-003-usage-api-adapters.md) | DONE | MVP-002 | 1 个开发日 | 2026-08-26T22:35:43.3840537+08:00 | [MVP-003-usage-api-adapters.md](./MVP-003-usage-api-adapters.md)：两个接口契约/外部鉴权聚焦测试通过，相关回归既有失败已记录 |
| MVP-004 | [MVP-004-admin-usage-view.md](./MVP-004-admin-usage-view.md) | DONE | MVP-003 | 1 个开发日 | 2026-08-26T22:49:06.0809887+08:00 | [MVP-004-admin-usage-view.md](./MVP-004-admin-usage-view.md)：页面测试、typecheck 和 build 通过 |

## 依赖说明

- 关键路径：MVP-001 → MVP-002 → MVP-003 → MVP-004。
- MVP-001 完成分组字段基础后，MVP-002 才能补全场景名称并查询。
- MVP-002 提供统一业务查询逻辑后，MVP-003 才能接入管理端和外部端。
- MVP-004 依赖管理端接口契约稳定。

## 规划假设

- 不自动创建动态 Token 统计项；部署使用前由管理员在 `/admin/token-statistics` 手动创建、发布并启用 `group_id + account_id + upstream_model`、`total_tokens`、日统计项。
- 管理端和外部端使用同一个后端查询 Service，仅鉴权和 HTTP 适配不同。
- 现有项目测试命令以 `backend/Makefile` 与 `frontend/package.json` 为准。
