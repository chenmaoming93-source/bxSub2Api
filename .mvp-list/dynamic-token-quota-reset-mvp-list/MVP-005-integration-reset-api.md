# MVP-005：提供 integrations 外部限额重置 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: `重置业务已由 MVP-003 完成，本切片聚焦外部 Handler、既有鉴权与 hardening 接入、日志脱敏及路由契约。`
- Dependencies: `MVP-003`

## 预期成果

公司内部外部系统能够通过现有 integrations Bearer Token 调用限额重置接口，并获得与管理员 API 相同的业务语义，无需用户 JWT、管理员会话、幂等键或新增复杂权限模型。

## 背景

- integrations 路由位于 `backend/internal/server/routes/integrations.go`，统一使用 `provAuth` 与 `provHardening`。
- 外部 Token 用量 Handler 模式可参考 `backend/internal/handler/external_token_usage_handler.go`。
- 现有接口在 provisioning 配置关闭时遵循统一隐藏/禁用行为。
- API Key 等敏感值不得在响应和日志中明文回显。

## 范围内

- 新增 `POST /api/v1/integrations/token-usage/reset`。
- 复用管理员接口相同的 tokenstat 请求和结果语义。
- 接入现有 integrations Bearer Token 和 hardening middleware。
- 对无认证、错误认证和 provisioning 关闭场景保持现有契约。
- 记录来源 IP、周期、指标和汇总结果，并对敏感维度脱敏。
- 更新 Handler/wiring/routes 及其单元和契约测试。

## 范围外

- 不接受 JWT、管理员 Cookie 或其他替代认证。
- 不增加 request_id 或幂等存储。
- 不实现外部系统内部的充值、审批或权限规则。
- 不实现前端页面。

## 实现说明

- 可新增专用 External Token Quota Reset Handler，也可在职责清晰的前提下扩展现有 External Token Usage Handler；接口边界必须便于 Stub 测试。
- 外部 DTO 不允许 projection、hash、shard、Redis Key 或 baseline。
- 重复调用按最新 raw 再次建立快照，这是已确认行为。
- 日志仅记录必要的维度摘要；若维度可能包含 API Key 明文，必须沿用现有遮罩策略或不记录该值。

## 验收标准

- [x] 新路由准确注册在 `/api/v1/integrations/token-usage/reset` 且只出现一次。
- [x] 无 Bearer Token 或错误 Token 无法调用；正确 Token 可以调用。
- [x] 路由复用 `provHardening`，配置关闭行为与现有 integrations 接口一致。
- [x] 四种业务状态与管理员 API 使用相同字段和语义。
- [x] 相同请求重复调用会重复重置，不要求幂等键。
- [x] 响应和日志不泄露 API Key 等敏感明文。
- [x] 现有 integrations 查询及 provisioning 路由回归通过。

## 验证计划

- `cd backend && go test ./internal/handler ./internal/server/routes`
- `cd backend && go test ./internal/service/tokenstat ./internal/repository/tokenstat`
- 扩展 `backend/internal/server/routes/integrations_test.go`，覆盖启用、关闭、无 Token、错误 Token 和正确 Token。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Handler | `backend/internal/handler/external_token_usage_handler.go` | 复用共享请求/结果语义，strict DTO，400/503 映射及仅汇总字段日志。 |
| Route | `backend/internal/server/routes/integrations.go` | 唯一注册 `/token-usage/reset`，沿用 group 的 Bearer auth 与 hardening。 |
| 契约测试 | `external_token_usage_handler_test.go`、`integrations_test.go` | 覆盖四状态字段、重复调用、未知字段、启用/关闭、无/错/正确 Token、唯一性及 nil provisioning。 |
| 命令 | `go test ./internal/handler ./internal/server/routes` | 全部通过。 |
| Wiring/服务 | `go generate ./cmd/server`；`go test ./cmd/server ./internal/service/tokenstat ./internal/repository/tokenstat` | 全部通过。 |

## 执行记录

2026-09-09 完成。调整 `RegisterIntegrationRoutes` 的旧 nil 早退：仅在全部 Handler 均为空时返回，各独立 integrations Handler 可单独注册；所有路由仍统一经过 `provAuth` 与 `provHardening`。重置响应和日志不回显 dimension values，因此不会泄露 API Key 明文。
