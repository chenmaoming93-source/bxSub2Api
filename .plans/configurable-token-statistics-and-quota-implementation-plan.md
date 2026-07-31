# 可配置多维 Token 统计与限额系统实施 Plan

**状态：Final — user approved**  
**版本：v0.1**  
**日期：2026-07-30**  
**目标项目：bxSub2Api**  
**变更摘要：** 首次形成独立的可配置多维 Token 统计、限额、历史查询及配套文档实施基线。

## 1. 引言

### 1.1 背景

平台需要按照模型调用产生的业务属性统计 Token 消耗，并在自然日、自然周、自然月范围内执行高频限额查询。

首期可用维度包括：

- 用户；
- API Key；
- 分组；
- 路由别名；
- 模型账号；
- 上游模型。

系统还需要允许后续增加新的维度和指标。管理员应当能够通过页面组合已经注册的维度和指标，创建新的统计投影和限额，无需重启服务。

### 1.2 新旧体系强制隔离

本 Plan 描述的新体系必须与项目中过往三套固定统计和限额逻辑完全隔离：

- 模型统计与限额；
- 用户＋模型统计与限额；
- 分组＋路由别名＋上游模型统计与限额。

新体系不得：

- 调用旧统计或旧限额 Service、Repository、Cache、Reader、Repairer；
- 使用旧 `TokenStatisticsType`、固定统计编码器或固定日限额接口；
- 读写旧 Redis Key；
- 读写旧日统计表或旧日限额配置表；
- 通过旧 API 或旧前端页面提供兼容；
- 进行新旧双写、历史迁移、数据回填或限额配置迁移；
- 将旧逻辑作为新体系失败时的兜底。

新体系必须拥有独立的：

- Domain 模型和服务接口；
- Redis 命名空间；
- MySQL 表；
- API 路径；
- 前端页面和状态管理；
- 配置项；
- 测试和运维指标。

旧三套统计与限额逻辑将由另一份专门的移除 Plan 负责删除。本文只定义隔离边界和新体系能力，不展开旧逻辑删除步骤。

### 1.3 目标用户

- 平台管理员：配置统计投影、限额并查询历史数据；
- 平台运维人员：监控同步、封账、Redis 清理和故障状态；
- 模型网关：产生用量事件并执行限额检查；
- 后续开发者或 AI：按照开发指引增加维度、指标和请求类型支持。

### 1.4 目标

- `G-01`：建立代码注册、页面组合的动态维度体系。
- `G-02`：建立可扩展指标体系，首期支持 `total_tokens`。
- `G-03`：管理员配置已有维度和指标的组合时无需部署或重启。
- `G-04`：按 Asia/Shanghai 自然日、周、月实时累计。
- `G-05`：请求主流程不等待统计 Redis 写入或 MySQL 同步。
- `G-06`：Redis 负责当前周期实时统计和限额，MySQL 保存长期历史。
- `G-07`：已结束周期完成最终同步后删除 Redis 数据。
- `G-08`：统计投影可以独立存在，限额必须依赖统计投影。
- `G-09`：限额采用简单后结算，允许少量并发超额。
- `G-10`：提供仅管理员可用的通用多维查询页面。
- `G-11`：提供新增维度/指标开发指引和管理员操作文档。
- `G-12`：保证新体系与旧固定体系不存在运行时和数据层交集。

### 1.5 成功标准

- 已注册维度和指标可在页面自由组合并秒级生效。
- 单次统计提交在请求线程中只执行有界、非阻塞内存操作。
- 统计队列满、Redis 写失败、MySQL 同步失败均不改变模型响应。
- 限额检查使用 Redis 精确读取，不执行全局扫描或临时大范围聚合。
- Redis 和 MySQL 重复同步不会重复增加 Token。
- 未确认落入 MySQL 的旧周期 Redis 数据不会删除。
- 通用查询页可查询任意已经采集的投影。
- 新模块不引用或读写旧三套统计限额体系。
- 两份配套文档可以指导后续人员完成扩展和日常操作。

### 1.6 非目标

- 不保存新的固定六维底账。
- 不预生成所有可能的维度组合。
- 不回填新投影启用前的数据。
- 不迁移旧统计历史或旧限额配置。
- 不实现滚动 24 小时、7 天或 30 天窗口。
- 不实现额度预占和严格零超额。
- 不向普通用户开放通用查询页。
- 不通过页面创建代码未注册的新维度或指标。
- 不在本 Plan 中删除旧三套统计和限额代码。

## 2. 假设与决策

### 2.1 已确认约束

| 主题 | 决策 |
|---|---|
| 用户规模 | 约 1000 |
| API Key 规模 | 每用户约 10 个 |
| API Key 与分组 | 每个 Key 仅绑定一个分组 |
| 路由规模 | 每个分组约 10 个路由别名 |
| 账号模型组合 | 每路由通常 1 个，最多个位数 |
| 周期 | 自然日、自然周、自然月 |
| 当前指标 | `total_tokens` |
| 实时存储 | Redis |
| 历史存储 | MySQL |
| 新投影历史 | 从启用时开始，不回填 |
| 限额方式 | 请求前检查，完成后按实际 Token 结算 |
| 并发超额 | 允许少量发生 |
| 查询页面 | 仅管理员 |
| 新旧关系 | 完全隔离，不迁移、不双写 |

### 2.2 周期定义

- 自然日：本地时间 00:00 至次日 00:00。
- 自然周：周一 00:00 至下周一 00:00。
- 自然月：每月 1 日 00:00 至次月 1 日 00:00。
- 区间均为左闭右开。
- 统计归属时间使用模型调用完成并获得最终用量的时间。
- 跨周期请求计入完成时所在周期。

### 2.3 可用性决策

统计和限额基础设施采用 fail-open：

- 用量事件构造失败：记录日志和指标，不修改模型响应。
- 本地统计队列已满：丢弃统计事件并告警，不阻塞请求。
- Redis 累计失败：有限重试后丢弃并告警，不影响模型响应。
- 限额 Redis 查询失败或超时：允许请求继续并告警。
- 只有成功读取有效计数并确认超限时，才拒绝请求或排除候选账号。

### 2.4 一致性决策

- Redis 是当前周期实时统计来源。
- MySQL 是历史查询和恢复来源。
- 两者最终一致。
- 同步写入 Redis 当前绝对值和单调版本，而不是无保护增量。
- Redis AOF、复制和备份保护尚未同步的数据。
- 极端故障下允许丢失未成功进入 Redis 或尚未持久化的数据。

### 2.5 非阻塞假设

- 既有 `usage_logs`、计费和余额逻辑不属于本体系，不因本 Plan 改变。
- 通用查询页默认读取 MySQL，并展示最后同步时间。
- 限额始终读取 Redis。
- MySQL 聚合历史首期长期保留，后续再依据数据量制定归档策略。

## 3. 概念功能设计

### 3.1 维度注册

首期注册：

```text
user_id
api_key_id
group_id
route_alias
account_id
upstream_model
```

每个维度定义包括：

- 稳定代码；
- 展示名称；
- 值类型；
- 规范化规则；
- 固定排序序号；
- 是否允许为空；
- 敏感级别；
- 可选值查询能力。

管理员只能组合已经注册的维度。新增代码未采集的维度需要开发、测试和发布；发布后即可参与任意新投影。

### 3.2 指标注册

首期注册：

```text
total_tokens
```

指标定义包括：

- 稳定代码；
- 展示名称；
- 单位；
- 整数值类型；
- 聚合方式；
- 是否允许配置限额；
- 适用请求类型；
- 口径说明和版本。

未来可以增加：

```text
input_tokens
output_tokens
cache_creation_tokens
cache_read_tokens
request_count
actual_cost_micros
```

已经发布的指标代码不得改变含义。口径变化应注册新指标代码。

### 3.3 统计投影

统计投影表示系统需要持续维护的一种维度组合和指标集合。

示例：

```text
名称：用户模型用量
维度：user_id、upstream_model
指标：total_tokens
```

投影状态：

```text
DRAFT → PUBLISHED → ACTIVE → DISABLED
```

- `DRAFT`：仅保存配置。
- `PUBLISHED`：已下发并开始采集。
- `ACTIVE`：所有实例完成配置加载，可以被强制限额引用。
- `DISABLED`：停止新统计，历史继续保留。

相同维度集合只允许存在一个有效投影，维度选择顺序不影响投影身份。

### 3.4 统一用量事件

模型调用完成后，新统计模块独立构造：

```json
{
  "dimensions": {
    "user_id": 1001,
    "api_key_id": 8001,
    "group_id": 10,
    "route_alias": "fast",
    "account_id": 301,
    "upstream_model": "gpt-5"
  },
  "metrics": {
    "total_tokens": 1200
  },
  "occurred_at": "2026-07-30T15:20:00+08:00"
}
```

事件构造器只从模型调用结果读取事实，不调用旧固定统计或旧限额代码。

### 3.5 非阻塞 Redis 累计

每个网关实例维护有界内存队列和异步统计 Worker。

请求主流程：

1. 构造事件；
2. 非阻塞尝试入队；
3. 记录入队成功或失败指标；
4. 立即结束统计提交。

异步 Worker：

1. 批量读取事件；
2. 获取本地缓存的活跃投影；
3. 从事件提取投影需要的维度和指标；
4. 计算日、周、月；
5. 通过一次 Lua 调用原子更新 Redis；
6. 失败时执行有限、短时重试；
7. 重试耗尽后记录丢失事件并结束。

### 3.6 限额管理

限额规则包含：

```text
统计投影
具体维度值
指标
周期
限额值
OBSERVE/ENFORCE
生效时间
失效时间
```

投影与限额关系：

- 投影可以没有限额；
- 一个投影可被多条限额复用；
- 限额必须引用一个投影及其已启用指标；
- 创建限额时若投影不存在，可以自动创建投影；
- 新投影当前周期数据不完整时，强制限额默认从下一个完整周期生效。

限额检查分为：

- 调度前：用户、API Key、分组、路由别名等已知维度；
- 账号选择后：模型账号、上游模型等调度后维度。

### 3.7 Redis 到 MySQL 同步

Lua 累计时同时：

- 增加统计值；
- 增加单调版本；
- 将统计身份加入脏集合。

同步任务：

1. 原子轮换当前脏集合；
2. 批量读取绝对值和版本；
3. 批量 UPSERT MySQL；
4. 仅当新版本不小于已存版本时更新；
5. 成功后移除处理记录；
6. 失败项重新进入待同步集合。

重复同步不会重复增加 Token。

### 3.8 周期封账和 Redis 清理

周期状态：

```text
OPEN → CLOSING → FINAL_SYNC → PERSISTED → DELETED
```

旧周期只有在以下条件全部满足后才删除：

- 周期已经结束；
- 异步队列不存在该周期待处理事件；
- 最大 Redis 重试窗口结束；
- 脏集合不存在该周期记录；
- MySQL 版本不低于 Redis 最终版本；
- 没有同步任务正在处理该周期。

满足条件后使用 `UNLINK` 删除计数、版本和同步辅助 Key。

TTL 仅作为孤儿 Key 兜底，不得代替持久化确认。

### 3.9 管理员通用查询

查询流程：

1. 选择投影；
2. 选择该投影已启用指标；
3. 选择时间范围和日/周/月粒度；
4. 根据投影动态显示筛选字段；
5. 选择投影内维度进行分组；
6. 查看汇总、趋势、排行榜和分页明细；
7. 查看统计起点、周期完整性和最后同步时间；
8. 导出 CSV。

页面不得：

- 查询投影中不存在的维度或指标；
- 跨投影拼接结果；
- 混合日、周、月重复相加；
- 将投影启用前的数据表示为完整历史；
- 将任意前端字段直接拼接为 SQL。

### 3.10 文档交付

必须交付：

```text
docs/token-statistics-development-guide.md
docs/token-statistics-operation-guide.md
```

开发指引覆盖：

- 系统术语和架构；
- 如何增加维度；
- 如何增加指标；
- 如何让新请求类型提供统计字段；
- 指标语义和版本约束；
- 编码兼容规则；
- 必测场景；
- 后续 AI 开发禁止事项。

操作文档覆盖：

- 创建、发布、停用投影；
- 创建、观察、启用和停用限额；
- 使用通用查询页面；
- 判断数据是否完整；
- 查看同步、封账和清理状态；
- 故障排查与恢复。

## 4. 详细技术设计

### 4.1 模块边界

新模块应使用独立、清晰的命名，例如：

```text
service/tokenstat
repository/tokenstat
handler/admin/tokenstat
```

如果现有项目分层不采用子包，则使用统一的 `DynamicTokenStat*` 类型前缀。

禁止新模块依赖旧固定统计和限额的具体类型。

### 4.2 Redis 命名空间

计数 Hash：

```text
sub2api:dynamic_token_stats:v1:
{period_type}:{period_start}:{projection_id}:{shard}
```

版本 Hash：

```text
sub2api:dynamic_token_stats_ver:v1:
{period_type}:{period_start}:{projection_id}:{shard}
```

Hash field：

```text
{dimension_hash}:{metric_code}
```

脏集合：

```text
sub2api:dynamic_token_stats_dirty:v1:current
sub2api:dynamic_token_stats_dirty:v1:processing:{batch_id}
```

配置：

```text
sub2api:dynamic_token_stats_config:v1:version
sub2api:dynamic_token_stats_config:v1:active
```

不得读写旧 `sub2api:token_stats:*` 命名空间。

### 4.3 维度编码

投影维度按注册表顺序规范排序。编码必须包含：

- 编码版本；
- 维度代码；
- 值类型；
- 明确的字符串长度；
- 规范化后的值。

编码结果计算 128 位哈希，用于 Redis 和 MySQL 精确定位。MySQL 同时保存原始维度 JSON。

若发现哈希相同但规范内容不同，必须拒绝覆盖并触发严重告警。

### 4.4 Lua 原子累计

一次 Lua 调用负责：

- 多投影；
- 多指标；
- 日、周、月；
- 计数递增；
- 版本递增；
- 脏集合标记；
- 设置兜底 TTL。

单次事件允许处理的投影和操作数量需要设上限，并对投影数量和写放大进行监控。

### 4.5 异步队列

配置项包括：

```text
queue_capacity
worker_count
batch_size
flush_interval_ms
redis_timeout_ms
redis_retry_count
```

队列满时不等待，直接丢弃并增加：

```text
token_statistics_events_dropped_total{reason="queue_full"}
```

服务优雅关闭时允许短时间排空，但不得无限阻止进程退出。

### 4.6 MySQL 表

#### 4.6.1 `token_stat_projections`

| 字段 | 类型 | Null | 约束 | 含义 |
|---|---|---:|---|---|
| id | BIGINT UNSIGNED | 否 | PK | 投影 ID |
| name | VARCHAR(128) | 否 |  | 名称 |
| dimension_codes | JSON | 否 |  | 有序维度代码 |
| dimension_signature | VARCHAR(512) | 否 | UNIQUE | 维度组合签名 |
| status | VARCHAR(20) | 否 | INDEX | 状态 |
| config_version | BIGINT UNSIGNED | 否 |  | 配置版本 |
| published_at | DATETIME(6) | 是 |  | 发布时间 |
| enabled_at | DATETIME(6) | 是 |  | 统计开始时间 |
| disabled_at | DATETIME(6) | 是 |  | 停用时间 |
| created_by | BIGINT UNSIGNED | 否 |  | 创建人 |
| created_at | DATETIME(6) | 否 |  | 创建时间 |
| updated_at | DATETIME(6) | 否 |  | 更新时间 |

停用后不删除，用于解释历史数据。

#### 4.6.2 `token_stat_projection_metrics`

保存投影启用的指标及启停时间。

唯一约束：

```text
UNIQUE(projection_id, metric_code)
```

#### 4.6.3 `token_stat_aggregates`

主要字段：

```text
period_type
period_start
period_end
projection_id
dimension_hash
dimension_values
metric_code
metric_value
source_version
user_id
api_key_id
group_id
route_alias
account_id
upstream_model
last_synced_at
created_at
updated_at
```

唯一约束：

```text
UNIQUE(
  period_type,
  period_start,
  projection_id,
  dimension_hash,
  metric_code
)
```

通用索引：

```text
(projection_id, metric_code, period_type, period_start)
```

常用维度可使用冗余列和针对性索引；不得为所有组合预建联合索引。

#### 4.6.4 `token_stat_quota_rules`

主要字段：

```text
name
projection_id
dimension_hash
dimension_values
metric_code
period_type
limit_value
enforcement_mode
status
effective_from
effective_until
created_by
created_at
updated_at
```

#### 4.6.5 `token_stat_period_states`

保存周期的：

```text
period_type
period_start
period_end
state
final_sync_version
closed_at
persisted_at
deleted_at
last_error
```

唯一约束：

```text
UNIQUE(period_type, period_start)
```

### 4.7 API

维度和指标：

```text
GET /admin/token-statistics/dimensions
GET /admin/token-statistics/metrics
```

投影：

```text
GET    /admin/token-statistics/projections
POST   /admin/token-statistics/projections
GET    /admin/token-statistics/projections/:id
PUT    /admin/token-statistics/projections/:id
POST   /admin/token-statistics/projections/:id/publish
POST   /admin/token-statistics/projections/:id/disable
```

限额：

```text
GET    /admin/token-statistics/quotas
POST   /admin/token-statistics/quotas
PUT    /admin/token-statistics/quotas/:id
POST   /admin/token-statistics/quotas/:id/enable
POST   /admin/token-statistics/quotas/:id/disable
```

查询：

```text
POST /admin/token-statistics/query
```

运行状态：

```text
GET  /admin/token-statistics/status
POST /admin/token-statistics/sync
POST /admin/token-statistics/periods/:type/:start/finalize
```

### 4.8 RBAC

复用：

```text
token_usage.read
token_quota.read
token_quota.update
```

新增：

```text
token_usage.manage
```

权限语义：

- `token_usage.read`：查看注册信息、投影和多维统计；
- `token_usage.manage`：创建、发布、停用投影及执行同步、封账操作；
- `token_quota.read`：查看通用 Token 限额；
- `token_quota.update`：创建和修改通用 Token 限额。

### 4.9 前端页面

统一入口：

```text
/admin/token-statistics
```

包含：

1. 多维查询；
2. 统计投影；
3. 限额规则；
4. 同步状态。

页面只调用新 `/admin/token-statistics/*` API，不调用任何旧统计或限额接口。

### 4.10 安全

- 不保存 API Key 明文，只保存 `api_key_id`。
- 所有管理接口要求管理员身份和 RBAC。
- 维度、指标、排序和分组字段必须使用后端白名单。
- 查询使用参数化 SQL。
- 投影和限额变更写入现有审计体系。
- CSV 导出继承查询权限和范围限制。
- 限制查询日期范围、分页大小和分组结果数。

### 4.11 可观测性

关键指标：

```text
token_statistics_events_enqueued_total
token_statistics_events_dropped_total
token_statistics_queue_depth
token_statistics_redis_write_duration
token_statistics_redis_write_failures_total
token_statistics_sync_lag_seconds
token_statistics_dirty_items
token_statistics_mysql_sync_failures_total
token_statistics_period_finalize_failures_total
token_statistics_quota_check_failures_total
token_statistics_quota_rejections_total
token_statistics_config_version
```

关键告警：

- 队列持续高水位或出现事件丢弃；
- Redis 写入持续失败；
- 同步延迟超过阈值；
- 已结束周期长时间无法删除；
- Redis/MySQL 抽样版本不一致；
- 配置版本未被全部实例加载；
- Redis 内存超过安全水位。

### 4.12 配置

使用新体系独立配置段，避免复用旧统计配置语义：

```yaml
gateway:
  dynamic_token_statistics:
    enabled: true
    timezone: Asia/Shanghai
    async_queue_capacity: 10000
    worker_count: 4
    batch_size: 100
    flush_interval_ms: 50
    redis_timeout_ms: 100
    redis_retry_count: 2
    shard_count: 256
    sync_interval_minutes: 1
    mysql_batch_size: 500
    sync_retry_count: 3
    finalize_check_interval_minutes: 1
    orphan_ttl_days: 7
```

数值为候选默认值，最终通过压测确定。

## 5. 关键流程伪代码

### 5.1 提交用量事件

```text
function finishModelCall(callResult):
  response = buildClientResponse(callResult)

  event = safelyBuildDynamicTokenStatEvent(callResult)
  if event invalid:
    record build failure
    return response

  if statisticsQueue.tryEnqueue(event) == false:
    record dropped event

  return response
```

### 5.2 异步累计

```text
function statisticsWorker():
  while running:
    events = read bounded batch
    projections = activeProjectionCache.current()

    for event in events:
      operations = []

      for projection in projections:
        dimensions = extract projection dimensions from event
        if required value missing:
          record extraction failure
          continue

        for enabled metric:
          if event does not provide metric:
            record metric missing
            continue

          for period in natural day, week, month:
            append exact increment

      execute one Redis Lua call with bounded retry
      if retry exhausted:
        record dropped event
```

### 5.3 限额检查

```text
function checkQuota(knownDimensions, phase):
  rules = match active rules
  if no rules:
    return allow

  try:
    values = Redis exact pipeline reads
  catch timeout or Redis error:
    record infrastructure failure
    return allow

  for each rule:
    if used >= limit:
      if OBSERVE:
        record violation
      if ENFORCE:
        reject request or exclude candidate

  return allow
```

### 5.4 发布投影

```text
function publishProjection(admin, projectionId):
  authorize token_usage.manage
  validate all dimensions and metrics are registered
  persist PUBLISHED state and increment config version
  publish Redis config notification
  wait for bounded instance propagation confirmation
  mark ACTIVE
```

### 5.5 创建限额

```text
function createQuota(admin, request):
  authorize token_quota.update
  validate metric is quota eligible

  projection = find exact dimension signature
  if projection missing:
    create and publish projection
    quota status = PENDING_PROJECTION

  if current period is incomplete:
    default effective time = next natural period start

  persist quota and publish config version
```

### 5.6 同步

```text
function syncDirtyStatistics():
  acquire distributed lease
  atomically rotate dirty set

  while processing set has items:
    read Redis values and versions
    begin MySQL transaction
    upsert only when incoming version >= stored version
    commit
    remove successful items

  release lease
```

### 5.7 封账和删除

```text
function finalizeExpiredPeriod(period):
  mark CLOSING
  wait for async writer watermark
  run final synchronization

  if dirty items remain:
    keep FINAL_SYNC and retry later

  verify MySQL final versions
  mark PERSISTED
  UNLINK all Redis keys for period
  mark DELETED
```

### 5.8 通用查询

```text
function queryStatistics(admin, request):
  authorize token_usage.read
  load projection
  validate metric belongs to projection
  validate filters and groupBy belong to projection
  validate date range, paging and sorting
  query MySQL using parameterized SQL
  calculate completeness from projection enabledAt
  return data, summary, pagination, completeness, lastSyncedAt
```

## 6. 验证策略

### 6.1 单元测试

- 维度和指标注册；
- 投影规范排序和去重；
- 维度编码和哈希稳定性；
- 日、周、月边界；
- Lua 计数和版本；
- 限额规则匹配；
- fail-open；
- 同步版本比较；
- 周期状态机；
- 查询白名单；
- 数据完整性判断。

### 6.2 集成测试

- 模型调用完成后异步写 Redis；
- 队列满时模型响应不受影响；
- Redis 不可用时模型响应不受影响；
- Redis 到 MySQL 幂等同步；
- 同步中持续写入不丢失脏标记；
- MySQL 失败时旧周期不删除；
- 最终同步后删除旧周期；
- 多实例配置刷新；
- 新投影不产生启用前数据；
- 限额自动创建依赖投影；
- 被限额引用的投影不能直接停用。

### 6.3 新旧隔离测试

通过依赖测试、源码审计或架构测试保证新模块：

- 不引用旧 `TokenStatisticsType`；
- 不调用旧固定日配额接口；
- 不读写旧 Redis 前缀；
- 不读写旧统计或限额表；
- 不调用旧固定报表 Repository；
- 不调用旧前端 API；
- 不将旧体系作为 fallback。

### 6.4 端到端测试

1. 管理员创建并发布投影；
2. 发起模型调用；
3. Redis 产生日、周、月统计；
4. 同步到 MySQL；
5. 通用查询页显示数据；
6. 创建观察限额；
7. 切换强制限额；
8. 超限时产生限制；
9. Redis 故障时请求 fail-open；
10. 周期结束后安全清理 Redis。

### 6.5 性能测试

覆盖：

- 1000 用户；
- 10000 API Key；
- 10～50 个活跃投影；
- 百万级 Redis Hash field；
- 峰值请求 QPS 的两倍；
- 多实例写入；
- 同步积压恢复；
- 通用查询大日期范围和排行榜。

候选目标：

- 请求线程统计提交 P99 小于 1ms；
- 限额 Redis 正常读取 P99 小于 5ms；
- 正常峰值下队列无丢弃；
- 同步长期处理能力高于统计产生速度；
- MySQL 正常时旧周期在配置窗口内完成删除。

### 6.6 安全测试

- 非管理员访问；
- 缺少 `token_usage.manage` 修改投影；
- 缺少 `token_quota.update` 修改限额；
- 非法维度、指标、排序和分组字段；
- 超大日期范围和分页；
- CSV 导出越权；
- API Key 明文泄漏。

### 6.7 文档验收

- 独立开发者或 AI 按开发指引完成一个示例指标接入演练。
- 管理员按操作手册完成投影创建、查询、限额和停用流程。

### 6.8 验收标准

| 编号 | 验收条件 |
|---|---|
| AC-01 | 已注册维度可页面组合，无需重启 |
| AC-02 | 已注册指标可配置给投影，无需重启 |
| AC-03 | 新维度或指标发布后可参与任意新投影 |
| AC-04 | 一次调用正确累计日、周、月 |
| AC-05 | 统计提交不等待 Redis 或 MySQL |
| AC-06 | 统计故障不改变模型响应 |
| AC-07 | 限额基础设施失败时 fail-open |
| AC-08 | 确认超限时产生限制效果 |
| AC-09 | 投影可在没有限额时独立运行 |
| AC-10 | 限额可自动创建缺失的依赖投影 |
| AC-11 | Redis/MySQL 重复同步不重复计数 |
| AC-12 | 未完成最终同步的旧周期不会删除 |
| AC-13 | 完成最终同步后旧周期 Redis 数据被删除 |
| AC-14 | 管理员可在一个页面查询全部已采集投影 |
| AC-15 | 页面不能查询未采集组合或启用前历史 |
| AC-16 | 通用查询仅具备 `token_usage.read` 的管理员可访问 |
| AC-17 | 投影管理要求 `token_usage.manage` |
| AC-18 | 新体系不引用、读写或兜底到旧三套统计限额逻辑 |
| AC-19 | 两份配套文档完成并通过演练 |

## 7. 实施顺序与拆分指南

### 阶段一：注册体系和数据模型

产出：

- 维度注册表；
- 指标注册表；
- 统一用量事件；
- 新 MySQL 表；
- 规范编码；
- RBAC。

### 阶段二：异步 Redis 统计

产出：

- 有界队列；
- Worker；
- Lua；
- 日、周、月 Key；
- 版本和脏集合；
- fail-open 监控。

### 阶段三：同步、封账和清理

产出：

- 绝对值版本 UPSERT；
- 分布式同步任务；
- 周期状态机；
- 最终同步；
- Redis 安全删除。

### 阶段四：投影管理

产出：

- 投影 API；
- 配置发布；
- 多实例刷新；
- 管理页面；
- 容量展示。

### 阶段五：通用限额

产出：

- 限额 API；
- 两阶段检查；
- 观察和强制模式；
- 自动创建依赖投影。

### 阶段六：通用查询

产出：

- 通用查询 API；
- 动态筛选和分组；
- 趋势、排行榜、分页；
- CSV 导出；
- 数据完整性提示。

### 阶段七：文档和运维

产出：

- 开发指引；
- 操作手册；
- 监控面板；
- 告警；
- 故障演练；
- 性能报告。

### 阶段八：新体系验收

产出：

- 新旧隔离审计；
- 完整验收报告；
- 管理员完成必要的新投影和限额配置。

旧逻辑删除不属于本阶段，由专门的旧体系移除 Plan 执行。

## 8. 发布与回滚

新体系使用独立功能开关：

```text
dynamic_token_statistics_enabled
dynamic_token_statistics_mysql_sync_enabled
dynamic_token_statistics_query_enabled
dynamic_token_statistics_quota_observe_enabled
dynamic_token_statistics_quota_enforce_enabled
```

上线顺序：

1. 新表和权限；
2. 新异步统计；
3. 新 MySQL 同步；
4. 投影管理；
5. 通用查询；
6. 限额观察；
7. 限额强制；
8. 完整验收。

本 Plan 不使用旧体系双写做灰度。新体系出现问题时：

- 关闭新限额强制；
- 关闭新统计采集；
- 保留已写入的新表；
- 不回退或调用旧体系作为兜底。

旧体系是否仍在项目中运行，由专门删除 Plan 的执行时点决定，但新体系代码始终与其隔离。

## 9. 风险与开放项

| 风险 | 概率/影响 | 缓解 |
|---|---|---|
| 异步队列满导致统计丢失 | 中/中 | 容量、批处理、监控、告警 |
| Redis 故障导致统计缺失 | 中/高 | AOF、主从、短重试、告警 |
| 异步写入造成限额短时滞后 | 高/低 | 接受少量超额并监控 |
| 投影过多造成写放大 | 中/中 | 容量估算、软上限、页面提示 |
| 旧周期过早删除 | 低/高 | 状态机、最终版本校验 |
| MySQL 同步积压 | 中/高 | 脏集合、批量写、积压告警 |
| 指标口径被修改 | 中/高 | 稳定代码，新口径新指标 |
| 通用查询产生低效 SQL | 中/中 | 白名单、范围限制、针对性索引 |
| 新旧代码意外交叉 | 中/高 | 独立命名空间、架构测试和审计 |
| 新投影当期数据不完整 | 高/中 | 展示启用时间，下周期强制 |

非阻塞开放项：

- 队列、批量、分片和超时默认值通过压测确定。
- MySQL 是否按月分区，根据真实增长量决定。
- CSV 首期可限制同步导出行数，后续再升级异步导出。

## 10. 追踪矩阵

| 目标 | 功能模块 | 技术组件 | 验收 |
|---|---|---|---|
| G-01 | 维度注册、投影 | 注册表、投影表 | AC-01、03 |
| G-02 | 指标注册 | 指标注册表 | AC-02、03 |
| G-03 | 配置发布 | Redis 版本、Pub/Sub | AC-01、02 |
| G-04 | 三周期统计 | 周期计算、Lua | AC-04 |
| G-05 | 非阻塞写入 | 有界队列、Worker | AC-05、06 |
| G-06 | 实时与历史存储 | Redis、MySQL | AC-11 |
| G-07 | 周期清理 | 状态机、最终同步 | AC-12、13 |
| G-08 | 投影与限额 | 投影和规则表 | AC-09、10 |
| G-09 | 简单后结算 | 限额检查器 | AC-07、08 |
| G-10 | 通用查询 | 查询 API 和页面 | AC-14～17 |
| G-11 | 配套文档 | 两份文档 | AC-19 |
| G-12 | 新旧隔离 | 模块、Key、表和测试 | AC-18 |

## 11. 评审记录

| 版本 | 状态 | 内容 |
|---|---|---|
| v0.1 | Final — user approved | 独立的新可配置多维统计与限额实施基线；明确与旧三套固定逻辑完全隔离，旧逻辑由专门 Plan 删除；用户已于 2026-07-30 审核通过 |

本 Plan 已经用户审核通过，可供后续 MVP 拆分和开发使用。
