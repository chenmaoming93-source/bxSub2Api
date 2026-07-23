# MVP-017：交付用户角色分配、默认绑定与权限响应

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `覆盖用户角色生命周期的创建、变更和登录读取闭环。`
- Dependencies: `MVP-003, MVP-006, MVP-015`

## 预期成果

现有和新用户都具有可靠角色绑定；管理端可分配多角色；`/auth/me` 向前端返回有效角色和权限。

## 背景

所有注册、OAuth、LDAP、外部开户和管理员创建路径都必须在事务内绑定 user 角色。

## 范围内

- `GET/PUT /api/v1/admin/users/:id/roles`。
- 用户角色全量替换、版本递增和缓存失效。
- 新用户默认 user 角色绑定。
- `users.role` 兼容同步。
- `/api/v1/auth/me` 扩展 roles、permissions、permission version。

## 范围外

- 前端用户角色弹窗。
- 删除 `users.role`。

## 实现说明

- 创建用户、默认角色和版本记录必须同事务。
- 分配 admin 仅允许超级管理员，禁止自我提权。
- 保留旧 `role` 响应字段。

## 验收标准

- [x] 所有用户创建入口自动绑定 user，不产生无角色半成品。
- [x] 用户可绑定多个有效角色并正确递增版本。
- [x] 最后 admin 保护和禁止自我提权生效。
- [x] `/auth/me` 对旧客户端兼容并返回新权限字段。

## 验证计划

- `cd backend && go test ./internal/service/... -run '(Provision|UserRole|Auth)'`
- `cd backend && go test ./internal/handler/... -run 'CurrentUser|UserRole'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 默认绑定 | `backend/internal/repository/user_repo.go` | 所有经统一 UserRepository 创建的注册、OAuth、LDAP、外部开户和管理创建，在同一 Ent 事务中绑定 legacy 对应的 admin/user 系统角色并初始化版本。 |
| 分配 API | `GET/PUT /api/v1/admin/users/:id/roles` | 支持多角色全量替换；版本递增、`users.role` 兼容同步、审计和最后 admin 锁在同一事务，成功后清理 Redis 用户权限 Key。 |
| 权限响应 | `backend/internal/handler/auth_handler.go` | `/auth/me` 保留原 `role` 与资料字段，并新增 `roles`、`permissions`、`permission_version`、`policy_version`。 |
| 测试 | `go test -tags unit ./internal/handler -run CurrentUser -count=1` | 通过；兼容字段及新权限字段同时存在。 |
| 测试 | `go test ./internal/service/... -run '(Provision|UserRole|Auth)' -count=1` | 通过。 |
| 测试 | `go test ./internal/repository -run 'UserRepositoryCreate|RBACLastAdmin' -count=1` | 通过；默认绑定与最后管理员保护未破坏创建并发约束。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过。 |

## 执行记录

创建入口统一落到 `UserRepository.Create`，因此不是逐个入口复制绑定逻辑。legacy `users.role=admin` 同步 admin 系统角色，否则默认 user；多自定义角色场景 legacy 字段保持 `user`。新增字段只扩展 `/auth/me`，旧 `role` 不删除。
