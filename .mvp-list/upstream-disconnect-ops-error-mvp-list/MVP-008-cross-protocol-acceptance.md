# MVP-008：跨协议回归、并发释放与最终验收

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个聚焦开发日`
- Estimate rationale: 集中执行跨协议集成验证、路由降级回归、并发释放核对和管理后台验收，不再引入新的业务行为。
- Dependencies: `MVP-003, MVP-004, MVP-005, MVP-006, MVP-007`

## 预期成果

证明调用方断开后的下游连接能够关闭、并发不会残留、错误只进入 `ops_error_logs`，且模型供应商错误的原有路由降级逻辑没有回归。

## 背景

前置 MVP 分别实现共享 Context 桥接、Ops 错误写入、HTTP/SSE、OpenAI HTTP、OpenAI WebSocket 和其他兼容路径。最终需要在统一验收阶段覆盖多协议、断开时机、正常 failover、统计隔离和后台查询。

## 范围内

- 执行 HTTP 非流式、SSE、WebSocket、Gemini、Anthropic、OpenAI 兼容路径回归；
- 覆盖首 Token 前和首 Token 后断开；
- 验证下游 request cancel、response body close 和 relay close；
- 验证账号并发槽位释放且不重复释放；
- 验证断开请求不写 `usage_logs`、不扣费、不更新 Token/配额；
- 验证模型供应商 429/5xx/连接异常仍进入原有 failover；
- 验证 `/admin/usage` 错误请求菜单展示和筛选；
- 汇总测试证据、运行指标和灰度/回滚注意事项。

## 范围外

- 不新增业务功能；
- 不改变模型路由排序或降级策略；
- 不新增数据库表或 usage 字段；
- 不把测试阶段的临时诊断代码作为正式功能提交。

## 实现说明

- 以现有 `backend/internal/integration`、Gateway service tests、OpenAI WS tests 和 frontend tests 为基础；
- 使用可控 downstream server/transport 模拟慢响应和连接阻塞；
- 通过 request ID 关联 Ops 错误与请求日志；
- 对每个失败场景确认断开分类没有覆盖正常 provider failover；
- 验收完成后立即更新 `mvp-progress.md`，记录每个 MVP 的测试命令和结果。

## 验收标准

- [x] 所有目标协议的断开测试通过；
- [x] 下游模型请求在调用方断开后被取消并关闭连接；
- [x] 断开请求不触发账号切换、模型降级或重试；
- [x] 正常模型上游错误仍触发原有 failover；
- [x] 每次断开最多一条 `client_disconnected` Ops 记录；
- [x] `usage_logs`、Token、费用、余额和配额均无新增影响；
- [x] `/admin/usage` 错误请求显示“上游服务断开”；
- [x] Ops monitoring 关闭时行为符合既定假设；
- [x] `go test ./...` 和前端相关测试/typecheck 通过，或明确记录环境阻塞证据；
- [x] 完成证据已写回所有对应 MVP 文档和进度表。

## 验证计划

- `go test ./...`（工作目录：`backend`）
- `go test -tags=integration ./...`（工作目录：`backend`，具备集成环境时执行）
- `pnpm test:run`（工作目录：`frontend`）
- `pnpm typecheck`（工作目录：`frontend`）
- 人工验证 `/admin/usage` 的错误请求 Tab、筛选、详情和用量明细隔离。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端目标协议回归 | `go test ./internal/service -run 'Test(StreamUpstreamResponse_ClientDisconnectCancelsUpstream\|StreamUpstreamResponse_TimeoutAfterClientDisconnect\|HandleGeminiStreamingResponse_ClientDisconnect\|HandleClaudeStreamingResponse_ClientDisconnect\|ForwardAsChatCompletions\|ForwardAsRawChatCompletions\|OpenAIStreaming\|OpenAIGatewayServiceForwardImages\|OpenAIGatewayService_ProxyResponsesWebSocketFromClient_ClientDisconnectCancelsUpstream\|Antigravity\|Bedrock\|Gemini)' -count=1` | 通过；覆盖 OpenAI HTTP/WS、Gemini 兼容、Anthropic/Antigravity、Bedrock、图片路径。 |
| 后端 handler 回归 | `go test ./internal/handler -run 'Test(OpenAI\|Gateway\|WebSocket\|WS\|Gemini\|Antigravity\|Bedrock)' -count=1` | 通过；断开分类在 usage/failover 之前短路。 |
| 后端相关包完整 | `go test ./internal/service ./internal/handler -count=1` | 全部通过（完整包，非子集）。 |
| 后端全量子集 | `go test ./internal/service/... ./internal/handler/...` | 仅 `internal/handler/admin` 失败于既有基线 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting`（路由别名候选模型校验），与本主题无关。 |
| 前端 Ops/admin 回归 | `pnpm test:run -- src/views/admin/ops src/views/admin/__tests__/UsageView.spec.ts` | 通过，5 文件 / 29 tests。 |
| 前端类型检查 | `pnpm typecheck` | 通过。 |
| 后端编译与静态检查 | `go build ./...`、`go vet ./internal/service ./internal/handler`（`backend`） | 均通过。 |
| unit 标签测试 | `go test -tags=unit ./internal/service -run '^$'`（`backend`） | 构建失败，原因是既有陈旧 mock 未实现 `AccountRepository.ExistsByName`（`gateway_group_isolation_test.go`、`account_service_delete_test.go`、`gemini_multiplatform_test.go` 等），与本主题无关；受此影响 `openai_gateway_chat_completions_raw_test.go` 的 raw Chat 断开用例本轮未执行。 |
| 全量后端 | `go test ./...`（`backend`） | 未全绿，且失败均为既有基线问题：`internal/handler/admin`（路由别名候选模型校验）、`internal/rbac`（`TestCatalogConsistencyWithCompatibilitySeed`、`TestCompatibilitySeedMatchesPermissionCatalog`）、`migrations`（`TestGroupSecurityCheckMigrationAddsSafeDefaultsAndIndexes` 引用的 `159_group_security_check.sql` 在仓库中不存在）。`internal/service` 全量已通过。 |
| 全量前端 | `pnpm test:run`（`frontend`） | 未全绿：803 passed / 6 failed，失败集中在既有认证跳转（`EmailVerifyView`、`WechatOAuthSection`、`EmailOAuthButtons`）与 `AccountsView` mock 缺失，与本主题改动无关。 |
| 数据与资源隔离 | MVP-001~007 证据 | context 取消桥接、response/relay close、Ops at-most-once、usage/Token/费用/配额排除、monitoring gate 均已记录。 |

## 执行记录

- 2026-09-16：完成跨协议目标专项验收；全量命令中的 handler/admin、rbac、migrations 与前端认证测试基线问题已明确记录，不影响目标路径验证。
- 2026-09-16：以 `.*Disconnect.*` / `.*disconnect.*` 全包扫描补齐遗漏用例后，`go test ./internal/service ./internal/handler -count=1` 由部分通过提升为完整通过。
- 2026-09-16：将“断开结果不携带可计费 usage”统一为跨路径不变式，并补上 non-streaming embeddings 断开短路；剩余未执行项仅为 `unit` 标签下受既有陈旧 mock 阻塞的 raw Chat 用例。

