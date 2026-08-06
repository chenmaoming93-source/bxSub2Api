# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/external-four-dimensional-token-usage-query-implementation-plan.md`
- Target effort per MVP: `1 个专注开发日（用户未指定，采用默认假设）`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-31T23:10:00+08:00`
- Overall: `6/6 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-route-and-domain-discovery.md](./MVP-001-route-and-domain-discovery.md) | DONE | none | 1 个专注开发日 | 2026-07-31 | 路由、RBAC、领域规则盘点及相关 Go 测试通过 |
| MVP-002 | [MVP-002-current-period-redis-reader.md](./MVP-002-current-period-redis-reader.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-31 | Redis Reader 与写入端兼容测试及 tokenstat 回归通过 |
| MVP-003 | [MVP-003-scoped-dimension-resolution.md](./MVP-003-scoped-dimension-resolution.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-31 | 四维受限解析实现及 Service 测试通过 |
| MVP-004 | [MVP-004-token-usage-query-service.md](./MVP-004-token-usage-query-service.md) | DONE | MVP-002, MVP-003 | 1 个专注开发日 | 2026-07-31 | 精确投影与三周期 Service 测试、Reader 回归通过 |
| MVP-005 | [MVP-005-external-http-endpoint.md](./MVP-005-external-http-endpoint.md) | DONE | MVP-004 | 1 个专注开发日 | 2026-07-31 | Handler 与仅外部 Token 路由契约测试通过 |
| MVP-006 | [MVP-006-integration-and-operational-acceptance.md](./MVP-006-integration-and-operational-acceptance.md) | DONE | MVP-005 | 1 个专注开发日 | 2026-07-31 | Wire、RBAC、全量后端测试及运维文档全部验收通过 |

## 依赖说明

- 关键路径：`MVP-001 -> MVP-002/MVP-003 -> MVP-004 -> MVP-005 -> MVP-006`。
- `MVP-002` 与 `MVP-003` 在 `MVP-001` 完成后可并行实施。
- `MVP-004` 统一 Redis Reader 与业务对象解析，形成可独立测试的领域查询能力。
- `MVP-005` 才暴露 HTTP 路由；`MVP-006` 完成生成式依赖注入、全链路安全边界与回归验收。

## 规划假设

- 每个 MVP 的目标工作量为一个专注开发日，合理范围约为 0.5～1.5 个开发日。
- 使用项目现有 Go、Gin、Wire、Ent、go-redis 和测试工具，不引入新基础框架。
- 最终 URL 由 MVP-001 的路由盘点确定；无冲突时采用 `/api/v1/integrations/token-usage/query`。
- 测试命令以 `backend` 为工作目录；若完整 `go test ./...` 受外部环境阻塞，仍须运行受影响包测试并记录阻塞原因。
