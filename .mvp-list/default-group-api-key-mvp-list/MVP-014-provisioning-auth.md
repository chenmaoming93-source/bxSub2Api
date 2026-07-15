# MVP-014：实现外部 Bearer 认证中间件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-005`

## 预期成果

外部供应路由可使用恒定时间 Bearer Token校验，并在功能关闭时不可探测。

## 背景

项目已有 Authorization Header惯例；本中间件只保护新 integrations路由。

## 范围内

- 严格解析 `Authorization: Bearer <token>`。
- 使用恒定时间比较。
- 功能关闭或 Token未配置时返回 404。
- 缺失/错误 Token返回 401。
- 确保日志不包含 Token值、前缀或长度。
- 添加中间件测试。

## 范围外

- 业务 Handler。
- 限流和审计。

## 实现说明

- 不接受 Query、Body或自定义 Header中的凭证。

## 验收标准

- [x] 正确 Token放行，其他传递方式和错误 Token均拒绝。
- [x] 认证测试证明响应与日志不泄露 Token。

## 验证计划

- `cd backend; go test ./internal/server/middleware/... ./internal/handler/... -run 'ProvisioningAuth|Bearer'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 规定测试 | `cd backend; go test ./internal/server/middleware/... ./internal/handler/... -run 'ProvisioningAuth|Bearer' -count=1` | 通过；正确 Token 放行，关闭/缺失/错误及非 Header 传递方式均按契约拒绝。 |
| 泄露检查 | `rg -n "log\\.|slog\\.|Printf|Println" internal/server/middleware/provisioning_auth.go` | 无匹配；响应测试同时确认不包含 Token 或其前缀。 |
| 额外回归 | `cd backend; go test ./internal/server/middleware/... -count=1` | 未通过：既有 `TestApiKeyAuthWithSubscriptionGoogle_InsufficientBalance` 期望 403、实际 200；与本中间件无调用关系，规定定向测试已通过。 |
| 实现路径 | `backend/internal/server/middleware/provisioning_auth.go` | 使用 `subtle.ConstantTimeCompare`，功能关闭或未配置返回 404，认证失败返回 401。 |

## 执行记录

实现与规定验证完成。计划原验证路径 `internal/middleware` 在仓库中不存在，已按真实目录修正为 `internal/server/middleware`。中间件不读取 Query、Body 或自定义 Header，也不记录凭证信息。
