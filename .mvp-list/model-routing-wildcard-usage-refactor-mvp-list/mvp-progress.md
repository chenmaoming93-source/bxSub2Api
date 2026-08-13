# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/model-routing-wildcard-usage-refactor-implementation-plan.md`
- Target effort per MVP: `假设为每个 MVP 一个专注开发日`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-05T10:48:20+08:00`
- Overall: `10/10 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-account-single-model-validation.md](./MVP-001-account-single-model-validation.md) | DONE | none | 1 个专注开发日 | 2026-08-05T09:42:37+08:00 | 唯一模型校验及写入口接入完成；`go test ./internal/service ./internal/handler/admin` 通过。 |
| MVP-002 | [MVP-002-model-routing-contract-and-editor.md](./MVP-002-model-routing-contract-and-editor.md) | DONE | none | 1 个专注开发日 | 2026-08-05T09:49:00+08:00 | 路由候选去 model 且兼容旧 JSON；后端、前端定向测试及 typecheck 通过。 |
| MVP-003 | [MVP-003-route-account-cache-first-loading.md](./MVP-003-route-account-cache-first-loading.md) | DONE | none | 1.5 个专注开发日 | 2026-08-05T09:52:58+08:00 | 显式账号批量 cache-first、一次回源及 best-effort 回填完成；定向测试通过。 |
| MVP-004 | [MVP-004-route-account-startup-warmup.md](./MVP-004-route-account-startup-warmup.md) | DONE | MVP-003 | 1 个专注开发日 | 2026-08-05T09:55:07+08:00 | 活动模型路由账号 128-ID 有界启动预热完成；失败降级与定向测试通过。 |
| MVP-005 | [MVP-005-account-bound-upstream-model-routing.md](./MVP-005-account-bound-upstream-model-routing.md) | DONE | MVP-001, MVP-002, MVP-003 | 1.5 个专注开发日 | 2026-08-05T10:06:27+08:00 | 账号 key 字典序模型绑定并贯穿选择/限额/日志；服务与集成定向测试通过。 |
| MVP-006 | [MVP-006-wildcard-quota-backend.md](./MVP-006-wildcard-quota-backend.md) | DONE | none | 1 个专注开发日 | 2026-08-05T10:09:22+08:00 | wildcard 适用性与具体请求 identity 分桶完成；tokenstat 测试通过。 |
| MVP-007 | [MVP-007-quota-wildcard-and-api-key-selector.md](./MVP-007-quota-wildcard-and-api-key-selector.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-08-05T10:14:35+08:00 | wildcard 表单及防抖 API Key 选择器完成；前后端定向测试与 typecheck 通过。 |
| MVP-008 | [MVP-008-usage-route-alias-backend.md](./MVP-008-usage-route-alias-backend.md) | DONE | none | 1 个专注开发日 | 2026-08-05T10:17:29+08:00 | usage/error 现有 model 参数统一为 requested_model 优先的路由别名语义；测试通过。 |
| MVP-009 | [MVP-009-usage-route-alias-ui.md](./MVP-009-usage-route-alias-ui.md) | DONE | MVP-008 | 1 个专注开发日 | 2026-08-05T10:31:50+08:00 | usage 搜索框原位改为路由别名文案，候选仅来自 requested-model；明细/错误/汇总/趋势/图表转发一致，重置清除；定向测试 8 例与 typecheck 通过。 |
| MVP-010 | [MVP-010-cross-module-regression-and-recovery.md](./MVP-010-cross-module-regression-and-recovery.md) | DONE | MVP-004, MVP-005, MVP-007, MVP-009 | 1 个专注开发日 | 2026-08-05T10:48:20+08:00 | 全量回归通过；修复 MVP-002 遗留的 ListGroupModelRoutes 契约回归；热态零查询/预热回源/快照/OAuth/调度回归证据齐备。 |

## 依赖说明

- 可并行起步：MVP-001、MVP-002、MVP-003、MVP-006、MVP-008。
- 模型路由关键链：MVP-003 → MVP-004，以及 MVP-001 + MVP-002 + MVP-003 → MVP-005。
- 限额链：MVP-006 → MVP-007。
- usage 链：MVP-008 → MVP-009。
- 最终集成检查：MVP-004、MVP-005、MVP-007、MVP-009 → MVP-010。

## 规划假设

- 用户未指定单个 MVP 工期，按一个专注开发日拆分；缓存和路由运行时改造因测试面较大允许 1.5 日。
- 不新增数据库列；JSON 契约、Redis 访问和查询语义变化均在现有存储结构上完成。
- `UpdateCredentials` 缺少 outbox 兜底保持为非本次范围。
- 验证命令以当前 Go module、Vitest 和 pnpm 脚本为准；涉及外部 Redis/MySQL 的恢复测试可使用仓库现有集成测试环境。
