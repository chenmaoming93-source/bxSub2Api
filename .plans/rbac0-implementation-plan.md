# RBAC0 权限系统改造实施 Plan

**状态：Final — user approved**  
**版本：v1.0**  
**日期：2026-07-23**  
**变更摘要：建立覆盖全部站内登录功能的 RBAC0，采用多角色、业务权限、声明式路由鉴权、SQL 兼容初始化、admin 通配权限、前端权限控制及版本化分布式缓存。**

## 1. 背景

当前项目只有 `admin` / `user` 两类硬编码角色：

- `admin` 通过 `AdminAuth` 访问全部管理接口。
- `user` 通过 JWT 访问个人接口和页面。
- 前端主要通过 `isAdmin` 和 `requiresAdmin` 控制页面、菜单和按钮。
- 后端不存在动态角色、多角色、统一权限目录和角色权限管理。

根据项目根目录 `RBAC_ENDPOINT_INVENTORY.md`，当前共有 522 条后端接口：

| 分类 | 数量 | RBAC 处理 |
|---|---:|---|
| 管理员接口 | 362 | 纳入 |
| 用户个人接口 | 65 | 纳入 |
| 公开及登录流程接口 | 54 | 明确排除 |
| 支付回调接口 | 6 | 明确排除 |
| 外部集成接口 | 3 | 明确排除 |
| 模型网关接口 | 32 | 明确排除 |
| 合计 | 522 | 427 条纳入，95 条排除 |

## 2. 目标与成功标准

### 2.1 目标

- `G-01`：建立标准 RBAC0：用户、角色、权限。
- `G-02`：一个用户可以绑定多个角色，权限按并集合并。
- `G-03`：内置 `admin` 定位为超级管理员，拥有不可移除的通配权限 `*`。
- `G-04`：内置 `user` 拥有当前普通用户的全部个人权限。
- `G-05`：升级后现有 admin/user 可访问的接口和页面与升级前完全一致。
- `G-06`：初始化、权限目录、内置角色和历史用户回填由独立可执行 SQL 完成。
- `G-07`：业务 Handler 不重复编写权限判断，鉴权集中在公共中间件。
- `G-08`：每个受控接口只声明权限编码，新增接口不能漏登记。
- `G-09`：页面、菜单和按钮使用统一权限判断，但后端始终是最终安全边界。
- `G-10`：RBAC 控制“能做什么”，现有数据归属逻辑继续控制“能操作谁的数据”。
- `G-11`：分布式部署下权限变更及时生效，不依赖单机缓存失效通知。
- `G-12`：角色和权限变更可审计、可灰度、可回滚。

### 2.2 成功标准

- 原管理员仍能访问全部原管理和个人功能。
- 原普通用户仍能访问全部原个人功能，且只能操作自己的数据。
- 自定义角色只能访问明确授予的功能。
- 427 条受控接口全部声明权限。
- 95 条排除接口全部声明排除原因。
- Gin 实际路由不存在“未分类”状态。
- 权限目录、初始化 SQL、后端声明和前端页面映射可以自动校验。
- RBAC 通用鉴权代码不散落到业务 Handler。

## 3. 范围

### 3.1 纳入

- RBAC 表结构、Repository、Service、缓存和审计。
- 内置权限、admin/user 角色和历史用户初始化。
- 管理员接口、用户个人接口及站内登录页面。
- 声明式路由注册和统一鉴权中间件。
- 角色管理、角色权限配置、用户多角色分配。
- 前端路由、菜单、页面、按钮权限控制。
- 完整性检查、影子鉴权、灰度和回滚。

### 3.2 排除

以下继续使用原有认证方式，不纳入站内用户 RBAC：

- 注册、登录、验证码、密码重置、OAuth 回调。
- 支付服务商 webhook。
- 模型网关 API Key 接口。
- 外部集成密钥接口。
- 健康检查和公开设置接口。
- 无法携带 JWT 的受控静态资源 URL。

### 3.3 非目标

- RBAC1 角色继承。
- RBAC2 互斥角色、职责分离和角色人数约束。
- ABAC 属性表达式和组织树授权。
- 显式拒绝权限。
- 按每条 URL 创建一个可编辑权限。
- 删除现有 `users.role` 字段。

## 4. 确定决策

| 编号 | 决策 |
|---|---|
| `D-01` | 使用 RBAC0：`用户 ↔ 角色 ↔ 权限` |
| `D-02` | 用户与角色多对多，角色与权限多对多 |
| `D-03` | 权限采用稳定业务编码，例如 `users.read` |
| `D-04` | HTTP 接口是权限执行点，页面是展示点，不作为 RBAC 数据层级 |
| `D-05` | 一个权限可保护多个接口、页面和按钮 |
| `D-06` | admin 角色拥有 `*`，自定义角色禁止获得 `*` |
| `D-07` | user 角色初始化为现有全部个人权限 |
| `D-08` | 多角色权限取并集，不支持显式 deny |
| `D-09` | 已停用角色不参与权限计算 |
| `D-10` | 系统权限由代码和 SQL 定义，管理界面不能创建、删除或改编码 |
| `D-11` | 角色可以动态创建、修改、停用和分配 |
| `D-12` | `users.role` 第一阶段保留为兼容字段，最终授权以 `rbac_user_roles` 为准 |
| `D-13` | 管理员 API Key 映射为超级管理员主体，但仍需认证、审计和合规控制 |
| `D-14` | 第一版只使用共享 Redis，不引入进程内 L1 权限缓存 |

## 5. 功能设计

### 5.1 权限计算

```text
用户
  ├─ 运营角色：users.read、usage.admin.read
  └─ 财务角色：billing.orders.read、users.balance.adjust

最终权限 = 两个角色的有效权限并集
```

任一有效角色拥有 `*` 时，主体拥有所有已登记权限。

### 5.2 权限与接口

```text
users.read
  ├─ GET /api/v1/admin/users
  ├─ GET /api/v1/admin/users/:id
  └─ GET /api/v1/admin/users/:id/usage
```

接口映射写在对应 Gin 路由旁，不保存为运行时可编辑数据库配置。

### 5.3 权限与页面

```text
/admin/users                   → users.read
“编辑用户”按钮                 → users.update
“删除用户”按钮                 → users.delete
“调整余额”按钮                 → users.balance.adjust
```

前端隐藏不能替代后端鉴权。

### 5.4 权限与数据归属

普通用户读取 API Key：

```text
RBAC：api_keys.self.read
归属：api_key.user_id == current_user.id
```

管理员读取指定用户 API Key：

```text
RBAC：users.api_keys.read
范围：允许指定目标用户
```

现有归属校验不得删除。

## 6. 权限目录

### 6.1 命名

```text
<resource>.<scope>.<action>
<resource>.<action>
```

示例：

```text
profile.self.read
profile.self.update
api_keys.self.read
api_keys.self.create
api_keys.self.update
api_keys.self.delete
users.read
users.create
users.update
users.delete
users.balance.adjust
users.roles.assign
roles.read
roles.create
roles.update
roles.delete
roles.permissions.assign
accounts.credentials.read
settings.update
```

### 6.2 粒度

- 列表和详情通常共用 `read`。
- 创建、修改、删除拆分。
- 余额、凭据、角色分配、系统更新等高风险动作单独授权。
- 页面复用其核心查询权限。
- 不为每条 URL 创建权限。
- 不把 HTTP 方法、URL 或 Vue 组件名写入权限编码。

### 6.3 代码目录

建议新增：

```text
backend/internal/rbac/
  permissions.go
  registry.go
  principal.go
  evaluator.go
  cache.go
  service.go
  errors.go
```

`permissions.go` 集中定义权限编码、名称、模块、描述和风险等级。

## 7. 数据模型

### 7.1 `rbac_roles`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `id` | BIGINT | 否 | 自动生成 | PK |
| `code` | VARCHAR(64) | 否 | 无 | UNIQUE，创建后不可改 |
| `name` | VARCHAR(100) | 否 | 无 | 展示名称 |
| `description` | VARCHAR(500) | 是 | NULL | 描述 |
| `is_system` | BOOLEAN | 否 | FALSE | 内置角色 |
| `status` | VARCHAR(20) | 否 | `active` | `active/disabled` |
| `created_at` | TIMESTAMP | 否 | 当前时间 |  |
| `updated_at` | TIMESTAMP | 否 | 当前时间 |  |
| `deleted_at` | TIMESTAMP | 是 | NULL | 软删除索引 |

### 7.2 `rbac_permissions`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `id` | BIGINT | 否 | 自动生成 | PK |
| `code` | VARCHAR(128) | 否 | 无 | UNIQUE |
| `name` | VARCHAR(100) | 否 | 无 | 名称 |
| `module` | VARCHAR(64) | 否 | 无 | INDEX |
| `description` | VARCHAR(500) | 是 | NULL | 描述 |
| `risk_level` | VARCHAR(20) | 否 | `low` | `low/medium/high/critical` |
| `is_system` | BOOLEAN | 否 | TRUE | 系统权限 |
| `status` | VARCHAR(20) | 否 | `active` | 状态 |
| `created_at` | TIMESTAMP | 否 | 当前时间 |  |
| `updated_at` | TIMESTAMP | 否 | 当前时间 |  |

特殊权限 `*` 的模块为 `system`、风险为 `critical`。

### 7.3 `rbac_user_roles`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `id` | BIGINT | 否 | 自动生成 | PK |
| `user_id` | BIGINT | 否 | 无 | FK、INDEX |
| `role_id` | BIGINT | 否 | 无 | FK、INDEX |
| `assigned_by` | BIGINT | 是 | NULL | 操作人 |
| `created_at` | TIMESTAMP | 否 | 当前时间 |  |

唯一约束：`UNIQUE(user_id, role_id)`。

### 7.4 `rbac_role_permissions`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `id` | BIGINT | 否 | 自动生成 | PK |
| `role_id` | BIGINT | 否 | 无 | FK、INDEX |
| `permission_id` | BIGINT | 否 | 无 | FK、INDEX |
| `created_at` | TIMESTAMP | 否 | 当前时间 |  |

唯一约束：`UNIQUE(role_id, permission_id)`。

### 7.5 `rbac_user_versions`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `user_id` | BIGINT | 否 | 无 | PK、FK |
| `authz_version` | BIGINT | 否 | `1` | 用户角色变更时递增 |
| `updated_at` | TIMESTAMP | 否 | 当前时间 |  |

独立表避免侵入现有 `users` 表的高频业务模型。

### 7.6 `rbac_policy_state`

| 字段 | 类型 | Null | 默认值 | 约束/说明 |
|---|---|---:|---|---|
| `id` | SMALLINT | 否 | `1` | PK，固定单行 |
| `policy_version` | BIGINT | 否 | `1` | 任一角色权限或角色状态变化时递增 |
| `updated_at` | TIMESTAMP | 否 | 当前时间 |  |

### 7.7 `rbac_audit_logs`

记录操作主体、动作、目标、修改前后快照、Request ID、IP 和时间。日志不可由普通角色修改或删除。

## 8. SQL 初始化

### 8.1 项目规约

严格遵循 `PROJECT_CONVENTIONS.md`：

- SQL 必须放在 `backend/sqlArchiving/`。
- 不得在 `backend/migrations/` 新增本次 SQL。
- 不修改已有 migration。
- 文件名使用当前最大编号的下一个编号。
- 实施前同时扫描两个 SQL 目录，不能只依赖文档编号。
- SQL 可脱离应用程序独立执行。
- 不含占位符，所有语句以分号结束。
- 优先提供重复执行保护。

建议拆分：

```text
backend/sqlArchiving/NNN_create_rbac_schema.sql
backend/sqlArchiving/NNN_seed_rbac_compatibility.sql
```

### 8.2 Schema SQL

负责：

- 创建 RBAC 表、索引、外键和唯一约束。
- 初始化 `rbac_policy_state` 单行。
- 不插入业务权限和角色。

### 8.3 Compatibility Seed SQL

负责：

1. 插入 `*` 和完整系统权限目录。
2. 插入内置 `admin`、`user`。
3. 给 `admin` 绑定 `*`。
4. 给 `user` 绑定当前全部个人权限。
5. 将现有 `users.role = 'admin'` 绑定到 admin。
6. 将现有 `users.role = 'user'` 绑定到 user。
7. 为现有用户初始化 `authz_version = 1`。
8. 使用唯一约束和幂等写法避免重复数据。

### 8.4 未知历史角色

上线前执行：

```sql
SELECT role, COUNT(*)
FROM users
GROUP BY role;
```

存在非 `admin/user` 值时阻止切换到强制模式，不静默降级。

### 8.5 新用户

注册、OAuth、LDAP、外部开户及管理员创建用户，统一在同一事务内：

```text
创建用户
绑定内置 user
初始化 authz_version
提交
```

任一步失败全部回滚。

### 8.6 `users.role` 兼容

- 第一阶段保留字段。
- 授权事实以 `rbac_user_roles` 为准。
- 用户角色服务同步维护兼容值：含 admin 则为 `admin`，否则为 `user`。
- 本期不删除字段。

## 9. 后端鉴权

### 9.1 执行链

```text
请求
 → 身份认证
 → Principal
 → RBAC 权限缓存/计算
 → RequirePermission
 → 原业务 Handler
 → 原数据归属和业务校验
```

### 9.2 声明式注册

```go
rbac.GET(users, "", permission.UsersRead, h.Admin.User.List)
rbac.GET(users, "/:id", permission.UsersRead, h.Admin.User.GetByID)
rbac.PUT(users, "/:id", permission.UsersUpdate, h.Admin.User.Update)
rbac.DELETE(users, "/:id", permission.UsersDelete, h.Admin.User.Delete)
```

`rbac.GET/POST/PUT/PATCH/DELETE` 内部统一：

1. 校验权限编码非空。
2. 校验权限存在于代码目录。
3. 登记 Method、完整路由和权限到 Registry。
4. 挂载 `RequirePermission`。
5. 注册原 Handler。

每个接口需要声明权限，但不编写独立鉴权逻辑。

### 9.3 中间件

```text
function RequirePermission(code):
    validate code exists
    principal = context.principal
    if missing: return 401
    if disabled: return 403

    effective = evaluator.getPermissions(principal)
    if effective.isSuperAdmin: continue
    if code not in effective.permissions:
        audit denial
        return 403

    continue
```

### 9.4 Principal

统一认证主体包含：

```text
type
user_id
status
auth_method
```

JWT 用户和管理员 API Key 使用同一授权接口。管理员 API Key 的有效权限为 `*`，但仍保留认证、合规和审计。

### 9.5 `AdminAuth` 拆分

现有 `AdminAuth` 将认证和 `role == admin` 授权耦合，需要拆成：

- 身份认证：确认用户或管理员 API Key。
- RBAC 授权：检查声明权限。

管理路由不再统一要求角色名为 admin。admin 因持有 `*` 保持全部能力，其他角色可访问被授权的部分管理接口。

### 9.6 防遗漏

所有路由必须处于以下两种状态之一：

```text
RBAC_CONTROLLED(permission)
EXCLUDED(reason)
```

启动和 CI 校验：

- 受控路由权限不能为空。
- 权限必须存在于代码目录和数据库。
- 排除路由必须有原因。
- 新增未分类路由直接失败。
- Gin 路由数必须等于受控数加排除数。

当前基线：

```text
522 = 427 + 95
```

## 10. 分布式权限缓存

### 10.1 原则

- 数据库是权限事实来源。
- Redis 是共享权限缓存。
- 不使用单机 L1 权限缓存。
- 不依赖通知每个应用节点删除本地缓存。
- 版本号保证权限变更及时失效，TTL 只负责回收冷用户缓存。

### 10.2 Key 与 Value

每个近期活跃用户一个固定 Key：

```text
rbac:user:{user_id}:permissions
```

Value：

```json
{
  "user_version": 7,
  "policy_version": 12,
  "roles": ["operator", "finance"],
  "permissions": ["users.read", "billing.orders.read"],
  "is_super_admin": false
}
```

全局策略版本：

```text
rbac:policy:version
```

约 1000 名用户时，最坏约 1000 个用户权限 Key 加少量控制 Key，容量压力可以忽略。

### 10.3 两类版本

- `user_version`：用户角色增加、移除时，在同一数据库事务中递增。
- `policy_version`：角色权限、角色状态发生变化时，在同一数据库事务中递增。

缓存命中条件：

```text
cached.user_version == current user authz_version
AND
cached.policy_version == current global policy_version
```

不一致时从数据库重新计算并覆盖同一个固定 Key，不产生旧版本 Key 堆积。

### 10.4 读取

```text
function getEffectivePermissions(userId):
    userVersion = repository.getUserAuthzVersion(userId)
    policyVersion = policyVersionProvider.getCurrent()
    cached = redis.get(fixedUserKey)

    if cached versions match:
        return cached

    roles = repository.getActiveRoles(userId)
    permissions = repository.getActivePermissions(roles)
    result = union(permissions)
    redis.set(fixedUserKey, result, TTL)
    return result
```

### 10.5 写入与失效

修改用户角色：

```text
begin transaction
replace user_roles
increment user authz_version
write audit
commit
delete fixed user permission key（优化，不依赖其正确性）
```

修改角色权限或状态：

```text
begin transaction
replace role_permissions or status
increment global policy_version
write audit
commit
update Redis policy version
```

旧用户缓存即使尚未删除，也因版本不匹配而不能使用。

### 10.6 Redis 故障

- Redis 不可用时回源数据库计算。
- 不使用无法确认版本的新旧缓存放行。
- 对数据库回源增加请求合并和限流保护。
- Redis 恢复后重新写入缓存。
- TTL 建议 10～30 分钟，主要用于清理长期不活跃用户。

## 11. 管理 API

### 11.1 角色

```text
GET    /api/v1/admin/rbac/roles
GET    /api/v1/admin/rbac/roles/:id
POST   /api/v1/admin/rbac/roles
PUT    /api/v1/admin/rbac/roles/:id
DELETE /api/v1/admin/rbac/roles/:id
PUT    /api/v1/admin/rbac/roles/:id/status
```

### 11.2 权限

```text
GET /api/v1/admin/rbac/permissions
GET /api/v1/admin/rbac/roles/:id/permissions
PUT /api/v1/admin/rbac/roles/:id/permissions
```

权限替换在事务中全量执行。

### 11.3 用户角色

```text
GET /api/v1/admin/users/:id/roles
PUT /api/v1/admin/users/:id/roles
```

角色替换在事务中全量执行。

### 11.4 当前用户

扩展 `/api/v1/auth/me`：

```json
{
  "user": {
    "id": 1,
    "role": "admin"
  },
  "roles": ["admin"],
  "permissions": ["*"],
  "permission_version": 7
}
```

保留旧 `role` 字段，避免破坏现有客户端。

## 12. 系统角色保护

### 12.1 admin

- 不能删除、停用或改编码。
- 不能移除 `*`。
- 自定义角色不能获得 `*`。
- 只有超级管理员能分配 admin。
- 禁止自我提权。
- 事务内锁定并检查，禁止移除最后一个有效超级管理员。

### 12.2 user

- 不能删除或改编码。
- 默认赋给所有新普通用户。
- 初始化权限保持现有个人功能完整。
- 修改其权限影响所有普通用户，必须二次确认并审计。

### 12.3 自定义角色

- 可创建、编辑、启停和软删除。
- 编码创建后不可修改。
- 仍绑定用户时默认禁止删除。
- 停用后立即不参与权限计算。

## 13. 前端

### 13.1 Auth Store

新增：

```text
roles
permissions
permissionVersion
can(permission)
canAny(permissions)
canAll(permissions)
isSuperAdmin
```

保留 `isAdmin` 兼容，但不再用于一般功能授权。

### 13.2 路由

```ts
{
  path: '/admin/users',
  requiredPermission: 'users.read'
}
```

支持 `requiredPermissions` 和 `any/all`。无权限进入统一 403 页面或安全默认页。

### 13.3 菜单与按钮

菜单声明 `requiredPermission`。按钮统一使用 `usePermission().can()` 或 `v-permission`。

禁止新增：

```ts
user.role === 'operator'
```

应使用：

```ts
can('users.update')
```

### 13.4 管理页面

新增角色管理页面，支持：

- 角色列表、创建、编辑和启停。
- 按模块配置权限。
- 高风险权限提示。
- 查看绑定用户数量。
- 用户管理页分配多个角色。
- 超级管理员保护提示。

## 14. 配置与灰度

建议配置：

```yaml
rbac:
  mode: enforce
  cache_ttl_seconds: 1800
  startup_validation: true
  audit_denied_requests: true
```

模式：

- `shadow`：计算 RBAC 结果并记录差异，最终沿用旧授权。
- `enforce`：RBAC 为最终授权。

按 `PROJECT_CONVENTIONS.md`，新增配置必须同步：

- `backend/config/config.yaml`
- 配置结构体和 `mapstructure`
- Viper 默认值
- `deploy/config.example.yaml`
- 相关环境变量示例
- 配置加载测试

## 15. 发布与回滚

### 15.1 顺序

1. 冻结接口、页面和权限矩阵。
2. 执行 RBAC Schema SQL。
3. 执行兼容 Seed SQL。
4. 部署 RBAC 核心和声明式路由。
5. 以 `shadow` 比较新旧鉴权。
6. 要求 admin/user 差异为零。
7. 部署前端权限控制和管理页面。
8. 切换 `enforce`。
9. 清理不再承担授权作用的硬编码角色判断。

### 15.2 回滚

- 从 `enforce` 切回 `shadow`。
- 保留 `users.role` 作为紧急兼容数据。
- 不自动删除 RBAC 表和审计日志。
- 不在正常升级 SQL 中包含破坏性回滚。
- 如需彻底删除，另行提供显式回滚 SQL。

## 16. 安全与审计

必须覆盖：

- 水平越权。
- 垂直越权。
- 伪造前端权限状态。
- 伪造 JWT 旧角色字段。
- 缓存版本不一致。
- 自定义角色获取 `*`。
- 移除最后超级管理员。
- 页面隐藏但接口直调。

必须审计：

- 角色创建、修改、启停和删除。
- 角色权限变化。
- 用户角色变化。
- admin 分配和移除。
- 内置 user 权限变化。
- 高频权限拒绝。

## 17. 可观测性

指标：

```text
rbac_authorization_total{permission,result}
rbac_authorization_denied_total{permission}
rbac_permission_cache_hit_total
rbac_permission_cache_miss_total
rbac_permission_db_fallback_total
rbac_registry_unclassified_routes
rbac_shadow_diff_total{legacy_result,rbac_result}
rbac_role_change_total{action}
```

日志包含 Request ID、主体、权限编码、路由模板和结果，不记录密码、Token、完整管理员 API Key 或敏感请求体。

## 18. 验证策略

### 18.1 单元测试

- 多角色权限并集。
- 停用角色排除。
- `*` 通配。
- 未知权限拒绝。
- 固定 Key 的版本匹配和失配。
- Redis 故障回源。
- admin/user 系统角色保护。
- 最后超级管理员保护。

### 18.2 Repository 和事务

- 角色、权限、用户角色关联。
- 全量替换。
- 唯一约束。
- 用户版本递增。
- 全局策略版本递增。
- 修改和版本更新同事务。
- 并发移除 admin。

### 18.3 SQL

- 空库执行。
- 历史数据执行。
- 重复执行。
- admin/user 回填数量。
- 未知角色预检查。
- 代码权限目录与数据库初始化一致。
- 编号和项目规约检查。

### 18.4 接口

- 未登录 401。
- 无权限 403。
- 有权限进入原 Handler。
- admin 访问全部原功能。
- user 访问全部原个人功能。
- user 不能访问管理功能。
- 自定义角色只访问被授予接口。
- 管理员 API Key 保持兼容。

### 18.5 前端

- admin 原菜单、页面和按钮不减少。
- user 原个人菜单、页面和按钮不减少。
- 自定义角色只显示授权入口。
- 前端隐藏不能绕过后端。

### 18.6 防遗漏

必须自动断言：

```text
实际 Gin 路由
= RBAC 受控路由
+ 明确排除路由
```

当前基线为 `522 = 427 + 95`。未来数字可以变化，但等式必须成立。

## 19. 验收标准

| 编号 | 条件 |
|---|---|
| `AC-01` | 现有 admin 升级前后接口和页面访问矩阵一致 |
| `AC-02` | 现有 user 升级前后个人接口和页面访问矩阵一致 |
| `AC-03` | admin 拥有不可移除的 `*` |
| `AC-04` | 用户可绑定多个角色，权限取并集 |
| `AC-05` | 普通用户资源归属校验保持有效 |
| `AC-06` | 427 条受控接口全部声明权限 |
| `AC-07` | 95 条排除接口全部有原因 |
| `AC-08` | 新增未分类路由导致 CI 失败 |
| `AC-09` | Handler 不重复实现通用鉴权 |
| `AC-10` | SQL 独立、可执行、可安全重复执行 |
| `AC-11` | 历史 admin/user 绑定由 SQL 完成 |
| `AC-12` | 新用户和默认角色在同一事务创建 |
| `AC-13` | 自定义角色不能获得 `*` |
| `AC-14` | 不能移除最后一个超级管理员 |
| `AC-15` | 权限变更通过版本校验及时生效 |
| `AC-16` | Redis 故障时安全回源数据库 |
| `AC-17` | 前端统一按权限控制 |
| `AC-18` | 角色和权限变更具有审计记录 |
| `AC-19` | shadow 模式下现有 admin/user 新旧结果差异为零 |

## 20. 实施顺序与 MVP 拆分边界

1. 权限目录、接口和页面矩阵。
2. RBAC Schema 与兼容 Seed SQL。
3. Repository、Service、版本和缓存。
4. Principal、Registry 和统一中间件。
5. 管理接口声明迁移。
6. 用户个人接口声明迁移。
7. 防遗漏和 shadow 对比。
8. 角色管理 API 与安全保护。
9. 前端权限基础设施。
10. 角色管理和用户角色页面。
11. enforce 切换、兼容验证和清理。

后续可使用 `plan-to-mvp-list` 按上述边界拆分，避免将 427 条接口迁移放进一个不可独立验证的任务。

## 21. 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| 接口漏登记 | 高 | Registry、启动检查、CI 闭合公式 |
| SQL 与代码权限不一致 | 高 | 执行 SQL 后对比代码目录 |
| admin/user 丢权限 | 极高 | `*`、完整 user Seed、shadow 零差异门禁 |
| 缓存保留旧授权 | 高 | 固定 Key + 双版本校验 |
| Redis 故障 | 中 | 数据库回源、请求合并和监控 |
| AdminAuth 拆分漏洞 | 高 | 认证授权分层测试、逐模块迁移 |
| 最后 admin 被移除 | 极高 | 事务锁与并发测试 |
| 权限过粗或过细 | 中 | 业务动作分组，高风险动作独立 |

## 22. 追踪矩阵

| 目标 | 组件 | 验收 |
|---|---|---|
| 完整 RBAC0 | RBAC 表、Service | `AC-04` |
| admin 超级管理员 | `*`、系统角色保护 | `AC-03`、`AC-14` |
| user 兼容 | Seed SQL、权限矩阵 | `AC-02` |
| SQL 初始化 | `sqlArchiving` | `AC-10`、`AC-11` |
| 低侵入鉴权 | Registry、路由包装器、中间件 | `AC-09` |
| 不漏接口 | 路由闭合检查 | `AC-06`～`AC-08` |
| 分布式缓存 | 固定 Key、双版本 | `AC-15`、`AC-16` |
| 前端权限 | Store、路由、菜单、按钮 | `AC-17` |
| 平滑上线 | shadow/enforce | `AC-19` |

## 23. 评审记录

| 版本 | 状态 | 内容 |
|---|---|---|
| v0.1 | Draft | 初始 RBAC0、多角色、admin 通配、SQL 初始化和声明式鉴权方案 |
| v0.2 | Draft | 明确普通用户功能也纳入 RBAC，资源归属逻辑继续保留 |
| v0.3 | Draft | 明确 Go/Gin 路由旁声明权限，通过 Registry 和统一中间件鉴权 |
| v0.4 | Draft | 完善分布式缓存，改为固定用户 Key、Value 双版本校验和数据库回源 |
| v1.0 | Final — user approved | 用户确认无疑问并批准输出最终 Plan |
