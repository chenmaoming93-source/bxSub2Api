# RBAC 使用指南

## 1. 模型与兼容原则

本项目采用 RBAC0：`用户 -> 角色 -> 权限`。接口和页面声明权限编码，用户通过一个或多个角色取得权限。

- `admin` 是系统超级管理员角色，固定拥有通配权限 `*`，不能删除或移除该权限。
- `user` 是系统普通用户角色，初始化后拥有原有个人中心、API Key、用量、订阅、支付等自助功能权限。
- 自定义角色只能取得明确勾选的权限，不能取得 `*`。
- `users.role` 暂时保留用于旧逻辑兼容；RBAC 表是新鉴权的权威数据源。
- 管理员 API Key 继续作为超级管理员身份使用。

## 2. 初始化与升级

数据库只支持项目约定的 MySQL 8 / GoldenDB 方言。按编号顺序执行：

```sql
SOURCE backend/sqlArchiving/162_create_rbac_schema.sql;
SOURCE backend/sqlArchiving/163_seed_rbac_compatibility.sql;
```

两个文件均可重复执行。第二个文件会：

1. 创建或修正系统 `admin`、`user` 角色；
2. 写入完整权限目录；
3. 给 `admin` 绑定 `*`，给 `user` 绑定全部原有自助权限；
4. 根据现有 `users.role` 回填用户角色；
5. 遇到未知旧角色时主动失败，避免用户被静默漏配。

上线前应在目标 GoldenDB 测试库连续执行两次，并确认第二次不产生错误或重复数据。不要使用 PostgreSQL 的 `SERIAL`、`RETURNING`、`ON CONFLICT`、`::type` 等语法。

## 3. 运行配置与回滚

`backend/config/config.yaml`：

```yaml
rbac:
  mode: "enforce"
  cache_ttl_minutes: 20
  audit_denials: true
```

- `mode: enforce`：权限不足返回 HTTP 403，是正式运行模式。
- `mode: shadow`：仍计算并审计拒绝，但放行请求，用于上线观察或紧急回滚。
- `cache_ttl_minutes`：Redis 权限快照 TTL；默认 20 分钟。
- `audit_denials`：是否输出结构化拒绝审计日志。

配置值会在启动时校验。紧急回滚只需将 `mode` 改为 `shadow` 并重启实例，不能删除 RBAC 表或初始化数据。问题修复并完成差异核对后再恢复 `enforce`。

## 4. 管理界面与接口

超级管理员可在“角色管理”页面：

- 新建、编辑、删除自定义角色；
- 按模块和风险等级查看、分配权限；
- 在用户管理页面为用户分配多个角色。

管理 API：

| 方法与地址 | 用途 |
|---|---|
| `GET/POST /api/v1/admin/rbac/roles` | 查询、新建角色 |
| `PUT/DELETE /api/v1/admin/rbac/roles/:id` | 修改、删除角色 |
| `GET /api/v1/admin/rbac/permissions` | 查询权限目录 |
| `GET/PUT /api/v1/admin/rbac/roles/:id/permissions` | 查询、替换角色权限 |
| `GET/PUT /api/v1/admin/users/:id/roles` | 查询、替换用户角色 |

角色、权限和用户角色变更会在同一数据库事务中写审计记录并推进版本号。系统会阻止删除最后一个管理员、移除管理员的 `*`，以及向自定义角色授予 `*`。

## 5. 后端新增受控接口

权限编码集中定义在 `backend/internal/rbac/permissions.go`，路由在注册位置声明所需权限，不需要在每个 Handler 内重复编写鉴权逻辑：

```go
rbacRoutes.GET(group, "/example", rbac.PermissionExampleRead, handler.List)
rbacRoutes.POST(group, "/example", rbac.PermissionExampleCreate, handler.Create)
```

管理员路由可使用 `adminGET`、`adminPOST`、`adminPUT`、`adminDELETE` 包装器。公共接口必须在路由覆盖规则中明确分类或排除。启动阶段会用 Gin 的实际路由表做闭合检查；受控接口缺少权限声明会直接失败，防止漏鉴权。

新增权限的完整步骤：

1. 在权限目录定义编码、模块、名称、说明和风险等级；
2. 在对应路由旁声明该编码；
3. 确定哪些系统角色应默认取得它；
4. 运行 `cd backend && go run ./internal/rbac/cmd/generatecompatseed` 更新兼容初始化 SQL；
5. 运行权限目录、路由闭合、SQL 一致性及 DDL 重复执行测试；
6. 在前端路由、菜单或按钮元数据中使用同一权限编码。

## 6. 前端使用

### 业务权限管理

“角色与权限”页面允许拥有相应管理权限的人员创建、编辑、停用和删除业务权限编码：

- 编码使用小写点分格式，例如 `report.export`，创建后不可修改；
- 内置系统权限带“内置”标识，不能通过界面修改或删除；
- 停用或删除业务权限会移除已有角色绑定，并推进全局策略版本；
- 创建后可立即在同一页面分配给角色。

界面创建的是可分配的业务能力编码。若要用它保护新的 HTTP 接口或页面，开发人员仍须在对应路由或页面权限矩阵中声明该编码；不能让数据库配置绕过启动时的接口覆盖校验。

`/auth/me` 在保留旧 `role` 字段的同时返回：

- `roles`
- `permissions`
- `permission_version`
- `policy_version`

页面逻辑使用认证 Store 的 `can`、`canAny`、`canAll` 和 `isSuperAdmin`。按钮可使用全局指令：

```vue
<button v-permission="'users.update'">编辑用户</button>
```

路由守卫负责阻止直接访问无权限页面，侧边栏会递归隐藏无权限菜单。前端控制只改善体验，后端中间件始终是安全边界。

## 7. 缓存与多实例

每个活跃用户最多一个主要 Redis Key：

```text
rbac:user:{user_id}:permissions
```

快照包含权限集合、用户授权版本和全局策略版本。每次读取都会以数据库版本为权威：

- 用户角色变更推进用户版本并删除该用户缓存；
- 角色权限变更推进全局策略版本，所有实例发现版本不匹配后自动重载；
- Redis 故障时回退数据库，不会把缓存可用性变成授权可用性的单点故障；
- 单实例内使用 singleflight 合并同一用户的并发回源。

约 1000 名用户时主要权限 Key 规模约为 1000 个，TTL 会回收不活跃用户数据。

## 8. 排障

- HTTP 401：没有有效登录身份，先检查 JWT、管理员 API Key 和身份中间件。
- HTTP 403：身份有效但缺少权限，检查用户角色、角色权限、角色状态和审计日志。
- HTTP 503 `AUTHORIZATION_UNAVAILABLE`：数据库授权版本或授权关系读取失败；检查数据库连接。Redis 单独故障通常只会触发数据库回源。
- 启动时路由闭合失败：新接口没有声明权限或没有被明确归为公共接口。
- 修改后短暂看到旧权限：确认事务已提交、版本号已推进，并检查各实例数据库连接及 Redis 日志。

推荐验证命令：

```bash
cd backend
go test ./internal/rbac ./internal/repository ./internal/server/middleware
go test ./...

pnpm --dir frontend run test:run
pnpm --dir frontend run lint:check
pnpm --dir frontend exec vue-tsc --noEmit
```

完整接口与页面映射以 `RBAC_ENDPOINT_INVENTORY.md`、后端实际 Gin 路由表和前端权限矩阵为准。
