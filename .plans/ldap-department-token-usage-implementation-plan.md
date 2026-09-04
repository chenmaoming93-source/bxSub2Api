# LDAP 用户全量同步与部门 Token 用量统计实现计划

> **状态：Final — user approved（最终版，用户已批准）**  
> **版本：1.1**  
> **日期：2026-09-04**  
> **变更摘要：**在保留 LDAP 用户部门资料链路的基础上，将部门报表改为基于 user_id Projection 并按 users.department 聚合，同时增加部门平均 Token、部门员工明细和员工消耗图表。

---

## 1. 引言

### 1.1 背景

项目已有 LDAP 登录、`local_login_accounts` 本地登录账号白名单、API Key 用户认证缓存、可配置 Token 统计 Projection，以及 `/admin/users` 和 `/admin/usage` 管理页面。

当前 LDAP 用户首次创建时主要保存登录账号到 `users.email`，并将 LDAP `displayName` 保存到 `users.username`。目前没有保存 LDAP 部门，也没有按部门统计 Token 用量。

### 1.2 目标

1. LDAP 用户首次登录时保存部门；
2. 在 `/admin/users` 增加“同步 LDAP 用户信息”按钮；
3. 使用 LDAP 服务账号全量同步 LDAP 用户信息；
4. 使用仅包含 `user_id` 维度的可配置 Token 统计 Projection；
5. 在 `/admin/usage` 保留部门 Token 环状图和列表，并增加部门人数、平均 Token 和员工明细图表。

### 1.3 成功标准

- LDAP 用户首次登录后部门正确保存；
- 非 LDAP 用户部门为空；
- 管理员可一键同步所有 LDAP 用户；
- 同步不需要用户 LDAP 密码；
- Token 统计继续使用现有 Projection、Redis 累加和 MySQL 聚合流程；
- 不修改 `usage_logs`；
- `/admin/usage` 可按日期查询当前部门的 Token 总量、占比、人数和平均值；
- 点击部门可懒加载该部门所有用户的 Token 数、占比和消耗排行；
- 统计始终以查询时 users.department 的当前值归类，不采用聚合记录中的历史部门。

### 1.4 非目标

本次不实现 LDAP 任意属性动态映射、LDAP 账号自动删除或禁用本地用户、修改 `signup_source` 语义、新增 LDAP `auth_identity`、修改 `usage_logs`，以及修改 Token 原始日志、创建 department + user_id Projection 或在页面上实现与本需求无关的部门筛选。

---

## 2. 假设与已确认决策

### 2.1 已确认事实

当前 `shouldUseLDAP` 规则为：LDAP 未启用或认证器不存在时使用普通登录；账号命中 `local_login_accounts` 时使用普通登录；LDAP 启用且账号未命中时使用 LDAP 登录。比较忽略大小写并去除首尾空格。

用户分类直接按照 `users.email` 与 `local_login_accounts` 比较，不使用 `signup_source`。模型调用中的 `user` 通常来自 `apiKey.User`，逻辑来源是 `users` 表，但 API Key 认证存在 L1 内存缓存、L2 Redis 缓存和数据库回源，因此不是每笔模型调用都会查询用户表。

### 2.2 决策

1. 部门保存为 `users.department`，不使用普通用户自定义属性表；
2. Token 报表只使用 `user_id` Dimension 的 Projection，`department` 不再作为新 Projection 的可配置维度；
3. 部门报表查询时通过 `user_id` 关联 `users.department`，始终使用用户当前部门；
4. 不增加 `usage_logs` 字段；
5. LDAP 同步采用同步批处理模式，返回成功、失败和未找到的汇总；
6. LDAP 查询失败时保留原部门并报告失败；
7. LDAP 查询成功但 department 缺失、为空或全为空格时清空部门；
8. 用户没有部门时在当前部门聚合中统一归类为 `未设置`；事件仍可携带 department 供兼容旧 Projection 使用。

---

## 3. 概念功能设计

### 3.1 LDAP 首次登录同步部门

当前入口为 `backend/internal/handler/auth_handler.go`。登录流程为：判断 LDAP、调用 LDAP 认证并读取完整 identity、使用用户输入账号查找 `users.email`，首次不存在时创建本地用户并保存资料。

映射关系：

| LDAP 信息 | 本地处理 |
|---|---|
| 用户输入账号 | 保存到 `users.email` |
| `UsernameAttribute` | displayName 缺失时的 username 回退值 |
| `EmailAttribute` | 不覆盖 `users.email` |
| `DisplayNameAttribute` | 保存到 `users.username` |
| `DepartmentAttribute` | 保存到 `users.department` |
| DN | 仅用于 LDAP bind，不落库 |

当前登录调用需要保留用户输入账号作为本地键，不再只使用 LDAP `identity.Username` 作为 `LoginLDAP` 的账号参数。

### 3.2 管理员全量同步 LDAP 用户

新增 `/admin/users` 操作按钮“同步 LDAP 用户信息”，需要 `users.update` 权限。

处理所有未软删除用户：命中 `local_login_accounts` 的用户清空部门；其他用户使用 `users.email` 通过 LDAP 服务账号查询并同步 `username` 与 `department`。单用户失败不阻断其他用户。

响应汇总包括用户总数、LDAP 候选数、成功数、清空数、未找到数、失败数、错误摘要和耗时。

### 3.3 可配置 Token 统计与当前部门归类

现有 Token 统计链路为：

```text
请求完成 → 构造内存 UsageLog → submitDynamicTokenUsage
→ 生成包含 user_id 的 UsageEvent → 读取启用的 Projection → Redis 累加
→ 同步 token_stat_aggregates → 管理端从 MySQL 查询
```

部门报表只依赖管理员在界面配置的 `user_id + total_tokens + day` Projection，不创建 `department` 或 `department + user_id` Projection，也不通过 SQL 初始化配置项。

查询部门报表时，MySQL 以 `token_stat_aggregates.user_id = users.id` 关联当前用户，并返回用户当前的 `users.department` 和日期范围内的 Token 汇总；服务层在内存中按部门聚合。这样用户改部门后，所选日期范围的 Token 会按当前部门重新归类，且无 Token 用户仍可计入人数和平均值。

API Key 认证缓存快照继续携带 `Department`，以保证请求事件和旧兼容 Projection 的行为；用户部门更新后调用 `InvalidateAuthCacheByUserID`，使后续请求加载最新值。该函数不查询数据库，也不修改 `usage_logs`。

### 3.4 `/admin/usage` 部门 Token 图

在“场景 Token 用量”上方保留“部门 Token 用量”区域：左侧环状图，右侧显示部门名称、Token 数、占比、用户数和平均 Token。点击部门后在弹窗中懒加载并展示所有当前属于该部门的员工、员工 Token 数、部门内占比和按 Token 排序的员工柱状图。

---

## 4. 详细技术设计

### 4.1 组件职责

| 组件 | 职责 |
|---|---|
| `AuthHandler` | 判断登录类型，传递完整 LDAP identity |
| `LDAPDirectory` | 使用服务账号查询 LDAP 用户资料 |
| `AuthService` | LDAP 用户首次创建和登录 |
| `LDAPUserSyncService` | 批量同步 LDAP 用户资料 |
| `APIKeyService` | 在认证缓存快照中携带部门 |
| `GatewayService` | 将当前用户部门传入 Token 统计事件 |
| `tokenstat` | 维护 Token 维度、按 user_id 查询并按当前部门聚合 |
| `UserHandler` | 提供管理员同步接口 |
| `UsageHandler` | 提供部门汇总和员工明细查询接口 |
| `UsersView` | 提供同步按钮和部门资料展示 |
| `UsageView` | 展示部门概览、部门平均值和员工 Token 图表 |
| `DepartmentDistributionChart` | 展示部门环状图、部门汇总列表并触发部门选择 |
| `DepartmentUserUsageChart` | 展示选定部门的员工 Token 消耗排行 |

### 4.2 数据模型

#### `users`

增加：

| 字段 | 类型 | Null | 默认值 | 说明 |
|---|---|---:|---|---|
| `department` | `VARCHAR(255)` | 否 | `''` | 当前部门，空表示未设置 |

不设置唯一约束，更新时不改变用户登录账号。

#### `token_stat_aggregates`

增加 Token 统计自身的冗余字段：

| 字段 | 类型 | Null | 默认值 | 说明 |
|---|---|---:|---|---|
| `department` | `VARCHAR(255)` | 是 | `NULL` | 部门维度冗余值 |

`user_id` Projection 的聚合记录使用现有 `user_id` 冗余字段和通用 `dimension_values`。`department` 冗余字段及其索引保留作旧事件/旧 Projection 兼容，但新的部门报表不读取它，也不新增部门查询索引。

明确不修改：

```text
usage_logs
 auth_identities
 user_attribute_values
```

### 4.3 Token 统计维度

管理员可配置维度列表只保留现有 `user_id` 等可用维度，不再展示 `department`。内部仍保留 `DimensionDepartment`、事件字段和聚合冗余字段，以兼容已经存在的事件、旧 Projection 和历史数据。

新的部门报表只要求配置：

```text
Dimension: user_id
Metric: total_tokens
Period: day
```

`AsyncPipeline` 继续使用通用 Projection 流程；部门报表不从事件中的历史 department 读取，而是使用聚合记录的 user_id 关联 users 表取得当前部门。

无部门时事件值必须为非空字符串：

```text
有效部门 → 实际部门名称
空部门 → 未设置
```

### 4.4 Projection

部门报表使用管理员在界面中创建、发布并激活的用户日统计 Projection：

```text
名称：用户 Token 用量
维度：user_id
指标：total_tokens
周期：day
```

不创建任何 Projection 或指标初始化 SQL。`department` 不再作为新 Projection 的可配置维度展示；内部仍可识别旧 department Projection，报表查询不会使用旧 Projection。未配置或未激活 user_id Projection 时接口返回明确的配置状态。

### 4.5 API-01：同步 LDAP 用户

```http
POST /api/v1/admin/users/sync-ldap
```

权限：`users.update`。

请求体：`{}`。

响应字段：`total_users`、`ldap_candidates`、`synced`、`local_users_cleared`、`not_found`、`failed`、`duration_ms`，并可附带受限错误摘要。

重复执行幂等。LDAP 未启用或配置不完整返回 `400`；无权限返回 `403`；内部数据库错误返回 `500`；LDAP 整体不可用返回 `503`。单用户错误作为汇总返回。

### 4.6 API-02：部门 Token 统计

```http
GET /api/v1/admin/usage/department-stats
```

参数：

```text
start_date=YYYY-MM-DD
end_date=YYYY-MM-DD
timezone=Asia/Shanghai
```

日期采用半开区间 `[start_date 00:00:00, end_date + 1 天 00:00:00)`。

接口使用激活的 `user_id + total_tokens + day` Projection，通过一条 MySQL 查询将 `users` 与用户维度聚合数据关联，得到每个未软删除用户的当前 department 和日期范围 Token 总量；服务层在内存中按部门聚合，返回部门 Token、占比、人数和平均 Token。

新增部门员工接口：

```http
GET /api/v1/admin/usage/department-stats/users
```

参数增加 `department`、`page`、`page_size`。接口只在点击部门后调用，返回该当前部门全部用户（无 Token 用户补 0）、部门内占比、分页信息和一致性状态。员工图表按 Token 倒序展示返回数据中的前 20 名。

### 4.7 LDAP 字段标准化

```text
normalizeDepartment(values):
    for value in values:
        value = trim(value)
        if value != "":
            return value
    return ""
```

LDAP 属性不存在、没有值、值全为空白均按无部门处理；多值取第一个非空值；查询失败不覆盖原值。

### 4.8 安全与权限

同步接口要求管理员和 `users.update` 权限。LDAP 服务密码只从服务端配置读取，不允许前端传递或记录。LDAP 查询属性固定来自配置，不接受任意 LDAP 查询表达式。日期范围和部门长度遵循现有限制，并记录同步操作汇总。

### 4.9 可靠性与性能

LDAP 同步使用有限并发、单用户超时和可取消 context。每个用户独立更新，不使用包围所有 LDAP 请求的全局事务。模型请求不增加部门数据库查询，继续使用 API Key 认证快照。Token 统计继续异步、失败不阻断模型请求。报表只读 MySQL 的 users 与 user_id 聚合关联结果，并返回 eventual consistency 信息；员工明细仅在部门点击后懒加载。

### 4.10 迁移、兼容和回滚

保留已完成的 `users.department`、`token_stat_aggregates.department` 迁移并提升 API Key 快照版本。本次不新增 SQL，不初始化 Projection 或指标；管理员手动配置 user_id 日 Projection。历史聚合中的 department 不参与新报表，报表以查询时 users.department 重新归类。旧 department Projection 可手动停用，但保留字段和内部识别能力以兼容旧数据。

---

## 5. 关键流程伪代码

### 5.1 LDAP 登录

```text
login(request):
    validate request
    if shouldUseLDAP(request.email):
        identity = LDAP.Authenticate(request.email, request.password)
        localAccount = normalize(request.email)
        user = findUserByEmail(localAccount)
        if user not found:
            user = createUser(
                email = localAccount,
                username = identity.displayName or identity.username,
                department = normalizeDepartment(identity.department),
                randomPassword = true,
                signupSource = "email"
            )
        return issueToken(user)
    return normalLocalLogin(request)
```

### 5.2 全量同步

```text
syncLDAPUsers(actor):
    authorize actor with users.update
    validate LDAP service configuration
    users = list all non-deleted users

    for user in bounded concurrency:
        if matchesLocalLoginAccount(user.email):
            clear department if needed
            invalidate cache if changed
            continue

        ldapUser, err = directory.LookupUser(user.email)
        if err:
            keep old department and record failure
            continue

        username = ldapUser.displayName or ldapUser.username
        department = normalizeDepartment(ldapUser.department)
        update users.username and users.department
        invalidate cache if department changed

    return summary
```

### 5.3 Token 统计事件

```text
submitDynamicTokenUsage(usageLog, department):
    dimensions = build existing dimensions from usageLog
    department = trim(department)
    if department == "":
        department = "未设置"
    dimensions["department"] = StringValue(department)
    enqueue UsageEvent(dimensions, total_tokens)
```

### 5.4 部门报表

```text
getDepartmentStats(actor, dates):
    authorize actor with usage read permission
    validate dates
    projection = resolve active daily user_id Projection
    if projection missing:
        return configuration unavailable
    userRows = MySQL users LEFT JOIN token_stat_aggregates
        ON aggregate.user_id = users.id
        AND projection/date/metric filters
    departments = in-memory group userRows by current users.department
    calculate total_tokens, user_count, average_tokens, percentage
    return departments sorted by total_tokens descending

getDepartmentUsers(actor, department, dates, page):
    authorize actor with usage read permission
    validate dates and department
    query current users in department with user_id aggregate totals
    include zero-token users
    calculate percentage against department total
    return paginated rows and top employee chart data
```

---

## 6. 验证策略

### 单元测试

覆盖 LDAP 判定、部门缺失/空值/空白/多值、部门标准化、Token 事件维度、Projection 行为、API Key 快照序列化及缓存版本。

### LDAP 集成测试

覆盖服务账号查询、用户不存在、LDAP 故障、部分失败继续、非 LDAP 用户清空部门、重复同步幂等。

### Token 统计集成测试

覆盖 user_id Projection 创建发布激活、用户事件写入 Redis、同步 MySQL、按当前 users.department 汇总、多日查询、“未设置”、零 Token 用户计数、平均值、部门占比、员工明细分页和员工占比；确认不依赖旧 department Projection。

### API 与前端测试

覆盖权限、配置错误、部分失败响应、日期校验、Projection 未激活、同步按钮 loading、部门图空数据/错误状态、部门平均值和占比、部门点击后的懒加载/分页、员工图表排序、图表与列表数值一致、响应式和深色模式。

### 验收标准

| 编号 | 标准 |
|---|---|
| AC-01 | LDAP 首次登录用户部门正确保存 |
| AC-02 | LDAP department 缺失或为空时显示“未设置” |
| AC-03 | 非 LDAP 用户同步后 department 为空 |
| AC-04 | 同步接口不需要用户 LDAP 密码 |
| AC-05 | 单用户同步失败不影响其他用户 |
| AC-06 | 模型调用不会因部门归类增加额外用户表查询 |
| AC-07 | Token 事件包含 user_id，且兼容携带 department |
| AC-08 | 现有 Token Projection 和旧聚合数据不被破坏 |
| AC-09 | 部门统计只使用激活的 user_id 日 Projection，并关联当前 users.department |
| AC-10 | `/admin/usage` 正确展示部门 Token 总量、占比、人数和平均值 |
| AC-11 | 点击部门后可懒加载全部当前员工、员工占比和消耗排行 |
| AC-12 | `usage_logs` 表结构保持不变 |

---

## 7. 实施顺序与拆分边界

1. **用户字段与 LDAP 资料模型**：`users.department`、LDAP department 配置、读取标准化、首次登录写入。
2. **API Key 缓存扩展**：认证快照部门字段、版本升级、缓存失效。
3. **管理员 LDAP 全量同步**：同步服务、API、权限、操作按钮和汇总反馈。
4. **Token 统计用户维度**：确认 user_id 事件、用户日 Projection、聚合查询和配置维度下架兼容。
5. **部门报表与员工明细**：当前部门 MySQL 关联查询、内存聚合、部门概览 API、员工懒加载 API、图表组件、Usage 页面、国际化和测试。

关键检查点：

```text
LDAP 同步 → users.department 更新 → API Key 缓存失效 → 下一次模型调用取得新部门
模型调用 → UsageEvent.user_id → user_id Projection → MySQL 与 users.department 关联 → 内存部门聚合 → 部门/员工图表
```

---

## 8. 风险与开放项

| 风险 | 缓解措施 |
|---|---|
| LDAP 用户量较大导致同步耗时 | 有限并发、单用户超时、汇总响应 |
| LDAP 查询键不一致 | 始终使用 `users.email` 作为查询键 |
| Redis 存在旧快照 | 提升快照版本并自动回源 |
| user_id Projection 未配置或未激活 | 由管理员在界面中配置/激活，并返回明确配置状态 |
| 旧 department Projection 或聚合数据存在 | 新报表只解析 user_id Projection，旧字段保留兼容，不参与新聚合 |
| 用户部门变化导致报表归类变化 | 明确采用查询时 users.department，不修改 Token 原始聚合记录 |
| LDAP 短暂不可用 | 保留原部门并报告失败 |
| LDAP displayName 覆盖手工 username | 明确 LDAP 用户资料以 LDAP 为准 |
| 部门过多影响图表可读性 | 列表滚动、颜色循环、按 Token 排序 |

非阻塞开放项：未来是否增加更多 LDAP 属性映射、是否升级为异步同步任务、是否增加部门筛选器。本次不影响实现。

---

## 9. 追踪矩阵

| 需求 | 模块 | 组件 | 验收标准 |
|---|---|---|---|
| FR-01 LDAP 首次保存部门 | LDAP 登录 | `AuthHandler`、`AuthService`、LDAP directory | AC-01、AC-02 |
| FR-02 区分 LDAP 与非 LDAP | 用户分类 | `local_login_accounts`、同步服务 | AC-03 |
| FR-03 全量同步 | LDAP 同步 | `LDAPUserSyncService`、UserHandler、UsersView | AC-03、AC-04、AC-05 |
| FR-04 用户加入 Token 统计 | Token 维度 | `tokenstat`、API Key snapshot、Gateway | AC-06、AC-07、AC-08 |
| FR-05 按当前部门统计 | 部门报表 | UsageHandler、MySQL users/aggregate query、内存聚合 | AC-09、AC-10 |
| FR-06 部门员工明细 | 员工报表 | UsageHandler、UsageView、员工图表 | AC-11 |
| FR-07 不修改 usage 表 | 数据边界 | users 和 tokenstat 表 | AC-12 |
| NFR-01 不增加每请求数据库查询 | 缓存性能 | API Key auth cache | AC-06 |
| NFR-02 LDAP 故障可局部降级 | 同步可靠性 | bounded concurrency、错误汇总 | AC-05 |
| NFR-03 管理员权限控制 | 安全 | RBAC、服务端配置 | API 契约测试 |

---

## 10. Review Record

### v1.0

用户已批准 LDAP 用户全量同步、`users.department`、API Key 缓存部门快照和基础部门 Token 报表。

### v1.1

用户批准将旧部门报表方案增强为：只配置 `user_id + total_tokens + day` Projection，查询时关联当前 `users.department`，并增加部门人数、平均 Token、部门员工明细、员工占比、员工消耗图表和懒加载；不新增 Projection 初始化 SQL。

状态：

```text
Final — user approved
```
