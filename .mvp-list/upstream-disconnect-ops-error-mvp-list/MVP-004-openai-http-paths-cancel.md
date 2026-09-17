# MVP-004：OpenAI HTTP 路径断开取消

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个聚焦开发日`
- Estimate rationale: OpenAI HTTP 包含 Responses、Chat Completions、Messages、raw/compat 和图片等多条入口，需分别接入并覆盖现有 OpenAI usage/failover 测试。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

OpenAI HTTP 网关的 Responses、Chat Completions、Messages 和适用的图片路径在调用方断开后取消模型请求、关闭下游连接并写入 Ops 错误，同时不改变模型上游错误的 failover。

## 背景

相关代码位于：

- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_gateway_chat_completions.go`
- `backend/internal/service/openai_gateway_messages.go`
- `backend/internal/service/openai_images.go`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/handler/openai_chat_completions.go`
- `backend/internal/handler/openai_images.go`

这些路径已有 `ClientDisconnect`、usage recording 和 failover 逻辑，但需要统一改为立即终止下游，而不是继续 drain。

## 范围内

- 接入 MVP-001 的 Context 取消桥接；
- 覆盖 OpenAI Responses HTTP；
- 覆盖 OpenAI Chat Completions HTTP；
- 覆盖 OpenAI Messages 兼容 HTTP；
- 覆盖适用的图片生成流式/非流式路径；
- 调用 MVP-002 写入 `client_disconnected`；
- 断开路径跳过 OpenAI normal usage recording；
- 保留正常 OpenAI upstream failover、account switch 和 retry；
- 验证 API Key、余额、Token 和图片计费不被触发。

## 范围外

- 不处理 OpenAI WebSocket；
- 不改变 OpenAI 模型选择或重试参数；
- 不改变正常响应协议；
- 不写入 `usage_logs`。

## 实现说明

- 重点检查 `OpenAIForwardResult.ClientDisconnect` 的生成和 handler 处理分支；
- 确保 `recordCyberPolicyIfMarked` 等已有特殊错误逻辑不会与本场景重复写入或误分类；
- 断开时不得进入 `RecordUsage` 或 `RecordCyberPolicyUsageLog`；
- 对图片请求使用现有 mandatory usage 记录机制的边界进行确认，断开错误必须明确排除；
- 保持 `UpstreamFailoverError` 对模型供应商错误的既有行为。

## 验收标准

- [x] Responses HTTP 断开后下游请求被取消；
- [x] Chat Completions HTTP 断开后下游请求被取消；
- [x] Messages 兼容 HTTP 断开后下游请求被取消；
- [x] 图片路径断开后下游请求被取消并关闭响应体；
- [x] 每个断开请求最多写一条 Ops 错误；
- [x] 断开路径不写 usage、不扣费、不更新 Token/配额；
- [x] 模型供应商错误且调用方仍在线时，原有 failover 测试通过；
- [x] 现有 OpenAI 响应和错误协议测试不回归。

## 验证计划

- `go test ./internal/service/... ./internal/handler/...`
- 重点运行 `openai_gateway_service_test.go`、`openai_gateway_chat_completions_test.go`、`openai_gateway_record_usage_test.go`、`openai_images_test.go` 以及相关 handler 测试；
- 使用可控 downstream transport 验证请求取消、Body Close 和未调用 usage recorder。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| OpenAI HTTP 专项 | `go test ./internal/service -run 'Test(ForwardAsChatCompletions|ForwardAsRawChatCompletions|OpenAIStreaming|OpenAIGatewayServiceForwardImages)' -count=1` | 通过；Responses、Chat、Images 断开不再 drain 并返回 `ClientDisconnect`。注意 `ForwardAsRawChatCompletions` 属于 `//go:build unit` 文件，默认标签下不参与执行（见下方 raw Chat 条目）。 |
| Handler 回归 | `go test ./internal/handler -run 'Test(OpenAI|Gateway|Failover)' -count=1` | 通过；Responses/Messages/Chat/Images 在 failover 与 usage 前统一排除断开。 |
| 服务/处理器编译 | `go test ./internal/service ./internal/handler -run '^$'` | 通过。 |
| OpenAI 断开单测 | `openai_gateway_chat_completions_test.go`、`openai_gateway_service_test.go`、`openai_images_test.go` | 旧 drain/ignore-cancel 断言已改为 upstream cancel、`ClientDisconnect` 和零 usage。 |
| Anthropic 兼容断开单测 | `openai_compat_model_test.go` | `TestForwardAsAnthropic_ClientDisconnectCancelsUpstream`、`TestForwardAsAnthropic_MissingTerminalAfterClientDisconnectSkipsOpsAndFailover`、`TestForwardAsAnthropic_UpstreamRequestCancelsOnClientCancel` 已改为取消上游、零 usage、`ClientDisconnect`，且不写上游 Ops、不触发 failover。 |
| OAuth 透传断开单测 | `openai_oauth_passthrough_test.go` | `..._StreamClientDisconnectCancelsUpstream`、`..._UpstreamRequestCancelsOnClientCancel`（Passthrough/Legacy）断言上游请求 Context 为 `context.Canceled`。 |
| 取消实现 | `openai_gateway_service.go`、`openai_gateway_chat_completions.go`、`openai_gateway_chat_completions_raw.go`、`openai_gateway_messages.go`、`openai_gateway_responses_chat_fallback.go`、`openai_images.go`、`openai_images_responses.go`、`openai_embeddings.go` | 统一使用 `detachOpenAIUpstreamContext`；writer/read failure 取消下游，HTTP response body 由 defer 关闭；断开结果统一不携带可计费 usage。 |
| Embeddings 断开（补充） | `openai_embeddings_test.go`、`handler/openai_embeddings.go` | `TestForwardEmbeddings_ClientDisconnectSkipsUsage` 通过；non-streaming embeddings 在入站 Context 取消时返回 `ClientDisconnect` 且 usage 归零，handler 在 `RecordUsage` 前短路。 |
| raw Chat 断开（unit 标签） | `openai_gateway_chat_completions_raw_test.go` | 两个用例已改为取消语义（零 usage、`ClientDisconnect`、上游 Context `context.Canceled`）；该文件带 `//go:build unit`，而 `-tags=unit` 因既有陈旧 mock 缺少 `AccountRepository.ExistsByName` 无法编译，故这两个用例本轮未能实际执行。 |
| 相关包完整命令 | `go test ./internal/service ./internal/handler -count=1` | 全部通过（非仅子集）。 |
| 相关包全量命令 | `go test ./internal/service/... ./internal/handler/...` | `internal/service`、`internal/handler` 及子包通过；`internal/handler/admin` 有既有基线失败 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting`（路由别名候选模型校验），与本 MVP 无关。 |

## 执行记录

- 2026-09-16：完成 OpenAI Responses、Chat Completions、Messages、raw Chat 与 Images HTTP 取消桥接；handler 在 failover/usage 前排除断开并进入 MVP-002 Ops pending marker，修正 Ops monitoring gate。
- 2026-09-16：补齐 `openai_compat_model_test.go`、`openai_oauth_passthrough_test.go` 中仍断言旧 drain / 忽略客户端取消行为的 5 个用例，改为验证取消上游、`ClientDisconnect`、零 usage；此后 `go test ./internal/service ./internal/handler` 全绿。
- 2026-09-16：统一“断开结果不携带可计费 usage”不变式（含 raw Chat 与 embeddings）；补充 non-streaming embeddings 断开短路及可运行的 `TestForwardEmbeddings_ClientDisconnectSkipsUsage`。`openai_gateway_chat_completions_raw_test.go` 的同类用例受既有 `unit` 标签构建阻塞影响未能执行，已如实记录。
- 纠正（本次）：`isOpenAIClientDisconnected` 原先只要入站 Context 已取消就判为断开（不校验 error、也不看是否真的断开），会把上游 429/5xx 乃至**已成功**的请求都当成客户端断开——既顶掉真实错误记录，又跳过计费。现收紧为：只有服务层显式标记 `result.ClientDisconnect`，或错误本身即 `context.Canceled`/`context.DeadlineExceeded` **且**入站 Context 已取消，才判定为断开（与既有 `isGatewayClientDisconnected` 同构）。服务层 21 处显式 `ClientDisconnect: true` 覆盖全部 OpenAI 路径，因此兜底推导是冗余的。同时 Ops 中间件的 pending 分支改为条件短路：仅当确需写入断开记录、且请求**没有**真实上游错误时才短路，否则完整回落到原有错误记录逻辑（真实错误优先）。新增 `TestIsOpenAIClientDisconnected_RequiresAnExplicitDisconnectSignal`、`TestOpsRequestCarriesRealUpstreamError`、`TestOpsErrorLoggerMiddleware_RealUpstreamErrorBeatsDisconnectRecord`、`TestOpsErrorLoggerMiddleware_RecordsDisconnectWithoutUpstreamError` 锁定行为。

