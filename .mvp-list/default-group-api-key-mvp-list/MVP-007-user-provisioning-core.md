# MVP-007：建立统一用户供应服务骨架

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-006`

## 预期成果

新增用户可通过单一服务完成用户落库、核心默认配置和默认 API Key创建。

## 背景

现有初始化散落在 `auth_service.go` 与 `admin_service.go`；需抽取但暂不切换所有入口。

## 范围内

- 新增 `UserProvisioningService` 输入、输出和依赖接口。
- 整合随机密码哈希、注册来源、余额、并发、RPM、订阅与额度初始化。
- 把用户与默认 Key核心创建纳入明确事务边界。
- 定义非核心提交后动作及错误记录。
- 增加核心成功、回滚和已有用户幂等测试。

## 范围外

- 迁移所有注册入口。
- LDAP查询。

## 实现说明

- 优先复用现有 helper，避免业务规则重写。
- 若现有 Repository不支持事务上下文，补足最小抽象。

## 验收标准

- [x] 默认 Key失败时不留下半初始化新用户。
- [x] 已有身份并发冲突能够回读用户，服务测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... -run 'UserProvisioning'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 服务骨架 | `backend/internal/service/user_provisioning_service.go` | 定义统一输入/输出和依赖端口；用户与默认 Key 位于同一事务回调，订阅及额度等非核心操作作为提交后动作并收集错误。 |
| 单元测试 | `backend/internal/service/user_provisioning_service_test.go` | 覆盖核心成功、默认 Key 失败回滚、已有用户幂等和创建冲突后回读。 |
| 定向验证 | `cd backend; go test ./internal/service/... -run 'UserProvisioning' -count=1` | 通过。 |

## 执行记录

已完成统一供应服务骨架。入口迁移仍按范围外留给后续 MVP；事务执行器由接入入口时注入，服务单元测试确认默认 Key 失败会回滚用户创建。
