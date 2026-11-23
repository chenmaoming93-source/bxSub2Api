# 成长计划 sub2api 侧实现 — 代码审查报告

| 项 | 内容 |
|---|---|
| 报告版本 | **v1.1**（第二轮） |
| 审查日期 | 2026-11-20 |
| 审查基线 | `b7d767e..f843933`（3 个提交，132 文件，+5070/−2050） |
| 审查范围 | 全量 diff：后端 growth service/handler、ent schema、GoldenDB SQL 脚本、前端 growth 页面与新增工具 |
| 对照文档 | `GROWTH_PLAN_DESIGN.md` v6.2、`GROWTH_PLAN_AIWORK_API.md` v4.0、`GROWTH_PLAN_SUB2API_DEV.md` v4.2、`GROWTH_PLAN_PAGES_DESIGN.md` v1.3 |
| 首轮结论 | v1.0 报告 18 项问题，**本轮确认全部已处置** |
| 本轮结论 | **仍有 2 个 P0 需在上线前修复**：ONCE 任务语义退化、物理商品履约队列饥饿 |

---

## 1. 两轮对比摘要

首轮提出的 18 项（3 P0 / 6 P1 / 9 P2）在 `b7d767e..f843933` 中已全部处理，修复质量整体较高：加锁顺序、账务不变量、并发条件更新都是按根因改的，不是打补丁。

但改动引入了两个新的 P0：

1. **`ONCE` 任务的"一次性"语义失效** —— 多奖励改造把"同周期只能一次"换成了"按 limit 计数"，ONCE 的周期键恒为 `once`，唯一索引不再拦截；页面还新增了"支持大于 1"的提示，等于引导管理员踩坑。
2. **物理商品订单占满履约队列** —— 物理订单会创建异步 job，但调度器直接跳过它们，且查询按 `next_retry_at ASC LIMIT 50`，累计 50 个物理订单后虚拟商品（token 充值）永久不履约。

另有 5 个 P1 与 4 个 P2，详见第 3 节。

---

## 2. 验证结果

| 检查项 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./...` | 通过 |
| 静态检查 | `go vet ./internal/service/growth/... ./internal/handler/growth/... ./internal/pkg/timezone/...` | 无告警 |
| 单元测试 | `go test -tags=unit ./internal/service/growth/... ./internal/handler/growth/... ./internal/pkg/timezone/... ./ent/schema/...` | 全绿 |
| 建表脚本结构 | 对照 `ddl/182` 与 `ddl/185-189` | 最终态一致（列名、`sequence`、4 列唯一键均对齐） |

> 集成测试（`-tags=integration`）依赖 MySQL 实例，本次未执行。新增的 `growth_mysql_upgrade_scripts_integration_test.go`（319 行）覆盖了升级脚本，建议在 CI 的 integration job 中确认其通过。

---

## 3. 首轮问题处置情况

### 3.1 已修复（18/18）

| 原编号 | 问题 | 修复方式 | 位置 |
|---|---|---|---|
| P0-1 | `daily_limit`/`per_user_limit` > 1 失效 | claim 增加 `sequence`，唯一键改 4 列；limit 判定移入事务 | `completion_service.go:270` `completionSequence` |
| P0-2 | ABBA 死锁 | 加锁顺序统一为 `growth_tasks → growth_budgets`，并在两处加注释固化 | `budget_service.go:191, 260` |
| P0-3 | 预算可核减到低于已发放 | 约束改为 `nextTotal >= max(allocated, distributed)`；新增 `LockedAmount` 派生字段 | `budget_service.go:145-152` |
| P1-1 | 概览全量拉内存 | 改 SQL `SUM` + 两个 `Count` | `read_service.go:274-306` |
| P1-2 | 发分悲观锁串行 | 改条件 `UPDATE ... WHERE allocated - distributed >= ?` + `RowsAffected` 判定 | `transfer_service.go:124-147` |
| P1-3 | Online 额度叠加 | 语义改为"重置额度"，`allocationDelta = 目标 − 当前` | `task_service.go:513` |
| P1-4 | `Publish` 用 rollback 返回值 | 改显式 `_ = tx.Rollback(); committed = true` | `task_service.go:319-321` |
| P1-5 | 调账 key 二次生成 | 入口固定 `key`，并强制 `request_id` 必填 | `admin_service.go:154, 235` |
| P1-6 | 调账混入消费统计 | 新增 `ToAccountType` / `IsAdjustment`，调账走 `ADJUST` 账户且不计 `spent_total` | `transfer_service.go:246-275` |
| P2-1 | 核减错误码语义错 | 拆分 `BUDGET_BELOW_ZERO` / `BUDGET_BELOW_LOCKED` | `budget_service.go:25-26` |
| P2-2 | Revoke 错误码混用 | 拆分 `TASK_STATE_INVALID` / `TASK_QUOTA_EXCEEDED` | `budget_service.go:271-279` |
| P2-3 | `findExisting` 跨用户 | 增加 `transferUserID != userID` 校验 | `transfer_service.go:515-520` |
| P2-4 | `crypto.randomUUID` 非安全上下文 | 新增 `utils/requestId.ts`，含 `getRandomValues` 与 `Math.random` 两级降级 | `frontend/src/utils/requestId.ts` |
| P2-5 | 403 卡片文案重复 | 新增 `permissionDeniedHint` | `growth.zh.ts:23` |
| P2-6 | 菜单硬编码中文 | 新增 `growth.admin.menu` / `growth.user.menu` | `AppSidebar.vue` |
| P2-7 | 缺长度校验 | 新增 `validateTaskTextFields` | `task_service.go:817` |
| P2-8 | `_ = claimedID` 死变量 | 已删除 | `completion_service.go` |
| P2-9 | `maxPoints=0` 被忽略 | 改 `*int64` 可空参数 + `COALESCE` | `admin_service.go:88-94` |

配套改动同样到位：`182` 建表脚本已同步为最终结构（新列名 + `sequence` + 无 CHECK），`185-189` 提供老环境增量路径，`growth.zh/en.ts` 补齐了 5 个新错误码文案。

---

## 4. 本轮新发现问题

### 4.1 P0 级

#### P0-1 `ONCE` 任务的"一次性"语义失效

**位置**

- `backend/internal/service/growth/completion_service.go:270-320`（`completionSequence`）
- `backend/internal/service/growth/transfer_service.go:515-531`（`BusinessCycleKey`：`ONCE → "once"`）
- `frontend/src/i18n/locales/growth.zh.ts:183`（`multiRewardHint`）
- `backend/ent/schema/growth_reward_claim.go:57`（4 列唯一键）

**现象**

改造后判定"能否再发一次"只依赖 `daily_limit` / `per_user_limit` 两个数值，`period_type` 不再参与。对 `ONCE` 任务：

```
cycle_key 恒为 "once"
sequence = 该周期已有 claim 数 + 1   → 1, 2, 3, ...
唯一键 (user, task, "once", sequence) → 永不冲突
```

因此 `ONCE + per_user_limit=5` 的任务会在 5 天内发 5 次分。改造前这段逻辑是"该 cycle_key 存在任意 claim 即拒绝"，`ONCE` 天然只能一次。

**为什么这不是"管理员配错"**

本轮同时新增了 i18n 提示：

```
multiRewardHint: '每日上限和每人上限支持大于 1；同一奖励周期内可按上限多次发分。'
```

即页面会主动告诉管理员"可以填大于 1"。一个有"一次性成就"类任务（首次绑定、首单、实名认证）的管理员，看到这句提示把 `per_user_limit` 填成 2 试一下，就会得到一个可以领两次的"一次性"任务。

**影响**

- `ONCE` 类任务可被重复领取，直接造成超发
- 与 `WEEKLY`/`DAILY` 的语义错位：后两者的 `cycle_key` 会随周期翻滚，`per_user_limit` 才有"跨周期累计"的意义；`ONCE` 没有下一周期，`per_user_limit>1` 本身就没有合理语义
- 三条链路受影响：发分（`completionSequence`）、展示（`read_service.taskProgress`）、对账（claim 数与发放额一致，查不出来）

**修复方案（推荐组合 A + B）**

A. `ONCE` 强制单次，`per_user_limit` 不参与判定：

```go
// completion_service.go completionSequence
if rule.PeriodType == "ONCE" {
    if cycleClaims > 0 {
        return 0, errCompletionNotEligible
    }
    return 1, nil
}
```

B. 保存规则时校验 `ONCE` 的 `per_user_limit` 必须为 1：

```go
// task_service.go validateTaskRule
if rule.PeriodType == "ONCE" && rule.PerUserLimit != 1 {
    return infraerrors.BadRequest("INVALID_RULE",
        "ONCE tasks must use per_user_limit=1")
}
```

C. 页面在 `period_type=ONCE` 时锁定 `per_user_limit=1` 并隐藏 `multiRewardHint`（`TasksTab.vue:534-540`）。

> 若产品确实需要"一次性但可多次领取"的语义，那它就不该叫 `ONCE`，应新增 `LIMITED` 周期类型并显式写入契约与页面文案。当前状态是两种语义混在一个枚举值里。

---

#### P0-2 物理商品订单占满履约队列，虚拟商品永久不履约

**位置**

- `backend/internal/service/growth/order_service.go:239`（无条件创建 job）
- `backend/internal/service/growth/fulfillment_service.go:114-116`（跳过 PHYSICAL）
- `backend/internal/service/growth/fulfillment_service.go:93-108`（查询 `ASC LIMIT 50`）
- `backend/ent/schema/growth_fulfillment_job.go:33,36`（`PENDING` / `next_retry_at` 默认 `now`）

**现象**

兑换订单创建时无条件建履约 job：

```go
_, err = callbackTx.GrowthFulfillmentJob.Create().SetOrderID(order.ID).Save(callbackCtx)
```

job 默认 `job_status='PENDING'`、`next_retry_at = NOW()`。而调度器每 10 秒执行一次：

```go
if productType(order) == "PHYSICAL" {
    continue          // 不 claim、不置状态、不推进 next_retry_at
}
```

物理订单的 job 因此永远停留在 `PENDING`，且 `next_retry_at` 永远是最初那个时间（最早的）。

调度查询：

```go
Where(JobStatusIn("PENDING","RETRY"), NextRetryAtLTE(now), ...).
Order(ent.Asc(growthfulfillmentjob.FieldNextRetryAt)).
Limit(50)
```

`next_retry_at` 越老越靠前 → 这 50 个位置被物理订单永久占住。

**影响**

累计 50 个未手动完成的物理订单后：

- 所有虚拟商品（token 充值等）订单进入 `FULFILLING` 后永不推进
- 用户已扣积分、订单卡在履约中，只能等管理员逐个手动完成物理订单来"释放"队列
- 单一用户即可触发：连续兑换 50 个物理商品（若库存/限购允许）

即使是少量物理订单（如 10 个），也会每 10 秒产生 10 次无效 `Order.Get` + 快照解析，并挤占 20% 的每轮处理配额。

**修复方案（推荐 B）**

A. 物理商品不建 job：`Create` 中按 `product_type` 判断，`PHYSICAL` 直接 `order_status=FULFILLING` 且跳过 job 创建。最简单，但丢失 job 记录，无法用 job 做超时告警。

B. 物理商品建 job 但置为独立状态：

```go
// schema 增加 MANUAL
field.String("job_status").Default("PENDING").
    Validate(growthEnum("job_status", "PENDING","PROCESSING","RETRY","SUCCEEDED","FAILED","CANCELLED","MANUAL"))

// order_service.Create
job := callbackTx.GrowthFulfillmentJob.Create().SetOrderID(order.ID)
if productType == "PHYSICAL" {
    job.SetJobStatus("MANUAL")
}
_, err = job.Save(callbackCtx)
```

调度查询天然排除 `MANUAL`，`ProcessDue` 里的 `productType(order) == "PHYSICAL"` 判断可删除（少一次 `Order.Get`）。`ManualComplete` 已经会把 job 置为 `SUCCEEDED`，无需改动。

C. 短期止血（不改状态机）：调度查询按 `next_retry_at ASC` 改为 `job_status, next_retry_at` 并跳过物理订单前先 `SetNextRetryAt(now+24h)` 推迟。不推荐，治标。

---

### 4.2 P1 级

#### P1-1 调账增加缺少 `distributed` 上限校验

**位置**：`admin_service.go:185`、`transfer_service.go:157`

发分路径有保护：

```go
if budget.DistributedAmount > budget.TotalAmount-command.Points {
    return TransferResult{}, ErrPoolExhausted
}
```

调账增加只有"未分配额度"校验：

```go
if budget.TotalAmount-budget.AllocatedAmount < command.Points {
    return AdjustmentResult{}, ErrBudgetInsufficient
}
```

**场景**：`total=100`，任务全部下线并回收（`allocated=0`），历史调账累计 `distributed=90`。此时再调账 20：`100-0 >= 20` 通过 → `distributed=110 > total=100`。

P0-3 的核减方向已经用 `max(allocated, distributed)` 保护，但增长方向漏了。**同一不变量只守了一半。**

**修复**：

```go
// admin_service.go increase
if budget.DistributedAmount > budget.TotalAmount-command.Points {
    return AdjustmentResult{}, ErrBudgetInsufficient
}
```

或统一成 `locked := maxInt64(allocated, distributed); if locked+points > total { ... }`，与 `changeTotal` 共用同一个辅助函数，避免两处口径漂移。

---

#### P1-2 调账扣减不回退预算，额度被逐步"泄漏"

**位置**：`admin_service.go:236-290`

调账增加会 `budget.DistributedAmount += points`，但调账扣减只退用户钱包（`reverseRedeem` 语义），不回退 `DistributedAmount`。

**场景**：总预算 1000。调账 +100（`distributed=100`），一周后扣回 100（`distributed` 仍 100）。此时预算被永久占用 100，但用户手上没有积分。反复"发-扣"五轮后，`distributed=500`，实际流通 0，剩余可用预算只剩 500。

**影响**：预算口径失真，运营会误以为预算不够而不断扩容；对账公式 `TaskDistributed + AdjustmentIn == DistributedAmount` 仍然成立，掩盖了问题。

**修复**：明确 `distributed` 语义并二选一落地：

- 若为"累计发放"（当前实现），增加 `NetDistributed` 派生字段用于预算占用展示，运营按净额判断余量
- 若为"当前占用"，扣减时 `budget.DistributedAmount -= points`（需同步调对账公式，改为 `TaskDistributed + AdjustmentIn - AdjustmentOut == DistributedAmount`）

`ReconciliationReport.AdjustmentOut` 已经采集，说明设计上倾向于后者，但扣减路径还没接上。

---

#### P1-3 超额请求仍走完整事务，锁竞争被放大

**位置**：`completion_service.go:145-170`（`AfterLedger` 内调用 `completionSequence`）

`errCompletionNotEligible` 在 `AfterLedger` 回调中返回，而该回调执行前已经完成：

1. `createPendingTransfer`（写 transfer）
2. 条件 `UPDATE growth_tasks`（**锁任务行**）
3. `SELECT ... FOR UPDATE growth_budgets`（**锁预算行**）
4. `lockOrCreateWallet`（**锁钱包行**）
5. `createLedgerPair`（写 2 条流水）

然后才判定超限并整体回滚。

**影响**：对 `daily_limit=1` 的签到类任务，用户重复点击或 AIWork 重试时，每次都会被判超限，但每次都完整走一遍三把行锁 + 5 条写入。高峰期这会显著加剧 P0-2 修复后本已缓解的锁竞争。

**修复**：在 `AwardFromTask` 之前做一次无锁预检查（读当前 claim 计数），超限直接返回；事务内的 `completionSequence` 保留作为最终一致性的守门人。

```go
// Complete() 中，在调用 AwardFromTask 之前
if quick, err := s.quickEligibility(ctx, command.UserID, task.ID, cycleKey, dayStart, rule); err == nil && !quick {
    result.Result = "NOT_ELIGIBLE"
    return s.persistNonAward(ctx, command, result, now)
}
```

预检查失败不视为错误（并发下可能误判可用），只用于拦截明显超额，正确性仍由事务内校验保证。

---

#### P1-4 兑换限购校验无锁，并发可突破

**位置**：`order_service.go:144-157`

```go
if version.PerUserLimit > 0 {
    count, err := tx.GrowthExchangeOrder.Query().
        Where(user, product, OrderStatusIn("CREATED","FULFILLING","SUCCEEDED")).
        Count(ctx)
    if count >= version.PerUserLimit {
        return ExchangeCreateResult{}, ErrPurchaseLimitReached
    }
}
```

`Count` 是普通快照读，没有 `ForUpdate`，也没有唯一索引兜底（`(user_id, product_id, 第 N 件)` 不存在）。

**影响**：`per_user_limit=1` 的商品，用户双击兑换按钮（前端 `exchanging` 有保护，但两个标签页、脚本或重试可绕过，且两次请求的 `idempotency_key` 不同即视为两笔）会生成 2 个订单，都扣分成功。

**修复**：按商品维度串行化限购校验。最小改动是对商品库存行（已有 `ForUpdate`）的锁覆盖下再做 Count —— 当前 `inventory` 的 `ForUpdate` 在 Count 之后，把 Count 移到锁之后即可让同一商品的兑换请求串行，天然避免超限：

```go
inventory, err := tx.GrowthProductInventory.Query().
    Where(growthproductinventory.ProductIDEQ(product.ID)).ForUpdate().Only(ctx)
// ... 库存检查
// 再把 perUserLimit 的 Count 放在这里，利用同一把库存行锁串行化
```

代价是同商品的兑换完全串行，对成长计划体量可接受。若要更高并发，需要引入 `(user_id, product_id, seq)` 唯一键。

---

#### P1-5 `sequence` 计算带 `claim_status` 过滤，存在永久阻塞隐患

**位置**：`completion_service.go:279-288`

```go
cycleClaims, err := tx.GrowthRewardClaim.Query().
    Where(user, task, CycleKeyEQ(cycleKey), ClaimStatusEQ("SUCCEEDED")).
    Count(ctx)
...
return cycleClaims + 1, nil
```

`sequence` 由"成功 claim 数 + 1"推导，但唯一索引 `(user,task,cycle,sequence)` **不区分状态**。

**隐患**：一旦后续引入"撤销/退款"把 claim 改为 `REVERSED`/`FAILED`（表中已预留 `CANCELLED` 之类的扩展空间），这类记录不再计入 `cycleClaims`，下次计算仍得到同一个 `sequence` → 撞唯一索引 1062 → 上层转成 `ALREADY_COMPLETED` → **该用户在该周期永久无法再获得奖励**。

当前写入路径只产 `SUCCEEDED`，所以是"埋雷"而非"现症"。

**修复**：`sequence` 用实际最大值推导，不受状态影响：

```go
maxSeq, err := tx.GrowthRewardClaim.Query().
    Where(user, task, CycleKeyEQ(cycleKey)).
    Aggregate(ent.Max(growthrewardclaim.FieldSequence)).
    Int(ctx)
// 或原生 SQL: SELECT COALESCE(MAX(sequence),0) FROM growth_reward_claims WHERE ...
return maxSeq + 1, nil
```

`dailyClaims` / `perUserClaims` 的 limit 判定继续带 `SUCCEEDED` 过滤，两者各司其职。

---

### 4.3 P2 级

| ID | 位置 | 问题 | 建议 |
|---|---|---|---|
| P2-1 | `frontend/src/api/client.ts:101` + 各 view 的 `errorMessage` | 前端 `errorMessage` 优先取 `cause.reason`，而后端 `reason` 是机器码。40+ 错误码中只映射了 9 个，`POOL_EXHAUSTED`、`ORDER_STATE_INVALID`、`TASK_RULE_MISSING`、`PRODUCT_VERSION_CHANGED`、`INVENTORY_UNDERFLOW` 等会直接以裸码展示给用户 | 在 `growthRules.ts` 增加统一的 `resolveGrowthError(reason)`，未命中时回落 `growth.common.requestFailed`，禁止透传机器码 |
| P2-2 | 全量 diff | 本次提交混入大量非 growth 改动：`pkg/timezone` 新增 SQL 偏移函数（32 行）、`dashboard_aggregation_repo.go`（28 行）、`TokenQuotaDimensionValueSelector.vue`（73 行）、`account_repo_temp_unsched_test.go` 等。`timezone.Init` 会写全局 `time.Local`，影响所有模块 | 拆分提交。growth 上线若出问题，回滚会把无关的时区/统计改动一起带走，扩大排查面 |
| P2-3 | `sqlArchiving/growth/ddl/*.sql` | 增量脚本无幂等保护：`185` 重复执行报 1060，`186`/`187`/`189` 重复执行报"列/约束不存在"；`182` 用 `CREATE TABLE IF NOT EXISTS`，老环境误跑会静默跳过 | README 已按"按实际旧结构选择脚本"说明，但没有校验脚本。建议补一个 `SELECT` 自检清单（查 `information_schema` 输出当前结构），由 DBA 执行前比对 |
| P2-4 | `ddl/187` + `ent/schema/*` | 为 GoldenDB 兼容删除了全部 CHECK 约束，DB 层不再兜底 `available_points >= 0`、`distributed <= allocated` 等不变量，完全依赖应用层 | 已是明确的兼容取舍。建议在 `growth_wallet` / `growth_tasks` 的写路径集中加断言（或用 `ent` 的 `hooks`），并让 `Reconcile` 覆盖负余额检测 |

---

## 5. 上线前检查清单

| 项 | 说明 |
|---|---|
| 修 P0-1 | `ONCE` 强制单次，或新增独立周期类型；同步页面在 ONCE 下锁定 `per_user_limit=1` |
| 修 P0-2 | 物理订单不建 job 或建为 `MANUAL`；修完后确认 `ProcessDue` 能正常消费虚拟订单 |
| 修 P1-1 | 调账增加补 `distributed + points <= total` 校验 |
| 结构自检 | 在生产库执行 README 的脚本前，先核对 `information_schema.COLUMNS`，确认列名（`task_status`/`remark`/`claim_status`/`action_type` 等）与 `uq_growth_reward_claims_cycle` 的 4 列定义 |
| 集成测试 | 确认 `growth_mysql_upgrade_scripts_integration_test.go` 在 CI integration job 通过 |
| 回归范围 | P2-2 的时区与 Token 统计改动需单独回归，尤其是依赖"当天/当月"边界的看板 |

---

## 6. 已确认符合设计的项（复述）

| 项 | 说明 |
|---|---|
| 无 Web 签到 | W 面无 `checkins`；`CheckinCard.vue` 只读并引导至 AIWork |
| 任务无删除 | 无 `DELETE` 路由，`task_code` 永久占用，1062 → `TASK_CODE_DUPLICATED` |
| M/W 面不走 SM2 | 仅 U 面挂 `decryptor.Middleware()` |
| 预算模型 v6.1 | 单行 `growth_budgets` + 四类流水 + 任务额度；对账接口已补 `AdjustmentOut` |
| 加锁顺序 | 全局统一 `growth_tasks → growth_budgets → growth_wallets`，代码内已有注释固化 |
| 幂等三层 | Redis coordinator + `unique(biz_type, idempotency_key)` + 业务唯一键 |
| 契约 v4.0 | U 面 11 接口齐全；`/exchanges` 按 `product_id` 分流读写 |
| 写死 SM2 私钥 | DESIGN v5.1 既定决策，非缺陷 |

---

## 7. 关键文件索引

| 关注点 | 文件 |
|---|---|
| 完成判定与发分序列 | `backend/internal/service/growth/completion_service.go` |
| 发分事务 / 账户流水 | `backend/internal/service/growth/transfer_service.go` |
| 预算与额度 | `backend/internal/service/growth/budget_service.go` |
| 任务状态机与规则版本 | `backend/internal/service/growth/task_service.go` |
| 兑换下单 | `backend/internal/service/growth/order_service.go` |
| 履约调度与退款 | `backend/internal/service/growth/fulfillment_service.go` |
| 调账与对账 | `backend/internal/service/growth/admin_service.go` |
| 读写视图 | `backend/internal/service/growth/read_service.go` |
| 表结构 | `backend/ent/schema/growth_*.go`（19 个） |
| 建表与升级脚本 | `backend/sqlArchiving/growth/{ddl,dml}/`、`README.md` |
| 前端请求 ID | `frontend/src/utils/requestId.ts` |
| 前端错误文案 | `frontend/src/i18n/locales/growth.{zh,en}.ts` |

---

*v1.1 基于 `f843933` 快照，对 `b7d767e..f843933` 全量 diff 逐文件审查。P0-1 与 P0-2 的触发路径已通过代码推导确认；P0-2 建议在集成环境构造 50 个物理订单后观察一次 `ProcessDue` 的返回值验证。*
