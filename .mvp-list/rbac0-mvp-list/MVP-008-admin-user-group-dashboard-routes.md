# MVP-008：迁移用户、分组与仪表盘管理路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `三个紧密关联的核心管理模块可形成第一个可观察的细粒度管理角色切片。`
- Dependencies: `MVP-006`

## 预期成果

自定义角色可仅凭明确权限访问用户、分组和管理仪表盘接口，admin 行为不变。

## 背景

路由集中在 `backend/internal/server/routes/admin.go` 的 dashboard、users、groups、user attributes 和模型配额分组。

## 范围内

- Dashboard 查询和聚合操作。
- 用户查询、编辑、余额、配额、订阅、API Key 和属性操作。
- 分组查询、编辑、倍率、RPM、模型候选和 API Key 操作。
- 对高风险动作使用独立权限。

## 范围外

- 账号、代理、系统设置和其他管理模块。
- 角色管理 API。

## 实现说明

- 查询与修改权限必须分离。
- 用户余额、删除、角色相关操作不得并入普通 update。
- 保留 Handler 的业务校验和管理员保护。

## 验收标准

- [x] 范围内所有路由进入 Registry。
- [x] 只读角色可查询但不能修改。
- [x] admin `*` 的响应与迁移前一致。
- [x] 用户和分组高风险操作有独立权限测试。

## 验证计划

- `cd backend && go test ./internal/server/... -run 'Admin.*(User|Group|Dashboard)'`
- `cd backend && go test ./internal/handler/admin/... -run '(User|Group|Dashboard)'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 路由与中间件 | `cd backend && go test ./internal/server/... -run 'RBAC|Admin.*(User|Group|Dashboard)' -count=1` | 通过；57 条 dashboard、users、groups、user-attributes 路由声明进入 Registry。 |
| Handler 回归 | `cd backend && go test ./internal/handler/admin/... -run '(User|Group|Dashboard)' -count=1` | 通过。 |
| 应用构建 | `cd backend && go test ./cmd/server -run '^$'` | 通过；独立管理身份认证、增量迁移兜底和 RBAC 依赖可编译。 |
| 高风险粒度 | `rbac_admin_core_routes_test.go` | 余额调整、用户删除、用户配额修改、分组删除和仪表盘回填均断言使用独立权限。 |

## 执行记录

共迁移 57 条。管理入口新增“只认证、不硬编码 admin 角色”的身份中间件；已登记路由交给 RBAC，尚未迁移的路由仍由 `RequireLegacyAdminForUnregistered` 强制要求旧 admin，避免过渡期越权。查询与修改分离，余额、删除、配额、聚合回填使用独立高风险权限。
