# MVP-003：实现 SingGuard 客户端、聊天文本格式化和规则判定

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: 需要覆盖四类协议输入结构、可读文本格式化、外部接口错误和完整规则判定，属于核心高风险逻辑。
- Dependencies: `MVP-001`

## 预期成果

在不接入具体 Gateway Handler 的情况下，能够通过独立服务完成一次 SingGuard Query 检查并返回结构化安全结果。

## 背景

SingGuard 接口由 `SINGGUARD_API_SPEC.md` 定义，使用 `POST {base_url}/classify`，请求体中的 `text` 必须是字符串，内容为可读聊天格式，`task` 固定为 `query`，不传 `threshold`。

## 范围内

- `base_url` 部署配置读取；
- SingGuard HTTP 客户端、独立连接池和超时；
- 解析五个 Query 风险维度；
- Anthropic/OpenAI Chat 的 `messages` 提取；
- OpenAI Responses 的 `input` 提取；
- Gemini 的 `contents` 提取；
- 将角色、文本、工具调用和工具结果转换为可读文本；
- 外部响应校验；
- `RuleAction`、`Decision`、`CheckStatus`、`isUnsafe` 结果生成；
- 超时、网络错误、HTTP 错误、非法 JSON 和维度缺失处理。

## 范围外

- Gateway 主流程接入；
- 采集和数据库写入；
- 图片生成和 embeddings；
- 真实内网 SingGuard 联调。

## 实现说明

- 不复用现有只提取最后一条用户消息的 `ExtractContentModerationInput` 作为完整安全检查输入；
- `text` 应为字符串，例如 `[user]\\n请查询订单`，不能将数组或对象直接作为 JSON 字段传输；
- 图片只生成类型/占位描述，不传 Base64；
- 文本超过 100000 字符时返回输入过长错误，由调用方执行异常决策；
- 本系统使用 `risk_prob > threshold`，不使用 SingGuard 返回体内的 threshold 或 label 作为最终规则依据；
- 规则按顺序判断，告警继续，阻断立即结束；
- `base_url` 不写死 IP。

## 验收标准

- [x] 正常请求发送的 JSON 中 `text` 的类型为字符串，内容为可读角色化聊天文本，`task` 为 `query` 且无 `threshold`。
- [x] 四类协议的完整聊天内容能够转换，工具调用不会导致格式化器崩溃。
- [x] 五个风险维度均能解析，缺失已配置维度时返回 `CheckStatus=error`。
- [x] 分数严格大于阈值才命中；告警继续判断，阻断停止判断。
- [x] 超时和外部异常能够返回可供调用方使用的异常状态及异常决策输入。
- [x] HTTP mock 契约测试和核心规则单元测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... ./internal/handler/...`
- 使用 `httptest` 模拟 `/classify` 的正常响应、超时、503、422、非法 JSON 和缺少维度响应。
- 对四种协议分别提供多轮消息、工具调用和空内容测试样例。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 定向测试 | `cd backend; go test ./internal/service -run 'TestFormatSecurityCheckTextSupportsChatResponsesAndGemini|TestSingGuardClient|TestSecurityCheckService' -count=1 -v` | 通过；协议格式化、请求契约、规则、缺维度和超时测试通过。 |
| 相关包测试 | `cd backend; go test ./internal/config ./internal/service ./internal/handler ./internal/handler/dto` | 通过。 |
| 实现路径 | `backend/internal/service/security_check.go` | 已实现独立 SingGuard client、Query 请求、五维响应、四协议格式化、图片占位、超限和规则判定。 |
| 配置路径 | `backend/internal/config/config.go` | 已增加 `singguard.base_url` 配置入口，未硬编码地址。 |
| 测试路径 | `backend/internal/service/security_check_test.go` | httptest 覆盖字符串 text、query/no threshold、正常/HTTP 错误/非法 JSON、四协议、严格阈值、warn 继续、block 短路、缺维度和超时。 |

## 执行记录

- `go test ./internal/handler/... ./internal/config` 中管理 Handler 的既有 `group_model_routing_test.go` 因 `daily_token_limit` 旧断言失败；该失败与 SingGuard 改动无关。非管理 Handler、service、config 和新增定向测试均通过。
- 由于真实 SingGuard 只在内网可用，本 MVP 使用 `httptest` 契约测试，不进行真实服务联调。
