# 模型路由同优先级账号池与路由级负载均衡实现计划

状态：Final，用户已批准  
版本：0.2  
日期：2026-08-17  
变更摘要：明确严格优先级降级过程，区分现有逻辑与本次改动，并将同优先级账号统一调度改为基于路由分配并发的负载均衡。

## 1. 背景与目标

当前 `groups.model_routing` 的候选处理方式是：

```text
按 candidate.priority 排序
→ 逐个候选处理
→ 当前候选内部选择账号
→ 当前候选不可用后才进入下一个候选
```

当前账号选择主要使用账号全局并发负载，因此同一模型路由下、相同候选优先级的账号无法组成统一负载池。

本次目标是：

- 将相同 `candidate.priority` 的所有候选账号合并成一个账号池；
- 使用该账号在当前 `group + route_alias` 下的路由级负载进行选择；
- 保持严格优先级，低数值 priority 的账号池全部无法使用后，才进入下一层；
- 当路由没有配置 `max_concurrency` 时，回退到账号全局并发负载；
- 使用账号自身 `model_mapping` 决定真实上游模型；
- 兼容现有旧格式和历史配置。

## 2. 本次范围与非目标

### 本次范围

- Gateway 模型路由选择逻辑；
- 同优先级候选合并；
- 路由级负载读取和计算；
- 路由级并发槽位参与调度；
- 路由配置重复账号校验；
- 候选 `model` 字段的兼容处理；
- 相关单元测试、集成测试和监控日志。

### 非目标

- 不改变账号自身 `model_mapping` 的语义；
- 不改变公开请求 API；
- 不实现不同 priority 之间的全局负载混合；
- 不实现按权重比例分流；
- 不让同一个 `group + route_alias + account` 同时属于多个 priority；
- 不改变普通非模型路由的账号调度策略。

## 3. 现状与本次变更

| 领域 | 当前现状 | 本次处理 |
|---|---|---|
| 候选 priority 排序 | 已存在，按数值升序 | 保留，并增加 priority 分层池 |
| 同优先级候选 | 当前仍逐候选处理 | 改为合并成统一账号池 |
| 账号负载 | 使用账号全局负载 | 优先改为路由级负载 |
| `max_concurrency` 为空 | 当前部分路径使用账号全局并发 | 明确为路由负载回退到账号全局负载 |
| 候选 `model` | 已解析、部分诊断路径仍携带 | 兼容读取但不参与实际模型选择 |
| 账号上游模型 | 使用账号 `model_mapping` | 保留 |
| 同别名重复账号 | 数据库唯一约束限制 | 增加配置层明确校验 |
| 路由/账号槽位获取 | 已存在 | 保留并用于最终原子确认 |
| 普通账号等待策略 | 已存在 | 本次不改变 |

## 4. 功能需求

以下 FR 只描述本次新增或修改的行为。

### FR-01：按 priority 形成调度层

匹配到一个路由别名后，系统需要：

1. 读取所有匹配的候选；
2. 按 `candidate.priority` 升序排序；
3. 将 priority 相同的候选划入同一个调度层；
4. 每个调度层形成一个独立账号池。

例如：

```json
[
  { "priority": 1, "account_ids": [101] },
  { "priority": 1, "account_ids": [102] },
  { "priority": 2, "account_ids": [103] }
]
```

应形成：

```text
层 1：priority=1，账号 [101,102]
层 2：priority=2，账号 [103]
```

### FR-02：同 priority 候选账号统一负载选择

同一个 priority 层中的所有账号必须统一参与负载比较。

不能再采用：

```text
candidate A 先选完
→ candidate B 再选
```

而应采用：

```text
priority=1 的所有账号
→ 统一计算 LoadRate
→ 选择最合适的账号
```

同一层中的选择顺序为：

```text
route LoadRate 最低
→ LastUsedAt 更早
→ 完全相同时随机打散
```

账号自身的全局 `Account.Priority` 不再作为同 priority 池的主要排序条件，避免账号被固定优先级长期偏置。

### FR-03：严格 priority 降级

严格 priority 降级的含义是：

> 只要当前 priority 层仍然存在一个可以成功获得所需槽位的账号，就绝不能选择更高数值 priority 层的账号。

具体流程如下。

#### 4.3.1 先处理最低 priority 层

系统首先处理数值最小的 priority 层。

对于该层的每个账号，依次进行：

1. 请求排除检查；
2. 账号是否存在；
3. 账号是否处于可调度状态；
4. 平台是否匹配；
5. 账号是否支持请求模型；
6. 模型级限流检查；
7. 配额检查；
8. 窗口费用检查；
9. RPM 检查；
10. 动态 token quota 检查；
11. 计算路由级或账号级回退 `LoadRate`。

不满足这些条件的账号只从当前层移除，不会直接导致整个请求失败。

#### 4.3.2 判断当前层是否有可用账号

经过过滤后：

- 如果当前层没有任何账号，当前层耗尽，进入下一层；
- 如果存在账号但全部 `LoadRate >= 100`，当前层耗尽，进入下一层；
- 如果存在可用账号，按照 `LoadRate` 排序并尝试获取槽位。

#### 4.3.3 处理并发抢占竞争

`LoadRate` 是读取时的快照，多个请求可能同时选择同一个账号。

因此排序后仍需逐个尝试：

```text
尝试获取路由槽位
→ 成功后尝试获取账号全局槽位
→ 两者都成功则选择该账号
→ 任一失败则释放已获取的槽位
→ 尝试当前层的下一个账号
```

只要当前层有一个账号最终成功获取槽位，就立即返回，不再查看下一 priority 层。

#### 4.3.4 当前层什么时候算“耗尽”

当前层只有在以下情况之一发生时，才算耗尽：

- 所有账号都被静态条件过滤；
- 所有账号路由负载达到上限；
- 所有账号账号级回退负载达到上限；
- 所有账号的路由槽位获取都失败；
- 所有账号的账号槽位获取都失败；
- 所有账号都被请求排除集合排除；
- 所有账号动态配额不可用。

此时才允许进入下一个 priority 层。

#### 4.3.5 上游请求失败后的重新选择

账号选择成功后，如果实际转发请求返回可重试的上游失败：

1. 当前账号加入本次请求的排除集合；
2. 重新进入同一路由别名的调度；
3. 仍优先尝试当前最低可用 priority 层的其他账号；
4. 当前层全部排除或不可用后，才进入下一 priority 层。

因此，上游错误不会让系统直接跳过整个 priority 层。

#### 4.3.6 所有层都耗尽

如果所有 priority 层都没有账号可以成功获取槽位：

- 返回模型候选耗尽错误或无可用账号错误；
- 不把更低优先级之外的普通账号池作为隐式兜底；
- 是否进入系统其他 fallback group，继续沿用现有外层 fallback 机制。

### FR-04：使用路由级 LoadRate

对于配置了 `max_concurrency` 的路由账号：

```text
route LoadRate =
route 当前活跃并发数
/
route max_concurrency
× 100
```

其中：

- 当前活跃并发数来自该路由账号的路由槽位；
- 不将路由等待数计入该指标；
- `LoadRate >= 100` 时，该账号在当前 priority 层不可用。

### FR-05：无路由并发配置时回退账号全局负载

如果对应的 `max_concurrency` 为 `null`：

```text
LoadRate =
账号全局当前活跃并发数
/
账号全局最大并发数
× 100
```

账号全局最大并发数使用：

```go
account.EffectiveLoadFactor()
```

该回退只影响负载比较和可用性判断，最终仍需通过账号全局并发槽位获取确认。

### FR-06：候选 `model` 字段降级为兼容字段

候选中的 `model`：

- 允许历史配置继续携带；
- 解析时不报错；
- 不参与候选匹配；
- 不参与上游模型选择；
- 不作为路由级并发或 quota 的模型标识；
- 可在兼容诊断信息中保留一段时间。

最终上游模型始终从选中账号的：

```go
account.FirstModelMappingValue()
```

获取。

### FR-07：同一别名账号唯一性校验

对于同一个 `group + route_alias`：

- 同一 priority 内重复账号应去重；
- 同一账号出现在不同 priority 中应报配置错误；
- 不允许依赖数据库唯一键异常来反馈配置错误；
- 错误信息需要指出 route alias、账号 ID 和冲突 priority。

## 5. 已有逻辑的保留范围

以下逻辑不是本次新增需求，但需要保持：

- `model_routing` 旧数组格式兼容；
- 候选 priority 数值越小越优先；
- 账号静态可调度性检查；
- 平台、模型能力、配额、RPM 和窗口费用检查；
- sticky session 的现有外层行为；
- route slot 和 account slot 的最终并发控制；
- 请求失败后的账号排除和重试机制；
- 普通非模型路由的负载选择；
- `group_model_route_accounts` 的现有配置投影机制。

## 6. 技术设计

### 6.1 主要组件调整

| 组件 | 调整 |
|---|---|
| Domain parser | 保留旧格式，增加重复账号语义校验 |
| GatewayService | 从逐候选循环改为 priority 分层账号池 |
| ConcurrencyService | 增加路由槽位批量负载读取 |
| Redis concurrency cache | 读取 route slot 当前活跃数 |
| Route repository | 继续读取 `max_concurrency` |
| Admin handler/UI | 提前校验跨 priority 重复账号 |
| Metrics/logging | 记录层级降级和 route LoadRate |

### 6.2 内部数据结构

建议新增内部结构：

```go
type RoutedAccountCandidate struct {
    Account             *Account
    Priority             int
    RouteMaxConcurrency *int
    CurrentConcurrency   int
    LoadRate             int
    UpstreamModel        string
}
```

其中：

- `Priority` 表示候选 priority；
- `RouteMaxConcurrency == nil` 表示使用账号全局负载回退；
- `UpstreamModel` 来自账号 mapping；
- 不再依赖候选 `model` 字段。

### 6.3 路由负载读取

Redis 路由槽位当前使用：

```text
concurrency:route:{groupID}|{routeAlias}|{accountID}
```

需要增加批量读取能力：

```go
GetRouteLoadsBatch(ctx, requests []RouteLoadRequest)
```

读取流程：

1. 批量清理过期槽位；
2. 批量读取各路由槽位当前数量；
3. 关联每个账号的 `max_concurrency`；
4. 计算 route LoadRate；
5. 返回给 GatewayService 排序。

### 6.4 最终槽位获取

负载读取只负责排序，不能替代并发控制。

最终流程仍然是：

```text
route LoadRate 排序
→ AcquireRouteSlot
→ AcquireAccountSlot
→ 成功后返回账号
```

如果已经获取路由槽位但账号槽位获取失败，必须释放路由槽位。

## 7. 数据模型与迁移

现有表继续使用：

```text
group_model_route_accounts
```

核心字段：

| 字段 | 含义 |
|---|---|
| `group_id` | 分组 ID |
| `route_alias` | 路由别名 |
| `account_id` | 账号 ID |
| `max_concurrency` | 该账号在该路由上的并发分配，可为空 |

现有唯一约束继续保留：

```text
(group_id, route_alias, account_id)
```

不新增数据库字段。

需要新增或调整：

- 配置保存前重复账号校验；
- 历史重复配置检查；
- 路由负载读取接口；
- Redis route slot 读取逻辑。

## 8. 核心伪代码

```text
function selectModelRoute(requestedModel):
    candidates = matchRoute(requestedModel)
    tiers = groupByPriority(candidates)

    for tier in tiers ordered by priority ascending:
        accounts = flatten(tier.account_ids)
        accounts = deduplicate(accounts)

        eligible = []

        for account in accounts:
            if excluded(account):
                continue
            if not staticSchedulable(account):
                continue

            routeConfig = getRouteConfig(routeAlias, account.id)

            if routeConfig.max_concurrency != null:
                current = getRouteCurrentConcurrency(routeKey)
                loadRate = current * 100 / routeConfig.max_concurrency
            else:
                current = getAccountCurrentConcurrency(account.id)
                loadRate = current * 100 / account.EffectiveLoadFactor()

            if loadRate >= 100:
                continue

            eligible.append(account, loadRate)

        sort eligible by:
            loadRate ascending
            LastUsedAt ascending
            random for exact ties

        for item in eligible:
            routeSlot = acquireRouteSlotIfConfigured(item)
            if not routeSlot.acquired:
                continue

            accountSlot = acquireAccountSlot(item.account)
            if not accountSlot.acquired:
                release(routeSlot)
                continue

            upstreamModel = item.account.FirstModelMappingValue()

            return selected account, upstreamModel,
                   release(routeSlot + accountSlot)

        continue to next priority tier

    return candidates exhausted
```

## 9. 验证策略

### 单元测试

新增或调整测试：

- priority 分层；
- 同 priority 账号合并；
- 同层 LoadRate 最低优先；
- route `max_concurrency` 计算；
- `max_concurrency == null` 回退账号负载；
- 当前层静态过滤后进入下一层；
- 当前层所有账号负载达到 100% 后进入下一层；
- 当前层并发抢占全部失败后进入下一层；
- 当前层一个账号成功后不访问下一层；
- 上游失败后重新选择仍遵循 priority；
- 候选 `model` 不影响最终上游模型；
- 跨 priority 重复账号报错。

### 集成测试

覆盖：

- Redis route slot 当前并发读取；
- 路由槽位和账号槽位同时获取；
- 槽位获取失败时正确释放；
- 多请求并发选择同一 priority 池；
- 路由级负载和账号级回退负载混合存在；
- 旧格式和新格式配置；
- 配置同步和唯一约束。

### 验收标准

- 只要 priority=1 中存在可成功获取槽位的账号，就不会选择 priority=2；
- priority=1 全部不可用后，系统才选择 priority=2；
- 同 priority 账号按 route LoadRate 而不是候选声明顺序选择；
- 路由 `max_concurrency` 配置生效；
- 未配置路由并发时能正确使用账号全局负载；
- 候选 `model` 的变化不会改变真实上游模型；
- 同别名跨 priority 重复账号会被明确拒绝；
- 所有并发槽位在失败路径中正确释放。

## 10. 实施顺序

### 阶段一：领域和配置校验

- 明确 priority 分层数据结构；
- 增加重复账号校验；
- 保持旧格式兼容；
- 标记候选 `model` 为兼容字段。

### 阶段二：路由负载读取

- 增加 route slot 批量读取；
- 实现 route LoadRate；
- 实现 `max_concurrency == null` 时的账号全局回退；
- 增加 Redis 和并发服务测试。

### 阶段三：Gateway 调度重构

- 替换逐候选处理为 priority 分层；
- 合并同 priority 账号；
- 按 route LoadRate 排序；
- 保留最终 route/account slot 原子获取；
- 处理层级降级和重试排除。

### 阶段四：管理端和运维

- 增加配置重复校验提示；
- 增加调度日志和指标；
- 完成端到端回归测试；
- 验证历史配置兼容性。

## 11. 风险与应对

| 风险 | 应对 |
|---|---|
| 负载快照与真实抢槽之间存在竞争 | 最终以原子槽位获取结果为准 |
| 路由槽位读取失败导致负载不准确 | 记录错误，按安全策略跳过或回退 |
| 历史配置存在跨 priority 重复账号 | 上线前检查并给出明确修复信息 |
| 路由分配并发超过账号总能力 | 保存时校验总量 |
| 同优先级账号容量不同 | 使用百分比 LoadRate |
| 路由槽位与账号槽位不一致 | 先获取路由槽位，再获取账号槽位，失败时释放已获取资源 |
| 删除候选 `model` 影响历史配置 | 保留兼容解析，不立即删除字段 |
| 同 priority 池变大后读取成本上升 | 批量读取 Redis 和路由配置 |

## 12. 需求追踪矩阵

| 需求 | 对应模块 | 验收标准 |
|---|---|---|
| 同 priority 合并账号 | FR-01、FR-02 | 同层统一比较 LoadRate |
| 严格 priority 降级 | FR-03 | 当前层成功时不进入下一层 |
| 路由级负载 | FR-04 | 使用 route 当前并发和分配上限 |
| 空配置回退 | FR-05 | 使用账号全局负载 |
| 删除候选模型依赖 | FR-06 | 上游模型只来自账号 mapping |
| 重复账号约束 | FR-07 | 跨 priority 重复配置被拒绝 |

## 13. Review Record

### v0.1

初始 Draft，包含同 priority 账号池、路由级负载、严格 priority 降级和候选 `model` 兼容处理。

### v0.2

根据用户反馈调整：

- 详细定义 FR-03 的层级耗尽、槽位竞争、上游失败重试和最终失败逻辑；
- 区分现有逻辑与本次新增/修改需求；
- 删除把已有行为重复描述为新 FR 的内容；
- 明确 `max_concurrency == null` 时回退到账号全局负载；
- 明确候选 `model` 不参与运行时模型选择。

### v0.2 审批

用户已明确批准本 Plan，可进入 MVP 拆解和后续实现。
