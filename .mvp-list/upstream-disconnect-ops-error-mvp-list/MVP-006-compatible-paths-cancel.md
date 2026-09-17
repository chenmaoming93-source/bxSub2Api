# MVP-006：Gemini、Anthropic 兼容及图片路径断开取消

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个聚焦开发日`
- Estimate rationale: 覆盖非 OpenAI WebSocket 的兼容、Antigravity、Bedrock 和图片流式实现，路径较多，拆为一个垂直协议组并配套专项测试。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

Gemini、Anthropic、Antigravity、Bedrock 及其他兼容转发路径在调用方断开后，能取消下游连接、停止 drain、释放并发资源并写入统一 Ops 错误；正常模型上游错误仍走原有 failover。

## 背景

仓库中存在多个独立流式实现，例如：

- `backend/internal/service/antigravity_gateway_service.go`
- `backend/internal/service/bedrock_stream.go`
- `backend/internal/service/gemini_chat_completions_compat_service.go`
- `backend/internal/service/gateway_forward_as_chat_completions.go`
- `backend/internal/service/gateway_forward_as_responses.go`
- 对应的 Gemini/Anthropic handler。

这些实现已有 `clientDisconnect` 或写失败处理，但部分路径仍可能为了 usage 继续读取上游。

## 范围内

- 梳理并接入 Gemini 兼容转发；
- 梳理并接入 Anthropic/Antigravity 流式转发；
- 梳理并接入 Bedrock 流式转发；
- 接入适用的图片流式路径；
- 使用统一 Context 取消桥接；
- 停止调用方断开后的下游 drain；
- 关闭下游 response body、relay 或 provider stream；
- 调用 MVP-002 写入 Ops 错误；
- 保留正常 provider error 的 failover 和账号健康处理。

## 范围外

- 不修改各平台请求/响应协议转换规则；
- 不修改模型账号选择策略；
- 不新增 usage_logs 字段；
- 不改变正常 Token/费用统计路径。

## 实现说明

- 逐个检查各实现对 `context.Canceled`、writer error、`clientDisconnect` 和 stream timeout 的处理；
- 只有确认入站调用方断开时才使用 `client_disconnected`；
- provider stream 自身断开且调用方仍在线时，必须保留原有 failover；
- 确认账号 release、response close 和 Ops record 的调用顺序；
- 对图片路径明确排除正常 mandatory usage recording。

## 验收标准

- [x] Gemini 兼容路径调用方断开后下游被取消；
- [x] Anthropic/Antigravity 路径调用方断开后不继续 drain；
- [x] Bedrock 路径调用方断开后连接和 response body 被关闭；
- [x] 图片流式路径调用方断开后不产生 usage/计费记录；
- [x] 各路径每次断开最多写一条 Ops 错误；
- [x] provider 错误且调用方仍在线时原有 failover 不回归；
- [x] 所有目标路径的并发 release 和资源关闭测试通过。

## 验证计划

- `go test ./internal/service/... ./internal/handler/...`
- 重点运行 `antigravity_gateway_service_test.go`、`bedrock_stream_test.go`、`gateway_streaming_test.go`、Gemini compatibility 测试及对应 handler 测试；
- 使用可控 stream/response 验证调用方取消后下游读循环终止。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 兼容路径专项 | `go test ./internal/service -run 'Test(StreamUpstreamResponse_ClientDisconnectCancelsUpstream|StreamUpstreamResponse_TimeoutAfterClientDisconnect|HandleGeminiStreamingResponse_ClientDisconnect|HandleClaudeStreamingResponse_ClientDisconnect|Antigravity|Bedrock|Gemini)' -count=1` | 通过；Gemini 兼容、Antigravity/Anthropic 流式及 Bedrock 相关测试通过，断开结果清空 usage。 |
| OpenAI/Gateway 兼容回归 | `go test ./internal/service -run 'Test(ForwardAsChatCompletions|ForwardAsRawChatCompletions|OpenAIStreaming|OpenAIGatewayServiceForwardImages)' -count=1` | 通过；既有 HTTP/图片取消路径保持不回归。 |
| Handler 回归 | `go test ./internal/handler -run 'Test(OpenAI|Gateway|WebSocket|WS|Gemini|Antigravity|Bedrock)' -count=1` | 通过；统一 ClientDisconnect Ops 短路生效。 |
| 编译验证 | `go test ./internal/service ./internal/handler -run '^$'` | 通过。 |
| 取消实现 | `antigravity_gateway_service.go`、`bedrock_stream.go`、`gemini_chat_completions_compat_service.go` | writer/read failure 立即返回断开结果，取消上游 context，停止 drain；Antigravity/Bedrock/Gemini 断开 usage 清零，正常 provider 错误仍保留 failover。 |

## 执行记录

- 2026-09-16：完成 Gemini compatibility、Antigravity/Anthropic 兼容、Bedrock 流式路径的 context bridge、立即停止 drain 与零 usage 结果分类。
- 纠正（本次）：`AntigravityGatewayService.Forward` 与 `GeminiMessagesCompatService.ForwardAsChatCompletions` 当时实际用的是 `detachUpstreamContext`（纯 `context.WithoutCancel`，**没有**取消桥），调用方断开只能靠"写客户端失败"察觉；上游静默且 keepalive 关闭时下游会继续运行。现已改用 `detachStreamUpstreamContext(ctx, true)`，并在该桥生效后必然出现的取消点上补齐断开语义：Antigravity 重试循环失败且调用方 Context 已取消时返回 `newClientDisconnectedForwardResult`（而不是当成上游失败去 failover）；Gemini 兼容路径在 `Do` 失败、非流式收集失败、流式读被取消时直接返回 `ClientDisconnect` 结果，不再为已离开的调用方重试，也不再把取消记成上游错误。

