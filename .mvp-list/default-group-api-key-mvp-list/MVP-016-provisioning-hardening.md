# MVP-016：加固外部供应接口的限流审计与指标

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-015`

## 预期成果

外部供应接口具备可运维的滥用防护、审计和低泄露可观测性。

## 背景

Plan要求区分认证失败与业务请求限流，并记录不含密钥的调用结果。

## 范围内

- 为来源 IP认证失败和已认证业务调用设置独立限流。
- 增加请求 ID贯穿、审计事件与指标。
- 审计只记录用户 ID、平台、来源 IP、是否创建及结果。
- 限制请求体大小和 Content-Type。
- 增加安全与限流测试。

## 范围外

- 外部 SIEM集成。
- Token管理 UI。

## 实现说明

- 指标平台 label需控制基数，只记录已通过正则的平台。
- 禁止记录完整 API Key、Bearer Token和 LDAP密码。

## 验收标准

- [x] 超限返回 429，审计与指标覆盖成功/失败且不含秘密。
- [x] 安全测试通过。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/middleware/... -run 'Provisioning.*Rate|Provisioning.*Audit|Provisioning.*Security'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 限流器 | `ProvisioningRateLimiter`（provisioning_hardening.go）| 令牌桶算法，auth 10/min + biz 60/min 独立维度 |
| 审计日志 | slog（handler + middleware）| user_id/platform/source_ip/result，不含 Key/Token/密码 |
| Body限制 | `MaxProvisioningBodySize = 4KB` | io.LimitReader 截断超限请求 |
| Content-Type | 中间件强制 `application/json` | 非 JSON 返回 415 |
| 429响应 | `AbortWithError(429, "RATE_LIMITED", ...)` | 超限时返回标准错误格式 |
| 请求ID | 依赖全局 `RequestLogger` 中间件 | 已有 `X-Request-ID` 透传 |
| 限流测试 | `TestProvisioningRateLimiter_*`（4个）| ALL PASS：令牌/隔离/并发/填充 |
| 加固测试 | `TestHardeningMiddleware_*`（3个）| ALL PASS：ContentType/BodySize/RateLimit |
| 审计测试 | `TestAuditLogger` | ALL PASS：成功/失败日志无秘密 |

## 执行记录

- **2026-07-08**：新增 `provisioning_hardening.go`（限流器+审计+加固中间件）和 `provisioning_hardening_test.go`（8个测试）。修改 `integrations.go` 路由注册使用加固中间件，修改 `external_provisioning_handler.go` 添加审计日志。审计仅记录 user_id/platform/source_ip/result，不含 API Key、Bearer Token 或 LDAP 密码。

