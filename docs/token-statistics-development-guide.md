# 可配置 Token 统计开发指引

## 1. 体系边界

本体系负责“注册维度与指标 → 配置投影 → 请求完成后异步累计 Redis → 定时同步 MySQL → 通用查询与动态限额”。它使用：

- Go 包：`backend/internal/service/tokenstat`、`backend/internal/repository/tokenstat`
- MySQL 表：`token_stat_*`
- Redis：`sub2api:dynamic_token_stats:*`、`sub2api:dynamic_token_stats_dirty:*`、`sub2api:dynamic_token_stats_config:*`、`sub2api:dynamic_token_stats_quota:*`
- 管理 API：`/admin/token-statistics/*`
- 管理页面：`/admin/token-statistics`

旧的模型、用户＋模型、分组＋路由别名＋上游模型固定统计及限额已经退役。本实现不得重新引入、读取、写入或兜底调用旧 `model_token_daily_*`、`user_model_token_daily_*`、`group_candidate_token_daily_*`、旧 Redis key 或 `/admin/token-usage/*`。

## 2. 增加新指标

指标是可累计的数值，例如未来的 `input_tokens`。投影配置只保存注册代码，代码负责把一次请求转换成指标值。

1. 在 `backend/internal/service/tokenstat/types.go` 增加稳定的 `MetricCode` 常量。代码一经生产使用不得改名。
2. 在 `registry.go` 注册 `MetricDefinition`：
   - `Code`：稳定机器标识；
   - `DisplayName`、`Unit`：页面展示；
   - `AllowQuota`：该指标是否可配置限额；
   - `Version`：语义版本。仅展示名变化不升级；计算语义或单位变化必须升级。
3. 在 `backend/internal/service/dynamic_token_usage.go` 的统一事件构造处填充该指标。不得在各协议转发器内分别写 Redis。
4. 如果新值无法从现有统一 usage 结果获得，先扩充统一 usage 领域对象，再由各协议适配；主请求路径仍只执行非阻塞入队。
5. 为注册表、事件校验、投影累计、Redis→MySQL 同步、查询和限额（若允许）补测试。
6. 前端不得硬编码可选指标；继续读取 `/admin/token-statistics/metrics`。

示例：

```go
const MetricInputTokens MetricCode = "input_tokens"

registerMetric(MetricDefinition{
    Code: MetricInputTokens, DisplayName: "输入 Token",
    Unit: "token", AllowQuota: true, Version: 1,
})
```

同时在统一事件中加入：

```go
Metrics: map[tokenstat.MetricCode]int64{
    tokenstat.MetricTotalTokens: total,
    tokenstat.MetricInputTokens: input,
}
```

如果指标加入后不要求历史回填，旧投影不自动获得该指标；管理员新建或调整配置后从启用时开始记录。

## 3. 增加新维度

维度会参与投影签名、维度哈希、Redis key field、MySQL冗余列和查询白名单，改动面比指标大。

1. 在 `types.go` 增加 `DimensionCode`。
2. 在 `registry.go` 注册类型、显示名、排序和语义版本。
3. 在统一 usage 事件构造处提供值。缺少投影要求的维度时，该事件对相应投影不会产生错误的半条记录。
4. `BuildDimensionIdentity` 必须继续按注册顺序、类型和规范化值生成稳定身份；不要使用展示名。
5. 若需要高频筛选，在 `token_stat_aggregates` Ent schema 添加可空冗余列、索引，并更新：
   - `backend/internal/repository/tokenstat/sync_engine.go`
   - `backend/internal/service/tokenstat/query.go`
   - `backend/sqlArchiving/NNN_*.sql`
6. DDL 只交付到 `backend/sqlArchiving/`，编号取 migrations 与 sqlArchiving 当前最大值加一。生产结构变更由管理员手动执行，应用不得自动执行该 SQL。
7. 前端筛选与分组继续由注册表和投影动态生成。

未来增加 `token_type` 时，可将其注册为字符串维度。新投影从启用时记录，既有聚合行和历史查询不回填；需要索引时再添加冗余列，不改变既有维度哈希语义。

## 4. 兼容性与版本

- 代码标识和已发布投影的维度顺序属于持久化协议。
- 计算语义变化优先新增指标代码；不要悄悄复用旧代码。
- 注册项 `Version` 用于表达语义版本，不等同于运行时配置版本。
- 投影配置版本发布到 Redis；多实例只消费完整、已验证的 ACTIVE 配置。
- Redis key 格式变化必须使用新命名空间版本，不能原地混写。
- 不做历史迁移时，应在发布说明中注明“仅对启用后的调用生效”。

## 5. 性能与失败策略

- 模型请求成功后只调用非阻塞 `TryEnqueue`；队列满、统计事件非法或 Redis 写入失败均不得改变模型响应。
- 限额只在成功读取 Redis 且确认 ENFORCE 超限时限制；Redis 和规则缓存异常 fail-open。
- 禁止在模型请求主逻辑中执行 MySQL、同步、封账、SCAN 或重试等待。
- 新指标不得引入按用户、账号或模型无限增长的进程内标签指标；观测数据使用固定字段快照。
- 查询必须由投影白名单生成 Ent 谓词，限制日期、分页和源行数。

## 6. 测试清单

至少运行：

```text
cd backend
go test ./internal/service/tokenstat ./internal/repository/tokenstat ./internal/handler/admin ./internal/server/routes

cd frontend
pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts
pnpm typecheck
pnpm build
```

新增指标还应覆盖：

- 所有协议经过统一 `recordUsage` 后产生相同事件；
- 队列满和 Redis 错误不影响调用结果；
- D/W/M 三个自然周期累计；
- 并发原子加、脏集合、同步幂等、封账版本一致；
- 查询拒绝未配置指标和非法维度；
- OBSERVE、ENFORCE 与 Redis fail-open；
- 新路径隔离扫描不命中旧表、旧 API 和旧 Redis 标识。

## 7. AI 后续开发禁止事项

- 不得为了“兼容”调用旧三套固定统计或限额。
- 不得自动迁移或回填旧数据，除非新的、已审核计划明确要求。
- 不得执行生产 DDL；只提交符合 `PROJECT_CONVENTIONS.md` 的 SQL 文件。
- 不得把 MySQL 写入、Redis 重试或聚合查询放到模型请求同步路径。
- 不得绕过注册表接受任意 SQL 列名、排序字段或 Redis key 片段。
- 不得改名复用已投产的维度、指标或命名空间版本。
- 不得把“测试未运行”记录为通过。

## 8. 限额用量重置

### 8.1 数据语义与 Redis namespace

限额重置不清零真实统计，也不修改 MySQL 聚合。它把重置瞬间某条 raw 统计值复制为 Redis-only baseline，限额读取使用：

```text
effective_usage = max(0, raw_usage - reset_baseline)
```

- raw 统计：`sub2api:dynamic_token_stats:v1:{period_type}:{period_start}:{projection_id}:{shard}`
- reset baseline：`sub2api:dynamic_token_quota_reset:v1:{period_type}:{period_start}:{projection_id}:{shard}`
- 两者使用同一个 field：`{dimension_hash_hex}:{metric_code}`。
- baseline 缺失时视为 0，因此旧数据及未重置限额仍保持 `effective_usage == raw_usage`。
- 每个 baseline Hash 的过期时间是自然周期结束时间加 `orphan_ttl_days`，与原统计孤儿 TTL 策略一致。
- 单条重置由 Lua 原子执行 `HGET raw → HSET baseline → EXPIREAT`；raw field 不存在时返回 `NO_USAGE`，不创建零 baseline。

### 8.2 一致性边界

部分维度请求会匹配所有包含这些维度的 ACTIVE 投影超集，并只保留 ENABLED、当前有效且完整维度值匹配的 quota 身份。发现过程严格按以下顺序执行：

1. 对 `sub2api:dynamic_token_stats_dirty:v1:current` 完整 SSCAN 一次；
2. 随后按页读取当前周期 MySQL `token_stat_aggregates` 身份；
3. 在内存联合、去重并执行完整 quota 匹配；
4. 按固定批次用普通 Redis pipeline 执行每条独立 Lua。

这里刻意不获取同步锁、不二次读取 dirty、不扫描 `processing:{token}`、不在写 baseline 前重新校验 quota。扫描后新增或暂处 processing 窗口的身份可能遗漏，这是已接受的最终一致性边界；管理员可再次重置。多个身份没有全局事务，单条失败不回滚成功条目，结果用 `PARTIAL_RESET` 和计数表达。

### 8.3 持久性、故障与运维

- baseline 只存在 Redis，不写入或恢复到 MySQL；应用进程重启且 Redis 数据保留时继续生效。
- 建议生产 Redis 开启并监控适当的 AOF/RDB 持久化和备份，但它只是运维增强，不构成应用层恢复保证。
- Redis baseline 丢失后，限额读取自动回到 raw 累计值；这不会污染或改写 MySQL 真实统计。
- 管理员通用查询以 MySQL `raw_value` 为真实累计，并按每个源统计身份的完整维度 hash 从 Redis pipeline 读取 baseline；分组查询会分别汇总 raw 与 baseline，返回 `reset_snapshot` 和 `effective_value=max(0,raw-reset_snapshot)`。Redis 读取失败时仍返回真实累计，并通过 `reset_snapshots_available=false` 明确标记快照不可用。
- Redis 读取异常时限额检查继续 fail-open；重置入口在无成功项时返回 `TOKEN_QUOTA_RESET_UNAVAILABLE`，已有成功项时优先返回 `PARTIAL_RESET`。
- 回滚应用版本前无需迁移或删除 baseline：旧版本不会读取独立 reset namespace。确认不再需要该能力后，可在变更窗口按前缀清理 baseline；禁止清理 raw、version 或 dirty namespace。

### 8.4 API 与禁止事项

- 管理员：`POST /api/v1/admin/token-statistics/quota-usage/reset`，要求 `token_quota.update`。
- integrations：`POST /api/v1/integrations/token-usage/reset`，只使用既有 Bearer Token 与 hardening。
- 管理员接口的 `dimension_values` 使用内部维度值；管理页面通过共享可搜索选择器把用户、API Key、分组和模型账号转换为对应 ID，不要求管理员手填 ID。
- integrations 接口的 `dimension_values` 使用人类可读字段：`username`（精确匹配用户 email）、`api_key`（完整 Key 值）、`group_name`、`account_name`、`route_alias`、`upstream_model`、`department`。服务解析前四项为内部 ID，后三项保留为字符串维度。
- integrations 示例：`{"dimension_values":{"username":"u@example.com","group_name":"public","route_alias":"default"},"metric_code":"total_tokens","period_type":"D"}`。
- 请求只接受具体维度值、允许 quota 的 `metric_code` 和 `D/W/M period_type`；不得接受 projection、hash、shard、Redis key、baseline 或 wildcard。只传单一维度可以有意匹配多个统计身份。
- 管理员重置响应额外返回 `matched_quotas` 和 `matched_entries`，用于展示匹配限额、统计身份维度、命中限额 ID 及每条重置结果；integrations 响应不返回这些内部排障详情。
- 同一请求重复调用会以当时最新 raw 覆盖 baseline，不提供幂等键。
- 不得把 reset baseline 放入 dirty 同步流，不得用 baseline 覆盖 raw 或 `token_stat_aggregates.metric_value`。

## 9. 外部四维当前用量查询

- 接口：`POST /api/v1/integrations/token-usage/query`。
- 只允许 integrations Bearer Token；不得添加 JWT、登录态、Admin Auth 或 RBAC 权限点。
- `username` 精确使用用户 `email` 查询；`api_key` 使用 `api_keys.key` 明文值（数据库唯一约束），并校验其属于目标用户和分组；路由别名必须属于目标分组。
- Key 值不存在/已删除返回 404 `API_KEY_NOT_FOUND`；Key 存在但用户/分组归属不匹配返回 400 `API_KEY_MISMATCH`（消息不泄露该 Key 实际归属）。
- 只匹配 `user_id,api_key_id,group_id,route_alias` 精确活动投影，指标为 `total_tokens`。
- 当前日、周、月使用 `CurrentUsageReader` 精确执行最多三个 `HGET`；禁止 Redis 扫描及 MySQL 回退。
- Field 缺失表示已配置但当前周期为 0；投影不存在返回 `dimension_configured=false`；Redis 故障返回 HTTP 503。
- Redis Key/Field 规则必须继续与 `RedisAccumulator` 共用 Builder，并保留读写兼容测试。
