# MVP-003：管理员 LDAP 全量同步后端

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Estimate rationale: 包含全量用户遍历、LDAP 有限并发、部分失败汇总、事务边界和缓存失效，复杂度高于单一 CRUD。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

管理员可通过后端接口同步所有 LDAP 用户的当前资料，同时将非 LDAP 用户部门统一清空。

## 背景

LDAP 用户分类规则固定为：`users.email` 不在 `local_login_accounts` 中即为 LDAP 用户。LDAP 查询只能使用服务账号，调用已有 `LookupUser`，不得调用需要用户密码的认证流程。

## 范围内

- 新增 `LDAPUserSyncService` 或等价服务。
- 新增 `POST /api/v1/admin/users/sync-ldap`。
- 使用 `users.email` 查询未命中本地账号白名单的用户。
- 同步 `username` 和 `department`。
- 本地账号白名单用户清空 department。
- 处理 LDAP 不存在、歧义、服务不可用和数据库错误。
- 有限并发、单用户失败继续、同步结果汇总。
- 仅更新非软删除用户；不删除或禁用 LDAP 用户。
- department 变化后调用 `InvalidateAuthCacheByUserID`。
- 接入 `users.update` 权限和操作日志。

## 范围外

- `/admin/users` 前端按钮。
- 异步任务和进度表。
- 任意 LDAP 属性映射。
- 修改 `usage_logs`、`auth_identities`。

## 实现说明

- 服务端读取配置中的 BindDN/BindPassword，绝不接收前端 LDAP 密码。
- 建议每个用户更新独立提交；LDAP 网络调用不包在全量全局事务内。
- LDAP 查询失败保留原部门；查询成功但部门为空时清空。
- 响应返回 total、LDAP candidates、synced、local cleared、not found、failed、duration 等字段。
- 重复执行必须幂等。

## 验收标准

- [x] 只有 `users.email` 不在本地账号白名单中的用户会查询 LDAP。
- [x] 本地账号用户 department 被清空。
- [x] LDAP 用户成功同步 displayName 和 department。
- [x] 单用户 LDAP 失败不会阻断其他用户。
- [x] 接口不需要用户 LDAP 密码且不泄露服务密码。
- [x] department 变化会失效 API Key 认证缓存。
- [x] 接口权限、错误码和汇总响应符合契约。
- [x] 后端同步和 handler 测试通过。

## 验证计划

- `go test ./internal/pkg/ldapauth ./internal/service/... ./internal/handler/admin/...`
- 针对 handler 使用 mock LDAP directory 验证分类、部分失败和权限。
- 人工检查路由注册和服务注入。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `backend: go test ./internal/service -run 'LDAPUserSync|DynamicToken|AuthCache|APIKeyService' -count=1` | 通过 |
| 测试 | `backend: go test ./internal/handler/admin -run 'SyncLDAP' -count=1` | 通过 |
| 测试 | `backend: go test ./internal/server/routes -run 'User|Route' -count=1` | 通过 |
| 编译 | `backend: go test ./cmd/server -run TestNonExistent -count=1` | 通过编译，无匹配测试 |
| 回归限制 | `backend: go test ./internal/service ./internal/handler/admin ./internal/server/routes ./cmd/server -count=1` | service/routes/cmd 通过；admin 存在既有 group_model_routing 测试失败，另有维度数量断言已按新部门维度修正 |
| 检查 | `backend/internal/service/ldap_user_sync_service.go`, `backend/internal/handler/admin/user_handler.go`, `backend/internal/server/routes/admin.go` | 服务账号 LookupUser、有限并发、分类/汇总、缓存失效和 `users.update` 路由已接入；不接收 LDAP 用户密码 |

## 执行记录

- 同步服务使用独立 `LDAPUserSyncService`，每个用户独立更新，LDAP 单用户失败继续执行。
- 路由为 `POST /api/v1/admin/users/sync-ldap`，权限为 `users.update`。

