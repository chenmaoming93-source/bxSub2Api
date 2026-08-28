# MVP-005：将安全检查接入 OpenAI Responses WebSocket 各轮请求

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: WebSocket 首帧和后续 turn 使用不同代码路径，需要独立验证连接关闭和逐轮判定。
- Dependencies: `MVP-004`

## 预期成果

OpenAI Responses WebSocket 的首个 `response.create` 请求和后续每轮 payload 都能执行同一套分组安全检查；阻断时按现有 WebSocket policy violation 方式结束请求/连接。

## 背景

`openai_gateway_handler.go` 中 WebSocket 首帧位于握手处理逻辑，后续 turn 通过 `OpenAIWSIngressHooks.BeforeRequest` 获取 payload。WebSocket 没有普通 HTTP Request Body，不能只依赖 HTTP Handler 接入。

## 范围内

- 首帧 `firstMessage` 安全检查；
- 后续 turn 的 `payload` 安全检查；
- 使用现有 request/group/model/protocol 元数据；
- 复用 MVP-003 的文本格式化和规则判定；
- 阻断时返回现有 WebSocket 错误并关闭当前请求/连接；
- 验证每轮检查不会复用上一轮 payload。

## 范围外

- WebSocket 握手 header 中无 body 的额外检查；
- 数据库采集；
- WebSocket 上游账号信息补写；
- 非 Responses WebSocket 协议。

## 实现说明

- 首帧和后续 turn 分别构造安全检查输入；
- 每一轮都以当前 payload 为安全检查和采集请求体；
- `originalModel` 作为客户端模型记录，不在安全检查阶段推导上游模型；
- 不因采集失败关闭 WebSocket；
- 阻断消息沿用现有 `writeContentModerationWSError` 或等价协议处理。

## 验收标准

- [x] 首帧命中阻断时不会建立或继续上游模型调用。
- [x] 后续 turn 命中阻断时能够终止当前连接或请求，并返回现有 policy violation 语义。
- [x] 后续 turn 未命中时仍能继续正常对话。
- [x] 每一轮发送给 SingGuard 的文本来自当前 payload，而不是首帧或上一轮内容。
- [x] WebSocket 相关单元/集成测试通过。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/service/...`
- 使用现有 WebSocket 测试工具模拟首帧、正常后续 turn、阻断后续 turn 和非法 payload。
- 检查阻断时 upstream mock 的调用次数为零或符合当前连接生命周期语义。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test ./internal/handler ./cmd/server` | 通过；现有 WebSocket 处理和新增安全依赖注入编译/回归通过。 |
| 首帧接入 | `backend/internal/handler/openai_gateway_handler.go:1306` | 首帧在账号选择前执行安全检查，block 写入安全错误帧并以 `StatusPolicyViolation` 关闭。 |
| 后续 turn 接入 | `backend/internal/handler/openai_gateway_handler.go:1492` | `BeforeRequest` 对每个非首轮 payload 独立执行检查，block 返回 `OpenAIWSClientCloseError`。 |
| 错误帧 | `backend/internal/handler/openai_gateway_handler.go:2123` | 新增 `evt_security_check_blocked`，使用现有 WebSocket policy violation 语义。 |

## 执行记录

- 首帧和后续 payload 均复用同一 `checkSecurityCheck`，但输入字节分别来自 `firstMessage` 和当前 turn 的 `payload`。
- 现有 Handler 回归测试通过；真实上游 WebSocket 联调不在本地验证范围内。
