# 去金额计费与 Redis Token 日统计改造 Plan

**状态：Final — user approved**  
**版本：v0.4**  
**日期：2026-07-13**  
**变更摘要：新增异常可诊断性要求。新功能的异常日志必须明确标识阶段、统计类型、业务维度和底层原因，避免使用无法定位问题的通用错误信息。**

## 1. 背景与目标

当前成功请求会执行金额计费、请求明细记录和三类每日 Token 统计更新。高并发集中更新相同统计行时，会造成 MySQL 行锁竞争和连接耗尽。

本次改造目标：

- 停止所有金额相关逻辑；
- 金额底层实现尽量保留，优先注释或旁路调用；
- `usage_logs` 保持现有逻辑和表结构；
- 保留三张每日 Token 统计表；
- 每天使用三个 Redis Hash 保存 Token 绝对累计值；
- 后台定期将 Redis 累计值覆盖到 MySQL；
- 不新增 MySQL 表；
- 不增加恢复、历史重建和 `usage_logs` 对账；
- 新功能发生异常时，日志必须包含可定位问题的具体信息，禁止大面积使用 `internal error`、`operation failed` 等通用描述。

## 2. 范围

### 2.1 保留

- API Key 身份认证；
- 用户、API Key、分组和账号状态检查；
- 并发与 RPM 限制；
- Token 配额配置和检查；
- `usage_logs` 请求明细；
- 三张每日 Token 统计表；
- 原有金额计费函数、类型、Repository 和数据库表实现。

### 2.2 停用

- 模型价格解析与 fallback pricing；
- 输入、输出、缓存和图片金额计算；
- 用户余额检查与扣减；
- 订阅金额检查与累计；
- API Key 金额额度检查与累计；
- 上游账号金额额度检查与累计；
- `applyUsageBilling` 调用；
- `usage_billing_dedup` 新增写入；
- 账号金额成本统计。

### 2.3 不做

- 不删除金额计费实现；
- 不重构计费模块；
- 不修改 `usage_logs` 表结构；
- 不增加 `route_alias` 字段；
- 不新增 MySQL 表；
- 不实现恢复或对账；
- 不考虑存量数据迁移；
- 不考虑上线切换期间计数一致性；
- Redis、MySQL 或应用异常造成的计数损失可以接受；
- 不为本功能新建数据库监控表。

## 3. 核心决策

### DEC-01 最小侵入停用金额逻辑

在金额逻辑最外层调用位置注释或旁路，不删除底层实现。

```go
// Monetary billing disabled for token-statistics-only operation.
// 原实现保留，后续如需恢复可重新启用。
// cost, err := calculateCost(...)
// billingApplied, err := applyUsageBilling(...)
```

### DEC-02 `usage_logs` 原样保留

- 不修改表和字段；
- 不增加恢复与对账职责；
- 按现有流程写入；
- 金额字段沿用现有零值。

### DEC-03 每日三个 Redis Hash

```text
sub2api:token_stats:model:{date}
sub2api:token_stats:user_model:{date}
sub2api:token_stats:group_candidate:{date}
```

不使用 `batch_id`、Hash 分片或每记录一个 Key。

### DEC-04 Redis 保存绝对累计值

Redis value 表示当天累计 Token。MySQL 使用绝对值覆盖，不再次累加。

### DEC-05 异常允许计数损失

- Redis 写入失败时记录具体错误；
- MySQL 同步失败时有限重试；
- 不回退逐请求 MySQL 写入；
- 不建设补偿和恢复机制。

### DEC-06 异常必须可诊断

每个新功能异常必须包含：

- 明确的事件名称；
- 失败阶段；
- 统计类型；
- 业务日期；
- Redis Key 或 MySQL 表；
- 当前游标或批次大小；
- 重试次数；
- 原始底层错误；
- 可用时附带 `request_id`、`user_id`、`group_id`、`model`。

不得仅记录：

```text
internal error
operation failed
redis error
database error
sync failed
```

对客户端仍可返回安全的通用信息，但服务端日志必须保留具体错误链。

## 4. 功能设计

### FR-01 请求前资格检查

保留：

- API Key 认证；
- 实体状态检查；
- 并发与 RPM；
- Token 配额。

停用：

- 余额；
- 订阅金额；
- API Key 金额额度；
- 账号金额额度。

余额不足不再返回 `billing_error`。

### FR-02 请求后使用明细

请求完成后：

1. 使用现有逻辑构建 `UsageLog`；
2. 跳过价格解析；
3. 跳过金额计算；
4. 跳过账号金额统计；
5. 跳过 `applyUsageBilling`；
6. 按现有逻辑写入 `usage_logs`；
7. 将 Token 写入 Redis。

### FR-03 Redis Token 累计

Token 口径保持不变：

```text
total_tokens =
    input_tokens
  + output_tokens
  + cache_creation_tokens
  + cache_read_tokens
```

通过一个 Pipeline 执行三个 `HINCRBY`。

### FR-04 定时同步 MySQL

后台 worker：

1. 获取 Redis 同步锁；
2. 使用 `HSCAN` 扫描三个 Hash；
3. 解码 field；
4. 分批覆盖 MySQL；
5. 有限重试；
6. 输出同步汇总或具体失败日志。

### FR-05 跨天与过期

- 业务时区使用 `Asia/Shanghai`；
- 新一天自动使用新 Key；
- 同步当天及前一天；
- 旧 Key 保留时间可配置；
- 默认保留 2 天。

## 5. Redis 数据设计

### 5.1 模型统计

```text
Key:
sub2api:token_stats:model:2026-07-13

Field:
gpt-5.2

Value:
125680
```

### 5.2 用户模型统计

```text
Key:
sub2api:token_stats:user_model:2026-07-13

Field:
1|gpt-5.2

Value:
86420
```

### 5.3 分组候选统计

```text
Key:
sub2api:token_stats:group_candidate:2026-07-13

Field:
2|yace|gpt-5.2

Value:
73190
```

### 5.4 Field 编码

采用稳定、可逆、无歧义的公共编码函数。字符串字段使用长度前缀或 URL-safe Base64，禁止直接假设模型名称不包含分隔符。

### 5.5 Redis 写入

```text
HINCRBY modelKey modelField totalTokens
HINCRBY userModelKey userModelField totalTokens
HINCRBY groupCandidateKey groupCandidateField totalTokens
```

通过同一个 Pipeline 提交，并设置固定绝对过期时间，不因后续请求延长旧日期 Key 的生命周期。

## 6. 配置设计

```yaml
gateway:
  token_statistics:
    redis_enabled: true
    sync_interval_minutes: 1
    hscan_count: 1000
    mysql_batch_size: 500
    sync_retry_count: 3
    redis_retention_days: 2
```

| 配置 | 默认值 | 说明 |
|---|---:|---|
| `redis_enabled` | `true` | 启用 Redis Token 累计 |
| `sync_interval_minutes` | `1` | MySQL 同步周期（分钟） |
| `hscan_count` | `1000` | `HSCAN COUNT` 提示值 |
| `mysql_batch_size` | `500` | MySQL 单批写入数量 |
| `sync_retry_count` | `3` | 同步失败重试次数 |
| `redis_retention_days` | `2` | 旧日期 Key 跨天后保留天数 |

配置非法时，启动阶段必须返回包含配置路径、非法值和有效范围的具体错误，例如：

```text
invalid gateway.token_statistics.redis_retention_days:
value=0, expected positive integer
```

## 7. MySQL 同步设计

### 7.1 表与唯一键

| 表 | 唯一统计键 |
|---|---|
| `model_token_daily_usages` | `model, usage_date` |
| `user_model_token_daily_usages` | `user_id, model, usage_date` |
| `group_candidate_token_daily_usages` | `group_id, route_alias, upstream_model, usage_date` |

只更新：

```text
used_tokens
updated_at
```

不得修改 `daily_limit_tokens`。

### 7.2 覆盖语义

```sql
used_tokens = GREATEST(
    used_tokens,
    VALUES(used_tokens)
)
```

不得执行二次累加。

### 7.3 批量写入

- 默认每批 500 条；
- 三张表分别使用小事务；
- 失败时有限重试；
- 保留原 `IncrementDailyTokenQuotas` 实现，但正常请求不再调用。

### 7.4 HSCAN

- 用户模型 Hash 禁止高频 `HGETALL`；
- 使用 `HSCAN`；
- 游标为 `0` 才表示完成；
- 重复 field 通过绝对值覆盖保证安全；
- 扫描期间变化的数据在下一周期同步。

## 8. Token 配额读取

1. 读取 `daily_limit_tokens`；
2. 优先从 Redis 读取当前 `used_tokens`；
3. Redis field 不存在时回退 MySQL；
4. 判断是否超限；
5. 请求完成后按实际 Token 执行 `HINCRBY`。

该限制仍属于软限制，并发请求可能短暂超额。

## 9. 异常日志设计

### 9.1 日志原则

- 使用结构化日志；
- 事件名固定且可搜索；
- 保留完整 `error` chain；
- 在最接近失败源的位置补充上下文；
- 上层不要把具体错误替换成 `internal error`；
- 同一个错误避免在每层重复打印完整堆栈；
- 成功请求不逐条打印日志；
- 不记录 API Key 明文、请求正文等敏感信息。

### 9.2 Redis 累加异常

事件名：

```text
token_statistics.redis_increment_failed
```

字段：

```text
stage=redis_increment
request_id
user_id
group_id
route_alias
model
upstream_model
usage_date
total_tokens
redis_keys
error
```

示例：

```text
token_statistics.redis_increment_failed
stage=redis_pipeline_exec
request_id=req_123
user_id=1
group_id=2
model=yace
upstream_model=gpt-5.2
usage_date=2026-07-13
total_tokens=1536
redis_keys=[
  sub2api:token_stats:model:2026-07-13,
  sub2api:token_stats:user_model:2026-07-13,
  sub2api:token_stats:group_candidate:2026-07-13
]
error="redis: connection pool timeout"
```

### 9.3 Field 编码或解码异常

事件名：

```text
token_statistics.field_encode_failed
token_statistics.field_decode_failed
```

字段：

```text
stage
statistics_type
usage_date
redis_key
encoded_field
cursor
error
```

日志不得输出无法控制长度的完整异常数据；field 可截断并附带 hash。

### 9.4 HSCAN 异常

事件名：

```text
token_statistics.redis_scan_failed
```

字段：

```text
stage=redis_hscan
statistics_type
usage_date
redis_key
cursor
count_hint
retry_attempt
error
```

### 9.5 MySQL 批量同步异常

事件名：

```text
token_statistics.mysql_sync_failed
```

字段：

```text
stage=mysql_batch_upsert
statistics_type
usage_date
table
row_count
batch_index
retry_attempt
max_retries
error_code
error
```

示例：

```text
token_statistics.mysql_sync_failed
stage=mysql_batch_upsert
statistics_type=user_model
usage_date=2026-07-13
table=user_model_token_daily_usages
row_count=500
batch_index=17
retry_attempt=3
max_retries=3
error_code=1205
error="Lock wait timeout exceeded; try restarting transaction"
```

### 9.6 分布式锁异常

区分：

- 锁已被其他实例持有：正常跳过，使用 Debug 或低频 Info；
- Redis 获取锁失败：Warning/Error。

事件名：

```text
token_statistics.sync_lock_held
token_statistics.sync_lock_failed
```

字段：

```text
instance_id
lock_key
lock_ttl
error
```

### 9.7 配额读取异常

事件名：

```text
token_statistics.quota_redis_read_failed
token_statistics.quota_mysql_fallback_failed
```

字段：

```text
request_id
statistics_type
user_id
group_id
model
usage_date
redis_key
fallback_attempted
error
```

若 Redis 读取失败后 MySQL 回退成功，可记录低频 Warning；两者都失败时必须记录 Error。

### 9.8 同步汇总日志

每轮同步只输出汇总，不逐 field 打印：

```text
token_statistics.sync_completed
```

字段：

```text
usage_date
statistics_type
scanned_fields
mysql_rows
scan_calls
duration_ms
retry_count
```

## 10. 可观测性

不新增数据库表，不向 MySQL 写监控指标。

使用：

- 结构化应用日志；
- 现有内存指标或 Prometheus 指标（如果项目已启用）；
- 不为本次功能额外引入新的监控系统。

建议指标：

```text
token_stats_redis_increment_success_total
token_stats_redis_increment_failed_total
token_stats_sync_success_total
token_stats_sync_failed_total
token_stats_sync_duration_seconds
token_stats_sync_fields_total
token_stats_sync_mysql_rows_total
token_stats_sync_lag_seconds
token_stats_redis_hash_fields
```

## 11. 关键伪代码

### 11.1 请求前检查

```text
function checkEligibility(request):
    authenticateAPIKey()
    checkEntityStatus()
    checkConcurrencyAndRPM()

    limit = readTokenLimit()

    used, redisError = readRedisUsage()
    if redisError:
        log quota_redis_read_failed with full context

        used, mysqlError = readMySQLUsage()
        if mysqlError:
            log quota_mysql_fallback_failed with full context
            return safe client error

    if tokenLimitExceeded(limit, used):
        reject request

    // 金额检查停用，具体实现保留
```

### 11.2 请求后记录

```text
function recordUsage(result, context):
    usageLog = buildUsageLogUsingExistingLogic()

    // 金额计算和计费调用停用，具体实现保留

    writeUsageLogUsingExistingLogic(usageLog)

    tokens = usageLog.totalTokens
    if tokens <= 0:
        return

    error = pipelineHIncrByForThreeHashes()
    if error:
        log redis_increment_failed with request and Redis context
        return
```

### 11.3 定期同步

```text
function syncTokenStatistics(date):
    lockResult = acquireDistributedLock()

    if lock held by another instance:
        return

    if lockResult is error:
        log sync_lock_failed with original Redis error
        return

    for each statisticsType:
        cursor = 0

        repeat:
            entries, nextCursor, error = HSCAN()

            if error:
                log redis_scan_failed with key and cursor
                stop current statisticsType

            decoded, errors = decodeEntries(entries)
            log each malformed entry with bounded context

            for each MySQL chunk:
                error = retryBatchUpsert()

                if error:
                    log mysql_sync_failed with table, batch and DB error
                    stop current statisticsType

            cursor = nextCursor
        until cursor == 0

        log one sync_completed summary
```

## 12. 可靠性边界

| 异常 | 行为 |
|---|---|
| Redis 累加失败 | 具体日志；本次计数可能丢失 |
| Redis 扫描失败 | 具体日志；等待下一周期 |
| Field 解码失败 | 记录统计类型、Key、field 摘要；跳过该 field |
| MySQL 同步失败 | 有限重试；记录表、批次及数据库错误 |
| 应用退出 | 不保证未完成操作恢复 |
| Redis 数据丢失 | 不从 `usage_logs` 恢复 |
| 多实例同步 | Redis 锁限制单一执行者 |
| 通用客户端错误 | 服务端日志保留完整具体原因 |

## 13. 安全与隐私

- Redis 不保存 API Key、请求正文、IP、User-Agent 或金额；
- 日志不输出 API Key 明文和请求正文；
- 模型等可变长字段需要长度限制；
- 错误日志保留底层错误，但需经过现有敏感信息过滤；
- 不改变现有 API 权限边界。

## 14. 验证策略

### AC-01 金额功能停用

确认余额、订阅、API Key 和账号金额数据不再变化，且不再新增计费去重记录。

### AC-02 非金额控制保留

确认身份、状态、并发、RPM 和 Token 配额仍生效。

### AC-03 `usage_logs` 不变

确认表结构和写入逻辑不变，不增加恢复、对账或 `route_alias`。

### AC-04 Redis 三 Hash

确认每日只有三个 Hash，并发 `HINCRBY` 累计正确。

### AC-05 MySQL 同步

确认重复同步不重复累计，旧值不覆盖新值，`daily_limit_tokens` 不变化。

### AC-06 Redis 保留时间

确认默认保留 2 天，修改配置后 TTL 正确变化。

### AC-07 热点压测

确认不再逐请求更新三张表，锁等待显著下降。

### AC-08 具体异常日志

分别注入以下故障：

- Redis Pipeline 失败；
- Redis HSCAN 失败；
- Field 解码失败；
- Redis 锁失败；
- MySQL 连接失败；
- MySQL `1205`；
- 配置非法；
- Redis 配额读取和 MySQL 回退同时失败。

每个故障必须验证：

- 事件名明确；
- `stage` 明确；
- 统计类型明确；
- Key 或表明确；
- 底层原始错误存在；
- 重试信息存在；
- 不仅输出 `internal error`；
- 不泄露敏感信息。

## 15. 实施顺序

### 阶段一：金额逻辑旁路

停用金额资格检查、价格计算和金额落库调用，保留具体实现。

### 阶段二：Redis 累计

实现三个每日 Hash、field 编解码、Pipeline、TTL 和具体异常日志。

### 阶段三：MySQL 批量覆盖

实现三张表批量 upsert、`GREATEST` 覆盖、有限重试及数据库错误日志。

### 阶段四：同步 worker

实现定时器、分布式锁、`HSCAN`、跨天同步和同步汇总日志。

### 阶段五：切换请求路径

停止逐请求 MySQL Token 增量，改为 Redis 累计；Token 配额读取优先 Redis。

### 阶段六：故障注入与压测

验证具体错误日志、每日三 Hash、TTL、MySQL 同步和热点压测结果。

## 16. 回滚方案

1. 停止 Redis 同步 worker；
2. 停止请求写 Redis；
3. 恢复原 `IncrementDailyTokenQuotas`；
4. 恢复金额资格检查；
5. 恢复价格计算和 `applyUsageBilling`；
6. 遗留 Redis Key 按 TTL 自动过期。

## 17. 风险与开放项

| 风险 | 影响 | 缓解 |
|---|---|---|
| 停用金额逻辑误伤 Token 检查 | Token 配额失效 | 分支级修改和回归测试 |
| HSCAN 期间数据变化 | 本轮同步滞后 | 下一轮重新扫描 |
| Redis 异常 | 计数损失 | 已接受；输出具体日志 |
| 错误日志过多 | 日志系统压力 | 成功不逐请求记录；相同失败限频 |
| 错误日志缺少上下文 | 排查困难 | 固定事件和必填字段 |
| 日志泄露敏感数据 | 安全风险 | 字段白名单、截断和脱敏 |
| 用户模型 Hash 未来过大 | 同步周期增加 | 监控 field 数，未来再评估分片 |

## 18. 追踪矩阵

| 需求 | 模块 | 验收 |
|---|---|---|
| 停用金额功能并保留实现 | Eligibility、RecordUsage | AC-01、AC-02 |
| `usage_logs` 原样保留 | UsageLog Repository | AC-03 |
| 每天三个 Redis Hash | Redis Statistics Repository | AC-04 |
| 可配置保留时间，默认 2 天 | Redis TTL | AC-06 |
| 定期覆盖三张表 | Sync Service、Quota Repository | AC-05 |
| 降低热点写入 | Redis 累计、批量覆盖 | AC-07 |
| 异常日志具体可查 | 所有新组件 | AC-08 |

## 19. 审核记录

- v0.1：采用 Redis 批次增量方案。
- v0.2：改为每天三个累计 Hash，取消 `batch_id` 和分片。
- v0.3：移除部署迁移考虑；旧日期 Key 保留时间可配置，默认 2 天。
- v0.4：新增异常可诊断性要求；所有新功能异常必须提供明确事件名、阶段、统计类型、业务维度和底层原因，避免通用错误信息；可观测性不新增数据库写入。
- 2026-07-13：用户明确审核通过，状态更新为 `Final — user approved`。
