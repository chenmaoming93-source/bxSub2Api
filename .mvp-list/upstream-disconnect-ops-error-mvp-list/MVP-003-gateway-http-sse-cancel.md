# MVP-003：Gateway HTTP/SSE 断开取消与资源释放

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个聚焦开发日`
- Estimate rationale: 聚焦 GatewayService 通用 HTTP/SSE 转发链路和已有流式测试，验证断开后取消、关闭响应体并跳过 failover。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

GatewayService 的普通 HTTP 和 SSE 转发在调用方断开后停止 drain，取消下游模型请求并写入一条 Ops 错误；模型供应商自身错误仍保留原有 failover。

## 背景

现有 `backend/internal/service/gateway_service.go`、`gateway_forward_as_chat_completions.go` 和 `gateway_forward_as_responses.go` 包含 SSE 读写、客户端断开和 upstream error 处理。当前部分逻辑在客户端写失败后继续读取下游，以便收集 usage。

## 范围内

- 将 MVP-001 的可取消 Context 接入 GatewayService HTTP/SSE 转发；
- 识别入站 Context 取消和向调用方写失败；
- 取消下游请求并关闭 HTTP response body；
- 停止后续 SSE drain；
- 标记 `client_disconnected` 并调用 MVP-002 的 Ops 写入入口；
- 在进入 failover 前增加调用方断开保护；
- 保持模型供应商 429/5xx/连接错误的原有 failover 行为；
- 保证并发槽位和请求资源正常释放。

## 范围外

- 不接入 OpenAI WebSocket；
- 不接入 Gemini/Anthropic 特殊兼容路径；
- 不新增 usage_logs 记录；
- 不修改模型账号选择策略和 failover 参数。

## 实现说明

- 重点检查 `gateway_service.go` 中 stream forwarding、`detachStreamUpstreamContext` 调用和 `UpstreamFailoverError` 分支；
- 重点覆盖 `gateway_forward_as_chat_completions.go`、`gateway_forward_as_responses.go`；
- 确保 writer error 产生的断开状态与模型上游读取错误区分；
- 断开后不能继续调用 normal `RecordUsage`；
- 关闭 response body 后仍需让 Ops 记录使用独立 Context 完成。

## 验收标准

- [x] SSE 首 Token 前调用方断开时，下游请求被取消且不执行 failover；
- [x] SSE 首 Token 后写出失败时，下游请求被取消且不继续 drain；
- [x] HTTP 非流式请求 Context 取消时，下游 response body 被关闭；
- [x] 模型供应商 429/5xx/连接异常且调用方仍在线时，原有 failover 测试通过；
- [x] 断开请求只产生 Ops 错误，不产生 usage 记录；
- [x] 账号并发 release 仍执行且不会重复 release；
- [x] 没有新增 goroutine 或 response body 泄漏。

## 验证计划

- `go test ./internal/service/... ./internal/handler/...`
- 重点运行 `gateway_streaming_test.go`、`gateway_service_streaming_test.go`、`gateway_forward_as_chat_completions_test.go` 和 `gateway_forward_as_responses_test.go`；
- 使用可控 `httptest` 上游验证 Context 取消和 Body Close。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 专项测试 | `go test ./internal/service -run 'TestHandle(CCStreamingFromAnthropic|ResponsesStreamingResponse)_CancelsUpstreamOnClientWriteFailure|TestStreamUpstreamContextControl' -count=1` | 通过；SSE writer failure 会取消桥接的 upstream context，并返回 `ClientDisconnect`。 |
| 回归测试 | `go test ./internal/service ./internal/handler -count=1` | 通过。 |
| Gateway/failover 回归 | `go test ./internal/service -run 'Test(Gateway|Handle|Forward)' -count=1`；`go test ./internal/handler -run 'Test(Gateway|Failover)' -count=1` | 均通过；正常供应商错误仍保持原有 failover。 |
| 相关包验证 | `go test ./internal/service/... ./internal/handler/...` | service、handler 主包及子包通过；`internal/handler/admin` 存在既有失败 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting`（路由别名候选账号约束），与本 MVP 无关。 |
| 代码路径 | `gateway_service.go`、`gateway_forward_as_chat_completions.go`、`gateway_forward_as_responses.go`、三个 Gateway handler | writer/context disconnect 取消 downstream、停止 drain、关闭 response body（defer）、短路 failover 与 usage；通过 Gin pending marker 进入 MVP-002 Ops 记录。 |
| 测试路径 | `internal/service/gateway_disconnect_stream_test.go`、`gateway_anthropic_apikey_passthrough_test.go` | 覆盖 Chat/Responses SSE 写失败取消、Anthropic passthrough 写失败立即返回；既有断开回归已更新为“不继续 drain”。 |

## 执行记录

- 2026-09-16：完成 Gateway 通用 HTTP/SSE 取消接入；复用 MVP-001 的 context bridge，writer error 和入站取消均绕过 failover，handler 在 usage 前短路并排队 `client_disconnected` Ops 记录。

