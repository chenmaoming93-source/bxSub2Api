# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/upstream-disconnect-ops-error-implementation-plan.md`
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发日；必要时允许 0.5–1.5 个开发日
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-09-16T21:10:00+08:00`
- Overall: `8/8 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-context-cancel-bridge.md](./MVP-001-context-cancel-bridge.md) | DONE | none | 1 个聚焦开发日 | 2026-09-16T18:03:53+08:00 | [验证证据](./MVP-001-context-cancel-bridge.md#完成证据) |
| MVP-002 | [MVP-002-ops-disconnect-error-record.md](./MVP-002-ops-disconnect-error-record.md) | DONE | none | 1 个聚焦开发日 | 2026-09-16T18:19:26+08:00 | [验证证据](./MVP-002-ops-disconnect-error-record.md#完成证据) |
| MVP-003 | [MVP-003-gateway-http-sse-cancel.md](./MVP-003-gateway-http-sse-cancel.md) | DONE | MVP-001, MVP-002 | 1 个聚焦开发日 | 2026-09-16T18:42:30+08:00 | [验证证据](./MVP-003-gateway-http-sse-cancel.md#完成证据) |
| MVP-004 | [MVP-004-openai-http-paths-cancel.md](./MVP-004-openai-http-paths-cancel.md) | DONE | MVP-001, MVP-002 | 1.5 个聚焦开发日 | 2026-09-16T19:20:10+08:00 | [验证证据](./MVP-004-openai-http-paths-cancel.md#完成证据) |
| MVP-005 | [MVP-005-openai-websocket-cancel.md](./MVP-005-openai-websocket-cancel.md) | DONE | MVP-001, MVP-002 | 1 个聚焦开发日 | 2026-09-16T19:30:20+08:00 | [验证证据](./MVP-005-openai-websocket-cancel.md#完成证据) |
| MVP-006 | [MVP-006-compatible-paths-cancel.md](./MVP-006-compatible-paths-cancel.md) | DONE | MVP-001, MVP-002 | 1.5 个聚焦开发日 | 2026-09-16T19:40:10+08:00 | [验证证据](./MVP-006-compatible-paths-cancel.md#完成证据) |
| MVP-007 | [MVP-007-admin-error-display.md](./MVP-007-admin-error-display.md) | DONE | MVP-002 | 0.5 个聚焦开发日 | 2026-09-16T19:50:10+08:00 | [验证证据](./MVP-007-admin-error-display.md#完成证据) |
| MVP-008 | [MVP-008-cross-protocol-acceptance.md](./MVP-008-cross-protocol-acceptance.md) | DONE | MVP-003, MVP-004, MVP-005, MVP-006, MVP-007 | 1 个聚焦开发日 | 2026-09-16T21:10:00+08:00 | [验证证据](./MVP-008-cross-protocol-acceptance.md#完成证据) |

## 依赖说明

- MVP-001 和 MVP-002 可并行启动。
- MVP-003、MVP-004、MVP-005、MVP-006 在共享取消能力和 Ops 写入能力完成后可并行实施。
- MVP-007 可在 MVP-002 完成后并行实施。
- MVP-008 是最终跨协议回归、并发释放和管理后台验收阶段。
- 关键路径为：`MVP-001 + MVP-002 → MVP-003/004/005/006 → MVP-008`。

## 规划假设

- 本拆分未收到用户指定的单个 MVP 工时，按每个 MVP 约 1 个聚焦开发日拆分。
- 估算包含实现和对应的自动化验证，不包含无关重构。
- 仅使用现有 `ops_error_logs`，不新增 `usage_logs` 字段或数据库表。
- Ops monitoring 关闭时沿用现有行为；开启时才要求错误记录可查询。
- `context.WithoutCancel` 保留为下游 Context 的基础语义，通过原始入站 Context 桥接显式取消。
- 正常模型上游错误继续使用原有 failover，不修改路由选择策略。

## 验收后纠正（本次）

MVP 全部标记 DONE 后发现了两个缺陷，已修复并把纠正记录写回对应 MVP 文档：

1. **P0 写入失败**：MVP-002 把批准方案字段表中的 `is_retryable` 误当成落库需求，新增 `migrations/149_ops_error_logs_add_is_retryable.sql` 并写入 INSERT。该列早已被迁移 136 随重试功能删除，**线上不存在**，导致所有 `ops_error_logs` 写入以 MySQL 1054 静默失败（错误记录全部丢失）。已彻底移除该字段、删除无效迁移文件，并把测试反转为"禁止出现该列"+列/占位符/参数计数一致性校验。参见 MVP-002。
2. **Gemini/Antigravity 缺取消桥**：MVP-006 声称"使用统一 Context 取消桥接"，但这两条路径实际用的是 `detachUpstreamContext`（无桥），断开只能靠写客户端失败察觉。已改用 `detachStreamUpstreamContext(ctx, true)` 并补齐相应断开语义。参见 MVP-006。

另：MVP-004/005 的断开判定原先允许"仅凭入站 Context 被取消"就判为断开，会顶掉真实上游错误记录并跳过计费；现已收紧为必须由服务层显式标记（或错误本身即 context 取消且入站 Context 已取消），并在 Ops 中间件中保证"存在真实上游错误时不写断开行"。参见 MVP-004。
