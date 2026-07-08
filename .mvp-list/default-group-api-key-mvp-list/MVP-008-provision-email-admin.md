# MVP-008：接入邮箱注册与管理员创建用户

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-007`

## 预期成果

邮箱注册和管理员创建的所有新用户统一获得默认 API Key。

## 背景

入口分别位于 `AuthService.RegisterWithVerification` 与 `adminServiceImpl.CreateUser`。

## 范围内

- 将两个入口接入 `UserProvisioningService`。
- 保持邀请码、优惠码、默认授权和管理员输入语义。
- 删除运行路径中的重复初始化，但不扩大无关重构。
- 补充两类入口的回归测试。

## 范围外

- OAuth与 LDAP入口。

## 实现说明

- 现有外部响应契约保持不变。
- 默认分组缺失不能阻止创建用户。

## 验收标准

- [x] 两类入口均恰好创建一个默认 Key。
- [x] 既有注册与管理员创建测试继续通过。

## 验证计划

- `cd backend; go test ./internal/service/... -run 'Register|CreateUser|UserProvisioning'`
- `cd backend; go test ./internal/handler/admin/... -run 'CreateUser'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 入口实现 | `backend/internal/service/auth_service.go`、`backend/internal/service/admin_service.go` | 邮箱注册与管理员创建接入统一供应服务，保留原有邀请码、优惠码、订阅和额度后置流程。 |
| 运行时装配 | `backend/internal/service/wire.go` | 为两个入口装配 Ent 事务供应服务和默认 Key 服务。 |
| 回归测试 | `backend/internal/service/user_provisioning_entrypoints_test.go` | 两类入口均验证只创建一个用户和一个默认 Key。 |
| 服务验证 | `cd backend; go test -tags=unit ./internal/service/... -run 'Register|CreateUser|UserProvisioning' -count=1` | 通过。 |
| Handler 验证 | `cd backend; go test ./internal/handler/admin/... -run 'CreateUser' -count=1` | 通过。 |

## 执行记录

已完成：外部响应契约不变；默认分组未配置或缺失时由默认 Key 服务创建 `group_id=NULL` 的 Key，不阻止用户创建。
