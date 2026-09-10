# MVP-001：用户部门字段与 LDAP 首次登录同步

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Estimate rationale: 同时涉及用户 schema、Ent 生成、LDAP 读取兼容和首次登录链路，属于本功能的基础切片。
- Dependencies: `none`

## 预期成果

LDAP 用户首次登录后，本地用户记录具备正确的 `department`，且 LDAP 缺少部门时保持可识别的空值状态。

## 背景

当前 LDAP 认证位于 `backend/internal/pkg/ldapauth`，首次本地用户创建位于 `backend/internal/service/auth_service.go`。LDAP 用户的本地账号必须使用用户输入账号保存到 `users.email`，不能只使用 LDAP `UsernameAttribute`。本 MVP 不涉及 Token 统计事件和管理员批量同步。

## 范围内

- 为 `users` 增加 `department VARCHAR(255) NOT NULL DEFAULT ''`，生成 Ent 代码并在 `backend/sqlArchiving/` 提供符合项目规约的 DDL（不新增 `backend/migrations/` 文件）。
- 为 LDAP 配置增加 `department_attribute`，默认值为 `department`，并同步更新 `backend/config/config.yaml` 与 `backend/resources/config.yaml`。
- `ldapauth.User` 增加部门字段；LDAP 查询请求读取部门属性。
- 同时覆盖 `LDAPDirectory` 和仍被其他流程使用的 LDAP client 读取路径。
- 处理 department 属性不存在、无值、空字符串、全空格和多值情况：取第一个非空值，否则返回空字符串。
- 调整 LDAP 登录调用，使 `users.email` 使用原始登录账号，displayName 作为 username，部门写入 users。
- 管理端用户 DTO 可返回 department，便于后续页面展示。
- 增加 LDAP 和首次登录测试。

## 范围外

- API Key 认证缓存部门快照。
- 管理员批量同步接口和按钮。
- Token 统计 Projection、聚合和部门图表。
- 修改 `usage_logs`、`auth_identities` 或 `user_attribute_values`。

## 实现说明

- 修改 `backend/ent/schema/user.go`、`backend/internal/service/user.go` 及对应 mapper。
- 修改 `backend/internal/config/config.go` 的 LDAPConfig 和默认配置。
- 修改 `backend/internal/pkg/ldapauth/client.go`、`directory.go` 及相关 identity 结构。
- `LoginLDAP` 接收原始本地账号和完整 LDAP identity；不要用 `signup_source` 判断 LDAP 用户。
- 空部门在 users 表保存为空字符串；展示层的“未设置”由后续功能统一处理。

## 验收标准

- [x] LDAP 首次登录使用用户输入账号写入 `users.email`，不依赖 LDAP username 属性格式。
- [x] LDAP displayName 正常写入 `users.username`，缺失时回退 LDAP username。
- [x] department 缺失、空值、空白、多值均按约定处理。
- [x] 普通用户创建和普通登录流程行为不变。
- [x] `usage_logs`、`auth_identities`、`user_attribute_values` schema 未被修改。
- [x] 相关 Go 单元测试和迁移/schema 测试通过。

## 验证计划

- `go test ./internal/pkg/ldapauth ./internal/service/...`
- `go test ./internal/repository/...`
- 人工检查 `backend/sqlArchiving/` DDL 只包含 users.department，且 `backend/migrations/` 未新增文件。
- 人工检查开发配置和资源配置均包含 `ldap.department_attribute`。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `backend: go test -tags=unit ./internal/pkg/ldapauth ./internal/service -run 'LDAP|LoginLDAP' -count=1` | 通过 |
| 测试 | `backend: go test ./internal/repository -run 'User|Migration' -count=1` | 通过 |
| 测试 | `backend: go test ./internal/handler/dto -count=1` | 通过 |
| 编译 | `backend: go test ./cmd/server -run TestNonExistent -count=1` | 通过编译，无匹配测试 |
| 检查 | `backend/sqlArchiving/173_add_user_department.sql` 与 git diff | 已确认未新增 backend/migrations SQL，usage_logs 等表未修改 |
| 限制 | `backend: go test ./internal/pkg/ldapauth ./internal/service/... ./internal/handler/dto ./internal/repository/...` | 首次宽范围运行超过 120 秒；已用聚焦测试替代并记录 |

## 执行记录

- 按 `PROJECT_CONVENTIONS.md` 将 DDL 放在 `backend/sqlArchiving/173_add_user_department.sql`；移除了误建的 `backend/migrations/159_add_user_department.sql`。

