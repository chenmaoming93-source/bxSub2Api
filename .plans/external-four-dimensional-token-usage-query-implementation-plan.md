# 四维 Token 消耗量外部查询接口实施 Plan

**状态：Final — user approved**  
**版本：v1.0**  
**日期：2026-07-31**  
**变更摘要：** 最终确定新增仅受 integrations 外部 Token 系统保护的四维 Token 用量查询接口；用户输入精确匹配 `email`；分别返回当前自然日、周、月结果；实施前必须确认 URL 不与现有路由冲突。

## 1. 引言

### 1.1 背景与问题

项目已有可配置多维 Token 统计体系：活动统计投影持续维护 `total_tokens`，当前自然日、自然周、自然月的实时值位于 Redis。现需向外部系统提供业务名称查询接口，入参为用户名、分组名、API Key 名和路由别名。其中“用户名”精确匹配用户 `email` 字段。

接口必须区分：业务对象不存在、四维统计投影未配置、投影已配置但当前周期无 Redis 数据、存在统计数据，以及 Redis 故障。

### 1.2 用户与目标

使用者是持有 integrations 外部 Access Token 的外部系统。

- `G-01`：提供稳定的四维 Token 用量外部查询接口。
- `G-02`：将业务名称安全解析为内部统计维度。
- `G-03`：只读取可配置 Token 统计体系的当前周期 Redis 数据。
- `G-04`：日、周、月分别计算、判断、查数和返回。
- `G-05`：明确区分“统计维度未配置”和“已配置但为 0”。
- `G-06`：用户、分组、API Key 或路由别名不存在时返回明确的 HTTP 404。
- `G-07`：接口只受 integrations 外部 Token 鉴权约束，不受登录、JWT、管理员身份或 RBAC 约束。
- `G-08`：新增 HTTP Method 与 URL 组合不得和已有路由重复。

### 1.3 成功标准

- 有效外部 Token 在没有登录 Cookie、JWT、管理员身份和 RBAC 权限时仍可调用。
- 用户通过 `email` 精确匹配，四类业务对象错误分别返回稳定的 404 错误码。
- 精确四维 `ACTIVE` 投影不存在时，日、周、月分别返回 `dimension_configured=false`。
- 投影存在但 Redis Field 不存在时返回 `total_tokens=0` 和 `data_present=false`。
- Redis 故障返回 503，不伪装成零值或未配置。
- 查询不扫描 Redis、不读取旧统计 namespace、不临时聚合其他投影。
- 最终路由在 Gin 路由表中只注册一次，且中间件边界符合要求。

### 1.4 范围

包含外部 HTTP 接口、路由冲突检查、外部 Token 鉴权和限流、对象解析及关系校验、活动投影发现、三个自然周期计算、Redis 精确查询、审计、测试、Wire 依赖注入与路由覆盖登记。

### 1.5 非目标

- 不查询历史周期或 MySQL 历史数据；
- 不新增或自动创建统计投影、维度、指标；
- 不聚合其他粒度投影；
- 不读取旧 `sub2api:token_stats:*`；
- 不修改统计写入流程；
- 不提供批量、分页、自定义时间范围查询；
- 不返回 API Key 密钥；
- 不新增数据库表或迁移；
- 不新增登录、JWT、Admin Auth 或 RBAC 权限点。

## 2. 假设与决策

### 2.1 已确认约束

| 编号 | 约束 |
|---|---|
| `C-01` | 复用 `/api/v1/integrations` 的外部 Bearer Token、开关、限流和 hardening |
| `C-02` | `username` 精确匹配用户 `email` |
| `C-03` | 四类业务对象不存在时返回 HTTP 404，并指出具体对象 |
| `C-04` | 日、周、月当前数据从 Redis 分别查询 |
| `C-05` | 投影存在但 Redis 无数据时视为 0 |
| `C-06` | 接口不受登录和 RBAC 约束，只受外部系统 Token 系统约束 |
| `C-07` | 新增 URL 不得与已有路由重复 |

### 2.2 路由决策

候选接口为：

```http
POST /api/v1/integrations/token-usage/query
```

实施前必须检查静态注册代码和 Gin 最终路由表，以 `HTTP Method + normalized absolute path` 判断冲突。只有候选组合未被占用时才采用。若冲突，不得覆盖或删除已有路由，应选择并评审未占用的等价 integrations 路径，并以自动化契约测试固定最终 URL。

### 2.3 鉴权与 RBAC 决策

目标路由只挂载：

1. 全局日志、CORS、安全 Header 等基础中间件；
2. integrations 外部 Token 鉴权；
3. integrations 限流与 hardening。

不得挂载 JWT Auth、Session 登录校验、Admin Auth、Admin Identity Auth、Gateway API Key Auth、RBAC `RequirePermission` 或用户/管理员权限中间件。若项目路由覆盖校验要求登记非 RBAC 路由，应将最终路由登记为已知排除项，理由为“由 integrations Bearer Token 保护”；排除登记不得引入 RBAC 判断。

### 2.4 统计投影决策

只接受状态为 `ACTIVE` 且规范化维度签名精确等于以下组合的投影：

```text
user_id,api_key_id,group_id,route_alias
```

部分维度投影、包含额外维度的投影及 `DRAFT`、`PUBLISHED`、`DISABLED` 投影均视为未配置。不得由多个投影临时推导结果。

### 2.5 状态语义

- 投影未配置：HTTP 200；各周期 `dimension_configured=false`、`data_present=false`、`total_tokens=null`。
- 投影已配置但 Redis Field 不存在：`dimension_configured=true`、`data_present=false`、`total_tokens=0`。
- Field 存在且值为 0：`dimension_configured=true`、`data_present=true`、`total_tokens=0`。
- 任一 Redis 查询失败或数据非法：整个请求返回 HTTP 503，不返回部分成功结果。

### 2.6 对象关系与 404 顺序

- API Key 必须属于目标用户并可用于目标分组；
- 路由别名必须存在于目标分组；
- 全局存在但关系不成立时，对外仍按对应资源不存在处理，避免资源枚举；
- 固定解析顺序为用户邮箱、分组名、API Key 名、路由别名，首次失败即返回对应 404。

### 2.7 实现阶段需验证的既有事实

- API Key 名在同一用户范围内的唯一性；
- 数据库 `email` 比较的大小写语义；
- API Key 与分组的现有授权规则；
- 路由别名的现有规范化规则；
- 动态统计实际使用的时区和 shard count。

这些事实必须沿用现有产品规则，不由本接口发明新规则。

## 3. 概念功能设计

### 3.1 `FR-01` 外部查询

请求示例：

```json
{
  "username": "user@example.com",
  "group_name": "public",
  "api_key": "sk-ldap-openai-key-0123456789",
  "route_alias": "gpt-main"
}
```

流程为外部 Token 鉴权与限流、请求校验、对象解析、活动投影匹配、周期计算、Redis 查询、响应与审计。请求链路不存在登录、JWT 或 RBAC 判断。

### 3.2 `FR-02` 路由唯一性

注册前搜索所有路由代码并检查测试 Router 的最终路由表；Method 与绝对路径组合必须唯一。Gin 的重复路由错误不得被捕获或忽略。

### 3.3 `FR-03` 业务对象解析

| 外部输入 | 匹配目标 | 统计维度 |
|---|---|---|
| `username` | 用户 `email` | `user_id` |
| `api_key` | `api_keys.key` 明文（数据库唯一约束），且必须属于目标用户和分组 | `api_key_id` |
| `group_name` | 分组名称 | `group_id` |
| `route_alias` | 目标分组路由配置 | `route_alias` |

### 3.4 `FR-04` 统计投影发现

从已初始化的活动投影快照中查找精确四维投影，并确认指标包含 `total_tokens`。快照未初始化属于内部错误，不能误报为未配置。

### 3.5 `FR-05` 当前周期查询

日、周、月分别使用写入端相同的周期、维度编码、哈希、分片、Key 和 Field 算法执行精确 `HGET`。允许用 pipeline 减少往返，但每个周期必须独立解析并生成状态。

### 3.6 `FR-06` 安全边界与审计

只接受 `Authorization: Bearer <integration-access-token>`。功能关闭返回 404，Token 缺失或错误返回 401。审计记录内部资源 ID、路由别名、投影 ID、结果、原因、来源 IP 和耗时，不记录 Authorization Header、外部 Token 或 API Key 密钥。

## 4. 详细技术设计

### 4.1 组件与数据流

```text
外部系统
  -> 全局基础中间件
  -> Integrations Token Auth / Rate Limit / Hardening
  -> External Token Usage Handler
  -> External Token Usage Service
      -> 业务 Repository / MySQL
      -> Active Projection Snapshot
      -> Dimension Identity Builder
      -> Redis Current Period Reader
```

JWT、Session、Admin Auth、Gateway API Key Auth 和 RBAC 不在该链路中。

### 4.2 组件职责

- `backend/internal/handler/external_token_usage_handler.go`：绑定、校验、错误映射、响应、审计。
- `backend/internal/service/external_token_usage_service.go`：对象解析、关系校验、投影匹配、周期查询编排。
- `backend/internal/repository/tokenstat/current_usage_reader.go`：精确构造动态统计 Redis Key/Field，区分缺失与故障。
- 既有业务 Repository：提供受范围约束的 email、分组名、用户内 API Key 名和分组路由别名查询；Service 不直接拼 SQL。
- `backend/internal/server/routes/integrations.go`：仅在 integrations 路由组注册最终接口。

### 4.3 API 契约

候选契约：

```http
POST /api/v1/integrations/token-usage/query
Authorization: Bearer <integration-access-token>
Content-Type: application/json
```

请求的四个字符串字段必填，去除首尾空白后非空，长度上限与对应存储字段一致。

成功响应：

```json
{
  "success": true,
  "data": {
    "query": {
      "username": "user@example.com",
      "group_name": "public",
      "api_key": "sk-ldap****6789",
      "route_alias": "gpt-main"
    },
    "resolved_dimensions": {
      "user_id": 101,
      "group_id": 12,
      "api_key_id": 880,
      "route_alias": "gpt-main"
    },
    "metric": "total_tokens",
    "timezone": "Asia/Shanghai",
    "periods": {
      "day": {
        "period_type": "D",
        "period_start": "2026-07-31T00:00:00+08:00",
        "period_end": "2026-08-01T00:00:00+08:00",
        "dimension_configured": true,
        "data_present": true,
        "total_tokens": 12500,
        "message": ""
      },
      "week": {
        "period_type": "W",
        "period_start": "2026-07-27T00:00:00+08:00",
        "period_end": "2026-08-03T00:00:00+08:00",
        "dimension_configured": true,
        "data_present": false,
        "total_tokens": 0,
        "message": "统计维度已配置，当前周期暂无数据"
      },
      "month": {
        "period_type": "M",
        "period_start": "2026-07-01T00:00:00+08:00",
        "period_end": "2026-08-01T00:00:00+08:00",
        "dimension_configured": false,
        "data_present": false,
        "total_tokens": null,
        "message": "统计维度未配置"
      }
    }
  }
}
```

当前投影配置并非按周期独立维护，因此正常情况下三个周期的 `dimension_configured` 一致；响应仍按要求逐周期携带该状态并独立查数。

错误契约：

| HTTP | 错误码 | 条件 |
|---:|---|---|
| 400 | `INVALID_REQUEST` | JSON、必填字段或长度无效 |
| 401 | `INVALID_ACCESS_TOKEN` | 外部 Token 缺失或错误 |
| 404 | `NOT_FOUND` | integrations 功能关闭 |
| 404 | `USER_NOT_FOUND` | email 精确匹配不到用户 |
| 404 | `GROUP_NOT_FOUND` | 分组不存在 |
| 404 | `API_KEY_NOT_FOUND` | 用户范围内 Key 不存在或关系不成立 |
| 404 | `ROUTE_ALIAS_NOT_FOUND` | 分组下路由别名不存在 |
| 429 | 既有限流错误码 | integrations 限流触发 |
| 503 | `TOKEN_USAGE_UNAVAILABLE` | Redis 不可用或统计值非法 |
| 500 | `INTERNAL_ERROR` | 其他非预期错误 |

接口只读且天然幂等，不需要 Idempotency Key 或分页。

### 4.4 Redis 契约

复用 `tokenstat.BuildDimensionIdentity`、自然周期算法和 shard 算法。Redis Hash Key 与写入端一致：

```text
sub2api:dynamic_token_stats:v1:{period_type}:{period_start}:{projection_id}:{shard}
```

Field 为：

```text
{dimension_hash_hex}:total_tokens
```

`redis.Nil` 映射为已配置但无数据；合法非负整数映射为存在数据；解析失败、负数、超时和连接错误映射为 503。禁止 `KEYS`、`SCAN`、遍历 shard 或 Field。

### 4.5 数据、事务与并发

本功能不新增数据表或迁移，使用既有用户、分组、API Key、模型路由、统计投影和 Redis Hash。查询为只读，不开启数据库事务。Redis `HGET` 与写入端 `HINCRBY` 并发安全；三个 Key 不构成原子快照，响应表示同一请求期间的近实时值。

用户 email 和 Token 用量属于业务敏感数据；日志优先记录 user ID。API Key 密钥和外部 Token 属于高敏感凭证，禁止读取、返回或记录。

### 4.6 可靠性、可观测性和性能

- Redis 不可用时不重试、不回退 MySQL，快速返回 503；客户端可按 503 退避重试。
- 建议指标：`integration_token_usage_queries_total{result}`、`integration_token_usage_query_duration_seconds`、`integration_token_usage_redis_reads_total{period,result}`、`integration_token_usage_dimension_missing_total`。
- 告警关注 Redis 错误率、非法统计值、401/429 异常升高和投影快照初始化失败。
- 单请求最多三个精确 `HGET`，额外内存为常数级；基础设施健康时接口自身 P95 目标不超过 100ms。

### 4.7 配置、发布与回滚

复用 `ExternalAPIKeyProvisioning.Enabled`、`AccessToken` 及现有限流配置，时区和 shard count 复用动态统计配置。发布前确认精确四维投影已按需要激活并验证 Redis 三周期数据。回滚只移除新路由和组件，无数据库或 Redis 数据回滚，不影响现有 integrations 接口和统计写入。

## 5. 伪代码

```text
function registerRoute(router):
  candidate = POST /api/v1/integrations/token-usage/query
  verify method+path does not conflict
  integrationGroup.use(externalTokenAuth, integrationHardening)
  integrationGroup.POST(finalPath, queryHandler)
  do not attach login, JWT, admin, gateway API-key, or RBAC middleware

function queryCurrentUsage(context, input):
  validate and trim input

  user = findUserByExactEmail(input.username)
  if absent: return USER_NOT_FOUND

  group = findGroupByName(input.group_name)
  if absent: return GROUP_NOT_FOUND

  apiKey = findAPIKeyByKey(input.api_key) // api_keys.key 唯一；并校验属于目标 user 与 group
  if absent or does not belong to user/group: return API_KEY_NOT_FOUND

  if routeAliasDoesNotExist(group.id, input.route_alias):
    return ROUTE_ALIAS_NOT_FOUND

  periods = naturalPeriods(clock.now(), statisticsTimezone)
  projection = findExactActiveProjection(
    user_id, api_key_id, group_id, route_alias,
    metric=total_tokens
  )

  if projection absent:
    return day/week/month each marked unconfigured

  identity = buildDimensionIdentity(projection.codes, resolvedValues)
  for each period in day, week, month:
    value = readExactRedisField(period, projection.id, identity)
    if redis failure or invalid value: return HTTP 503
    if field absent: result = configured, no-data, zero
    else: result = configured, data-present, value

  return complete three-period response
```

## 6. 验证策略与验收标准

### 6.1 测试范围

- 路由：候选 URL 盘点、Method+Path 唯一、最终路由只注册一次、原 integrations 路由回归。
- 安全边界：无外部 Token 401；功能关闭 404；有效外部 Token 在无 Cookie、JWT、管理员身份、RBAC 权限时成功；JWT 或 Gateway API Key 不能替代外部 Token。
- Service：email 精确匹配、四类 404 及固定顺序、跨用户/分组关系、精确活动投影匹配、未配置三周期结果。
- Redis Reader：日周月 Key、hash、Field、shard 与写入端兼容；缺失、零值、正值、非法值、故障；不访问旧 namespace、不扫描。
- Handler/契约：字段校验、错误映射、三周期固定结构、RFC 3339 时间、敏感信息不泄露。
- 构建：Wire 依赖图、RBAC Route Coverage 和相关后端测试通过。

### 6.2 验收标准

| 编号 | 验收条件 |
|---|---|
| `AC-01` | 最终 Method+URL 不冲突且只注册一次 |
| `AC-02` | 路由只位于 integrations 组，只接受外部 Bearer Token |
| `AC-03` | 无登录、JWT、管理员身份和 RBAC 权限仍可凭有效外部 Token 调用 |
| `AC-04` | 用户通过 `email` 精确匹配 |
| `AC-05` | 四类对象不存在时分别返回明确的 404 |
| `AC-06` | 仅精确四维 `ACTIVE` 投影视为已配置 |
| `AC-07` | 未配置时三个周期分别返回 `dimension_configured=false` 和 `total_tokens=null` |
| `AC-08` | 投影存在但 Redis Field 不存在时返回 0 和 `data_present=false` |
| `AC-09` | 日、周、月分别计算和查询，读写 Redis 规则一致 |
| `AC-10` | Redis 故障返回 503，不返回零值或未配置 |
| `AC-11` | 不扫描 Redis、不读取旧统计 namespace |
| `AC-12` | RBAC Coverage 通过，目标接口只登记外部 Token 排除而不执行 RBAC |
| `AC-13` | 原有 integrations 接口回归测试通过 |
| `AC-14` | 响应和日志不泄露 API Key 密钥或外部 Token |
| `AC-15` | Wire 构建及相关后端测试全部通过 |

## 7. 实施顺序与拆分指导

1. 路由、安全边界和既有数据规则盘点，确定最终 URL 与 Repository 语义。
2. 定义领域契约、错误类型及受范围约束的对象解析能力。
3. 实现与写入端共享规则的 Redis 当前周期 Reader。
4. 实现对象解析、活动投影匹配和三周期编排 Service。
5. 实现 Handler、仅外部 Token 的 integrations 路由及审计。
6. 完成 Wire、RBAC 排除登记、契约测试、回归测试和运维文档。

推荐按上述边界拆成约一个专注开发日的 MVP；Redis 读写兼容、业务查询 Service 和外部安全边界必须分别具备独立验证路径。

## 8. 风险与开放项

| 风险 | 影响 | 缓解 |
|---|---|---|
| 候选 URL 被间接注册 | 高 | 同时检查源码和最终 Gin 路由表 |
| 路由误挂 JWT/RBAC | 高 | 只扩展 integrations 组并测试中间件边界 |
| RBAC 排除导致无鉴权 | 严重 | 强制断言无外部 Token 返回 401 |
| API Key 名不唯一或分组关系复杂 | 高 | 先验证既有约束并复用产品规则 |
| email 比较语义不明确 | 中 | 通过 Repository 测试固化现状 |
| Redis 读写 Key 规则漂移 | 高 | 共用 Builder 并做端到端兼容测试 |
| 活动投影快照未初始化 | 高 | 启动初始化；未初始化返回内部错误 |
| Redis 故障误判零值 | 高 | 严格区分 `redis.Nil` 和其他错误 |

当前没有阻塞实施的开放决策；最终 URL 和既有数据规则由第一个实施阶段通过仓库事实确认。

## 9. 追踪矩阵

| 目标 | 功能 | 技术组件 | 验收 |
|---|---|---|---|
| `G-01`、`G-02` | `FR-01`、`FR-03` | Handler、Service、业务 Repository | `AC-04`、`AC-05` |
| `G-03`、`G-04` | `FR-04`、`FR-05` | Projection Snapshot、Redis Reader | `AC-06`～`AC-11` |
| `G-05` | `FR-04`、`FR-05` | Period Result | `AC-07`、`AC-08`、`AC-10` |
| `G-06` | `FR-03` | Scoped Lookup | `AC-05` |
| `G-07` | `FR-06` | Integrations Auth、RBAC Exclusion | `AC-02`、`AC-03`、`AC-12` |
| `G-08` | `FR-02` | Router、Route Contract Test | `AC-01`、`AC-13` |
| 安全审计 | `FR-06` | Audit、Response | `AC-14` |
| 可部署性 | 全部 | Wire、测试与文档 | `AC-15` |

## 10. 评审记录

| 版本 | 状态 | 变更 |
|---|---|---|
| v0.1 | 已修订 | 初版；确认 integrations 鉴权、email 精确匹配和四类 404 |
| v0.2 | 已批准 | 增加 URL 冲突检查；明确无登录、无 RBAC且只受外部 Token 系统约束 |
| v1.0 | Final — user approved | 将批准内容整理为最终实施基线并交付 |
