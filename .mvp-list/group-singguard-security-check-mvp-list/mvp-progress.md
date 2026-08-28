# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: [../..//.plans/group-singguard-security-check-implementation-plan.md](../../.plans/group-singguard-security-check-implementation-plan.md)
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发者日；如实现或验证明显超过 1.5 个开发者日，应继续拆分。
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-27T20:09:02+08:00`
- Overall: `9/9 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-config-data-model.md](./MVP-001-config-data-model.md) | DONE | none | 1 个开发者日 | 2026-08-26T15:06:11+08:00 | [证据](./MVP-001-config-data-model.md#完成证据) |
| MVP-002 | [MVP-002-config-cache-propagation.md](./MVP-002-config-cache-propagation.md) | DONE | MVP-001 | 1 个开发者日 | 2026-08-26T15:14:45+08:00 | [证据](./MVP-002-config-cache-propagation.md#完成证据) |
| MVP-003 | [MVP-003-singguard-check-core.md](./MVP-003-singguard-check-core.md) | DONE | MVP-001 | 1.5 个开发者日 | 2026-08-26T15:30:14+08:00 | [证据](./MVP-003-singguard-check-core.md#完成证据) |
| MVP-004 | [MVP-004-http-gateway-integration.md](./MVP-004-http-gateway-integration.md) | DONE | MVP-002, MVP-003 | 1 个开发者日 | 2026-08-26T15:43:27+08:00 | [证据](./MVP-004-http-gateway-integration.md#完成证据) |
| MVP-005 | [MVP-005-websocket-integration.md](./MVP-005-websocket-integration.md) | DONE | MVP-004 | 1 个开发者日 | 2026-08-26T15:46:27+08:00 | [证据](./MVP-005-websocket-integration.md#完成证据) |
| MVP-006 | [MVP-006-async-collection-protection.md](./MVP-006-async-collection-protection.md) | DONE | MVP-003, MVP-004 | 1.5 个开发者日 | 2026-08-26T16:07:50+08:00 | [证据](./MVP-006-async-collection-protection.md#完成证据) |
| MVP-007 | [MVP-007-admin-configuration-ui.md](./MVP-007-admin-configuration-ui.md) | DONE | MVP-001, MVP-002 | 1 个开发者日 | 2026-08-26T16:19:33+08:00 | [证据](./MVP-007-admin-configuration-ui.md#完成证据) |
| MVP-008 | [MVP-008-log-query-retention-ui.md](./MVP-008-log-query-retention-ui.md) | DONE | MVP-006, MVP-007 | 1.5 个开发者日 | 2026-08-26T16:35:16+08:00 | [证据](./MVP-008-log-query-retention-ui.md#完成证据) |
| MVP-009 | [MVP-009-security-log-ui-refinement.md](./MVP-009-security-log-ui-refinement.md) | PENDING | MVP-008 | 1 个开发者日 |  |  |

## 依赖说明

- 关键链路：`MVP-001 → MVP-002 → MVP-003 → MVP-004 → MVP-005`。
- 采集链路从 `MVP-003` 和 `MVP-004` 开始，最终与管理页面汇合到 `MVP-008`，界面可读性和独立页面由 `MVP-009` 补充。
- `MVP-007` 与 `MVP-004` 至 `MVP-006` 可在配置接口稳定后并行推进。

## 规划假设

- 使用现有 Go/Gin/Ent、MySQL/GoldenDB、Redis、Vue 3/TypeScript 技术栈。
- 不引入消息队列或对象存储。
- SingGuard 真实服务仅在内网可访问，自动化测试使用本地 HTTP mock，不要求真实联调。
- 第一阶段只覆盖聊天模型请求，不覆盖图片生成、图片编辑和 embeddings。
- 请求体存储字段采用大字段并在应用层提前截断；具体迁移语法需兼容项目支持的 MySQL/GoldenDB 方言。
- 每个 MVP 的验证命令以当前仓库 Makefile 和现有测试结构为基础；若命令因环境缺失无法执行，不得标记为 DONE。
