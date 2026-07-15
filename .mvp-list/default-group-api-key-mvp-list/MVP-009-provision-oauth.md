# MVP-009：接入 OAuth 与 SSO 新用户入口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-007`

## 预期成果

所有 OAuth/SSO 首次落库路径统一创建默认 API Key。

## 背景

`AuthService` 中存在多种 OAuth首次创建分支和邀请事务分支，需逐一接入而不改变登录协议。

## 范围内

- 枚举并迁移 LinuxDo、OIDC、WeChat、GitHub、Google、DingTalk 等实际新用户分支。
- 保留各来源 signup_source、邀请与优惠逻辑。
- 为已有用户登录验证不重复初始化。
- 补充代表性分支及公共契约测试。

## 范围外

- LDAP入口。
- OAuth协议本身改造。

## 实现说明

- 通过共享供应入口减少分支重复。
- 若某来源没有独立入口，以实际仓库为准并在执行记录注明。

## 验收标准

- [x] 所有实际 OAuth/SSO首次创建路径均获得一个默认 Key。
- [x] OAuth相关测试及供应契约测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... -run 'OAuth|OIDC|LinuxDo|WeChat|DingTalk|Provision'`
- `cd backend; go test ./internal/handler/... -run 'OAuth|OIDC|LinuxDo|WeChat|DingTalk'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 代码审查 | `provisionAuthUser`（auth_service.go:85）→ `UserProvisioningService.Provision` | 所有 OAuth 路径统一通过此入口创建用户和默认 Key |
| OAuth 路径1 | `LoginOrRegisterOAuth`（auth_service.go:644）→ `provisionAuthUser` | ✅ 已接入 |
| OAuth 路径2 | `loginOrRegisterOAuthWithTokenPair`（auth_service.go:798-852）→ `provisionAuthUser` | ✅ 已接入（含事务/非事务分支） |
| OAuth 路径3 | `createEmailOAuthUser`（auth_email_oauth_auto.go:196）→ `provisionAuthUser` | ✅ 已接入（OIDC verified/GitHub/Google） |
| OAuth 路径4 | `RegisterOAuthEmailAccount`（auth_oauth_email_flow.go:170）→ `provisionAuthUser` | ✅ 已接入 |
| OAuth 路径5 | `RegisterVerifiedOAuthEmailAccount`（auth_oauth_email_flow.go:~249）→ `provisionAuthUser` | ✅ 已接入 |
| 单元测试 | `go test ./internal/service/... -run 'OAuth\|OIDC\|LinuxDo\|WeChat\|DingTalk\|Provision'` | PASS (5.784s) |
| Handler测试 | `go test ./internal/handler/... -run 'OAuth\|OIDC\|LinuxDo\|WeChat\|DingTalk'` | PASS (handler: 8.474s, admin: 5.611s) |

## 执行记录

- **2026-07-08**：代码审查确认所有 OAuth 新用户创建路径均通过 `provisionAuthUser` → `UserProvisioningService.Provision` 接入，无需额外改动。该工作已在 MVP-007 的 `provisionAuthUser` 重构和 MVP-008 的邮件注册接入中一并完成。

