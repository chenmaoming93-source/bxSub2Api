# MVP-005：交付仅受外部 Token 保护的 HTTP 接口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `包含 Handler、错误契约、integrations 路由、中间件边界、审计和接口级测试，是一个完整可调用的垂直切片。`
- Dependencies: `MVP-004`

## 预期成果

外部系统可通过最终确认的非冲突 URL 和 integrations Bearer Token 查询四维日、周、月 Token 用量；接口不需要登录、JWT、管理员身份或 RBAC 权限。

## 背景

现有 integrations 路由已复用外部 provisioning Token、功能开关、限流和 hardening。新接口必须加入该边界，不能误挂用户、管理员或 Gateway 鉴权。

## 范围内

- 实现请求绑定、裁剪、长度校验和响应 DTO。
- 映射四类 404、400、503 和 500。
- 注册最终 Method + URL，并断言唯一性。
- 只使用 integrations Token Auth、限流和 hardening。
- 按项目机制登记 RBAC known exclusion，但不执行 RBAC。
- 增加结构化审计，过滤所有凭证。
- 编写 Handler、路由和中间件边界测试。

## 范围外

- 不新增登录或 RBAC 权限配置。
- 不允许 JWT、用户 API Key 或管理员身份替代外部 Token。
- 不改变现有 integrations 接口契约。

## 实现说明

- 优先扩展 `backend/internal/server/routes/integrations.go`。
- 无外部 Token必须返回 401；功能关闭返回 404。
- 有效外部 Token 在无 Cookie、JWT、管理员或 RBAC 上下文时必须成功。
- 响应不得包含 API Key 密钥；日志不得包含 Authorization Header。

## 验收标准

- [x] 最终 Method + URL 在路由表中只出现一次。
- [x] 有效外部 Token 可调用；无效或缺失 Token 返回 401。
- [x] 功能关闭返回 404。
- [x] 无登录、JWT、管理员身份和 RBAC 权限不影响有效外部 Token 请求。
- [x] JWT、Gateway API Key 或登录 Cookie不能替代外部 Token。
- [x] HTTP 错误码和三周期 JSON 契约稳定。
- [x] 审计和响应不泄露凭证。

## 验证计划

- `go test ./internal/handler ./internal/server/routes ./internal/server/middleware -run 'External.*TokenUsage|IntegrationRoutes'`
- 检查测试 Router 的 `Routes()` 和目标中间件行为。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Handler | `backend/internal/handler/external_token_usage_handler.go` | 请求校验、四类领域错误、503、响应 DTO 和无凭证审计已实现 |
| 路由 | `backend/internal/server/routes/integrations.go` | 注册 `POST /api/v1/integrations/token-usage/query`，复用外部 Token/hardening |
| 安全契约测试 | `go test ./internal/handler ./internal/server/routes ./internal/server/middleware -run 'External.*TokenUsage|IntegrationRoutes'` | PASS |
| 唯一性证据 | `TestIntegrationRoutes_TokenUsageExternalTokenOnlyAndUnique` | Routes() 中目标 Method+Path 恰好一次；Cookie/假 JWT 均无法替代外部 Token |

## 执行记录

- 2026-07-31：目标接口仅加入 integrations 组；成功请求无需任何登录或 RBAC 上下文。
- 审计只记录内部 ID、路由别名、来源 IP 和结果，不记录 Authorization 或 API Key 密钥。
- 2026-07-31（契约变更）：请求参数 `api_key_name` 改为 `api_key`（`api_keys.key` 明文值）；响应 `query.api_key` 仅回显脱敏形式（如 `sk-ab****efgh`），完整明文不出现在响应与日志中。
- 2026-07-31（错误契约）：Key 值存在但与请求用户/分组不匹配返回 HTTP 400 `API_KEY_MISMATCH`（区别于不存在/已删除的 404 `API_KEY_NOT_FOUND`）。

