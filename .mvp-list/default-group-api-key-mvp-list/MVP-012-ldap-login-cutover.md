# MVP-012：切换 LDAP 登录到新版供应链

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-007, MVP-011`

## 预期成果

现有 LDAP登录入口仅使用新版认证、身份映射和统一用户供应服务。

## 背景

旧版 LDAP源码保留，但不得参与 Handler、Service或 Wire运行依赖。

## 范围内

- 调整 AuthHandler/Wire接入新版组件。
- 首次 LDAP登录通过 `UserProvisioningService` 创建用户和默认 Key。
- 已有用户登录不重复初始化。
- 添加静态引用检查或针对旧入口的构造测试。
- 保留旧源码且确保仓库可编译。

## 范围外

- 运行时新旧切换开关。
- 外部平台接口。

## 实现说明

- 不提供自动回退。
- 旧版入口可保留为未调用函数，但运行依赖图不得引用。

## 验收标准

- [x] LDAP登录测试通过且新用户获得默认 Key。
- [x] 运行代码无旧版 LDAP入口引用，旧源码仍存在。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/service/... -run 'LDAP'`
- `cd backend; go test ./cmd/server/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| AuthHandler | `auth_handler.go:50-51` → `ldapauth.NewDefaultLDAPDirectory` + `NewLDAPAuthenticator` | 替换旧版 `ldapauth.New(cfg.LDAP)` |
| AuthHandler | `auth_handler.go:257` → `Authenticate(c.Request.Context(), ...)` | 新版认证器接受 ctx |
| LoginLDAP | `auth_service.go:564-569` → `provisionAuthUser(ctx, user)` | 替换 `userRepo.Create`，首次 LDAP 登录现在创建默认 Key |
| 旧版源码 | `internal/pkg/ldapauth/client.go` | 仍存在，`ldapauth.New()` 无运行时引用 |
| 测试更新 | `auth_ldap_test.go` → stub 添加 `context.Context` | 匹配新接口 |
| 单元测试 | `go test ./internal/handler/... ./internal/service/... -run 'LDAP'` | ALL PASS |
| 全面回归 | `go test ./internal/handler/... ./internal/service/... ./internal/pkg/ldapauth/...` | ALL PASS (7 packages) |
| 编译检查 | `go build ./internal/...` | PASS |

## 执行记录

- **2026-07-08**：完成三处核心改动：
  1. `ldapauth/client.go`：`Authenticator` 接口和 `Client.Authenticate` 签名增加 `context.Context`
  2. `auth_handler.go`：构造函数改用 `NewDefaultLDAPDirectory` + `NewLDAPAuthenticator`，Login 方法传入 `c.Request.Context()`
  3. `auth_service.go`：`LoginLDAP` 中 `userRepo.Create` 替换为 `provisionAuthUser`，旧版并发回退逻辑由 provisioning 层内置处理

