# MVP-011：新增 LDAP 认证与身份映射组件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-010`

## 预期成果

新版 LDAP密码登录可认证用户并生成统一的本地身份输入。

## 背景

新版运行逻辑需独立于旧 `Authenticator.Authenticate`，同时保持当前用户名、显示名属性语义。

## 范围内

- 新增 `LDAPAuthenticator` 密码认证实现。
- 新增 `LDAPIdentityService` 规范化用户名、显示名和本地账户键。
- 复用新版连接基础设施。
- 测试密码错误、属性缺失、用户名长度和规范化。

## 范围外

- 切换 Login Handler。
- 外部供应接口。

## 实现说明

- 旧版代码仅保留源码，新类型不得引用旧版用户创建流程。

## 验收标准

- [x] 新版认证返回可直接交给 `UserProvisioningService` 的身份。
- [x] 认证与身份映射单元测试通过。

## 验证计划

- `cd backend; go test ./internal/pkg/ldapauth/... ./internal/service/... -run 'LDAPAuthenticator|LDAPIdentity'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 规定测试 | `cd backend; go test ./internal/pkg/ldapauth/... ./internal/service/... -run 'LDAPAuthenticator|LDAPIdentity' -count=1` | 通过；认证与身份映射测试均退出码 0。 |
| LDAP 回归 | `cd backend; go test ./internal/pkg/ldapauth/... -count=1` | 通过，旧版与新版测试均成功。 |
| 认证实现 | `backend/internal/pkg/ldapauth/authenticator.go` | 服务账户查询 DN 后使用独立连接执行目标用户 Bind；密码错误返回 `ErrInvalidCredentials`。 |
| 身份映射 | `backend/internal/service/ldap_identity_service.go` | 输出规范化用户名、邮箱、显示名、`ldap:<username>` 本地账户键与 `ldap` 来源。 |

## 执行记录

实现与验证完成。新版类型不调用旧 `Client.Authenticate` 或旧用户创建流程；用户名最大 100 字符，显示名缺失时回退用户名，超长显示名按 Unicode 字符安全截断。
