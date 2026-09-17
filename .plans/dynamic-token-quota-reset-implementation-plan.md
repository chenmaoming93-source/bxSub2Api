# 可配置 Token 限额重置能力实现 Plan

> **状态：Final — user approved（最终版，用户已审核通过）**

## 1. 文档元数据

| 项目 | 内容 |
|---|---|
| 文档名称 | 可配置 Token 限额重置能力实现 Plan |
| 版本 | v1.0 |
| 日期 | 2026-09-09 |
| 状态 | 最终版，用户已审核通过 |
| 变更摘要 | 明确统计身份发现采用单次 dirty identity 快照且不上锁、不复查；接受同步 processing 窗口遗漏；批量重置取消数量超限报错，改为分页、分批、逐条处理 |

---

## 2. 背景与目标

### 2.1 背景

现有可配置 Token 统计与限额体系使用同一份累计值完成：

1. Token 使用量统计；
2. 当前周期限额检查；
3. Redis 到 MySQL 的定时同步；
4. 历史统计及报表查询。

如果直接将 Redis 统计值归零，后续同步会把错误值写入 MySQL，破坏真实 Token 使用记录。

### 2.2 目标

- 支持重置指定维度组合的当前周期限额判定用量。
- 重置后重新获得完整周期额度。
- 不修改真实累计量、历史统计及 MySQL 同步结果。
- 支持日、周、月自然周期。
- 输入部分维度时，重置所有包含这些维度且值相等的适用额度记录。
- 提供管理员页面和外部接口。
- 尽量减少新增 Redis 存储。
- 采用符合内部系统定位的最终一致性策略。

### 2.3 核心公式

```text
raw_usage       = 现有 Redis 真实累计量
reset_baseline  = Redis 重置快照；不存在时为 0
effective_usage = max(0, raw_usage - reset_baseline)
```

限额判断：

```text
effective_usage >= limit_value
```

---

## 3. 范围

### 3.1 当前范围

- 新增独立 Redis 重置快照命名空间。
- 调整限额检查的用量读取逻辑。
- 新增管理员重置接口。
- 新增外部 integrations 重置接口。
- 在 Token 统计管理员页面增加重置界面。
- 支持部分维度匹配及多条快照创建。
- 返回 `RESET`、`PARTIAL_RESET`、`NO_QUOTA`、`NO_USAGE`。
- 增加日志、监控和测试。

### 3.2 非目标

- 不修改或归零 `token_stat_aggregates.metric_value`。
- 不修改现有 Redis 统计 Key、field 和版本 Key。
- 不把重置快照同步到 MySQL。
- 不新增重置快照数据库表。
- 不保证 Redis 数据完全丢失后的快照恢复。
- 不保证一次请求内多个条目整体原子。
- 不对统计身份发现加同步锁。
- 不重复读取 dirty identity。
- 不扫描同步任务的 processing set。
- 不因为匹配条目较多而直接拒绝整个重置请求。
- 不实现外部接口幂等。
- 不按模型请求完成时间严格划分重置边界。

---

## 4. 已确认决策

| ID | 决策 |
|---|---|
| DEC-01 | 重置快照只保存在 Redis。 |
| DEC-02 | 允许任意时刻、多次重置。 |
| DEC-03 | 以重置时已经写入 Redis 的统计用量为边界。 |
| DEC-04 | 外部接口不要求幂等。 |
| DEC-05 | 管理员页面允许自行选择维度并输入值。 |
| DEC-06 | 外部接口沿用现有 integrations Bearer Token。 |
| DEC-07 | 没有适用限额时不创建快照，返回 `NO_QUOTA`。 |
| DEC-08 | 有适用限额但当前周期没有用量时返回 `NO_USAGE`。 |
| DEC-09 | 输入部分维度时，匹配维度集合为输入维度超集的统计条目。 |
| DEC-10 | dirty identity 只读取一次，不加锁、不复查。 |
| DEC-11 | dirty identity 读取后发生的变化不影响本次已确定的候选集合。 |
| DEC-12 | 接受同步 processing 窗口、新增记录等原因导致的少量遗漏。 |
| DEC-13 | 限额及候选记录完成一次验证后，不在写快照前二次确认。 |
| DEC-14 | 多条快照逐条处理，允许部分成功，不执行回滚。 |
| DEC-15 | 不设置因匹配数量过多而直接失败的业务限制，改用分页和分批处理。 |

---

## 5. 匹配语义

输入：

```json
{
  "dimension_values": {
    "user_id": 42,
    "group_id": 7
  },
  "metric_code": "total_tokens",
  "period_type": "D"
}
```

应匹配：

```json
{"user_id":42,"group_id":7}
```

以及：

```json
{
  "user_id":42,
  "group_id":7,
  "upstream_model":"deepseekv4pro"
}
```

不匹配：

```json
{
  "group_id":7,
  "upstream_model":"deepseekv4pro"
}
```

因为它不包含输入中的 `user_id`。

以下月统计也不匹配：

```text
period_type = M
```

---

## 6. 现有数据结构与身份发现

### 6.1 MySQL 聚合记录

当前真实统计同步到：

```text
token_stat_aggregates
```

关键字段：

| 字段 | 用途 |
|---|---|
| `period_type` | `D`、`W`、`M` |
| `period_start` | 自然周期开始时间 |
| `period_end` | 自然周期结束时间 |
| `projection_id` | 统计投影 ID |
| `dimension_hash` | 完整维度组合 hash |
| `dimension_values` | 完整维度值 JSON |
| `metric_code` | 指标代码 |
| `metric_value` | 上次同步时的真实累计量 |
| `source_version` | Redis 同步版本 |
| `user_id` 等 | 高频查询使用的冗余维度字段 |

示例：

```text
period_type      = D
period_start     = 2026-09-09 00:00:00
projection_id    = 12
dimension_hash   = 0xABCDEF...
dimension_values = {
    "user_id": 42,
    "group_id": 7,
    "upstream_model": "deepseekv4pro"
}
metric_code      = total_tokens
metric_value     = 90000
```

MySQL 中的 `metric_value` 可能落后于 Redis，因此只用于发现统计身份，不能作为快照值。

### 6.2 Redis 实时统计

统计 Key：

```text
sub2api:dynamic_token_stats:v1:
{period_type}:{period_start}:{projection_id}:{shard}
```

示例：

```text
sub2api:dynamic_token_stats:v1:D:20260909T000000+0800:12:3
```

field：

```text
{dimension_hash_hex}:{metric_code}
```

示例：

```text
abcdef0123456789abcdef0123456789:total_tokens
```

value 是当前实时累计量：

```text
100000
```

### 6.3 Dirty identity

当前 dirty set：

```text
sub2api:dynamic_token_stats_dirty:v1:current
```

member 中包含完整身份：

```json
{
  "period_type": "D",
  "period_start": "2026-09-09T00:00:00+08:00",
  "period_end": "2026-09-10T00:00:00+08:00",
  "projection_id": 12,
  "shard": 3,
  "field": "abcdef0123456789abcdef0123456789:total_tokens",
  "dimension_hash": "abcdef0123456789abcdef0123456789",
  "dimension_values": {
    "user_id": 42,
    "group_id": 7,
    "upstream_model": "deepseekv4pro"
  },
  "metric_code": "total_tokens"
}
```

Redis 统计 Hash 中只有 hash，无法反推出完整维度，因此需要通过 MySQL `dimension_values` 和 dirty identity 找到具体统计身份。

### 6.4 身份发现顺序

#### 第一步：读取一次 dirty identity

首先读取一次：

```text
sub2api:dynamic_token_stats_dirty:v1:current
```

并立即在内存中过滤：

- 当前周期；
- 请求指标；
- 候选投影；
- 包含全部输入维度；
- 输入维度值相等。

一旦 identity 被读取进本次候选集合，即使随后同步任务将其从 dirty set 删除，也不影响本次处理。

#### 第二步：查询 MySQL

然后分页查询 `token_stat_aggregates`：

```text
period_type = 请求周期
period_start = 当前自然周期开始
metric_code = 请求指标
projection_id 属于候选投影
输入维度值相等
```

MySQL 负责补充以前已同步、目前已经不在 dirty set 中的统计身份。

#### 第三步：去重

使用以下身份去重：

```text
period_type
period_start
projection_id
dimension_hash
metric_code
```

#### 第四步：验证限额

统计身份只有在存在对应的有效限额时才进入待重置集合。

有效限额需要满足：

- 状态为 `ENABLED`；
- 当前时间处于生效区间；
- `metric_code` 相同；
- `period_type` 相同；
- projection 相同；
- 请求提供的维度值与规则具体值一致，或者规则值为 wildcard。

`OBSERVE` 和 `ENFORCE` 均属于有效限额。

完成这次验证后，待重置集合即确定。写入快照前不再：

- 重新查询限额；
- 重新读取 dirty set；
- 检查记录是否刚刚同步；
- 检查是否出现新记录；
- 获取同步锁。

### 6.5 明确接受的遗漏窗口

#### 新记录遗漏

第一次读取 dirty identity 后才出现的新统计身份可能不会被本次重置。

处理方式：

```text
接受遗漏；必要时再次点击重置。
```

#### Processing 窗口遗漏

同步任务会将：

```text
dynamic_token_stats_dirty:v1:current
```

切换为：

```text
dynamic_token_stats_dirty:v1:processing:{token}
```

如果统计身份已经进入 processing、但尚未写入 MySQL，则本次可能：

- 无法从 current dirty set 找到；
- 也无法从 MySQL 找到。

该情况同样被接受，不扫描 processing set，也不上锁。

#### 已经读取后又被同步

这种情况不会造成遗漏：

1. identity 已经保存在本次请求内存中；
2. 同步只会更新 MySQL 并删除 dirty member；
3. 原统计 Redis field 仍然存在；
4. 服务仍可根据已保存的 Key、field 和 hash 创建快照。

---

## 7. 重置快照设计

### 7.1 Key

新增：

```text
sub2api:dynamic_token_quota_reset:v1:
{period_type}:{period_start}:{projection_id}:{shard}
```

示例：

```text
sub2api:dynamic_token_quota_reset:v1:D:20260909T000000+0800:12:3
```

### 7.2 Field

与统计 field 完全一致：

```text
abcdef0123456789abcdef0123456789:total_tokens
```

### 7.3 Value

value 为执行该条重置时的实时统计累计量：

```text
100000
```

### 7.4 TTL

```text
period.End + orphanTTL
```

快照不会：

- 写入 dirty set；
- 增加统计 version；
- 参与 MySQL 同步；
- 修改真实统计值。

---

## 8. 批量执行方式

### 8.1 总体原则

不因为匹配到多个条目而直接报错。

如果一次匹配到 10、100 或更多条目，则：

1. 分页读取 MySQL 身份；
2. 分批处理候选条目；
3. 每一批使用 Redis pipeline 降低网络往返；
4. pipeline 中每个条目执行独立的小型 Lua；
5. 某一条失败不阻止后续条目继续处理；
6. 不回滚已经成功的条目。

### 8.2 单条 Lua

每条快照执行：

```text
raw = HGET statistics_key field

if raw exists:
    HSET reset_key field raw
    EXPIREAT reset_key expire_at
    return RESET
else:
    return NO_USAGE
```

单条 Lua 只是保证该条记录的“读取当前值并建立快照”不会被并发累加插入中间，不代表整个请求需要原子。

### 8.3 Pipeline 含义

Pipeline 只用于减少网络请求。

例如存在 100 条待重置记录：

```text
不使用 pipeline：
应用与 Redis 往返约 100 次

使用每批 20 条的 pipeline：
应用与 Redis 往返约 5 次
```

每条记录仍然独立成功或失败，不会因为其中一条失败而撤销其他记录。

### 8.4 大批量处理

不设置 `TOO_MANY_MATCHES` 业务错误。

实现应采用：

- dirty set 使用 `SSCAN` 或等效渐进读取；
- MySQL 使用稳定分页；
- Redis 使用固定大小 pipeline；
- 控制单批大小，而不是限制总匹配数量。

如果处理中发生超时或连接失败：

- 已成功的记录保留；
- 尽量继续处理当前可处理记录；
- 最终能返回响应时返回 `PARTIAL_RESET`；
- 调用方可再次执行相同重置。

---

## 9. 限额读取

限额检查读取：

```text
原统计 Key/field
重置快照 Key/field
```

计算：

```text
max(0, raw_usage - reset_baseline)
```

建议使用短 Lua：

```text
raw = HGET statistics_key field or 0
baseline = HGET reset_key field or 0
return max(0, raw - baseline)
```

规则：

- 原统计 field 不存在：有效用量为 0。
- 快照不存在：基线为 0。
- 快照大于统计值：钳制为 0。
- Redis 错误：保持现有限额 `fail-open`。
- 模型请求主路径不访问 MySQL。

---

## 10. API 设计

### 10.1 管理员接口

```http
POST /api/v1/admin/token-statistics/quota-usage/reset
```

权限：

```text
token_quota.update
```

请求：

```json
{
  "dimension_values": {
    "user_id": {
      "type": "int64",
      "int64": 42
    },
    "group_id": {
      "type": "int64",
      "int64": 7
    }
  },
  "metric_code": "total_tokens",
  "period_type": "D"
}
```

### 10.2 外部接口

```http
POST /api/v1/integrations/token-usage/reset
```

鉴权：

```http
Authorization: Bearer <integration-access-token>
```

调用方不得指定：

- `projection_id`
- `dimension_hash`
- shard
- Redis Key
- 快照值

### 10.3 完全成功

```json
{
  "status": "RESET",
  "matched_quota_count": 2,
  "matched_usage_count": 3,
  "reset_count": 3,
  "failed_count": 0,
  "no_usage_count": 0
}
```

### 10.4 部分成功

```json
{
  "status": "PARTIAL_RESET",
  "matched_quota_count": 2,
  "matched_usage_count": 100,
  "reset_count": 98,
  "failed_count": 2,
  "no_usage_count": 0
}
```

部分成功不回滚。调用方可以再次重置。

### 10.5 没有限额

```json
{
  "status": "NO_QUOTA",
  "matched_quota_count": 0,
  "reset_count": 0,
  "message": "No enabled quota matches the supplied dimensions, metric and period"
}
```

### 10.6 有限额但无用量

```json
{
  "status": "NO_USAGE",
  "matched_quota_count": 1,
  "matched_usage_count": 0,
  "reset_count": 0,
  "message": "Matching quota exists, but no usage exists in the current period"
}
```

### 10.7 错误

| HTTP 状态 | 错误码 | 含义 |
|---|---|---|
| 400 | `INVALID_REQUEST` | 输入参数非法 |
| 400 | `DIMENSION_NOT_REGISTERED` | 包含未注册维度 |
| 503 | `TOKEN_QUOTA_RESET_UNAVAILABLE` | 核心依赖不可用且没有完成任何重置 |
| 401/403 | 沿用现有错误 | 鉴权或 RBAC 失败 |

匹配记录数量多本身不是错误。

---

## 11. 关键流程伪代码

```text
function resetQuotaUsage(request):
    validate period, metric and dimension values
    reject wildcard values in request

    period = currentNaturalPeriod(request.periodType)

    candidateProjections =
        active projections containing every input dimension

    matchingQuotaScopes =
        enabled effective quotas
        matching projection, metric, period and supplied values

    if matchingQuotaScopes is empty:
        return NO_QUOTA

    dirtyIdentities =
        read current dirty identities once
        filter by period, projection, metric and input values

    databaseIdentities =
        query current-period token_stat_aggregates in pages
        filter by projection, metric and input values

    candidates =
        union and deduplicate dirtyIdentities and databaseIdentities

    candidates =
        validate against matchingQuotaScopes once

    if candidates is empty:
        return NO_USAGE

    do not read dirty again
    do not lock synchronization
    do not revalidate quota rules

    for candidates in bounded batches:
        pipeline independent reset Lua calls

        accumulate:
            resetCount
            failedCount
            noUsageCount

        continue processing remaining batches when practical

    if failedCount > 0:
        return PARTIAL_RESET

    if resetCount == 0:
        return NO_USAGE

    return RESET
```

---

## 12. 应用及 Redis 崩溃行为

### 12.1 应用重启、Redis 正常

快照继续存在并生效，不需要数据库恢复。

### 12.2 应用在批量处理中崩溃

允许部分成功：

```text
A 已重置
B 已重置
应用崩溃
C 尚未重置
```

A、B 不回滚。调用方可以再次提交。

### 12.3 Redis 暂时不可用

- 尚未完成任何重置时返回 503。
- 已有部分成功时尽量返回 `PARTIAL_RESET`。
- 如果连接中断导致无法响应，调用方可以重试。
- 限额检查维持现有 `fail-open`。

### 12.4 Redis 恢复持久化数据

Redis AOF/RDB 成功恢复时，快照继续有效。

### 12.5 Redis 快照丢失

由于不写数据库：

- 已执行的重置无法恢复；
- 原统计仍在时，有效用量恢复为原累计量；
- 原统计和快照都丢失时，有效用量暂时为 0；
- 无法区分“没有重置”和“快照丢失”。

建议生产 Redis 开启 AOF，并监控 key eviction。

---

## 13. 安全、日志与监控

### 13.1 管理员接口

- 使用现有管理员认证；
- 使用 `token_quota.update`；
- 前端按权限展示入口；
- 后端独立验证权限。

### 13.2 外部接口

- 仅使用 integrations Bearer Token；
- 复用 provisioning hardening；
- 复杂业务权限由外部系统负责。

### 13.3 日志

记录：

- 调用入口；
- 操作者或来源 IP；
- 周期和指标；
- 输入维度；
- dirty identity 候选数；
- MySQL 候选数；
- 去重后候选数；
- 成功、失败和无用量数量；
- 总耗时。

不得记录 API Key 等敏感值明文。

### 13.4 指标

```text
token_quota_reset_requests_total
token_quota_reset_entries_total
token_quota_reset_partial_total
token_quota_reset_no_quota_total
token_quota_reset_no_usage_total
token_quota_reset_failures_total
token_quota_reset_duration_ms
```

不得使用用户或模型等高基数值作为监控标签。

---

## 14. 兼容、发布与回滚

### 14.1 兼容性

没有快照时：

```text
effective_usage = raw_usage
```

现有行为不变。

### 14.2 发布顺序

1. 增加快照 Key Builder。
2. 增加单条快照 Lua 和 Repository。
3. 增加 reset-aware 限额读取。
4. 实现单次 dirty identity 和 MySQL 身份发现。
5. 实现分批处理。
6. 实现管理员及 integrations 接口。
7. 实现管理员页面。
8. 增加监控并灰度启用。

### 14.3 回滚

旧版本忽略新增快照：

- 真实统计不受影响；
- 已执行重置暂时失效；
- 快照按 TTL 自然过期；
- 不需要数据库回滚。

---

## 15. 验证策略

### 15.1 身份发现

- 第一次读取 dirty identity 后不再复查。
- 不获取统计同步锁。
- dirty member 读取后被删除不影响后续快照。
- MySQL 能补充已完成同步的身份。
- dirty 和 MySQL 重复身份正确去重。
- processing 窗口遗漏被视为允许的一致性边界。
- 新增记录未进入本次候选集合被视为允许行为。
- 完成限额验证后不做二次确认。

### 15.2 Redis 与批量处理

- 单条快照读取和写入原子。
- 多条快照独立成功或失败。
- 单条失败不阻止后续条目。
- 已成功条目不回滚。
- 大量条目通过分页和 pipeline 处理，不因数量直接失败。
- 多次重置覆盖为最新累计量。
- 快照不写 dirty set，不修改 version Key。
- TTL 正确。

### 15.3 同步与限额

- 重置后 MySQL 仍同步真实累计量。
- 限额检查正确计算 `raw - baseline`。
- Redis 读取异常时保持 fail-open。
- 日、周、月快照互不影响。

### 15.4 API 和前端

- 正确区分 `RESET`、`PARTIAL_RESET`、`NO_QUOTA`、`NO_USAGE`。
- 管理员和 integrations 鉴权正确。
- 页面支持动态维度输入。
- 页面正确显示成功、失败和无用量数量。

---

## 16. 验收标准

| ID | 验收标准 |
|---|---|
| AC-01 | 重置成功后对应额度有效用量变为 0。 |
| AC-02 | 原 Redis 统计值不发生变化。 |
| AC-03 | MySQL 继续记录真实累计量。 |
| AC-04 | 输入部分维度时匹配维度超集和值相同的适用统计身份。 |
| AC-05 | 周期不同、值不同或缺少输入维度的条目不受影响。 |
| AC-06 | 没有限额返回 `NO_QUOTA`。 |
| AC-07 | 有限额但无用量返回 `NO_USAGE`。 |
| AC-08 | dirty identity 只读取一次且不获取同步锁。 |
| AC-09 | 已读取身份随后完成同步仍可以建立快照。 |
| AC-10 | 接受 processing 窗口和新记录产生的少量遗漏。 |
| AC-11 | 多条额度部分失败时已成功条目不回滚。 |
| AC-12 | 匹配数量较多时采用分页和分批处理，不因数量直接报错。 |
| AC-13 | 应用重启且 Redis 正常时快照继续有效。 |
| AC-14 | Redis 数据丢失不会污染 MySQL 统计。 |
| AC-15 | 快照按需创建并按周期过期。 |
| AC-16 | 管理页面与外部接口具有相同重置语义。 |

---

## 17. 风险

| 风险 | 影响 | 处理 |
|---|---|---|
| Redis 数据丢失导致快照失效 | 高 | 建议 AOF；明确 Redis-only 边界 |
| processing 窗口遗漏身份 | 低 | 明确接受；再次调用重置 |
| dirty 读取后出现新身份 | 低 | 明确接受；再次调用重置 |
| 大批量处理耗时较长 | 中 | SSCAN、数据库分页、Redis pipeline |
| 多条记录部分成功 | 低 | 返回 `PARTIAL_RESET`，不回滚 |
| 应用中途退出 | 低 | 保留已成功结果，允许重试 |
| 异步事件在重置后入账 | 中 | 以 Redis 入账时刻为边界 |
| 外部调用重试导致再次重置 | 低 | 已确认接受 |

---

## 18. 追踪矩阵

| 需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| 不破坏统计 | 快照式重置 | 独立 Redis Key | AC-02、AC-03 |
| 重置额度 | 限额读取 | `raw - baseline` | AC-01 |
| 部分维度匹配 | 身份发现 | projection、aggregate、dirty | AC-04、AC-05 |
| 简化最终一致性 | 单次身份快照 | 不加锁、不复查 | AC-08～AC-10 |
| 批量尽量完成 | 分批处理 | 分页、SSCAN、pipeline | AC-11、AC-12 |
| 区分业务状态 | API | `NO_QUOTA`、`NO_USAGE` | AC-06、AC-07 |
| 故障边界 | Redis-only | TTL、AOF 建议 | AC-13～AC-15 |
| 双入口 | 管理及集成 | Admin/Integrations API | AC-16 |

---

## 19. 审核记录

| 版本 | 状态 | 内容 |
|---|---|---|
| v0.1 | 已修订 | 首次草案 |
| v0.2 | 已修订 | 补充实际数据结构，取消多条额度整体原子性 |
| v0.3 | 审核通过 | dirty identity 只读取一次且不上锁、不复查；接受 processing 窗口遗漏；取消匹配数量超限失败，改为分页和分批尽量处理 |
| v1.0 | 最终版 | 用户明确审核通过并形成正式 Plan |
