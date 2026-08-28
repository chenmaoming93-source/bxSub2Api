# MVP-004：将安全检查接入普通 HTTP 模型调用主流程

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: 复用现有 Handler 已读取的 body 和内容审计接入点，覆盖主 HTTP 协议但不包含 WebSocket。
- Dependencies: `MVP-002, MVP-003`

## 预期成果

分组开启安全检查后，普通 HTTP 模型请求会在上游账号选择/模型调用前完成同步判定；阻断请求不会调用上游，告警请求继续原流程。

## 背景

当前多个 Handler 已通过 `ReadRequestBodyWithPrealloc` 读取请求体，并在模型调用前调用 `checkContentModeration`。安全检查不能通过再次读取 `c.Request.Body` 的 Middleware 实现，应直接复用现有 `body []byte`。

## 范围内

- 接入 Anthropic Messages；
- 接入 OpenAI Chat Completions；
- 接入 OpenAI Responses；
- 接入 Gemini `generateContent`；
- 获取 request ID、client request ID、用户、API Key、分组、provider、协议、模型和入站 endpoint；
- 将安全检查放在上游账号选择和转发前；
- 阻断、告警、放行响应；
- 配置关闭、规则为空、缓存异常时保持主流程可用；
- 记录安全判定耗时和结构化日志。

## 范围外

- WebSocket；
- 图片生成、图片编辑、embeddings；
- 异步数据库采集；
- 上游账号 ID、上游请求 ID和最终映射模型记录。

## 实现说明

- 安全检查应在现有基本请求解析之后执行；
- 采集使用原始逻辑请求体，OpenAI 紧凑路径若在检查前规范化 body，应在规范化前保留 `originalBody`；
- 阻断响应应沿用当前协议对应的错误响应方式；
- 外部异常和超时使用分组 `exception_action`；
- 只允许 `decision=block` 终止模型主流程；`decision=warn` 仅告警。

## 验收标准

- [x] 四类普通 HTTP 协议在配置关闭或规则为空时不调用 SingGuard。
- [x] 命中阻断规则时，在上游调用前返回协议兼容的阻断错误。
- [x] 只命中告警规则时打印 warn 且模型请求继续。
- [x] SingGuard 超时、异常和返回字段错误按分组异常决策处理。
- [x] 安全检查位置使用已有 `body []byte`，不会重复读取空的 `c.Request.Body`。
- [x] HTTP 主流程回归测试和安全检查接入测试通过。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/service/...`
- 使用现有 HTTP Handler 测试模式模拟 allow、warn、block、timeout 和 exception 场景。
- 检查阻断场景中上游账号选择/转发 mock 未被调用。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 回归测试 | `cd backend; go test ./internal/handler ./internal/service ./internal/config ./cmd/server` | 通过。 |
| 接入测试 | `cd backend; go test ./internal/handler -run TestRunSecurityCheckLoadsGroupConfigAndBlocksBeforeUpstream -count=1 -v` | 通过；按分组配置执行 SingGuard 并返回 block。 |
| Handler 接入 | `backend/internal/handler/gateway_handler.go`, `gateway_handler_chat_completions.go`, `gateway_handler_responses.go`, `openai_chat_completions.go`, `openai_gateway_handler.go`, `gemini_v1beta_handler.go` | 已在上游选择/转发前使用已有 `body []byte` 执行检查，block 返回协议错误，warn 不终止。 |
| Wire 接入 | `backend/internal/service/wire.go`, `backend/internal/repository/wire.go`, `backend/internal/handler/wire.go`, `backend/cmd/server/wire_gen.go` | 已注入 provider、client 和 checker；空 base URL 不阻塞启动。 |

## 执行记录

- 图片生成/编辑路径 `openai_images.go` 保持排除，符合第一期范围。
- `go test ./internal/handler/... ./internal/service/...` 的管理 Handler 全量命令仍受既有 `group_model_routing_test.go` 失败影响；本次相关 Handler、service、config 和 cmd/server 目标包均已单独通过。
