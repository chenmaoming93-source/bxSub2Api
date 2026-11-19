# 成长计划 sub2api 侧实现 — 代码审查报告

| 项 | 内容 |
|---|---|
| 报告版本 | v1.0 |
| 审查日期 | 2026-11-18 |
| 审查基线 | `b7d767e` feat: 成长计划初版 |
| 审查范围 | `backend/internal/{service,handler,server}/growth`、`backend/ent/schema/growth_*`、`frontend/src/{views,components,api,i18n}/growth`、路由与 RBAC 接入点 |
| 对照文档 | `GROWTH_PLAN_DESIGN.md` v6.2、`GROWTH_PLAN_AIWORK_API.md` v4.0、`GROWTH_PLAN_SUB2API_DEV.md` v4.2、`GROWTH_PLAN_PAGES_DESIGN.md` v1.3 |
| 审查结论 | **不建议直接全量上线**。存在 3 个 P0 级账务/并发缺陷，其中 1 个会导致已配置功能静默失效 |

---

## 1. 结论摘要

实现完成度高于预期：契约 U 面 11 个接口全部落地，M 面 34 个管理接口、W 面 8 个用户接口均超出页面设计 v1.3 的规划范围；预算模型从"多池"收敛为"总预算 + 任务额度"已完整实现，含对账接口与审计流水。

但存在 3 个必须先修的缺陷：

1. **`daily_limit` / `per_user_limit` 配置大于 1 时静默失效** —— 唯一索引与周期键设计冲突，用户只能拿 1 次，且接口返回成功。
2. **额度划拨与任务发分加锁顺序相反** —— 并发时触发 MySQL ABBA 死锁（1213）。
3. **预算可被核减到低于已发放额** —— 破坏"总预算 ≥ 已发放"账务不变量，前端进度条可溢出到 400%。

P1 中另有 2 项建议本迭代内处理：W 面概览的全量内存聚合（随数据增长必然劣化）、任务重新上线的额度叠加。

---

## 2. 验证方法与结果

### 2.1 自动化检查

| 检查项 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./...` | 通过，无错误 |
| 静态检查 | `go vet ./internal/service/growth/... ./internal/handler/growth/...` | 无告警 |
| 单元测试 | `make test-unit`（等价 `go test -tags=unit ./...`） | 全绿 |
| 前端类型 | `npx vue-tsc --noEmit -p tsconfig.json` | 无错误 |
| CI 配置 | `.github/workflows/backend-ci.yml` | 已执行 `make test-unit` + `make test-integration` + `make test-frontend` |

> **执行注意**：growth 全部测试文件带 `//go:build unit` 或 `//go:build integration` 标签。直接跑 `go test ./internal/service/growth/...` 会显示 `[no test files]`，容易被误判为"没有测试"。本地验证必须使用 `make test-unit`。CI 已正确配置，不构成缺口。

### 2.2 代码规模

| 层 | 文件数 | 行数 |
|---|---|---|
| 后端 service（含测试） | 40 | 9,432 |
| 后端 handler / routes | 16 | — |
| ent schema（growth_*） | 19 | — |
| 前端 views / components / api | 26 | 3,452 |

### 2.3 覆盖度核对（对照契约 v4.0）

| 契约条目 | 实现位置 | 结论 |
|---|---|---|
| #1 `POST /growth/me` | `routes/growth.go:13` → `UMe` | 一致 |
| #2 `POST /growth/tasks` | `:14` → `UTasks` | 一致 |
| #3 `POST /growth/tasks/{task_id}` | `:15` → `UTask` | 一致 |
| #4 `POST /growth/ledger` | `:17` → `ULedger` | 一致，游标分页 |
| #5 `POST /growth/products` | `:18` → `UProducts` | 一致 |
| #6 `POST /growth/products/{product_id}` | `:19` → `UProduct` | 一致 |
| #7 `POST /growth/exchanges`（列表） | `:20` → `UExchanges` | 一致 |
| #8 `POST /growth/checkins` | `:22` → `UCheckin` | 一致 |
| #9 `POST /growth/tasks/{task_id}/completion-requests` | `:16` → `UCompletion` | 一致，同步发分 |
| #10 `POST /growth/exchanges`（命令） | `:20` 按 `params.product_id > 0` 分流 | 一致 |
| #11 `POST /growth/exchanges/{order_id}` | `:21` → `UOrder` | 一致 |

> #7 与 #10 共用同一 `POST /growth/exchanges`，由 `read_handler.go:199` 的 `params.ProductID > 0` 判定分流。实现符合契约，但这是契约层面的隐含约定：若 AIWork 侧在列表请求里误带 `product_id`，会被当作兑换命令执行并扣分。建议在契约 v4.1 中显式写明该分流规则，或在服务端加 `status`/`page` 与 `product_id` 互斥校验。

---

## 3. 问题清单

严重程度定义：

- **P0** —— 账务错误、数据不一致、配置功能静默失效，上线前必须修复
- **P1** —— 性能劣化、并发瓶颈、异常路径报错，本迭代内处理
- **P2** —— 语义、文案、健壮性问题，可排期

---

### 3.1 P0 级

#### P0-1 `daily_limit` / `per_user_limit` 大于 1 时静默失效

**位置**

- `backend/internal/service/growth/completion_service.go:238-251`（创建 claim）
- `backend/internal/service/growth/completion_service.go:271-292`（1062 兜底分支）
- `backend/ent/schema/growth_reward_claim.go:56`（唯一索引）
- `backend/internal/service/growth/transfer_service.go:515-531`（`BusinessCycleKey`）

**现象**

管理员在任务规则里配置 `daily_limit=3`，用户当天第 2、3 次完成任务时：

- 接口返回 HTTP 200，`result=ALREADY_COMPLETED`
- `awarded_points=0`，钱包无变动
- 无任何错误日志或告警

用户侧看到"已完成"，管理员侧看到"任务已配置 3 次"，双方都不会察觉异常。

**根因**

`growth_reward_claims` 的唯一索引是 `(user_id, task_id, cycle_key)`，而 `cycle_key` 的粒度由周期类型决定：

```
DAILY  → "2026-11-18"      （一天一个值）
WEEKLY → "2026-W47"        （一周一个值）
ONCE   → "once"            （永久一个值）
```

同一用户在同一周期内的第 2 次发分，必然违反该唯一索引并返回 MySQL 1062。该错误在 `completion_service.go:271` 被识别为"重复提交"，转换为 `ALREADY_COMPLETED` 正常响应返回。

结论：**`daily_limit > 1` 与 `per_user_limit > 1` 在当前数据模型下不可能生效**，只有恰好等于 1 时才与唯一索引语义自洽。而 `validateTaskRule`（`task_service.go:797-808`）只校验 `>= 0`，前端表单（`TasksTab.vue:181-182`）也允许任意正整数，等于埋了一个静默失效的开关。

**影响**

- 所有"每日可完成 N 次"类任务从第 2 次起不发分
- 用户投诉时无从排查：服务端无错误、流水无记录、对账平衡
- 一旦上线并产生用户预期，后期修复需补发历史积分

**修复方案（二选一，需产品确认）**

方案 A —— 支持一个周期内多次（功能完整，改动较大）

在 claim 表增加序号维度，唯一索引改为 `(user_id, task_id, cycle_key, sequence)`：

```go
// ent/schema/growth_reward_claim.go
index.Fields("user_id", "task_id", "cycle_key").Unique(),  // 原
index.Fields("user_id", "task_id", "cycle_key", "sequence").Unique(), // 新
```

`completion_service.go` 建 claim 前先取同周期已发次数作为 `sequence`，同时删除 `dailyClaims` / `perUserClaims` 的"先查后写"逻辑（改为在事务内基于 `sequence` 计数判断，避免竞态）。

方案 B —— 显式限制为每周期一次（改动最小，约 30 分钟）

```go
// task_service.go validateTaskRule
if rule.DailyLimit > 1 {
    return infraerrors.BadRequest("INVALID_RULE",
        "daily_limit cannot exceed 1: one reward per business cycle")
}
if rule.PerUserLimit > 1 {
    return infraerrors.BadRequest("INVALID_RULE",
        "per_user_limit cannot exceed 1: one reward per cycle key")
}
```

同步调整前端表单上限与 i18n 提示，并在任务列表把"每日 N 次"列改为"每周期 1 次"。

> **建议**：若短期内无"一天多次签到/多次完成"的业务需求，选 B —— 它把错误提前到配置阶段，而不是发分阶段。若确实需要多次，选 A，但需同步评估 U 面 `/growth/me` 的签到状态查询与 `checkinStreak` 的连续性计算。

---

#### P0-2 额度划拨与任务发分加锁顺序相反，存在 ABBA 死锁

**位置**

- `backend/internal/service/growth/budget_service.go:188-195`（`Allocate`：先 budget 后 task）
- `backend/internal/service/growth/budget_service.go:255-261`（`Revoke`：先 budget 后 task）
- `backend/internal/service/growth/transfer_service.go:122-132`（`awardFromTaskTx`：先 task 后 budget）

**现象**

运营在后台给某任务追加额度，同时有用户完成该任务触发发分，两个事务互相等待对方持有的行锁，MySQL 检测到后回滚其中一个，返回错误 1213 `Deadlock found when trying to get lock`。

**根因**

```
Allocate / Revoke:   LOCK growth_budgets(id=1)  →  LOCK growth_tasks(id=X)
awardFromTaskTx:     LOCK growth_tasks(id=X)    →  LOCK growth_budgets(id=1)
```

两条路径对同两个资源的加锁顺序相反，构成教科书式的 ABBA 死锁。由于 `growth_budgets` 是单行账户（`id=1`），所有预算类操作本身已经串行，真正交叉点是与发分事务的锁顺序冲突。

**影响**

- 发分失败：U 面返回 500，AIWork 侧需重试；若重试逻辑不完善，用户当次奖励丢失（额度未扣，可人工补）
- 划拨失败：管理员看到"系统错误"，重复点击可能造成重复划拨（虽有 `request_id` 幂等，但依赖前端每次生成新 UUID）
- 触发条件是并发，测试环境难复现，生产高峰期概率显著上升

**修复方案**

统一为 `awardFromTaskTx` 的 **task → budget** 顺序，改动集中在 `Allocate` 与 `Revoke`：

```go
// budget_service.go Allocate：调换顺序
task, err := tx.GrowthTask.Query().Where(growthtask.ID(taskID)).ForUpdate().Only(ctx)
if ent.IsNotFound(err) {
    return BudgetAccount{}, ErrGrowthTaskNotFound
}
if err != nil {
    return BudgetAccount{}, err
}
budget, err := ensureBudgetForUpdate(ctx, tx)   // 移到 task 之后
if err != nil {
    return BudgetAccount{}, err
}
```

`Revoke` 同样处理。`changeTotal` 只锁 budget，无交叉，无需改动。

同时建议在 `budget_service.go` 顶部加注释固化该约定，避免后续新增事务再次反向加锁：

```go
// 加锁顺序约定：growth_tasks → growth_budgets。
// 任何同时触达两张表的事务必须按此顺序加锁，否则会与发分路径形成 ABBA 死锁。
```

---

#### P0-3 预算可被核减到低于已发放金额

**位置**

- `backend/internal/service/growth/budget_service.go:140-148`（`changeTotal` 的 REDUCE 校验）
- `backend/internal/service/growth/admin_service.go:202-204`（调账增加推高 distributed）
- `frontend/src/views/growth/admin/BudgetTab.vue:41`（`maxReduce`）

**现象**

设总预算 100、全部分配给任务后任务下线并回收剩余 100（`allocated=0`）、管理员通过人工调账给用户发了 80（`distributed=80`）。此时前台"核减"按钮的可用上限是 `total - allocated = 100`，把预算核减到 20 会成功。结果：

- `total=20 < distributed=80`
- `DistributedRate = distributed / total = 4.0`，前端进度条宽度 400%
- `UnallocatedAmount = total - allocated = 20`，仍可继续向新任务分配 20，相当于总共发出 100 分而总预算只有 20

**根因**

核减校验只覆盖"已分配"维度：

```go
nextTotal = budget.TotalAmount - mutation.Points
if nextTotal < budget.AllocatedAmount {
    return BudgetAccount{}, ErrBudgetAllocatedLocked
}
```

但 `DistributedAmount` 的构成是"任务发分 + 人工调账增加"，其中人工调账部分不占用 `AllocatedAmount`（设计如此：调账直接消耗未分配预算）。因此 `distributed` 可以大于 `allocated`，而校验遗漏了这一维度。

**影响**

账务不变量 `0 ≤ distributed ≤ total` 被破坏，且对账接口（`admin_service.go:302`）只校验 `TaskDistributed + AdjustmentIn == DistributedAmount`，不会发现该问题。属于"能通过对账但账目已经错了"的类型。

**修复方案**

后端：

```go
// budget_service.go changeTotal
locked := budget.AllocatedAmount
if budget.DistributedAmount > locked {
    locked = budget.DistributedAmount
}
nextTotal = budget.TotalAmount - mutation.Points
if nextTotal < locked {
    return BudgetAccount{}, ErrBudgetBelowLocked   // 新增错误码，见 P2-1
}
```

前端同步：

```ts
// BudgetTab.vue
const maxReduce = computed(() => {
  const total = budget.value?.total_amount ?? 0
  const locked = Math.max(
    budget.value?.allocated_amount ?? 0,
    budget.value?.distributed_amount ?? 0,
  )
  return Math.max(0, total - locked)
})
```

并建议给 `BudgetAccount` 增加派生字段 `locked_amount`（= max(allocated, distributed)），前端只消费该字段，避免口径在两端各写一遍。

---

### 3.2 P1 级

#### P1-1 W 面概览全量拉取内存聚合

**位置**：`backend/internal/service/growth/read_service.go:281-306`

```go
monthEntries, err := s.client.GrowthAccountLedger.Query().
    Where(/* user_id, WALLET, IN, created_at >= monthStart */).
    All(ctx)                       // 当月全部流水拉进内存
for _, entry := range monthEntries {
    result.monthEarned += entry.Points
}

userTasks, err := s.client.GrowthUserTask.Query().
    Where(growthusertask.UserIDEQ(userID)).
    All(ctx)                       // 用户历史全部 user_task 拉进内存
for _, task := range userTasks {
    if task.Status == "IN_PROGRESS" { result.inProgress++ }
    if task.RewardedAt != nil && !task.RewardedAt.Before(dayStart) { result.completedToday++ }
}
```

**影响**：`growth_user_tasks` 的行数按"用户 × 任务 × 周期"增长，DAILY 任务一天一行。活跃用户一年可积累上千行，且该查询在每次打开用户积分页时执行（`/growth` 与 `/api/v1/user/growth/overview` 共用）。随数据增长必然劣化，属于"上线即开始倒计时"的问题。

**修复**：改用 SQL 聚合，分别走已有索引 `(user_id, created_at, id)` 与 `(user_id, task_id, cycle_key)`。

```go
monthEarned, err := s.client.GrowthAccountLedger.Query().
    Where(/* 同上条件 */).
    Aggregate(ent.Sum(growthaccountledger.FieldPoints)).
    Int(ctx)

inProgress, err := s.client.GrowthUserTask.Query().
    Where(growthusertask.UserIDEQ(userID), growthusertask.StatusEQ("IN_PROGRESS")).
    Count(ctx)

completedToday, err := s.client.GrowthUserTask.Query().
    Where(growthusertask.UserIDEQ(userID), growthusertask.RewardedAtGTE(dayStart)).
    Count(ctx)
```

三条查询替代两次全量加载，时间复杂度从 O(行数) 降为 O(1) 索引扫描。

---

#### P1-2 发分在任务行锁上串行化

**位置**：`backend/internal/service/growth/transfer_service.go:122-149`

当前实现用悲观锁 + 读改写：

```go
task, err := tx.GrowthTask.Query().Where(growthtask.ID(command.TaskID)).ForUpdate().Only(ctx)
if task.AllocatedPoints-task.DistributedPoints < command.Points {
    return TransferResult{}, ErrPoolExhausted
}
task, err = tx.GrowthTask.UpdateOneID(task.ID).
    SetDistributedPoints(task.DistributedPoints + command.Points).Save(ctx)
```

**影响**：同一任务的所有发分事务在 `growth_tasks` 行锁上排队。对签到、每日任务这类"早高峰集中完成"的场景，会成为明显瓶颈（串行度 = 任务数量，而非并发用户数）。

**修复**：改为设计文档要求的条件更新，由影响行数判定额度是否充足。ent 无法表达列与列比较，直接用原生 SQL：

```go
res, err := tx.ExecContext(ctx, `
UPDATE growth_tasks
SET distributed_points = distributed_points + ?, updated_at = CURRENT_TIMESTAMP(6)
WHERE id = ? AND allocated_points - distributed_points >= ?`,
    command.Points, command.TaskID, command.Points)
rows, _ := res.RowsAffected()
if rows == 0 {
    return TransferResult{}, ErrPoolExhausted
}
```

> 注意：改完后 `remainingTaskQuota` 的计算需重新读一次任务行（或直接用 `allocated - distributed` 的返回值），当前依赖读改写后的局部变量。

---

#### P1-3 任务重新上线时额度叠加

**位置**：`backend/internal/service/growth/task_service.go:503-508`

```go
task, err = tx.GrowthTask.UpdateOneID(taskID).
    SetAllocatedPoints(task.AllocatedPoints + initialAllocation).   // 累加
    SetStatus("ACTIVE").
```

**现象**：任务下线时若未勾选"回收剩余额度"（`Offline` 的 `revokeRemaining=false`），`allocated_points` 会保留剩余值；重新上线时再叠加 `initialAllocation`，实际可用额度 = 旧剩余 + 新划拨，与管理员的心理预期（重新划拨就是"这次给这么多"）不一致。

**影响**：预算被隐形多占，直到任务额度用完才暴露。

**修复（二选一）**

- 方案 A：`SetAllocatedPoints(initialAllocation)`，语义改为"重新设定额度"，与 `Revoke` 之后再 `Allocate` 等价
- 方案 B：保持累加，但在在线弹窗中显式回显"当前剩余 X 分，本次追加 Y 分，上线后可用 X+Y 分"，由管理员确认

建议 A，语义最简单，且避免"下线后忘记回收"这一类操作失误。

---

#### P1-4 `Publish` 依赖 `tx.Rollback()` 的返回值表达成功

**位置**：`backend/internal/service/growth/task_service.go:309-317`

```go
if ent.IsNotFound(err) {
    if task.Status == "ACTIVE" {
        current, currentErr := currentPublishedRuleTx(ctx, tx, taskID)
        if currentErr == nil {
            return buildTaskView(task, current, nil), tx.Rollback()
        }
    }
    return TaskListView{}, ErrTaskRuleMissing
}
```

**问题**：用 `tx.Rollback()` 的 `error` 作为函数返回值。rollback 成功时返回 `nil`，handler 于是把"没有待发布规则"当作成功响应 200；rollback 失败时返回非 nil，反而变成错误。语义倒置且脆弱 —— 一旦 rollback 返回 `sql.ErrTxDone`（外层 defer 已执行），行为会翻转。

同时 `committed` 仍为 `false`，`defer` 中会再次 rollback（返回 `ErrTxDone`，被 `_ =` 忽略）。

**修复**：

```go
if currentErr == nil {
    _ = tx.Rollback()
    committed = true        // 阻止 defer 二次 rollback
    return buildTaskView(task, current, nil), nil
}
```

或更清晰：把"已是终态、无需发布"作为显式哨兵值返回，由 handler 映射为 200。

---

#### P1-5 调账扣减在无 `request_id` 时已扣款却报错

**位置**：`backend/internal/service/growth/admin_service.go:230-279`

```go
_, err = s.transfers.debitWalletTx(ctx, tx, WalletDebitCommand{
    IdempotencyKey: adjustmentKey(command),   // 第 1 次生成
    ...
})
// ... commit ...
transfer, err := s.client.GrowthTransfer.Query().
    Where(growthtransferPredicate(command)).  // 第 2 次生成
    Only(ctx)
if err != nil {
    return AdjustmentResult{}, err          // 已提交，钱已扣
}
```

```go
func adjustmentKey(command AdjustmentCommand) string {
    if strings.TrimSpace(command.RequestID) != "" {
        return "adjust:" + command.RequestID
    }
    return fmt.Sprintf("adjust:%d:%d:%d", command.UserID, command.OperatorID, time.Now().UnixNano())
}
```

**问题**：`request_id` 为空时，`adjustmentKey` 每次调用都生成不同值（含 `UnixNano`）。第二次查询必然 `NotFound`，函数返回错误，但事务在上一行已经 commit。

**影响**：前端 `UserPointsTab.vue:152` always 传 `crypto.randomUUID()`，生产路径被掩盖；但 M 面接口本身允许不传 `request_id`（`adjustmentRequest.RequestID` 无 `binding:"required"`），一旦有脚本/第三方接入就会触发"扣款成功但报错"，进而重复调用造成重复扣款。

**修复**：入口处固定一次 key。

```go
func (s *AdminGrowthService) decrease(ctx context.Context, command AdjustmentCommand) (AdjustmentResult, error) {
    key := adjustmentKey(command)          // 只生成一次
    // ... 后续全部使用 key
}
```

同时给 `adjustmentRequest` 的 `RequestID` 加 `binding:"required"`，与 `CreateBudgetEntry`（其 `RequestID` 亦为可选）保持一致策略 —— 建议所有写接口的 `request_id` 一律必填。

---

#### P1-6 人工调账扣减与兑换消费混算

**位置**：`backend/internal/service/growth/admin_service.go:250-261`

`decrease` 复用 `debitWalletTx`，该函数把资金的去向固定为 `REDEEMED` 账户，并累加 `spent_total`。

**影响**：

- 用户端"已消费"包含被管理员扣回的积分，语义错误
- 流水里 `account_type=REDEEMED` 同时承载"兑换消耗"和"人工扣减"，后续按账户维度统计会失真
- 对账接口的 `AdjustmentIn` 只统计增加方向，扣减方向无对应字段，报表不对称

**修复**：为调账扣减引入独立账户类型 `ADJUST`（不改表结构，`account_type` 是字符串枚举），或至少在 ledger 的 `biz_type='ADJUST'` 基础上，前端把两类消费分列展示。建议前者，一致性最好：

```go
// debitWalletTx 增加 toAccount 参数，默认 "REDEEMED"，调账传 "ADJUST"
func (s *TransferService) debitWalletTx(ctx context.Context, tx *ent.Tx, command WalletDebitCommand, toAccount string) (TransferResult, error)
```

---

### 3.3 P2 级

| ID | 位置 | 问题 | 建议 |
|---|---|---|---|
| P2-1 | `budget_service.go:141-143` | 核减金额超过总预算时返回 `BUDGET_ALLOCATED_LOCKED`（"被任务锁定"），实际原因是"不能减到负数" | 拆分错误码 `BUDGET_BELOW_ZERO`，与 P0-3 新增的 `BUDGET_BELOW_LOCKED` 并列 |
| P2-2 | `budget_service.go:269-275` | `Revoke` 的"任务非 OFFLINE"与"超过剩余额度"共用 `TASK_QUOTA_INVALID` | 拆为 `TASK_STATE_INVALID` + `TASK_QUOTA_EXCEEDED`，前端才能给准确提示 |
| P2-3 | `transfer_service.go:480-513` | `findExisting` 仅按 `(biz_type, idempotency_key)` 查询，不校验用户归属；跨用户幂等键冲突时会把他人 `points` / `current_balance` 返回 | 增加 `transfer.ToUserID == command.UserID` 校验，不符返回 409 `IDEMPOTENCY_CONFLICT` |
| P2-4 | 11 处（`UserGrowthView.vue:113`、`UserPointsTab.vue:152`、`TasksTab.vue`×6、`ProductsTab.vue`×4、`OrdersTab.vue`×2） | `crypto.randomUUID()` 仅在安全上下文（HTTPS / localhost）可用。内网 HTTP 部署时抛 `TypeError`，调账、兑换、发布全部失效 | 抽 `utils/requestId.ts`，`crypto.randomUUID?.() ?? manualUUID()`，`manualUUID` 用 `crypto.getRandomValues` 拼 v4 |
| P2-5 | `AdminGrowthView.vue:65-66` | 403 卡片的标题与正文都渲染 `t('growth.common.permissionDenied')` | 正文改用 `growth.common.permissionDeniedHint` |
| P2-6 | `AppSidebar.vue:682, 739` | 菜单文案硬编码中文 `'我的积分'` / `'成长计划'`，i18n 已有 `growth.*` 命名空间却未使用 | 改为 `t('growth.user.menu')` / `t('growth.admin.menu')`。注意：同文件 `'角色权限'` 等亦为硬编码，属既有惯例，可一并对齐 |
| P2-7 | `task_service.go:781-795` | `validateTaskCreate` 未校验 `title` / `description` / `icon` / `task_code` 长度与字符集 | 增加长度上限（与 schema 的 `MaxLen` 对齐），超限返回 400 而非触发数据库错误 500 |
| P2-8 | `completion_service.go:179, 299` | `claimedID` 声明后从未使用，用 `_ = claimedID` 消音 | 删除变量 |
| P2-9 | `admin_service.go:92` | `if maxPoints >= 0 && maxPoints != 0` 导致 `max_points=0` 被忽略，无法筛选"0 积分用户" | 改用 `c.Query` 是否存在判断，或统一为 `-1` 表示不限 |

---

## 4. 已确认符合设计的项

| 项 | 说明 |
|---|---|
| 无 Web 签到 | W 面无 `checkins` 路由；`CheckinCard.vue` 只读并引导至 AIWork，符合决策① |
| 任务无删除 | 无 `DELETE` 路由、schema 无 `deleted_at`；`task_code` 永久占用，1062 映射为 `TASK_CODE_DUPLICATED`(409) |
| M/W 面不引入 SM2 | 仅 U 面挂载 `decryptor.Middleware()`；浏览器侧不持有密钥，符合同源会话模型 |
| 预算模型 v6.1 | `growth_budgets` 单行 + `growth_budget_entries` 四类流水，`allocated` / `distributed` 分列，表数 17 |
| 完成判定在 AIWork | `completion_service` 不查 `usage_logs`，直接按规则发分；契约 v4.0 的 `POOL_EXHAUSTED` 语义已标注为"任务额度耗尽" |
| 幂等三层 | Redis `IdempotencyCoordinator` + `unique(biz_type, idempotency_key)` + 业务唯一键（`completion_ref` / `cycle_key`） |
| 非对称加密 v5.1 | SM2 私钥写死服务端常量（`growth_payload.go:25`），无 `kid` 轮换 —— 与 DESIGN v5.1 的既定决策一致，非缺陷 |
| RBAC 接入 | `growth.read` / `growth.manage` / `growth.adjust` / `growth.self.read` 四码齐备，`ValidateRouteCoverage` 通过 |
| 页面操作唯一性 | 页头仅保留"刷新"，无重复主操作按钮，符合页面设计 v1.3 的验收项 |

> **关于写死私钥的补充说明**：该方案是 DESIGN v5.1 的明确决策（"私钥保密靠代码库访问控制"），本报告不作为缺陷列出。但建议在代码库层面补充：仓库访问审计、禁止 fork 公开、以及 `growthSM2PrivateKeyHex` 的变更流程文档（换密钥需与 AIWork Main 同步发版，过渡期跨版本请求会报 `CRYPTO_KEY_UNWRAP_FAILED`）。

---

## 5. 修复排期建议

| 优先级 | 项 | 工作量 | 依赖 |
|---|---|---|---|
| 第 1 批（上线前必须） | P0-1 方案 B（限制为 1） | 0.5 天 | 需产品确认"是否支持一天多次" |
| | P0-2 加锁顺序统一 | 0.5 天 | 无 |
| | P0-3 核减约束 + 错误码 | 0.5 天 | 需新增 `BUDGET_BELOW_LOCKED` |
| 第 2 批（本迭代） | P1-1 概览聚合改 SQL | 0.5 天 | 无 |
| | P1-3 上线额度语义 | 0.5 天 | 需确认"重设 vs 追加" |
| | P1-5 调账 key 固定 | 0.2 天 | 无 |
| 第 3 批（下迭代） | P1-2 条件更新替代悲观锁 | 1 天 | 需压测验证 |
| | P1-4 / P1-6 | 1 天 | P1-6 涉及账务口径 |
| 第 4 批（随手） | P2-1 ~ P2-9 | 1 天 | 可合并为一个清理 PR |

**验证要求**

1. P0-2 需补并发测试：N 个 goroutine 同时发分 + 1 个划拨，断言无 1213（`make test-integration` 环境）。
2. P0-1 与 P0-3 需补集成测试覆盖边界：`daily_limit=1` 正常、`=3` 被拒绝（方案 B）或正常发 3 次（方案 A）；`total` 核减至 `distributed` 边界。
3. 所有新增错误码需同步补充前端 i18n（`growth.zh.ts` / `growth.en.ts`），否则用户看到裸错误码。

---

## 6. 附录：关键文件索引

| 关注点 | 文件 |
|---|---|
| U 面路由（11 接口） | `backend/internal/server/routes/growth.go` |
| W 面路由（8 接口） | `backend/internal/server/routes/user_growth.go` |
| M 面路由（34 接口） | `backend/internal/server/routes/admin.go:165-200` |
| SM2 信封中间件 | `backend/internal/server/middleware/growth_payload.go` |
| 加解密实现 | `backend/internal/service/growth/crypto/envelope.go` |
| 预算与划拨事务 | `backend/internal/service/growth/budget_service.go` |
| 任务状态机与规则版本 | `backend/internal/service/growth/task_service.go` |
| 发分事务（核心） | `backend/internal/service/growth/transfer_service.go` |
| 完成请求（同步发分） | `backend/internal/service/growth/completion_service.go` |
| 签到 | `backend/internal/service/growth/checkin_service.go` |
| 调账与对账 | `backend/internal/service/growth/admin_service.go` |
| 读写视图 | `backend/internal/service/growth/read_service.go` |
| 表结构 | `backend/ent/schema/growth_*.go`（19 个） |
| 管理端页面 | `frontend/src/views/growth/AdminGrowthView.vue` + `admin/*.vue` |
| 用户端页面 | `frontend/src/views/growth/UserGrowthView.vue` + `user/*.vue` |
| i18n | `frontend/src/i18n/locales/growth.{zh,en}.ts` |

---

*报告基于 `b7d767e` 快照的静态审查与自动化检查。P0-1 / P0-2 的触发条件已通过代码路径推导确认，建议在集成环境补充复现用例后再合入修复。*
