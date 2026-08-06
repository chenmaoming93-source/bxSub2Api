# 模型路由、动态限额与用量检索重构实施 Plan

**状态：最终版——用户已批准（Final — user approved）**

| 项目 | 内容 |
|---|---|
| 文档版本 | v0.3 |
| 日期 | 2026-08-04 |
| 项目 | `bxSub2Api` |
| 当前状态 | 用户已批准 |
| 变更摘要 | 明确区分现有能力、本次开发、回归验证和非本次范围 |

## 1. 背景与目标

### 1.1 待解决问题

1. 非 OAuth 模型账号可以在 `credentials.model_mapping` 中配置多个 key。
2. `groups.model_routing` 候选对象重复保存 `model` 和账号关系。
3. 纯 `model_routing.account_ids` 路径目前每笔请求都会通过 `GetByIDs` 查询账号数据库。
4. 动态 Token 限额不支持“匹配任意值，但按实际值分别限额”的通配符。
5. API Key 限额要求管理员手工输入数据库自增 ID。
6. usage 现有模型搜索没有统一按 `requested_model` 查询。

### 1.2 本次目标

- `G-01`：非 OAuth 模型账号最多保存一个 `model_mapping` key。
- `G-02`：模型路由候选仅保存 `account_ids/priority`。
- `G-03`：模型路由从每个账号自身取得上游模型。
- `G-04`：模型路由账号热态读取不再访问数据库。
- `G-05`：限额维度支持 wildcard，并按请求具体值读取统计。
- `G-06`：API Key 限额使用搜索选择组件。
- `G-07`：重构 usage 现有搜索框，统一按路由别名查询。

### 1.3 非目标

- 不改变数据库中 `credentials`、`model_routing`、`dimension_values` 的列类型。
- 不改变 OAuth 账号行为。
- 不改变非模型路由路径的模型解析方式。
- 不改变现有账号调度排序策略。
- 不重做已有账号快照同步和 scheduler outbox。
- 不修复 `UpdateCredentials` 缺少 outbox 兜底的问题。
- 不增加 wildcard 聚合统计桶。
- 不新增第二个 usage 搜索框。

## 2. 工作分类总览

| ID | 内容 | 分类 | 本次动作 |
|---|---|---|---|
| `BASE-01` | Redis 单账号快照 `sched:acc:<id>` | 已有 | 复用，不重建 |
| `BASE-02` | 账号完整更新后立即刷新单账号快照 | 已有 | 不修改 |
| `BASE-03` | `scheduler_outbox/account_changed` 异步同步 | 已有 | 不修改 |
| `BASE-04` | 分组调度 bucket 初始重建 | 已有 | 保留 |
| `BASE-05` | 候选优先级、粘性、账号 priority、负载、LRU、并发调度 | 已有 | 不修改算法 |
| `BASE-06` | Redis 分块 `MGET` 基础实现 | 已有内部能力 | 复用 |
| `DEV-01` | 模型账号唯一模型校验 | 新开发 | 必须实施 |
| `DEV-02` | 路由候选删除 `model` | 新开发 | 必须实施 |
| `DEV-03` | 每账号解析上游模型 | 新开发 | 必须实施 |
| `DEV-04` | 路由账号批量 cache-first 加载 | 新开发 | 必须实施 |
| `DEV-05` | 回源账号写入现有单账号缓存 | 新开发 | 必须实施 |
| `DEV-06` | Redis 启动后预热模型路由账号 | 新开发 | 必须实施 |
| `DEV-07` | wildcard 类型和限额检查 | 新开发 | 必须实施 |
| `DEV-08` | API Key 搜索选择 | 新开发 | 必须实施 |
| `DEV-09` | usage 路由别名查询重构 | 新开发 | 必须实施 |
| `REG-01` | OAuth 流程 | 回归 | 确认不受影响 |
| `REG-02` | 账号快照即时更新与 outbox | 回归 | 确认仍正常 |
| `REG-03` | 现有多账号调度算法 | 回归 | 确认结果未被改变 |
| `OUT-01` | `UpdateCredentials` 无 outbox | 非本次 | 保留现状 |

## 3. 现有能力基线

本节只描述本次方案依赖的现有能力，不代表需要重新开发。

### 3.1 `BASE-01`：Redis 单账号快照

现有 Redis key：

```text
sched:acc:<account_id>
```

其中保存精简后的 `Account`，并已包含：

```text
credentials.model_mapping
```

现有单账号快照没有自然 TTL，依靠主动同步和重建维护。

**本次不需要：**

- 新建另一套账号缓存。
- 新增用于保存模型名的 Redis key。
- 将模型名额外复制到 `groups.model_routing`。

### 3.2 `BASE-02`：账号完整更新后的即时同步

管理员修改账号及 `model_mapping` 时，当前走：

```text
accountRepo.Update
→ 更新数据库
→ enqueue account_changed
→ syncSchedulerAccountSnapshot
→ 覆盖 sched:acc:<id>
```

这是现有能力。

**本次只需确保：**

- 唯一模型校验发生在数据库更新前。
- 新增的路由账号读取逻辑复用更新后的同一缓存。
- 不破坏原同步调用。

### 3.3 `BASE-03`：数据库 outbox

现有数据库表：

```text
scheduler_outbox
```

`account_changed` 由后台服务消费后，会重新加载账号并刷新 Redis。

Redis 中已有消费进度：

```text
sched:outbox:watermark
```

**本次不需要：**

- 新建 outbox 表。
- 新建 `account_changed` 事件。
- 重写 outbox 消费器。
- 修改管理员完整账号更新的事件生产逻辑。

### 3.4 `BASE-04`：正常分组调度快照

现有 scheduler 会按分组、平台和模式重建调度 bucket。

它可以恢复通过正常分组关系参与调度的账号，但不会天然覆盖只出现在：

```text
model_routing.account_ids
```

中的账号。

本次仅补充模型路由账号预热，不重写正常 bucket 重建。

### 3.5 `BASE-05`：现有多账号调度算法

候选对象之间：

```text
candidate.priority 数值更小者先尝试
```

同一候选内：

```text
过滤不可用账号
→ 优先有效粘性账号
→ 账号 priority 更小
→ 负载率更低
→ 从未使用或 LastUsedAt 更早
→ 完全同组随机打散
→ 获取并发槽
→ 必要时进入等待逻辑
```

`account_ids` 数组不表示固定调用顺序。

本次不得重新定义这套选择算法。

## 4. 本次开发内容

### 4.1 `DEV-01`：模型账号唯一模型校验

#### 需求

非 OAuth 模型账号最终保存的：

```text
credentials.model_mapping
```

最多只能包含一个有效 key。

#### 实现要求

增加统一服务层校验函数：

```text
validateSingleUpstreamModel(accountType, credentials)
```

覆盖以下写入入口：

- 管理员新增账号。
- 管理员修改账号。
- 账号导入。
- 其他能够写入完整 `model_mapping` 的非 OAuth 入口。

#### 不修改内容

- `credentials` 仍为 Map/JSON。
- 空 Map 是否允许沿用现有规则。
- OAuth 账号跳过本规则。
- 账号查询、测试、普通调度和模型列表的既有逻辑不修改。

#### 错误

```text
code: ACCOUNT_MULTIPLE_UPSTREAM_MODELS
message: 模型账号只能配置一个上游模型
```

### 4.2 `DEV-02`：模型路由候选删除 `model`

#### 新格式

```json
{
  "claude-code": [
    {
      "account_ids": [12, 18],
      "priority": 0
    },
    {
      "account_ids": [20],
      "priority": 1
    }
  ]
}
```

#### 后端修改

调整 `ModelRouteCandidate` 的业务结构：

```text
account_ids
priority
```

兼容读取旧结构：

```json
{
  "model": "legacy-model",
  "account_ids": [12],
  "priority": 0
}
```

旧 `model`：

- 可以被反序列化。
- 不参与运行时逻辑。
- 新保存和新响应中不再输出。

继续兼容旧的纯账号 ID 数组格式。

#### 前端修改

`GroupModelRoutingEditor.vue`：

- 删除候选上游模型选择。
- 保留账号多选。
- 保留候选 priority。
- 提交数据不再含 `model`。

### 4.3 `DEV-03`：按账号解析上游模型

#### 原因

删除候选级 `model` 后，同一候选的不同账号可能分别配置不同模型：

```text
账号12 → model-a
账号18 → model-b
```

因此不能为整个候选只计算一个模型。

#### 新局部结构

在模型路由调用路径中增加类似结构：

```text
RoutedAccountCandidate {
  Account
  UpstreamModel
}
```

#### 模型取得规则

```text
读取 Account.Credentials["model_mapping"]
→ 取得有效 key
→ 按字符串字典序升序排序
→ 取第一个
```

新账号原则上只有一个 key；排序用于兼容历史多 key 数据并保证确定性。

#### 使用位置

每个账号自己的 `UpstreamModel` 用于：

- 模型支持检查。
- 模型限流检查。
- 动态 Token 限额。
- 实际上游请求。
- selection result。
- `upstream_model` 日志。
- 现有模型映射记录。

#### 禁止事项

- 不得根据账号 ID 再次查询数据库获取模型名。
- 不得修改非模型路由路径的模型解析。
- 不得使用旧候选中的 `model`。

### 4.4 `DEV-04`：路由账号批量 cache-first 加载

#### 当前问题

`mergeExplicitRouteAccounts` 当前对路由引用账号直接调用：

```go
accountRepo.GetByIDs(ctx, ids)
```

纯 `model_routing.account_ids` 使用场景下，每笔请求都会查数据库。

#### 目标流程

```text
收集匹配路由的全部候选 account_ids
→ 去重
→ 批量读取 sched:acc:<id>
→ 收集缓存未命中的 ID
→ 未命中部分一次性 GetByIDs
→ 合并结果
→ 构建 accountByID
```

#### 新增服务能力

在 scheduler cache/service 边界增加批量账号读取，例如：

```go
GetAccounts(
    ctx context.Context,
    accountIDs []int64,
) (map[int64]*Account, error)
```

#### Redis 读取

复用现有分块 `MGET` 基础能力：

```text
MGET sched:acc:12 sched:acc:18 sched:acc:20
```

不循环执行单个 `GET`。

#### 数据库回源

例如本次需要 `[12,18,20]`，Redis 命中 12、20，则只执行：

```go
accountRepo.GetByIDs(ctx, []int64{18})
```

要求：

- 多个未命中 ID 只执行一次批量查询。
- 回源受现有 scheduler DB fallback 限流和 timeout 保护。
- Redis 与数据库都不可用时沿用现有调度失败出口。

### 4.5 `DEV-05`：回源结果写回现有缓存

#### 需求

数据库回源成功后，把账号写入现有：

```text
sched:acc:<id>
```

稳定状态下后续请求直接命中 Redis。

#### 行为

```text
数据库批量回源成功
→ 本次请求立即使用回源 Account
→ best-effort 写入 Redis 单账号快照
```

Redis 写回失败时：

- 不应丢弃已成功查询的本次账号数据。
- 本次请求可继续调度。
- 记录错误指标和日志。
- 后续请求可能再次回源。

#### 不新增内容

- 不新建缓存表。
- 不新建缓存 key 格式。
- 不修改现有账号更新同步机制。

### 4.6 `DEV-06`：模型路由账号启动预热

#### 当前缺口

Redis 清空并重启后，正常分组 bucket 可以恢复正常分组账号，但纯模型路由账号不一定被加载。

#### 新增预热步骤

在 scheduler 初始恢复阶段追加：

```text
读取启用了 model_routing 的 groups
→ 解析全部 account_ids
→ 去重
→ 分批调用 accountRepo.GetByIDs
→ 写入 sched:acc:<id>
```

#### 加载范围

只预热：

- 正常调度 bucket 已有逻辑所需账号。
- 启用的 `model_routing` 引用账号。

不加载完全未参与调度的数据库账号。

#### 可靠性

- 预热采用有限批次，避免启动时数据库峰值。
- 预热失败不阻止服务启动。
- 请求路径的批量 cache-first 回源继续作为兜底。
- 记录预热数量、耗时和失败数。

### 4.7 `DEV-07`：限额 wildcard

#### 类型扩展

后端 `DimensionValue` 增加：

```text
ValueTypeWildcard = "wildcard"
```

JSON：

```json
{
  "type": "wildcard"
}
```

前端改为联合类型：

```ts
type DimensionValue =
  | { type: 'int64'; int64: number }
  | { type: 'string'; string: string }
  | { type: 'wildcard' }
```

#### 适用边界

wildcard：

- 允许用于限额规则的 `dimension_values`。
- 不允许进入 Usage Event。
- 不改变普通统计查询 filters 的语义。
- 不生成 wildcard 聚合统计桶。

#### 检查逻辑

规则：

```text
group_id=3
route_alias=wildcard
account_id=18
limit=100万
```

请求：

```text
group=3
route=deepseek
account=18
```

限额检查创建本地临时查询值：

```text
group=3
route=deepseek
account=18
```

然后查询该具体组合的统计。规则本身仍永久保持 `route_alias=wildcard`。

#### 实现约束

- 不修改 `rule.DimensionValues`。
- 新建局部 `lookupValues`。
- 固定维度使用规则值。
- wildcard 维度使用请求实际值。
- 请求缺少 wildcard 维度实际值时，规则不适用。
- 不同实际值生成不同 `dimension_hash`。

### 4.8 `DEV-08`：API Key 搜索选择

#### 前端

把 `api_key_id` 数字输入改成可搜索选择框：

- 允许输入名称或具体 API Key。
- 400ms 防抖。
- 最少 2 个字符。
- 取消上一次未完成请求。
- 只采用最后一次响应。
- 候选显示名称和脱敏 Key。
- 用户必须选择候选，不能直接提交自由文本。
- 搜索文本变化时清除旧 ID。

#### 后端

优先复用或扩展现有管理员 API Key 列表接口。

候选响应：

```json
{
  "id": 123,
  "name": "生产环境",
  "masked_key": "sk-abcd…wxyz"
}
```

#### 最终提交

限额接口不变：

```json
{
  "api_key_id": {
    "type": "int64",
    "int64": 123
  }
}
```

#### 安全

- 不返回完整 Key。
- 不把完整 Key 写入日志。
- 不把完整 Key 写入限额规则。
- 如果数据库保存哈希，则复用现有精确认证查询方式。

### 4.9 `DEV-09`：usage 路由别名查询重构

#### UI

直接重构现有“模型”搜索框：

- 标签改为“路由别名”。
- placeholder 改为路由别名语义。
- 不增加第二个搜索字段。

#### 参数

本次暂时保留现有参数名：

```text
model
```

但业务语义改成 route alias/requested model，降低接口兼容风险。

#### 查询表达式

统一为：

```sql
COALESCE(NULLIF(TRIM(requested_model), ''), model) = ?
```

#### 覆盖接口

- 用量明细。
- 错误请求。
- 汇总统计。
- 趋势数据。
- usage 页面受当前过滤条件影响的图表。

#### 候选来源

现有下拉候选改为 requested-model 统计值，不再使用 upstream model 或含义不明确的原始字段。

## 5. 不需要开发但必须回归验证的内容

### 5.1 `REG-01`：OAuth

确认：

- OAuth 新增、刷新和凭证更新不因唯一模型校验失败。
- 本次不修改 `UpdateCredentials` 同步策略。
- 非模型路由 OAuth 调度行为不变。

### 5.2 `REG-02`：账号快照同步

这是已有能力，只需回归确认：

- 管理员修改 `model_mapping` 后仍调用完整 `accountRepo.Update`。
- `sched:acc:<id>` 被现有即时同步覆盖。
- `account_changed` 仍能作为完整 Update 的异步兜底。
- 删除账号仍会删除单账号快照。

这些不是本次新增功能，不应重新实现。

### 5.3 `REG-03`：现有多账号调度

新 `RoutedAccountCandidate` 只能让“账号与自身模型绑定”，不得改变：

- candidate priority。
- 粘性会话优先。
- 账号 priority。
- 负载率排序。
- LRU。
- 同组随机打散。
- 并发槽。
- 等待队列。
- 原有账号健康和限额过滤。

## 6. 关键伪代码

### 6.1 唯一模型校验

```text
function validateSingleUpstreamModel(accountType, credentials):
  if accountType is OAuth:
    return success

  mapping = credentials.model_mapping
  if mapping is absent:
    return success under existing rules

  if mapping is not an object:
    return invalid credentials

  keys = non-empty trimmed keys(mapping)
  if count(keys) > 1:
    return ACCOUNT_MULTIPLE_UPSTREAM_MODELS

  return success
```

### 6.2 批量加载路由账号

```text
function loadRouteAccounts(accountIDs):
  ids = deduplicatePreservingOrder(accountIDs)

  cachedValues = schedulerCache.MGetAccounts(ids)
  accountsByID = decode cache hits
  missedIDs = ids not present in accountsByID

  if missedIDs is not empty:
    verify existing DB fallback policy allows query
    fallbackAccounts = accountRepo.GetByIDs(missedIDs)

    for account in fallbackAccounts:
      accountsByID[account.id] = account

    bestEffort schedulerCache.WriteAccounts(fallbackAccounts)

  return accountsByID
```

### 6.3 每账号模型路由

```text
function selectModelRoute(routeCandidates, accountsByID):
  for routeCandidate ordered by candidate.priority:
    routedAccounts = []

    for accountID in routeCandidate.account_ids:
      account = accountsByID[accountID]
      if account is absent:
        continue

      upstreamModel = firstSortedModelMappingKey(account)
      if upstreamModel is empty:
        record candidate failure
        continue

      if account fails existing checks using upstreamModel:
        continue

      routedAccounts.append(account, upstreamModel)

    selected = existingAccountSelectionAlgorithm(routedAccounts)

    if selected exists:
      return selected.account with selected.upstreamModel

  return candidates exhausted
```

### 6.4 路由账号预热

```text
function warmRouteAccountsAtStartup():
  groups = list enabled model-routing groups
  accountIDs = parse and deduplicate all routing account IDs

  for batch in boundedBatches(accountIDs):
    accounts = accountRepo.GetByIDs(batch)
    schedulerCache.WriteAccounts(accounts)

  record metrics
```

### 6.5 wildcard 限额

```text
function evaluateQuota(rule, requestDimensions):
  lookupValues = new map

  for code in rule.dimension_codes:
    configured = rule.dimension_values[code]
    actual = requestDimensions[code]

    if configured.type == wildcard:
      if actual is missing:
        return not applicable
      lookupValues[code] = actual
    else:
      if actual != configured:
        return not applicable
      lookupValues[code] = configured

  identity = BuildDimensionIdentity(rule.dimension_codes, lookupValues)
  used = Redis.Read(identity)

  if read fails:
    return existing fail-open result

  return compare used with rule.limit_value
```

## 7. 数据与接口影响

### 7.1 数据库结构

本次不新增列、不修改列类型。

| 数据 | 变化 |
|---|---|
| `accounts.credentials` | JSON 内容增加写入约束 |
| `groups.model_routing` | 新候选不再输出 `model` |
| `token_stat_quota_rules.dimension_values` | JSON 允许 wildcard 类型 |
| usage/error logs | 不改结构，只改查询语义 |
| `scheduler_outbox` | 不改 |

### 7.2 Redis

不新增账号缓存 key。本次新增的只是对现有 key 的批量使用和回源写回：

```text
sched:acc:<id>
```

### 7.3 API

| 接口领域 | 变化 |
|---|---|
| 账号新增/修改 | 多模型时新增业务校验错误 |
| 分组路由 | 新候选不再提交 `model` |
| 限额 | `DimensionValue` 接受 wildcard |
| API Key 列表 | 复用或补充搜索能力 |
| usage | 现有 `model` 参数改为 requested-model 语义 |

## 8. 验证策略

### 8.1 新功能测试

#### `DEV-01`

- 非 OAuth：零个、一个、两个 key。
- OAuth 多 key 不触发新校验。
- 新增、修改、导入均不能绕过。

#### `DEV-02/03`

- 新候选不含 `model`。
- 旧候选含 `model` 可读取但被忽略。
- 同一候选两个账号分别使用自身模型。
- 字典序取第一个结果稳定。
- 账号无模型时被过滤。

#### `DEV-04/05`

- Redis 全命中时数据库调用次数为 0。
- 部分命中时只查询缺失 ID。
- 多个缺失 ID 只调用一次 `GetByIDs`。
- 回源后写入现有缓存。
- Redis 写回失败时本次账号仍可使用。
- 数据库 fallback 限流继续有效。

#### `DEV-06`

- Redis 清空后启动预热模型路由账号。
- 没有 `account_groups` 关系的路由账号仍能预热。
- 预热失败时请求回源可以恢复。
- 预热分批执行。

#### `DEV-07`

- wildcard JSON 序列化。
- 不同实际值生成不同 hash。
- 不合并不同路由值。
- 检查前后规则内容不变。
- wildcard 维度缺失时规则不适用。

#### `DEV-08`

- 防抖后才请求。
- 旧请求不会覆盖新结果。
- 未选择候选无法保存。
- 最终提交 ID。
- 响应不含完整 Key。

#### `DEV-09`

- 优先匹配 `requested_model`。
- `requested_model` 为空时回退 `model`。
- 不根据 `upstream_model` 误匹配。
- 明细、错误和统计结果口径一致。

### 8.2 回归测试

- 现有账号即时快照刷新。
- `account_changed` 消费。
- OAuth 流程。
- 非模型路由调用。
- 账号测试和可用模型查询。
- 多账号粘性、priority、负载、LRU 和并发选择。
- Redis 故障时现有调度降级策略。

### 8.3 性能标准

- 路由账号缓存热态：账号数据库查询数为零。
- 路由账号冷态：每批未命中账号最多一次 `GetByIDs`。
- 启动预热不能形成无界数据库并发。
- wildcard 不增加统计写入桶。
- API Key 连续输入不会逐字符请求后端。

## 9. 验收标准

| ID | 验收内容 | 类型 |
|---|---|---|
| `AC-01` | 非 OAuth 模型账号不能保存多个模型 key | 新功能 |
| `AC-02` | 新模型路由候选不包含 `model` | 新功能 |
| `AC-03` | 旧候选仍能被兼容读取 | 新功能 |
| `AC-04` | 每个路由账号使用自身模型 key | 新功能 |
| `AC-05` | 获取账号模型不新增数据库查询 | 新功能 |
| `AC-06` | 热缓存下纯模型路由账号不查数据库 | 新功能 |
| `AC-07` | 缓存未命中只批量查询缺失账号 | 新功能 |
| `AC-08` | 回源账号写回现有 `sched:acc` 缓存 | 新功能 |
| `AC-09` | Redis 冷启动会预热模型路由账号 | 新功能 |
| `AC-10` | wildcard 规则按请求具体值分别检查 | 新功能 |
| `AC-11` | wildcard 检查不修改规则 | 新功能 |
| `AC-12` | API Key 可搜索选择并提交 ID | 新功能 |
| `AC-13` | usage 统一按路由别名查询 | 新功能 |
| `AC-14` | 现有账号快照同步机制未被破坏 | 回归 |
| `AC-15` | 现有多账号调度顺序未被改变 | 回归 |
| `AC-16` | OAuth 和非模型路由行为未改变 | 回归 |

## 10. 实施顺序

### 阶段一：数据契约与领域约束

本次开发：

- `DEV-01` 唯一模型校验。
- `DEV-02` 路由候选格式。
- `DEV-07` wildcard 类型基础。

### 阶段二：路由账号加载

本次开发：

- `DEV-04` 批量 cache-first。
- `DEV-05` 回源写回。
- 相关缓存和数据库调用测试。

依赖已有：

- Redis `sched:acc`。
- 分块 `MGET`。
- DB fallback limiter。
- `GetByIDs`。

### 阶段三：路由运行时

本次开发：

- `DEV-03` 每账号模型解析。
- 账号与上游模型绑定参与调度。

只做回归：

- 现有粘性、负载、LRU、priority 和并发算法。

### 阶段四：冷启动恢复

本次开发：

- `DEV-06` 扫描和预热路由账号。

依赖已有：

- scheduler 启动流程。
- 账号缓存写入。
- 正常 bucket 重建。

### 阶段五：管理端与限额

本次开发：

- wildcard 检查和 UI。
- API Key 搜索选择。
- 路由编辑器删除模型字段。

### 阶段六：usage 重构

本次开发：

- 统一 requested-model 查询表达式。
- 现有搜索框文案和候选来源。
- 明细、错误和图表一致性测试。

### 阶段七：集成与发布

- 新功能验收。
- 已有机制回归。
- Redis 清空恢复测试。
- 热态数据库查询验证。
- 回滚准备。

## 11. 风险与边界

| ID | 风险 | 缓解 |
|---|---|---|
| `R-01` | 历史路由 `model` 与账号模型不同 | 明确账号模型为事实来源 |
| `R-02` | 历史账号存在多个 key | 字典序取第一；新写入禁止 |
| `R-03` | Redis 冷启动缓存击穿 | 启动预热、批量回源、现有限流 |
| `R-04` | 预热造成数据库峰值 | 分批、限速、异步执行 |
| `R-05` | 新局部候选结构意外改变排序 | 固定现有算法并做回归测试 |
| `R-06` | wildcard 增加 Redis 读取 | 请求内 identity 去重 |
| `R-07` | API Key 搜索泄露 | 脱敏响应、禁止日志记录原文 |
| `R-08` | usage 表达式影响索引 | 检查执行计划，必要时后续优化 |
| `R-09` | `UpdateCredentials` 刷新失败无 outbox | 已确认保留，不属于本次 |
| `R-10` | 旧后端不能读取新路由格式 | 明确发布和回滚顺序 |

## 12. 追踪矩阵

| 目标 | 本次开发项 | 依赖的已有能力 | 验收 |
|---|---|---|---|
| `G-01` | `DEV-01` | 现有账号写入流程 | `AC-01` |
| `G-02` | `DEV-02` | 现有路由 JSON 字段 | `AC-02～03` |
| `G-03` | `DEV-03` | 现有 Account credentials | `AC-04～05` |
| `G-04` | `DEV-04～06` | `sched:acc`、MGET、outbox、GetByIDs | `AC-06～09` |
| `G-05` | `DEV-07` | 现有具体维度统计桶 | `AC-10～11` |
| `G-06` | `DEV-08` | 现有限额 ID 数据结构 | `AC-12` |
| `G-07` | `DEV-09` | 已有 requested/model 日志字段 | `AC-13` |
| 稳定性 | 无新开发 | 现有同步和调度算法 | `AC-14～16` |

## 13. 评审记录

| 版本 | 状态 | 修订内容 |
|---|---|---|
| v0.1 | 已被替代 | 首版方案 |
| v0.2 | 已被替代 | 加入路由账号缓存和恢复 |
| v0.3 | 用户已批准 | 将已有能力、本次开发、回归项和非本次范围严格分离；明确账号快照同步与 outbox 不需要重新开发 |
