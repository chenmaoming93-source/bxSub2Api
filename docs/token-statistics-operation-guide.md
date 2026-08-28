# 可配置 Token 统计与限额操作手册

## 1. 上线前准备

1. 由数据库管理员审阅并手动执行 `backend/sqlArchiving/165_create_dynamic_token_statistics_tables.sql`。应用不会自动执行生产 DDL。
2. 配置 `gateway.dynamic_token_statistics`，确认 Redis、MySQL、时区 `Asia/Shanghai`、队列和同步周期符合环境要求。
3. 启动后访问 `/admin/token-statistics`。页面要求 `token_usage.read`；编辑统计项要求 `token_usage.manage`；编辑限额要求 `token_quota.update`。
4. 新体系不迁移旧统计或限额。首次配置后从启用时重新记录。

## 2. 建立统计项

在“统计项”页签填写名称，勾选维度并保存草稿。维度组合即查询和限额的统计粒度，例如：

- `upstream_model`
- `user_id + upstream_model`
- `group_id + route_alias + upstream_model`

草稿可以编辑；发布后进入待启用状态；启用后开始接收新调用。页面显示统计起点，启用前的历史不会补齐。停用不会删除 MySQL 历史数据。被 ENABLED 限额引用的统计项不能停用，应先停用限额。

## 3. 建立限额

在“限额”页签选择维度并填写每个维度的精确值、自然周期、Token 上限和模式：

- `OBSERVE`：记录超限判定，不阻断；
- `ENFORCE`：成功读取 Redis 且已达到上限时限制请求或排除候选账号。

限额创建后立即作用于当前自然日、自然周或自然月，并使用当前周期已经累计的 Token 用量进行判断。若维度组合尚无统计项，系统自动创建统计草稿，限额显示 PENDING；管理员发布并启用该统计项后，等待限额自动变为 ENABLED 并立即生效。停用后重新启用的限额也会立即生效。

统计项可以不配置限额。限额不能脱离统计项生效。

## 4. 通用查询

在“通用查询”页签：

1. 选择一个已采集投影；
2. 选择日、自然周或自然月；
3. 设置日期范围；
4. 按需填写筛选值和分组维度；
5. 查看汇总、趋势、排行榜和分页明细，或导出 CSV。

页面只提供该投影实际包含的维度，不能跨投影拼接。自定义日期范围使用日粒度；自然周和自然月必须使用对齐的自然周期。查询读取 MySQL，不合并 Redis 当前值，因此会显示最后同步时间、统计开始时间和完整性提示。

## 5. 同步与封账

“同步状态”显示最近周期状态和最后同步时间：

- `OPEN`：周期进行中；
- `CLOSING`：已进入关闭流程，等待队列排空；
- `FINAL_SYNC`：最终 Redis→MySQL 同步和版本校验；
- `PERSISTED`：MySQL 已确认持久化；
- `DELETED`：旧周期 Redis key 已 UNLINK。

Redis 只保留当前自然日、周、月和封账中的短暂数据。只有达到 PERSISTED 后才删除旧周期 key，因此正常情况下可等待系统自动过期或封账清理，不应人工批量删除。

## 6. 监控与告警

同步状态接口 `/admin/token-statistics/status` 返回固定基数 metrics：

- `enqueued`、`dropped_queue_full`、`invalid_events`
- `redis_write_failures`
- `synced_rows`、`sync_failures`
- `finalized_periods`、`finalize_failures`
- `quota_checks`、`quota_exceeded`、`quota_fail_open`
- `config_version`

建议告警：

- 5 分钟内 `dropped_queue_full` 或 `redis_write_failures` 增量大于 0；
- 连续两个同步周期 `sync_failures` 增长，或 `last_synced_at` 落后超过 3 个同步周期；
- 自然周期结束后 30 分钟仍未进入 `PERSISTED/DELETED`；
- `quota_fail_open / quota_checks` 持续超过 1%；
- 多实例看到的 `config_version` 长时间不一致；
- Redis 内旧周期 key 在确认 PERSISTED 后仍持续增长。

这些计数是进程生命周期内累计值，实例重启会归零；监控系统应按实例抓取增量。

## 7. 故障处理

### 队列丢弃或 Redis 写失败

模型请求仍应成功。先检查 Redis 延迟、连接池和队列容量，再恢复服务；未入队的统计不会自动补记。不要改为同步写 MySQL。

### MySQL 同步失败

脏集合会重新入队，Redis 仍保留计数。修复 MySQL 后等待下个同步周期，核对 `synced_rows`、`last_synced_at` 和页面数据。

### 封账失败

查看周期 `last_error`、pending queue、脏集合和版本校验。未达到 PERSISTED 前禁止人工删除 Redis key。修复后允许定时任务重试。

### 限额未限制

依次检查限额状态、统计项 ACTIVE、`effective_from`、维度值、Redis 当前周期计数和 `quota_fail_open`。基础设施异常时设计为 fail-open。

### 页面无历史数据

确认查询时间晚于投影 `enabled_at`，周期粒度对齐，MySQL 已同步，并确认筛选维度属于当前投影。系统不会读取旧统计历史。

## 8. 演练步骤

测试环境可按以下顺序模拟：

1. 建立 `user_id + upstream_model` 投影并启用；
2. 发起一次成功模型调用，确认请求延迟不等待统计写入；
3. 等待同步，查询 MySQL 结果和 `last_synced_at`；
4. 建立 OBSERVE 小额度，确认超限不阻断；
5. 建立或重新启用 ENFORCE，确认其立即根据当前周期累计用量执行超限限制；
6. 暂停 Redis，确认模型请求 fail-open 且 `quota_fail_open`/写失败增长；
7. 恢复 Redis/MySQL，确认同步重试；
8. 对已结束周期执行封账流程，确认状态最终为 DELETED 且 MySQL 数据保留；
9. 运行隔离扫描，确认新体系未引用旧表、旧 API 或旧 Redis key。

生产环境不执行破坏性故障演练；可用预发布环境或依赖代理模拟超时。

## 9. 外部当前用量接口运维

接口路径为 `POST /api/v1/integrations/token-usage/query`，复用 `ExternalAPIKeyProvisioning` 的启用开关、Access Token 和限流配置。调用方不需要站内登录或 RBAC 权限。

- HTTP 401：检查 `Authorization: Bearer <token>`，JWT、登录 Cookie 和模型 Gateway API Key 均不能替代 integrations Token。
- HTTP 400 `API_KEY_MISMATCH`：API Key 值存在，但不属于请求中的用户/分组（或无分组），请核对 `username`/`group_name` 与 Key 的实际归属。
- HTTP 404：功能可能关闭，或用户 email、分组、API Key 值（不存在/已删除）、路由别名不存在。
- HTTP 200 且 `dimension_configured=false`：启用精确四维活动投影 `user_id,api_key_id,group_id,route_alias`。
- HTTP 200 且 `data_present=false,total_tokens=0`：投影存在，但该当前周期 Redis Field 尚无数据，属于合法零值。
- HTTP 503：Redis 当前周期读取失败或值非法；接口不会回退到可能滞后的 MySQL。

发布前验证目标 URL 只注册一次、RBAC coverage 将其归类为 external integration exclusion、三个周期读取正确。回滚只需撤回接口代码，不应删除 Redis 数据或统计投影。

## 10. 外部历史按天用量接口运维

接口路径为 `POST /api/v1/integrations/token-usage/query/group-api-key/daily`，入参 `group_name`、`api_key`、`start_date`、`end_date`（`YYYY-MM-DD`，跨度 ≤ 366 天）。同样复用 `ExternalAPIKeyProvisioning` 的启用开关、Access Token 和限流配置，数据来自 MySQL `token_stat_aggregates`（最终一致，同步间隔见 `gateway.dynamic_token_statistics.sync_interval_minutes`）。路径中 `group-api-key` 为维度组合标识，未来其他维度组合的历史按天查询沿用该模式（如 `user-model/daily`）。

- HTTP 400 `INVALID_REQUEST`：JSON/字段/日期格式非法，`end_date < start_date`，或跨度超过 366 天。
- HTTP 400 `API_KEY_MISMATCH`：API Key 值存在，但不属于目标分组（或无分组）。
- HTTP 404 `GROUP_NOT_FOUND` / `API_KEY_NOT_FOUND`：分组或 API Key 值不存在/已删除；或 integrations 功能关闭（`NOT_FOUND`）。
- HTTP 200 且 `dimension_configured=false`、`days=[]`、`message="统计维度未配置"`：不存在**维度签名精确等于 `api_key_id,group_id`** 且启用中的统计投影（在「可配置 Token 统计」页面新建、发布、启用；注意含额外维度的投影不会被本接口采用）。
- HTTP 200 且 `dimension_configured=true`、`days=[]`：统计项已配置，但该范围内无任何已同步数据（数据从投影启用时刻起采集）。
- HTTP 200 且 `days` 非空：只含有数据的天（升序），缺失天不出现、不补 0；结合 `complete=false`（仍在最终一致同步中）与 `last_synced_at` 判断时效。
- HTTP 500：MySQL 聚合查询失败；接口不返回部分结果。

投影选择规则：**强制要求**存在 ACTIVE 且维度签名精确等于 `api_key_id,group_id` 的统计投影（维度组合必须且只能是「API Key + 分组」两项）。超集投影（如四维 `user_id,api_key_id,group_id,route_alias`）不会被采用，因为异步管道会跳过缺少额外维度（如 `route_alias` 为空）的事件，导致该投影漏记、求和偏小。发布前验证目标 URL 只注册一次、RBAC coverage 将其归类为 external integration exclusion。回滚只需撤回接口代码，不删除 MySQL 聚合数据或统计投影。

### 10.1 CSV 下载接口

`POST /api/v1/integrations/token-usage/query/group-api-key/daily/csv`：入参与 JSON 接口完全一致（`group_name`、`api_key`、`start_date`、`end_date`），鉴权方式相同，返回 `text/csv` 附件下载。

- CSV 列：`date,total_tokens`，`date` 为统计时区日期（升序）。
- **逐日补 0**：范围内每一天都有一行，无记录的天 `total_tokens` 为 0（与 JSON 接口"缺失天不出现"不同）。
- HTTP 409 `STATISTICS_NOT_CONFIGURED`：未配置精确 `api_key_id,group_id` 投影时返回该错误，**不会**输出全 0 的 CSV（避免与"已配置但无数据"混淆）。
- 其余错误语义与 JSON 接口一致（400 / 401 / 404 / 500）。
