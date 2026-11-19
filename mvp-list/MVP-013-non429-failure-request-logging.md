# MVP-013: 记录非 429 失败请求的脱敏请求报文

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 需要请求体旁路捕获、条件日志、敏感字段脱敏、大小限制和多种网关协议测试，但无需修改业务路由。
- Dependencies: `none`

## Outcome

AI 网关请求最终失败且状态码不是 429 时，系统会在现有失败日志中记录可排查的请求报文摘要，同时保证 429、成功请求和超大请求体不被直接完整写入。

## Context

`backend/internal/handler/ops_error_logger.go` 当前只捕获错误响应 `body`，并把响应内容写入 `ErrorBody`。请求体可能包含提示词和凭据，必须限制大小并脱敏。现有 `ops_error_logs` 虽然包含请求体字段，但本需求明确规定**不得使用这些字段**。另外，普通 warn/error 日志可能被 `OpsSystemLogSink` 索引到 `ops_system_logs`，因此也不能直接复用普通日志级别输出请求体。

## In Scope

- 在 AI 网关路由增加有界请求体旁路捕获，最多保留 64 KiB UTF-8 前缀，不改变 handler 对原始 body 的读取。
- 仅在最终状态码 `>=400 && !=429` 时，将脱敏后的请求报文写入**独立的 file-only 结构化日志**。
- 递归脱敏 `authorization`、`x-api-key`、`api_key`、`token`、`access_token`、`refresh_token`、`password`、`secret` 等字段；不记录请求头全集。
- 日志字段包含 `request_body_bytes`、`request_body_truncated`、`request_id`、endpoint、status、user_id、group_id。
- 捕获或日志失败必须 fail-open，不影响原始错误响应。

## Out of Scope

- 429 请求体记录。
- 成功请求体持久化。
- 上传文件或二进制内容的完整日志；只保留可安全展示的 JSON/text 摘要。
- 为观测看板新增独立请求正文表。

## Implementation Notes

- 使用 `io.TeeReader`/受限 buffer 包装 `c.Request.Body`，避免 `io.ReadAll` 读取最大 256 MiB 请求体。
- 只在网关请求进入 handler 前包装 body；管理员接口不纳入本需求。
- 请求体已经是压缩/二进制或非 JSON 文本时，记录摘要或省略内容并写原因，避免二进制污染日志。
- 脱敏后再输出结构化 JSON 日志，防止换行注入。
- 不得调用 `logger.WriteSinkEvent`，也不得通过会被 `OpsSystemLogSink.shouldIndex()` 捕获的 warn/error 路径输出请求体。
- 实现独立的 file-only sink（复用文件轮转配置），并确保该事件不进入 `ops_system_logs`。
- 建议增加配置开关和日志级别，但默认按需求开启非 429 失败记录。

## Acceptance Criteria

- [x] 400/401/403/404/5xx 等非 429 失败请求会记录请求报文到日志文件；429 不记录。
- [x] 成功请求不写请求体。
- [x] 请求体超过 64 KiB 时只保存前缀，并正确标记 `truncated=true` 和原始字节数。
- [x] 鉴权、密码、token 类字段在日志文件中均被替换为脱敏值。
- [x] `ops_error_logs.request_body` 保持为空，`ops_system_logs` 不新增该请求报文事件。
- [x] HTTP、chat、responses、Gemini 至少各有一个覆盖用例，WebSocket 握手/错误路径有明确策略。
- [x] 捕获失败不改变响应状态码、响应体或重试行为。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/handler
go test -tags=unit ./internal/service
go test -tags=integration ./internal/repository
go build ./cmd/server
```

使用固定 payload 验证 429 排除、64 KiB 截断、嵌套敏感字段脱敏和日志内容与客户端响应无关。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Bounded capture | `backend/internal/handler/gateway_request_failure_logger.go` | PASS: primes at most 64 KiB + 1 byte, then replays the prefix plus the untouched remainder; request bodies rejected before auth reads still have a captured prefix |
| Failure filter | `OpsErrorLoggerMiddleware` + unit tests | PASS: logs status 400, 401, 403, 404, 500, 502, 503 and 529; excludes every 2xx/3xx response and HTTP 429 |
| File-only sink | `backend/internal/pkg/logger/request_failure_logger.go` | PASS: standalone JSON logger writes `request-failures.log` beside the configured main log, reuses rotation settings, and does not register with or emit to the global logger sink |
| Database exclusion | logger sink test + `migration 136` + repository insert paths | PASS: request body event generates zero `ops_system_logs` sink events; legacy `ops_error_logs.request_body` columns were removed and no insert path reintroduces them |
| Truncation | 70 KiB text fixture and truncated secret fixture | PASS: `request_body_truncated=true`, original `Content-Length` reported, captured prefix remains bounded, and a secret beginning before the cutoff is fully redacted |
| Recursive redaction | Nested JSON fixture | PASS: `authorization`, `api_key`, `access_token`, `password` and `client_secret` become `***REDACTED***`; non-sensitive `max_tokens` and message content remain |
| Protocol coverage | `/v1/messages`, `/v1/chat/completions`, `/v1/responses`, `/v1beta/models/*` | PASS for all four inbound protocol paths |
| WebSocket policy | `GET /v1/responses` non-429 failure | PASS: logs request metadata with no invented body and records `request_body_unavailable` |
| Fail-open | forced panic in file writer test hook | PASS: original 502 response status is unchanged |
| Focused tests | `go test -tags=unit ./internal/handler ./internal/pkg/logger` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Changed paths | `gateway_request_failure_logger.go`, `ops_error_logger.go`, logger request-failure sink and tests | Implemented |

## Execution Notes

该功能会把用户提示词写入日志，必须同时确认日志访问权限与保留周期；凭据仍必须脱敏。
