# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/integration-api-key-group-implementation-plan.md`
- Target effort per MVP: `40 分钟`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-16T22:12:40+08:00`
- Overall: `7/7 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-platform-key-triple-idempotency.md](./MVP-001-platform-key-triple-idempotency.md) | DONE | none | 40 分钟 | 2026-07-16T19:08:47+08:00 | [服务实现与定向测试](./MVP-001-platform-key-triple-idempotency.md#完成证据) |
| MVP-002 | [MVP-002-repository-triple-lookup.md](./MVP-002-repository-triple-lookup.md) | DONE | MVP-001 | 40 分钟 | 2026-07-16T19:10:55+08:00 | [仓储实现与定向测试](./MVP-002-repository-triple-lookup.md#完成证据) |
| MVP-003 | [MVP-003-api-key-composite-index.md](./MVP-003-api-key-composite-index.md) | DONE | none | 40 分钟 | 2026-07-16T19:12:03+08:00 | [索引归档与静态测试](./MVP-003-api-key-composite-index.md#完成证据) |
| MVP-004 | [MVP-004-group-access-validation.md](./MVP-004-group-access-validation.md) | DONE | MVP-001 | 40 分钟 | 2026-07-16T19:15:04+08:00 | [分组准入实现与测试](./MVP-004-group-access-validation.md#完成证据) |
| MVP-005 | [MVP-005-provisioning-api-contract.md](./MVP-005-provisioning-api-contract.md) | DONE | MVP-004 | 40 分钟 | 2026-07-16T19:17:38+08:00 | [HTTP 契约与 Handler 测试](./MVP-005-provisioning-api-contract.md#完成证据) |
| MVP-006 | [MVP-006-wiring-and-default-group-cleanup.md](./MVP-006-wiring-and-default-group-cleanup.md) | DONE | MVP-002, MVP-004, MVP-005 | 40 分钟 | 2026-07-16T19:21:18+08:00 | [依赖注入与编译验证](./MVP-006-wiring-and-default-group-cleanup.md#完成证据) |
| MVP-007 | [MVP-007-integration-and-regression-verification.md](./MVP-007-integration-and-regression-verification.md) | DONE | MVP-003, MVP-006 | 40 分钟 | 2026-07-16T22:12:40+08:00 | [全量 unit 与 integration 回归](./MVP-007-integration-and-regression-verification.md#完成证据) |

## 依赖说明

- 关键路径：`MVP-001 -> MVP-004 -> MVP-005 -> MVP-006 -> MVP-007`。
- `MVP-002` 在 `MVP-001` 明确三元组仓储契约后实施，并在 `MVP-006` 汇合。
- `MVP-003` 与 `MVP-001` 可并行，最终由 `MVP-007` 验证索引归档和回归结果。

## 规划假设

- 每项以一个专注的 40 分钟实现与验证窗口为目标；若遇到现有接口连锁编译修改，允许在 20–60 分钟范围内完成。
- 后端测试命令以 `backend` 为工作目录执行；优先运行定向包测试，再在最终 MVP 运行 `go test -tags=unit ./...`。
- 表结构变更 SQL 必须新增到 `backend/sqlArchiving/`，不得新增或修改 `backend/migrations/*.sql`。
- 本功能不修改配置，因此不涉及 `backend/config/config.yaml` 或 `deploy/config.example.yaml`。
- `group_name` 使用去除首尾空格后的精确名称查询；订阅分组直接拒绝。
