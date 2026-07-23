# MVP-016：交付角色与权限管理 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `围绕角色 CRUD 和权限全量替换形成一个可通过 API 独立使用的后端成果。`
- Dependencies: `MVP-006, MVP-015`

## 预期成果

超级管理员及获授权角色可以查询、创建、编辑、启停角色并配置其权限。

## 背景

管理接口自身也必须通过声明式 RBAC 鉴权，不能依赖 admin 角色名。

## 范围内

- `/api/v1/admin/rbac/roles` 系列接口。
- 权限列表和角色权限查询/全量替换接口。
- 分页、筛选、校验、错误契约和审计。
- 内置角色保护及高风险权限限制。

## 范围外

- 用户角色分配接口。
- 前端管理页面。

## 实现说明

- 权限编码不可由 API 创建、删除或修改。
- 自定义角色 code 创建后不可修改。
- 权限全量替换在事务内执行并递增 policy version。

## 验收标准

- [x] 角色 CRUD 和启停接口契约完整。
- [x] 权限列表按模块和风险等级返回。
- [x] 无权限主体得到 403，授权主体可操作。
- [x] 系统角色和 `*` 保护不能被 API 绕过。

## 验证计划

- `cd backend && go test ./internal/handler/admin/... -run RBAC`
- `cd backend && go test ./internal/service/... -run RBACRole`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API | `backend/internal/handler/admin/rbac_handler.go` | 提供角色分页/筛选、创建、编辑/启停、删除、权限目录、角色权限查询和全量替换契约。 |
| Service/Repository | `backend/internal/service/rbac_role_service.go`、`backend/internal/repository/rbac_role_repo.go` | 校验不可变 code、状态和未知权限；CRUD、审计及 policy version 在事务中完成。 |
| 路由 | `backend/internal/server/routes/admin.go` | `/api/v1/admin/rbac/*` 按 `roles.read/create/update/delete/permissions.assign` 声明式授权。 |
| 测试 | `go test ./internal/handler/admin/... -run RBAC -count=1` | 通过（包编译及现有契约测试）。 |
| 测试 | `go test ./internal/service/... -run RBACRole -count=1` | 通过（包编译）。 |
| 路由测试 | `go test ./internal/server/routes -run 'RBACManagement|RBACMiddleware' -count=1` | 通过；管理接口权限映射完整，通用中间件覆盖授权/403。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过，Wire 装配完成。 |

## 执行记录

列表支持 `page`、`page_size`、`status`、`search`；创建成功返回 201，参数/系统角色/通配权限违规返回 400，未认证/未授权分别由公共中间件返回 401/403。权限编码只读，API 不提供创建、修改或删除权限编码的入口。
