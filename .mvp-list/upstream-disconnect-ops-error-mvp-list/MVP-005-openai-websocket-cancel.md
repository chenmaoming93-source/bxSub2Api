# MVP-005：OpenAI WebSocket 断开取消

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个聚焦开发日`
- Estimate rationale: 聚焦 OpenAI WebSocket relay 的 client disconnect、drain、close 和既有会话测试，边界清晰但需要验证双向连接释放。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

OpenAI WebSocket 调用方断开后，立即关闭模型侧 relay/连接，不再继续 drain 或重试，并写入一条 `client_disconnected` Ops 错误记录。

## 背景

`backend/internal/service/openai_ws_forwarder.go` 当前在检测到客户端断开后会继续 drain upstream，以便获取后续事件和 usage。OpenAI WebSocket 还有独立的 relay、会话和 retry 逻辑，需要单独接入统一断开语义。

## 范围内

- 接入 MVP-001 的显式取消桥接；
- 识别 WebSocket 读、写、close 和 relay 错误中的调用方断开；
- 取消模型侧 HTTP/WebSocket 请求；
- 关闭 relay、response body 和模型连接；
- 停止客户端断开后的 drain；
- 调用 MVP-002 写入 Ops 错误；
- 断开后不启动账号重试或路由切换；
- 保留模型供应商错误且客户端仍在线时的现有 retry/failover。

## 范围外

- 不修改 OpenAI WebSocket 会话协议；
- 不修改账号 sticky/session 选择策略；
- 不写入 `usage_logs`；
- 不调整模型供应商错误的重试次数。

## 实现说明

- 重点检查 `openai_ws_forwarder.go` 中 `clientDisconnected`、`emitStreamMessage`、`ReadMessageWithContextTimeout` 和 relay 退出分支；
- 区分调用方 close 与模型供应商 read error；
- 断开路径必须终止双向 relay，不能只停止下游写出；
- 确认 release 和 close 操作幂等；
- 确认 Ops 入库使用独立 Context，不受 WebSocket close 影响。

## 验收标准

- [x] WebSocket 调用方主动关闭后模型侧连接被取消；
- [x] WebSocket 写失败后不再继续 drain upstream；
- [x] WebSocket 断开后不切换账号、不重试；
- [x] 每个断开请求只写一条 `client_disconnected` Ops 错误；
- [x] relay、response body 和模型连接均关闭；
- [x] 模型供应商错误且调用方仍在线时，原有 retry/failover 测试通过；
- [x] 不产生 usage、Token、费用或配额统计。

## 验证计划

- `go test ./internal/service/...`
- 重点运行 `openai_ws_forwarder_test.go`、`openai_ws_forwarder_success_test.go`、`openai_ws_forwarder_ingress_test.go`、`openai_ws_http_bridge_test.go`；
- 使用现有 WebSocket 测试辅助和可控 relay 验证双方连接关闭。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| WebSocket 专项 | `go test ./internal/service -run 'Test(OpenAI.*WS|.*WS.*|OpenAIWS)' -count=1` | 通过；OpenAI WS protocol/pool/relay/retry 回归通过。 |
| Ingress 断开专项 | `go test ./internal/service -run 'TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_ClientDisconnectCancelsUpstream' -count=1` | 通过；客户端写失败立即返回 `ClientDisconnect`，不 drain，上游 lease 标记 broken 并释放，usage 为零。 |
| Handler WebSocket | `go test ./internal/handler -run 'Test(OpenAI|Gateway|WebSocket|WS)' -count=1` | 通过；WS `AfterTurn` 在 usage 前识别断开并只排队一条 Ops marker。 |
| 取消实现 | `openai_ws_forwarder.go` | 普通 OpenAI WS relay 与 Responses ingress relay 在 context/read/write/client-close 路径停止上游读取、标记 broken、释放连接；断开结果不进入 retry/failover。 |
| 完整相关包 | `go test ./internal/service/... ./internal/handler/...` | MVP-004 记录的结果仍适用：相关服务/处理器包通过；`internal/handler/admin` 仅有既有模型路由基线失败。 |

## 执行记录

- 2026-09-16：完成 OpenAI WS relay 与 Responses ingress relay 的 client disconnect 取消；handler WS usage hook 增加断开短路，避免 usage/token/cost/quota 写入。

