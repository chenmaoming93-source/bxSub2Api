# 按天 Token 消耗量外部查询接口实施 Plan

**状态：Draft — 待评审**  
**版本：v0.1**  
**日期：2026-08-12**  
**变更摘要：** 新增仅受 integrations 外部 Bearer Token 保护的历史按天查询接口：入参为分组名、API Key 值、起止日期（日粒度），返回该时间范围内每一天该 API Key 在该分组下的 `total_tokens` 消耗量。响应语义三态：**未配置统计项**（`dimension_configured=false` + message，区别于空列表）、**已配置但无数据**（`days=[]`）、**已配置且有数据**（`days` 只含有数据的天，缺失天不出现、不补 0）。数据来自可配置 Token 统计的 MySQL 聚合表（`token_stat_aggregates`），完全复用现有统计投影与通用查询引擎 `ProjectionAdminService.QueryUsage`，不新增数据库表。

## 1. 引言

### 1.1 背景与问题

项目已有两套可复用能力：

1. **可配置 Token 统计（动态统计投影）**：管理员在前端配置「统计项（Projection）」——维度组合 + 指标 `total_tokens`；网关请求结束后由异步管道按启用中的投影聚合，Redis 保存当前周期实时值，MySQL 同步保存日/周/月聚合行（`token_stat_aggregates`）。通用查询引擎 `ProjectionAdminService.QueryUsage`（`backend/internal/service/tokenstat/query.go`）已支持按投影 + 指标 + 周期粒度 + 时间范围 + 维度筛选查询 MySQL 聚合，返回每周期一行、汇总与同步完整性元数据。
2. **外部查询接口**：`POST /api/v1/integrations/token-usage/query`（`external_token_usage.go` 等）实现了"分组名 / API Key 值 → 内部 ID"的业务解析，但**只查当前自然日/周/月**（Redis 实时值），不支持历史时间范围。

现需新增一个查询接口：输入**分组名、时间范围（日粒度）、API Key 值**，返回该时间范围内每一天、该 API Key 调用该分组的 Token 消耗量。该需求正是"复用可配置统计 + 通用查询引擎 + 既有维度解析"的典型场景，新增代码量很小。

### 1.2 用户与目标

使用者是持有 integrations 外部 Access Token 的外部系统。

- `G-01`：提供稳定的历史按天 Token 消耗外部查询接口。
- `G-02`：入参使用业务名称（分组名、API Key 值），服务端安全解析为内部统计维度（`group_id`、`api_key_id`）。
- `G-03`：只读取可配置 Token 统计的 MySQL 聚合数据，不扫描 Redis、不读取旧统计 namespace。
- `G-04`：时间范围按统计时区切分为自然日，返回范围内**有数据的天**的消耗量；范围内完全无数据时返回空列表（区别于"未配置"）。
- `G-05`：明确区分「统计维度未配置」「范围在投影启用之前」「已同步但无数据」「数据仍在最终一致同步中」。
- `G-06`：分组或 API Key 不存在、API Key 不属于该分组时返回明确错误；响应与日志不泄露 API Key 密钥。
- `G-07`：接口只受 integrations 外部 Bearer Token 鉴权约束，不受登录、JWT、管理员身份或 RBAC 约束。
- `G-08`：新增 HTTP Method + URL 组合不得与已有路由重复。

### 1.3 成功标准

- 有效外部 Token 在没有登录 Cookie、JWT、管理员身份和 RBAC 权限时仍可调用。
- `group_name` + `api_key` 解析为 `group_id` + `api_key_id`；API Key 不属于该分组时按不匹配处理，不泄露归属。
- 未找到可用统计投影时返回 `dimension_configured=false` + `message="统计维度未配置"`，**不得**以空列表伪装（区别于"已配置但无数据"）。
- 找到可用投影但范围内完全无聚合数据时返回 `days=[]`。
- 找到可用投影且有数据时，`days` 只含有数据的天（升序），缺失天不出现、不补 0。
- MySQL 同步滞后时返回 `complete=false` 与 `last_synced_at`，由调用方自行判断时效。
- 最终路由在 Gin 路由表中只注册一次，且中间件边界符合要求；原有 integrations 接口回归通过。

### 1.4 范围

包含外部 HTTP 接口、路由冲突检查、外部 Token 鉴权与限流、分组与 API Key 解析及归属校验、活动投影发现、按天查询编排与三态响应、响应与审计、测试、Wire 依赖注入与 RBAC 路由覆盖登记。

### 1.5 非目标

- 不查询 Redis 实时数据（当天未同步部分由 `complete` 标记，不合并）；
- 不新增或自动创建统计投影、维度、指标；
- 不修改统计写入流程、不新增数据库表或迁移；
- 不提供周/月/自定义周期粒度（本接口固定日粒度，与需求一致）；
- **不补零**：`days` 只含有数据的天，缺失天不出现；
- 不返回 API Key 密钥；
- 不新增登录、JWT、Admin Auth 或 RBAC 权限点；
- 不做 CSV 导出（管理端通用查询已具备）。

## 2. 假设与决策

### 2.1 已确认约束

| 编号 | 约束 |
|---|---|
| `C-01` | 复用 `/api/v1/integrations` 的外部 Bearer Token、开关、限流和 hardening 中间件 |
| `C-02` | `group_name` 按分组名称匹配（`FindGroupByName`，已删除分组不可见） |
| `C-03` | `api_key` 按 `api_keys.key` 明文精确匹配（数据库唯一约束），且必须属于目标分组 |
| `C-04` | 不要求 `username` 与 `route_alias`；API Key 归属校验只检查 `key.GroupID == group.ID`（`GroupID` 为 nil 视为不匹配，沿用现有 `ErrAPIKeyMismatch` 语义） |
| `C-05` | 历史数据只读 MySQL `token_stat_aggregates`，经 `ProjectionAdminService.QueryUsage` 查询 |
| `C-06` | 日桶边界按 `gateway.dynamic_token_statistics.timezone`（默认 `Asia/Shanghai`）划分 |
| `C-07` | 接口不受登录和 RBAC 约束，只受 integrations 外部 Token 系统约束 |
| `C-08` | 新增 URL 不得与已有路由重复 |

### 2.2 路由决策

候选接口：

```http
POST /api/v1/integrations/token-usage/query/group-api-key/daily
POST /api/v1/integrations/token-usage/query/group-api-key/daily/csv   （CSV 下载，逐日补 0）
```

实施前必须检查静态注册代码和 Gin 最终路由表，以 `HTTP Method + normalized absolute path` 判断冲突。当前 `integrations.go` 已注册 `/api-keys/getOrCreate`、`/model-routes/list`、`/model-routes/list-attributes`、`/token-usage/query`，候选组合未被占用（RBAC_ENDPOINT_INVENTORY.md 第 539 行残留的 `token-usage/user-group-model/daily` 为历史陈旧登记，当前路由表不存在，实施时一并修正该清单）。路径中 `group-api-key` 为维度组合标识：历史按天查询统一采用 `/token-usage/query/{dims}/daily` 模式（CSV 版追加 `/csv` 后缀），未来其他维度组合（如 `user-model`、`route-alias`）按同一模式扩展。若冲突，不得覆盖或删除已有路由，应选择并评审未占用的等价 integrations 路径，并以自动化契约测试固定最终 URL。

### 2.3 鉴权与 RBAC 决策

目标路由只挂载：

1. 全局日志、CORS、安全 Header 等基础中间件；
2. integrations 外部 Token 鉴权（`ExternalProvisioningAuth`）；
3. integrations 限流与 hardening。

不得挂载 JWT Auth、Session 登录校验、Admin Auth、Gateway API Key Auth 或 RBAC `RequirePermission`。目标路由在 RBAC 路由覆盖清单中登记为已知排除项，理由为"由 integrations Bearer Token 保护"；排除登记不得引入 RBAC 判断。`external_api_key_provisioning.enabled=false` 或 `access_token` 为空时，integrations 组整体不可达，接口自然返回 404（与现有接口一致）。

### 2.4 统计投影决策

**只接受**状态为 `ACTIVE`、规范化维度签名**精确等于** `api_key_id,group_id`、指标含 `total_tokens` 的投影。这是强制要求：本接口的统计口径就是"分组 × API Key"，投影必须与口径完全一致。

`DRAFT`、`PUBLISHED`、`DISABLED` 投影，以及维度是 `{api_key_id, group_id}` **超集**的投影（例如 `user_id,api_key_id,group_id,route_alias`、`api_key_id,group_id,upstream_model`）一律视为未配置。若不存在可用投影：HTTP 200，`dimension_configured=false`，`days` 为空数组，`message` 提示"统计维度未配置"。不得由多个投影临时推导结果，**不得回退到超集投影计算**。

> 为什么不接受超集投影（精度问题）：异步管道对每个事件按投影逐一判断，投影要求的任一维度在该事件中缺失，该事件就被**整体跳过**该投影（`async_pipeline.go` `operations()`：`if !complete { continue }`）。而 `submitDynamicTokenUsage` 只把"有值"的维度放进事件——`route_alias` 为空、`user_id` 缺失、`upstream_model` 未记录等都很常见。于是含额外维度的超集投影会**漏记**这些事件，用它求"该 key 该分组总量"会系统性**少算**。精确二维投影只依赖 `api_key_id` 与 `group_id`（key 归属分组时必然存在），不会漏。因此超集回退不可接受，必须强制配置精确投影。

### 2.5 数据来源与状态语义

- 数据源：`token_stat_aggregates`（MySQL），最终一致（同步间隔 `sync_interval_minutes`，默认 5 分钟）。
- **未配置投影**（情况 1）：`dimension_configured=false`、`days=[]`、`message="统计维度未配置"` —— 明确标识，区别于"已配置但无数据"。
- **已配置但范围内完全无数据**（情况 2）：`dimension_configured=true`、`days=[]`。
- **已配置且有数据**：`days` 只含有数据的天（升序），缺失天不出现、不补 0。
- 同步滞后：`complete=false`；`last_synced_at` 返回最近聚合行同步时间；`projection_enabled_at` 返回投影启用时间（早于该时间的日期不会有数据，天然不会出现在 `days` 中，不视为错误）。
- MySQL 查询失败：HTTP 500（`INTERNAL_ERROR`），不返回部分结果。

### 2.6 归属校验与 404 顺序

- 固定解析顺序：`group_name` → `api_key`，首次失败即返回对应错误；
- 分组不存在 → `GROUP_NOT_FOUND`（404）；API Key 值不存在或已删除 → `API_KEY_NOT_FOUND`（404）；
- API Key 存在但 `key.GroupID == nil || *key.GroupID != group.ID` → `API_KEY_MISMATCH`（400），不泄露该 Key 实际归属；
- 分组或 Key 已删除（`deleted_at` 非空）一律按不存在处理。

### 2.7 实现阶段需验证的既有事实

- `FindGroupByName` / `FindAPIKeyByKey` 的精确匹配与删除过滤语义（沿用 `external_token_usage_repo.go`）；
- `TokenStatAggregate.period_start` 的存储时区语义（ent `time.Time` 按 UTC 瞬时存储）与 `QueryUsage` 的 `PeriodStartGTE/LT` 过滤（以统计时区构造的起止瞬时直接可用）；
- 统计时区（`gateway.dynamic_token_statistics.timezone`）与 shard/sync 配置取值；
- RBAC 路由覆盖清单中 integrations 排除项的登记格式。

这些事实必须沿用现有产品规则，不由本接口发明新规则。

## 3. 概念功能设计

### 3.1 `FR-01` 外部按天查询

请求示例：

```json
{
  "group_name": "public",
  "api_key": "sk-ldap-openai-key-0123456789",
  "start_date": "2026-07-01",
  "end_date": "2026-07-31"
}
```

流程为外部 Token 鉴权与限流 → 请求校验 → 对象解析与归属校验 → 活动投影发现 → 按天查询 → 三态响应 → 审计。请求链路不存在登录、JWT 或 RBAC 判断。

### 3.2 `FR-02` 路由唯一性

注册前搜索所有路由代码并检查测试 Router 的最终路由表；Method 与绝对路径组合必须唯一。Gin 的重复路由错误不得被捕获或忽略。

### 3.3 `FR-03` 业务对象解析与归属校验

| 外部输入 | 匹配目标 | 统计维度 | 校验 |
|---|---|---|---|
| `group_name` | 分组名称（未删除） | `group_id` | 必填，≤100 字符 |
| `api_key` | `api_keys.key` 明文（唯一约束） | `api_key_id` | 必填，≤128 字符；`key.GroupID == group.ID` |

### 3.4 `FR-04` 统计投影发现

从活动投影快照（`ProjectionAdminService.ActiveProjections()`）中查找维度签名精确等于 `api_key_id,group_id` 且指标含 `total_tokens` 的投影（见 2.4，强制精确、不接收超集）。快照未初始化属于内部错误，不得误报为未配置。

### 3.5 `FR-05` 按天查询

- 将 `start_date` / `end_date`（`YYYY-MM-DD`）在统计时区解释为 `[start 00:00:00, end+1日 00:00:00)`；
- 调用 `ProjectionAdminService.QueryUsage`：`PeriodType=PeriodDay`、`Filters={api_key_id: key.ID, group_id: group.ID}`、`GroupBy=[]`、`Sort=time_asc`、`PageSize=1000`（单投影单 key+分组下每日至多一行，行数 ≤ 天数 ≤ 366，不会触顶）；
- 将查询结果行映射为 `days[]`：**只含有数据的天**（`period_start` 按统计时区格式化为 `YYYY-MM-DD`），按日期升序；结果行为空则 `days=[]`。缺失天不出现、不补 0。

### 3.6 `FR-06` 安全边界与审计

只接受 `Authorization: Bearer <integration-access-token>`。功能关闭返回 404，Token 缺失或错误返回 401。审计记录内部资源 ID（group_id、api_key_id、投影 ID）、结果、原因、来源 IP 和耗时；响应 echo 中 API Key 必须脱敏（复用 `maskAPIKey`），不记录 Authorization Header、外部 Token 或 API Key 密钥。

## 4. 详细技术设计

### 4.1 组件与数据流

```text
外部系统
  -> 全局基础中间件
  -> Integrations Token Auth / Rate Limit / Hardening
  -> ExternalTokenUsageHandler.DailyQuery（新方法）
  -> ExternalTokenUsageService.QueryDailyUsage（新方法）
      -> 业务 Repository：FindGroupByName / FindAPIKeyByKey（既有）
      -> ProjectionAdminService.ActiveProjections()（既有，选投影）
      -> ProjectionAdminService.QueryUsage（既有，查 MySQL 聚合）
  -> MySQL token_stat_aggregates
```

JWT、Session、Admin Auth、Gateway API Key Auth 和 RBAC 不在该链路中。

### 4.2 组件职责

- `backend/internal/handler/external_token_usage_handler.go`：新增 `DailyQuery` 方法；绑定、校验、错误映射、响应脱敏、审计日志。
- `backend/internal/service/external_token_usage.go`：新增 `QueryDailyUsage(ctx, ExternalDailyUsageInput) (ExternalDailyUsageResult, error)`；对象解析、归属校验、投影发现、范围切日、调用 QueryUsage、三态响应映射。
- `backend/internal/service/external_token_usage.go` 结构体：新增字段保存 `*tokenstat.ProjectionAdminService`（wire 的 `ProvideExternalTokenUsageService` 已接收该依赖，仅需在 Configure 阶段存入具体类型；现有 `projections` 接口字段保留给 Redis 当前周期查询用）。
- `backend/internal/server/routes/integrations.go`：注册 `POST /token-usage/query/group-api-key/daily`。
- `backend/internal/repository/external_token_usage_repo.go`：无需改动（`FindGroupByName`、`FindAPIKeyByKey` 已满足）。
- RBAC 路由覆盖清单（`RBAC_ENDPOINT_INVENTORY.md`）：登记新路由为 integrations Bearer Token 排除项，并顺手修正 539 行陈旧登记。

### 4.3 API 契约

```http
POST /api/v1/integrations/token-usage/query/group-api-key/daily
Authorization: Bearer <integration-access-token>
Content-Type: application/json
```

请求（四个字段必填；`start_date`、`end_date` 为 `YYYY-MM-DD`；`end_date >= start_date`；跨度 ≤ 366 天，与 `QueryUsage.maxQueryDays` 一致；超出返回 400）：

```json
{
  "group_name": "public",
  "api_key": "sk-ldap-openai-key-0123456789",
  "start_date": "2026-07-01",
  "end_date": "2026-07-31"
}
```

成功响应（有数据）：

```json
{
  "success": true,
  "data": {
    "query": {
      "group_name": "public",
      "api_key": "sk-ld****6789",
      "start_date": "2026-07-01",
      "end_date": "2026-07-31"
    },
    "resolved_dimensions": { "group_id": 12, "api_key_id": 880 },
    "projection_id": 7,
    "metric": "total_tokens",
    "timezone": "Asia/Shanghai",
    "dimension_configured": true,
    "projection_enabled_at": "2026-07-01T00:00:00+08:00",
    "last_synced_at": "2026-07-31T23:55:00+08:00",
    "complete": true,
    "message": "",
    "days": [
      { "date": "2026-07-01", "total_tokens": 12500 },
      { "date": "2026-07-03", "total_tokens": 9800 }
    ]
  }
}
```

响应三态（调用方判读口径）：

| 状态 | `dimension_configured` | `days` | `message` |
|---|---|---|---|
| 未配置统计项（情况 1） | `false` | `[]` | `统计维度未配置` |
| 已配置但范围内无数据（情况 2） | `true` | `[]` | 空（或 `范围内暂无已同步数据`） |
| 已配置且有数据 | `true` | 只含有数据的天，升序 | 空 |

`days` 为空列表**只**出现在"已配置但无数据"时；"未配置"必须以 `dimension_configured=false` + message 标识，不得伪装成空列表。

错误契约：

| HTTP | 错误码 | 条件 |
|---:|---|---|
| 400 | `INVALID_REQUEST` | JSON、必填字段、日期格式或范围（>366 天 / end < start）无效 |
| 400 | `API_KEY_MISMATCH` | API Key 值存在，但不属于目标分组（或无分组） |
| 401 | `INVALID_ACCESS_TOKEN` | 外部 Token 缺失或错误 |
| 404 | `NOT_FOUND` | integrations 功能关闭 |
| 404 | `GROUP_NOT_FOUND` | 分组不存在或已删除 |
| 404 | `API_KEY_NOT_FOUND` | API Key 值不存在或已删除 |
| 429 | 既有限流错误码 | integrations 限流触发 |
| 500 | `INTERNAL_ERROR` | MySQL 查询失败或其他非预期错误 |

接口只读且天然幂等，不需要 Idempotency Key 或分页。

#### 4.3.1 CSV 下载接口

`POST /api/v1/integrations/token-usage/query/group-api-key/daily/csv`：入参与 JSON 接口完全一致（`group_name`、`api_key`、`start_date`、`end_date`），鉴权与限流相同，返回 `text/csv; charset=utf-8` 附件（`Content-Disposition: attachment`）。

- 数据内容与 JSON 接口一致（date + total_tokens），但**逐日补 0**：范围内每一天一行，无记录的天 `total_tokens=0`（JSON 接口保持"缺失天不出现"，两者语义由各自文档明确）。
- 未配置精确投影时返回 HTTP 409 `STATISTICS_NOT_CONFIGURED`（JSON 接口以 `dimension_configured=false` 表达；CSV 无法用数据行表达该状态，故以错误区分，避免输出误导性的全 0 下载）。
- 实现：Service 新增 `QueryDailyUsageFilled`（与 `QueryDailyUsage` 共享 `runDailyQuery` 公共核心，仅在结果映射时逐日补 0）；Handler 新增 `DailyQueryCSV`；CSV 序列化复用 `encoding/csv`（先例见管理端 `QueryUsage` 的 `format=csv` 分支）。

### 4.4 查询契约（复用 QueryUsage）

```text
input := tokenstat.UsageQueryInput{
  ProjectionID: selectedProjection.ID,
  MetricCode:   tokenstat.MetricTotalTokens,
  PeriodType:   tokenstat.PeriodDay,
  Start:        dateAt(startDate, location),   // start_date 00:00:00（统计时区）
  End:          dateAt(endDate+1日, location), // 开区间
  Filters:      { api_key_id: Int64Value(key.ID), group_id: Int64Value(group.ID) },
  GroupBy:      nil,   // 按周期聚合即可
  Sort:         "time_asc",
  Page:         1, PageSize: 1000,
}
```

说明：

- `validateUsageQuery` 对日粒度 + 自定义范围合法；起止瞬时按统计时区构造，与聚合行 `period_start` 的存储瞬时一致（ent `time.Time` 存 UTC 瞬时，比较不受影响）。
- `GroupBy=nil` 时 `queryGroupKey` 为空 key，同一 `period_start` 的多个聚合行自动求和（精确二维投影下同一 key+分组每天至多一行，实际不会出现多行，求和仅为兜底）。
- 结果映射：`days` 直接由查询结果行生成（`date` 用 `period_start.In(location).Format("2006-01-02")`），**不做补零**；结果行为空则 `days=[]`。三态语义见 2.5 与 4.3。
- 范围保护：日期跨度 > 366 天在 handler 层直接拒绝（400），不进入 QueryUsage。

### 4.5 数据、事务与并发

本功能不新增数据表或迁移，使用既有分组、API Key、统计投影和 `token_stat_aggregates`。查询只读，不开启数据库事务。SQL 由 Ent 生成、参数化，筛选值只允许注册表内的维度（`aggregateDimensionPredicate` 白名单），不存在注入面。API Key 密钥属于高敏感凭证，禁止读取、返回或记录（日志仅记录 `api_key_id`）。

### 4.6 可靠性、可观测性和性能

- MySQL 不可用时返回 500，不重试、不回退 Redis、不伪装成零值。
- 建议指标：`integration_token_usage_daily_queries_total{result}`、`integration_token_usage_daily_query_duration_seconds`、`integration_token_usage_daily_missing_projection_total`。
- 告警关注 MySQL 错误率、`complete=false` 持续、401/429 异常升高。
- 单请求最多 366 行结果 + 常数级投影匹配开销；基础设施健康时接口自身 P95 目标不超过 200ms。

### 4.7 配置与运维要求（含页面操作）

**（1）后端配置**（开发 `backend/config/config.yaml`；部署 `deploy/config.example.yaml`）：

```yaml
gateway:
  dynamic_token_statistics:
    enabled: true                 # 统计运行时开关；关闭则不再产生新聚合（历史数据仍可查）
    timezone: Asia/Shanghai       # 日桶边界时区（默认值即 Asia/Shanghai）

external_api_key_provisioning:
  enabled: true                   # 外部接口开关；false 时新接口返回 404
  access_token: "至少32字节且不含空白的随机串"
```

**（2）管理员页面配置统计投影（前置条件，投影不启用就采不到数据）**：

1. 管理员登录，侧边栏进入 **「可配置 Token 统计」**（`/admin/token-statistics`，需 `token_usage.manage` 权限）。
2. 切到 **「统计项」** tab → 点 **「新建统计项」**：
   - **名称**：如 `分组-APIKey 日统计`；
   - **维度**：勾选 `API Key` 与 `分组` 两项（**必须且只能这两项**；含额外维度的投影不会被本接口采用，见 2.4 精度说明）；
   - **指标**：页面固定 `total_tokens`，无需修改；
   - 点 **「保存草稿」**。
3. 在统计项列表中对该项依次点 **「发布」** → **「启用」**，状态变为「运行中 / ACTIVE」。
4. （可选）切到 **「查询」** tab 手动核对：选该统计项、周期「日」、起止日期、筛选条件填 API Key 与分组的 **ID**（页面此处填数值 ID，非 key 明文/分组名），比对按天趋势。

**（3）数据口径注意**：

- 数据从投影**启用时刻**起采集，`projection_enabled_at` 之前的日期无数据、不会出现在 `days`（接口不补零）；
- MySQL 聚合按 `sync_interval_minutes`（默认 5 分钟）同步，`complete=false` 表示可能不完整；
- 启用后的投影不可直接改维度，调整需停用后新建；
- 调用方需持有 `external_api_key_provisioning.access_token`，以 `Authorization: Bearer <token>` 调用。

**（4）发布与回滚**：先启用投影并确认 MySQL 聚合产生数据，再发布路由。回滚只移除新路由与方法，无数据库或 Redis 数据回滚，不影响现有 integrations 接口与统计写入。

## 5. 伪代码

```text
function registerRoute(router):
  candidate = POST /api/v1/integrations/token-usage/query/group-api-key/daily
  verify method+path does not conflict（源码 + 最终 Gin 路由表）
  integrationGroup.use(externalTokenAuth, integrationHardening)
  integrationGroup.POST(candidate, dailyQueryHandler)
  // 不挂载 login/JWT/admin/gateway-api-key/RBAC 中间件

function queryDailyUsage(context, input):
  validate: group_name/api_key/start_date/end_date 必填、trim、长度
  validate: 日期为 YYYY-MM-DD；start <= end；跨度 <= 366 天
  location = statsTimezone()

  group = findGroupByName(input.group_name)          // 不存在 -> GROUP_NOT_FOUND
  apiKey = findAPIKeyByKey(input.api_key)            // 不存在 -> API_KEY_NOT_FOUND
  if apiKey.groupID == nil or apiKey.groupID != group.id:
    return API_KEY_MISMATCH

  projection = findDailyProjection(activeProjections):   // 见 2.4
    candidates = ACTIVE 且维度签名精确 == api_key_id,group_id 且含 total_tokens
    // 超集投影一律不算（管道会漏记缺维度的事件，见 2.4 精度说明）
  if projection == nil:
    return dimension_configured=false, days=[], message="统计维度未配置"  // 情况 1

  rangeStart = dateAt(input.start_date, location)    // 00:00:00
  rangeEnd   = dateAt(input.end_date + 1, location)  // 开区间
  result = projectionAdmin.queryUsage({
    projectionID, metric=total_tokens, periodType=D,
    start=rangeStart, end=rangeEnd,
    filters={api_key_id: apiKey.id, group_id: group.id},
    groupBy=[], sort=time_asc, page=1, pageSize=1000
  })                                                 // MySQL 失败 -> INTERNAL_ERROR

  days = []
  for row in result.rows:                            // 只含有数据的天，不补零
    days.append({ date: row.periodStart.in(location).format("YYYY-MM-DD"),
                  total_tokens: row.value })
  sort days by date asc                              // 无数据时 days 保持 []（情况 2）

  return { resolved_dimensions, projection_id, metric, timezone,
           dimension_configured=true, projection_enabled_at,
           last_synced_at, complete, days }
```

## 6. 验证策略与验收标准

### 6.1 测试范围

- 路由：候选 URL 盘点、Method+Path 唯一、最终路由只注册一次、原 integrations 路由回归、RBAC 清单登记。
- 安全边界：无外部 Token 401；功能关闭 404；有效外部 Token 在无 Cookie、JWT、管理员身份、RBAC 权限时成功；JWT 或 Gateway API Key 不能替代外部 Token。
- Service：分组/Key 解析与 404 顺序、Key 不属分组 → `API_KEY_MISMATCH`、投影选择规则（**强制精确签名，超集视为未配置**）、**三态语义**（未配置 vs 已配置无数据 vs 有数据）、`complete/last_synced_at/projection_enabled_at` 透传、范围超限拒绝。
- Handler/契约：字段与日期校验、错误映射、固定响应结构、API Key 脱敏、敏感信息不泄露。
- 构建：Wire 依赖图、RBAC Route Coverage 和相关后端测试通过。

### 6.2 验收标准

| 编号 | 验收条件 |
|---|---|
| `AC-01` | 最终 Method+URL 不冲突且只注册一次 |
| `AC-02` | 路由只位于 integrations 组，只接受外部 Bearer Token |
| `AC-03` | 无登录、JWT、管理员身份和 RBAC 权限仍可凭有效外部 Token 调用 |
| `AC-04` | 分组/API Key 不存在分别返回明确 404；Key 不属于分组返回 `API_KEY_MISMATCH` |
| `AC-05` | 仅 ACTIVE 且维度签名**精确等于** `api_key_id,group_id` 的投影可用；超集/部分/非 ACTIVE 投影一律视为未配置 |
| `AC-06` | 未配置投影（情况 1）返回 `dimension_configured=false`、`days=[]`、`message="统计维度未配置"`，HTTP 200；不得伪装成空列表 |
| `AC-07` | 已配置但范围内完全无数据（情况 2）返回 `dimension_configured=true`、`days=[]` |
| `AC-08` | 有数据时 `days` 只含有数据的天（升序），缺失天不出现、不补 0；`date` 为统计时区日期 |
| `AC-09` | 超集投影存在但无精确投影时返回 `dimension_configured=false`（不回退计算） |
| `AC-10` | 日期跨度 > 366 天或格式非法返回 400 |
| `AC-11` | `complete`、`last_synced_at`、`projection_enabled_at` 正确透传 |
| `AC-12` | MySQL 失败返回 500，不返回部分结果或零值 |
| `AC-13` | RBAC Coverage 通过，目标接口只登记外部 Token 排除而不执行 RBAC |
| `AC-14` | 原有 integrations 接口回归测试通过 |
| `AC-15` | 响应和日志不泄露 API Key 密钥或外部 Token |
| `AC-16` | Wire 构建及相关后端测试全部通过 |

## 7. 实施顺序与拆分指导

1. 路由、安全边界与既有数据规则盘点：确认最终 URL 无冲突、RBAC 清单登记格式、时区与查询语义。
2. Service：`ExternalTokenUsageService` 增加 `QueryDailyUsage` 与投影选择、日范围切分、三态响应映射；Wire 注入 `*tokenstat.ProjectionAdminService`。
3. Handler：`DailyQuery` 方法（校验、错误映射、脱敏、审计）与 `integrations.go` 路由注册。
4. 测试：Service / Handler / 路由契约 / 安全边界 / RBAC 覆盖，全量后端回归。
5. 运维文档：更新 `docs/token-statistics-operation-guide.md` 与 `RBAC_ENDPOINT_INVENTORY.md`，修正陈旧登记。

推荐按上述边界拆成一个约一个专注开发日的 MVP；Service 查询语义与外部安全边界必须分别具备独立验证路径。

## 8. 风险与开放项

| 风险 | 影响 | 缓解 |
|---|---|---|
| 候选 URL 被间接注册 | 高 | 同时检查源码和最终 Gin 路由表；契约测试固定 |
| 路由误挂 JWT/RBAC | 高 | 只扩展 integrations 组并测试中间件边界 |
| RBAC 排除导致无鉴权 | 严重 | 强制断言无外部 Token 返回 401 |
| 时区边界导致日桶错位 | 高 | 起止瞬时与日期标签统一按统计时区构造；过滤与标签同源 |
| 超集投影被误当作可用口径 | 中 | 强制精确签名匹配（`findDailyProjection`），文档与响应 `projection_id` 明确口径 |
| MySQL 聚合与 Redis 实时值口径不一致 | 中 | 明确 `complete=false` 语义，不合并 Redis |
| 活动投影快照未初始化 | 高 | 启动初始化；未初始化返回内部错误 |
| 空列表被误读为"未配置" | 中 | 三态语义显式化：`dimension_configured=false` + message 标识未配置；响应携带 `projection_enabled_at` 说明数据起点 |

开放决策（实施首阶段以仓库事实确认）：最终 URL 冲突检查结果；`TokenStatAggregate.period_start` 存储时区与过滤语义的测试固化；RBAC 清单登记格式。

## 9. 追踪矩阵

| 目标 | 功能 | 技术组件 | 验收 |
|---|---|---|---|
| `G-01`、`G-02` | `FR-01`、`FR-03` | Handler、Service、业务 Repository | `AC-04` |
| `G-03`、`G-04` | `FR-04`、`FR-05` | Projection 快照、QueryUsage、三态映射 | `AC-05`～`AC-11` |
| `G-05` | `FR-05` | Result 元数据 | `AC-06`、`AC-07`、`AC-11` |
| `G-06` | `FR-03`、`FR-06` | Scoped Lookup、maskAPIKey | `AC-04`、`AC-15` |
| `G-07` | `FR-06` | Integrations Auth、RBAC Exclusion | `AC-02`、`AC-03`、`AC-13` |
| `G-08` | `FR-02` | Router、Route Contract Test | `AC-01`、`AC-14` |
| 可部署性 | 全部 | Wire、测试与文档 | `AC-16` |

## 10. 评审记录

| 版本 | 状态 | 变更 |
|---|---|---|
| v0.1 | Draft | 初版；确认复用可配置统计 + `QueryUsage` + 既有维度解析；外部 Bearer 鉴权；只校验 Key 归属分组；缺日补零；超集投影回退规则 |
| v0.2 | Draft | 按用户确认修订响应三态语义：未配置统计项 → `dimension_configured=false` + `message="统计维度未配置"`（不得伪装成空列表）；已配置但无数据 → `days=[]`；有数据 → `days` 只含有数据的天（缺失天不出现、不补 0）；同步更新 API 契约、伪代码、验收标准与追踪矩阵 |
| v1.0 | Implemented | 已实现：`ExternalTokenUsageService.QueryDailyUsage` + `findDailyProjection`（`internal/service/external_token_usage.go`，Wire 注入 `ConfigureHistoryQuery`）；Handler `DailyQuery`（`internal/handler/external_token_usage_handler.go`）；路由 `POST /api/v1/integrations/token-usage/query/group-api-key/daily`（`internal/server/routes/integrations.go`，路径采用 `/token-usage/query/{dims}/daily` 维度组合模式）；新增 Service/Handler/路由契约测试；全量后端回归除 4 个改动前即失败的环境性用例（Redis 不可用、RBAC 目录种子、模型路由校验）外全部通过；更新 `RBAC_ENDPOINT_INVENTORY.md` 与 `docs/token-statistics-operation-guide.md` |
| v1.1 | Implemented | 按用户要求改为**强制精确投影**：仅接受 ACTIVE 且维度签名精确等于 `api_key_id,group_id` 的投影，删除超集回退（`pickDailyProjection` → `findDailyProjection`）；超集/部分维度一律返回「统计维度未配置」；同步更新 Service 测试、2.4 投影决策（含精度问题说明）、FR-04、伪代码、AC-05/AC-09 与运维文档 |
| v1.2 | Implemented | 新增 **CSV 下载接口** `POST /token-usage/query/group-api-key/daily/csv`：入参与 JSON 一致，返回 `date,total_tokens` 附件并**逐日补 0**；未配置投影返回 409 `STATISTICS_NOT_CONFIGURED`；Service 抽取 `runDailyQuery` 公共核心 + `QueryDailyUsageFilled`，Handler 新增 `DailyQueryCSV`，路由/契约/补零/未配置测试齐备；更新 RBAC 清单、运维文档（§10.1）与本节（§2.2、§4.3.1） |
