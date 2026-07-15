# Token 消耗数据实时读取改造 Plan

**状态：Final — user approved**  
**版本：v1.0**  
**日期：2026-07-14**  
**变更摘要：明确 MySQL 与 Redis 的集合式读取、合并和读修复流程；不使用“不存在标记/负缓存”；禁止 Redis miss 后逐项额外查询 MySQL。**

## 1. 背景

目前 Token 消耗存在两套读取方式：

- 请求限额判断优先使用 Redis 中的当天实时数据；
- Token 消耗统计页面和部分限额配置页面仍然使用 MySQL 中的定时同步数据。

因此，一次请求完成后可能出现：

- 限额判断已经使用最新消耗量；
- 页面仍显示上一次同步到 MySQL 的旧值。

本次改造需要统一这些页面和限额判断的数据读取口径。

## 2. 改造目标

### 2.1 主要目标

- 历史日期的统计数据继续完全由 MySQL 提供。
- 今天的统计数据优先使用 Redis。
- Redis 缺失、MySQL 存在时，使用 MySQL 数据并反向修复 Redis。
- Redis 存在、MySQL 尚未同步时，保留 Redis 实时数据。
- 三个 Token 消耗统计页面和相关配置页面使用相同的数据来源规则。
- 汇总、使用率、状态、排序和分页基于最终合并结果计算。

### 2.2 成功标准

- 查询不包含今天时，不读取 Redis。
- 查询包含今天时，今天的数据能够实时反映 Redis 累计值。
- 限额判断与页面展示的当天 `used_tokens` 一致。
- Redis 缺少 MySQL 已有数据时，查询结果正确，并自动修复 Redis。
- 查询不存在的统计项时，不因 Redis miss 额外执行一次 MySQL 点查。
- 不改变现有 API 请求参数和响应结构。

## 3. 改造范围

### 3.1 包含范围

- 模型 Token 消耗统计页面。
- 路由候选 Token 消耗统计页面。
- 用户模型 Token 消耗统计页面。
- 全局模型限额配置中的“今日已使用”。
- 用户模型限额配置中的“今日已使用”。
- 其他经代码检索发现的、返回上述三类 `used_tokens` 的管理端配置接口。
- 当天默认统计目标及当天筛选候选项。
- 请求限额判断中的 Redis 缺失修复逻辑。

### 3.2 不包含范围

- 不修改 Token 累加逻辑。
- 不修改 Redis → MySQL 定时同步周期。
- 不修改 Redis field 编码格式。
- 不新增数据库表或字段。
- 不改变限额配置的存储位置。
- 不修改前端页面布局。
- 不通过查询接口强制触发一次完整 Redis → MySQL 同步。
- 不增加“不存在标记”或负缓存。

## 4. 已确认的数据规则

### 4.1 日期规则

“今天”使用项目现有业务时区计算。

假设当前日期为 2026-07-14：

| 查询范围 | 数据来源 |
|---|---|
| 7 月 10 日～7 月 12 日 | 完全使用 MySQL |
| 7 月 10 日～7 月 14 日 | 10～13 日使用 MySQL，14 日合并 Redis 与 MySQL |
| 7 月 14 日 | 合并 Redis 与 MySQL |
| 未来日期 | 按现有接口规则返回空结果或参数错误 |

### 4.2 当天数据规则

| Redis | MySQL | 最终结果 | 动作 |
|---|---|---|---|
| 有 | 有 | Redis | 不相加，不覆盖 Redis |
| 有 | 无 | Redis | 视为正常同步延迟 |
| 无 | 有 | MySQL | 返回 MySQL，并反向修复 Redis |
| 无 | 无 | 无数据 | 返回空或零值，不修复 Redis |

“MySQL 最终权威”仅适用于：

```text
Redis 缺失 + MySQL 存在
```

MySQL 中暂时没有、但 Redis 已经存在的当天数据，不得删除、清零或忽略。

### 4.3 Redis 故障规则

Redis 连接失败、超时与 Redis field 不存在是两种情况：

- Redis 成功响应，但 field 不存在：属于数据缺失，可以使用 MySQL 并执行读修复。
- Redis 整体读取失败：无法确认 field 是否真的缺失，只降级使用 MySQL，不执行覆盖式修复；后续查询重新尝试。

### 4.4 聚合规则

以下业务口径保持不变：

- `used_tokens` 按现有统计维度求和；
- 使用率为 `used_tokens / daily_limit_tokens`；
- 使用率达到 80% 为 `warning`；
- 使用率达到 100% 为 `exceeded`；
- 无有效限额时为 `unlimited`；
- 三种统计维度和业务键不变。

## 5. 功能设计

### FR-01：历史查询保持原有路径

查询范围不包含今天时：

- 继续调用现有 MySQL 报表仓储；
- 继续由 MySQL 完成聚合、汇总、排序和分页；
- 不访问 Redis；
- 不进入服务层混合逻辑。

这样可以避免影响历史查询性能和已有行为。

### FR-02：当天数据采用集合式读取

查询包含今天时，不按每个筛选项逐个执行：

```text
Redis HGET
→ miss
→ MySQL 点查
```

而是执行：

```text
一次查询 MySQL 得到符合条件的当天集合
+
一次批量读取 Redis 得到当天集合
↓
在服务内比较两个集合
```

模型、路由候选和用户模型分别处理，不混合业务维度。

### FR-03：按业务键合并

三类业务键如下：

| 维度 | 业务键 |
|---|---|
| 模型 | `usage_date + model` |
| 路由候选 | `usage_date + group_id + route_alias + upstream_model` |
| 用户模型 | `usage_date + user_id + model` |

相同业务键下：

- Redis 数据覆盖 MySQL 当天累计值；
- Redis 与 MySQL 绝不相加；
- Redis 独有项保留；
- MySQL 独有项用于回退和读修复。

### FR-04：Redis 缺失时执行读修复

当 Redis 成功读取，但 MySQL 集合中某个业务项在 Redis 集合中不存在：

1. 本次查询使用 MySQL 的 `used_tokens`。
2. 将 MySQL 累计值写入对应 Redis Hash。
3. 确保 Redis key 继续遵守现有过期策略。
4. 修复采用绝对值写入，不进行累加。
5. 修复失败不影响本次查询结果。
6. 后续请求会再次发现缺失并尝试修复。

### FR-05：防止读修复覆盖并发实时数据

不能直接无条件执行 `HSET`。

可能出现：

1. 查询读取 Redis，发现 field 不存在；
2. 新请求完成，使用 `HINCRBY` 创建了该 field；
3. 查询用较旧 MySQL 值执行 `HSET`；
4. Redis 实时累计值被覆盖。

因此读修复必须采用原子“仅不存在时写入”：

```text
field 仍不存在 → 写入 MySQL 值
field 已经存在 → 保留 Redis 现值
```

建议使用 Lua 或等价原子操作，同时设置正确 TTL。

### FR-06：不存在统计项不触发额外查询

管理员可以手动构造接口参数，查询不存在的模型、路由或用户。

例如：

```text
model=not-exists
```

执行规则：

1. 报表正常查询一次 MySQL。
2. 正常批量读取 Redis。
3. 两边结果均不存在时返回空结果。
4. 不再针对 `not-exists` 单独查询 MySQL。
5. 不向 Redis 写入该不存在项。
6. 不创建负缓存。

因此，反复查询不存在项会产生报表接口本来就需要的正常 MySQL 查询，但不会因为 Redis miss 增加数据库查询次数。

### FR-07：三个报表使用最终合并结果

查询包含今天时，下列内容必须基于最终数据计算：

- 明细记录；
- `summary.used_tokens`；
- `pagination.total`；
- 使用率；
- 状态；
- 多字段排序；
- 分页。

不允许先使用 MySQL 数据完成分页，再只覆盖当前页的 Redis 数值。

### FR-08：配置页面显示实时消耗

全局模型和用户模型限额配置：

- 限额配置仍从 MySQL 读取；
- MySQL 同时批量读取今天已有消耗；
- Redis 批量读取今天实时消耗；
- 按相同业务键合并；
- 页面中的“今日已使用”使用最终合并值；
- 保存限额后的返回结果也使用相同规则。

配置页面只处理 MySQL 中已经存在的配置项，不针对任意不存在项执行额外查询。

### FR-09：当天选项与默认目标

对于依赖当天数据的查询选项和默认目标：

- Redis 中存在但尚未同步到 MySQL 的统计项应可进入候选集合；
- 当天默认目标应基于最终合并后的消耗量选择；
- 历史日期的默认目标继续使用 MySQL；
- 用户名、分组名、优先级等信息继续由 MySQL 提供；
- 无法关联到有效业务实体的 Redis 孤立项不作为可选择目标，并记录诊断信息。

## 6. 技术设计

### 6.1 当前架构问题

当前限额仓储通过 Redis 覆盖 MySQL 快照，但：

- 报表仓储直接查询三张 MySQL usage 表；
- 管理配置仓储也从 MySQL 读取 `used_tokens`；
- 各模块没有共享“今天的数据来源”规则。

### 6.2 新增统一读取能力

在 service 层定义统一接口，在 repository 层提供实现，例如：

```text
CurrentTokenUsageReader
├─ ReadModelUsage
├─ ReadGroupCandidateUsage
└─ ReadUserModelUsage
```

该组件负责：

- 批量读取 Redis；
- 复用现有 field 编解码；
- 接收已经查询出的 MySQL 当天集合；
- 比较 MySQL 和 Redis 集合；
- 执行缺失回退；
- 批量执行原子读修复；
- 返回最终当天集合。

该组件不负责：

- 单独根据每个 Redis miss 查询 MySQL；
- 历史日期查询；
- 前端分页参数解析；
- 业务权限判断；
- 修改限额配置。

### 6.3 Repository 职责

MySQL 报表 Repository 拆分为两类能力：

- 现有历史快速查询：聚合、汇总、排序、分页。
- 包含今天时使用的集合查询：查询符合过滤条件的 MySQL 日报数据、限额配置以及用户、分组、路由描述信息；不针对 Redis miss 进行二次点查。

### 6.4 Service 职责

`TokenUsageReportService` 负责：

- 校验日期和筛选条件；
- 判断是否包含今天；
- 选择纯 MySQL 路径或混合路径；
- 合并历史与今天的日报级记录；
- 计算使用率和状态；
- 生成汇总；
- 执行最终排序和分页。

服务层处理的是已经按天累计的数据，不读取原始请求流水。

### 6.5 Redis 读取方式

使用三类现有 Hash：

```text
sub2api:token_stats:model:<date>
sub2api:token_stats:group_candidate:<date>
sub2api:token_stats:user_model:<date>
```

根据数据规模选择 `HSCAN` 或受控批量读取，避免阻塞 Redis。

当查询条件已经明确到少量业务键时，可以采用批量 `HMGET`，但仍需保证：

- 不逐项访问 Redis；
- 不在 Redis miss 后逐项访问 MySQL；
- 一次查询的 Redis 往返次数受控。

### 6.6 读修复操作

建议实现批量原子修复：

```text
repairMissingFields(redisKey, mysqlOnlyRows, expireAt)
```

每个 field 的语义：

```text
if field does not exist:
    HSET field mysqlUsedTokens
else:
    keep current Redis value
ensure key expiration
```

修复值必须是 MySQL 中的绝对累计值，不能使用 `HINCRBY`。

### 6.7 查询数据流

#### 不包含今天

```text
Handler
→ Report Service
→ MySQL Report Repository
→ MySQL 聚合、排序和分页
→ 返回
```

#### 包含今天

```text
Handler
→ Report Service
→ MySQL 查询历史日报集合
→ MySQL 查询今天日报及配置集合
→ Redis 批量读取今天集合
→ 比较 MySQL 与 Redis 集合
→ Redis 缺失项使用 MySQL 并读修复
→ Redis 独有项保留
→ 合并历史与今天
→ 计算使用率和状态
→ 汇总、排序、分页
→ 返回
```

### 6.8 API 兼容性

以下接口保持现有路径、参数和响应结构：

- `/admin/token-usage/models`
- `/admin/token-usage/routes`
- `/admin/token-usage/users`
- `/admin/token-usage/options/...`
- `/admin/token-usage/default-target`
- 全局模型限额接口
- 用户模型限额接口

前端不需要为数据源变化调整调用方式。

### 6.9 数据模型

不修改以下 MySQL 表结构：

- `model_token_daily_usages`
- `group_candidate_token_daily_usages`
- `user_model_token_daily_usages`
- `model_token_daily_limit_configs`
- `group_candidate_token_daily_limit_configs`
- `user_model_token_daily_limit_configs`

现有唯一键继续作为集合合并依据：

- `model + usage_date`
- `group_id + route_alias + upstream_model + usage_date`
- `user_id + model + usage_date`

不需要数据库迁移和历史数据回填。

### 6.10 配置规约

预计本次不新增配置。

如果实现时需要新增或修改配置：

- 更新 `backend/config/config.yaml`；
- 为新增配置增加含义及单位注释；
- 时间间隔类配置使用分钟；
- 同步维护 `PROJECT_CONVENTIONS.md` 中的开发规约。

## 7. 核心伪代码

### 7.1 报表主流程

```text
function getReport(query):
    validate(query)
    today = businessToday()

    if query.endDate < today:
        return mysqlRepository.queryHistoricalReport(query)

    historyRows = []
    if query.startDate < today:
        historyRows = mysqlRepository.queryDailyRows(
            query.startDate,
            today - 1 day,
            query.filters
        )

    mysqlTodayRows = mysqlRepository.queryTodayRowsAndConfigs(today, query.filters)
    redisTodayResult = redisReader.readToday(query.dimension, today, query.filters)

    if redisTodayResult.failed:
        todayRows = mysqlTodayRows
        recordRedisFallback()
    else:
        todayRows, repairRows = mergeTodaySets(mysqlTodayRows, redisTodayResult.rows)
        repairRedisMissingFields(repairRows)

    allRows = historyRows + todayRows
    calculateUsageRateAndStatus(allRows)
    summary = sum(allRows.usedTokens)
    sort(allRows, query.sortRules)
    resultPage = paginate(allRows, query.page, query.pageSize)

    return resultPage, summary, count(allRows)
```

### 7.2 集合合并

```text
function mergeTodaySets(mysqlRows, redisRows):
    mysqlMap = indexByBusinessKey(mysqlRows)
    redisMap = indexByBusinessKey(redisRows)
    finalMap = empty
    repairRows = empty

    for each redisRow in redisMap:
        if mysqlMap contains redisRow.key:
            finalMap[redisRow.key] = redisRow enriched with mysql metadata and limit
        else:
            finalMap[redisRow.key] = redisRow

    for each mysqlRow in mysqlMap:
        if redisMap contains mysqlRow.key:
            continue

        finalMap[mysqlRow.key] = mysqlRow
        if mysqlRow represents persisted usage:
            repairRows.add(mysqlRow)

    return finalMap.values, repairRows
```

### 7.3 不存在查询项

```text
function queryNonexistentFilter(filter):
    mysqlRows = mysqlRepository.query(filter)
    redisRows = redisReader.query(filter)

    if mysqlRows empty and redisRows empty:
        return empty report

    // 不执行额外 MySQL 点查
    // 不写入 Redis
    // 不创建负缓存
```

### 7.4 原子修复

```text
function repairRedisMissingFields(rows):
    for each controlled batch:
        atomically:
            for each row:
                if redis hash field does not exist:
                    write row.usedTokens
                else:
                    preserve existing Redis value
            ensure existing expiration policy
```

## 8. 异常处理

### 8.1 Redis 不可用

- 返回 MySQL 结果；
- 不将 Redis 故障误判为 field 缺失；
- 不执行可能覆盖数据的修复；
- 记录降级日志和指标；
- 后续查询重新尝试 Redis。

### 8.2 Redis field 无法解码

- 记录统计类型、日期和错误阶段；
- 不让损坏记录进入最终统计；
- 能够从 MySQL 明确识别业务键时，允许按缺失规则修复正确 field；
- 不自动删除无法识别的 Redis field。

### 8.3 Redis value 非法

- 非整数或负数不得覆盖 MySQL；
- 记录错误；
- 使用 MySQL 对应记录降级；
- 能确认业务键时可重写正确值。

### 8.4 MySQL 查询失败

- 历史查询无法由 Redis 完整替代，接口返回错误；
- 不把 MySQL 错误当作空结果；
- 不执行读修复。

## 9. 性能与可靠性

### 9.1 性能原则

- 历史查询保留现有 MySQL 快速路径。
- Redis 采用批量读取，不产生 N+1。
- MySQL 当天数据采用集合查询，不产生 Redis miss 后的逐项点查。
- 配置页面批量关联消耗数据。
- 读修复使用 pipeline 或受控批次。
- 服务层只处理日报累计值，不处理原始请求日志。

### 9.2 可观测性

增加以下指标或结构化日志：

- Redis 批量读取耗时；
- MySQL 历史查询耗时；
- MySQL 当天集合查询耗时；
- Redis 读取失败次数；
- MySQL 降级次数；
- MySQL 独有记录数量；
- Redis 独有记录数量；
- 读修复成功、失败和并发跳过数量；
- Redis field 解码失败数量；
- 混合结果记录总数；
- 混合排序与分页耗时。

正常的每条命中不单独输出日志，避免日志膨胀。

## 10. 验证策略

### 10.1 日期路径测试

- 查询 7 月 10 日～12 日仅访问 MySQL。
- 查询 7 月 10 日～14 日访问 MySQL 历史、MySQL 今天集合和 Redis 今天集合。
- 查询仅包含今天时不读取无关历史数据。

### 10.2 合并规则测试

三种维度分别验证：

- Redis 与 MySQL 都有时使用 Redis。
- Redis 有、MySQL 无时使用 Redis。
- Redis 无、MySQL 有时使用 MySQL 并修复 Redis。
- 两边都没有时返回空，不写 Redis。
- 两边累计值不相加。

### 10.3 不存在项测试

通过公开管理接口传入不存在的模型名、路由别名、上游模型、用户 ID 和用户模型组合，验证：

- 返回空结果；
- 每个报表请求只执行设计内的 MySQL 集合查询；
- 不因 Redis miss 追加 MySQL 点查；
- 不创建 Redis field；
- 不创建负缓存；
- 多次请求行为一致。

### 10.4 并发测试

- 读取发现 field 缺失后，并发请求创建该 field；
- 读修复不得覆盖并发产生的 Redis 实时值；
- 多个查询同时修复同一 field 时结果幂等；
- 修复不得重复累加。

### 10.5 报表正确性测试

- `summary.used_tokens` 使用合并结果。
- `pagination.total` 使用合并结果。
- Redis-only 记录能够进入结果列表。
- 按 `used_tokens` 排序正确。
- 按使用率和状态排序正确。
- 多字段排序优先级保持不变。
- 使用率和超限状态基于最终值计算。

### 10.6 配置页面测试

- 全局模型限额显示 Redis 实时消耗。
- 用户模型限额显示 Redis 实时消耗。
- Redis 缺失时显示 MySQL 值并修复。
- 保存限额后返回最新消耗值。
- Redis 故障时页面仍能打开。

### 10.7 接口兼容测试

- API 路径不变。
- 请求参数不变。
- 响应 JSON 字段和类型不变。
- 前端三个报表页面无需修改调用协议。
- 已删除用户显示规则不变。

## 11. 验收标准

| 编号 | 验收条件 |
|---|---|
| AC-01 | 不包含今天的查询完全由 MySQL 处理，不访问 Redis |
| AC-02 | 包含今天时，今天两边都有的数据使用 Redis |
| AC-03 | Redis 独有的实时数据正常显示 |
| AC-04 | Redis 缺失、MySQL 存在时使用 MySQL 并反向修复 Redis |
| AC-05 | 两边都不存在时返回空，不产生额外 MySQL 点查和 Redis 写入 |
| AC-06 | 读修复不会覆盖并发产生的 Redis 实时数据 |
| AC-07 | 三个报表的明细、汇总、排序和分页均基于最终合并结果 |
| AC-08 | 配置页面中的“今日已使用”与限额判断一致 |
| AC-09 | Redis 不可用时降级 MySQL，且不误执行缺失修复 |
| AC-10 | 不产生逐项 Redis 访问或逐项 MySQL 回退查询 |
| AC-11 | 现有 API 契约保持兼容 |
| AC-12 | 不修改数据库结构、Redis 编码和统计业务口径 |
| AC-13 | 关键降级、缺失和修复行为具有日志及测试证据 |

## 12. 实施顺序

### 阶段一：建立统一读取和修复能力

- 定义三种统计维度的批量读取接口。
- 复用现有 Redis key 和 field codec。
- 实现集合比较。
- 实现原子“仅缺失时写入”。
- 完成缺失、异常和并发测试。

### 阶段二：统一限额与配置页面

- 将请求限额判断接入统一读修复语义。
- 改造全局模型限额配置。
- 改造用户模型限额配置。
- 检索并改造其他展示 `used_tokens` 的配置接口。

### 阶段三：改造三个统计报表

- 保留纯历史 MySQL 路径。
- 增加包含今天的混合路径。
- 实现三个维度的集合合并。
- 在合并后计算汇总、状态、排序和分页。
- 改造当天选项和默认目标。

### 阶段四：验证与交付

- 完成后端单元、仓储、接口和并发测试。
- 完成前端 API 及页面回归。
- 进行 Redis 故障降级测试。
- 验证不存在筛选项不会造成额外点查。
- 进行混合查询性能验证。
- 检查日志和指标。

## 13. 发布与回滚

### 13.1 发布

API 契约不变，优先发布后端。

上线后观察：Redis 缺失率、MySQL 回退率、读修复成功率、Redis-only 数据量、混合查询耗时和接口错误率。

### 13.2 回滚

本次没有数据库迁移，可直接回滚代码：

- 恢复统计页面纯 MySQL 查询；
- 停止读修复；
- 已修复到 Redis 的数据来自 MySQL 已有绝对累计值，无需清理；
- Redis → MySQL 定时同步不受影响。

## 14. 风险

| 风险 | 影响 | 缓解措施 |
|---|---|---|
| 旧 MySQL 值覆盖并发 Redis 值 | 实时消耗倒退 | 原子“仅不存在时写入” |
| 合并后分页增加服务层负担 | 查询变慢 | 保留历史快速路径，仅处理日报累计数据 |
| Redis-only 项缺少描述信息 | 页面信息不完整 | 从 MySQL 关联实体；无效孤立项记录诊断 |
| Redis 故障被误判为数据缺失 | 错误修复 | 严格区分读取失败和 field 不存在 |
| 按 miss 逐项回查 MySQL | 数据库放大 | 强制集合式查询并增加调用次数测试 |
| 只覆盖当前页 | 汇总和排序错误 | 合并完成后统一汇总、排序和分页 |

## 15. 追踪矩阵

| 需求 | 功能 | 技术组件 | 验收标准 |
|---|---|---|---|
| 历史数据使用 MySQL | FR-01 | 历史快速路径 | AC-01 |
| 今天优先 Redis | FR-02、FR-03 | CurrentTokenUsageReader | AC-02、AC-03 |
| Redis 缺失时读修复 | FR-04、FR-05 | 原子修复组件 | AC-04、AC-06 |
| 不存在项不穿透 | FR-06 | 集合式查询 | AC-05、AC-10 |
| 报表实时一致 | FR-07 | Report Service | AC-07 |
| 配置页面实时一致 | FR-08 | Quota Admin Service | AC-08 |
| Redis 故障降级 | 异常处理 | Repository | AC-09 |
| 接口兼容 | API 兼容设计 | Handler/DTO | AC-11 |
| 保持数据结构和口径 | 数据模型 | 现有 schema/codec | AC-12 |
| 可观测 | 可观测性 | 日志与指标 | AC-13 |

## 16. 评审记录

| 版本 | 状态 | 内容 |
|---|---|---|
| v0.1 | 已被修订 | 初始 Redis/MySQL 合并及读修复方案 |
| v0.2 | 用户审核通过 | 明确集合式读取；删除负缓存；禁止 Redis miss 后逐项查询 MySQL；补充不存在筛选项和并发修复规则 |
| v1.0 | Final — user approved | 保留用户批准的 v0.2 实质内容并完成最终交付 |
