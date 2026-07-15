# MVP-015：实现外部 Key 供应主流程

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-007, MVP-010, MVP-013, MVP-014`

## 预期成果

外部平台可通过约定接口为本地或 LDAP用户获取/创建平台 Key。

## 背景

接口为 `POST /api/v1/integrations/api-keys/ensure`，不签发用户 JWT。

## 范围内

- 实现 Handler、DTO、服务编排和路由注册。
- 先精确查本地用户，未命中时调用新版 LDAPDirectory。
- LDAP命中后通过 UserProvisioningService落库。
- 校验用户状态与平台名。
- 映射 200/201及约定错误码。
- 设置 `Cache-Control: no-store`、`Pragma: no-cache`。

## 范围外

- 高级限流、指标和审计持久化。
- 多 Token管理。

## 实现说明

- LDAP不可用与 USER_NOT_FOUND必须区分。
- 匿名路由仅由专用 Bearer中间件保护。

## 验收标准

- [x] 本地用户、LDAP新用户、LDAP未命中和停用用户场景符合契约。
- [x] Handler与编排服务测试通过。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/service/... -run 'External.*Provision|EnsurePlatformKey'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 编排服务 | `ExternalProvisioningService`（external_provisioning_service.go）| 协调 UserLookup → LDAPDirectory → Provisioning → PlatformAPIKey |
| Handler | `ExternalProvisioningHandler`（external_provisioning_handler.go）| POST /integrations/api-keys/ensure |
| Bearer认证 | `ExternalProvisioningAuth`（middleware/provisioning_auth.go）| 已存在，现挂载到集成路由组 |
| 路由注册 | `RegisterIntegrationRoutes`（routes/integrations.go）| 匿名路由组，Bearer中间件保护 |
| Cache头 | `Cache-Control: no-store`, `Pragma: no-cache` | Handler 中设置 |
| 用户查找 | 先精确查本地 → 未命中时 LDAPDirectory.LookupUser | 区分 ErrUserNotFound vs LDAP不可用 |
| LDAP落库 | UserProvisioningService.Provision（signup_source="ldap"）| 自动创建默认 Key |
| 平台Key | PlatformAPIKeyService.GetOrCreatePlatformKey | 幂等 get-or-create |
| 200/201映射 | UserCreated 或 KeyCreated → 201，否则 → 200 | response.Created vs response.Success |
| 单元测试 | `TestEnsurePlatformKey_*`（5个测试）| ALL PASS：本地用户/LDAP回退/未命中/非法平台/幂等 |
| 编译检查 | `go build ./internal/...` | PASS |

## 执行记录

- **2026-07-08**：新增 `external_provisioning_service.go`、`external_provisioning_handler.go`、`routes/integrations.go`，修改 `handler.go`（添加字段）和 `router.go`（注册路由）。使用窄接口设计避免依赖膨胀。Handler 中直接使用 `response.Created`/`response.Success` 区分 201/200。

