# sub2api 成长计划｜页面设计（管理端 + 用户端）

> 日期：2026-11-17 ｜ 依赖：总设计 [GROWTH_PLAN_DESIGN.md](GROWTH_PLAN_DESIGN.md) v6.1、后端设计 [GROWTH_PLAN_SUB2API_DEV.md](GROWTH_PLAN_SUB2API_DEV.md) v4.1
> 本文只描述 **sub2api 自有前端页面**（管理端 + Web 用户端），不涉及 AIWork 界面。
> **核心约束：低侵入**——以新增文件为主；4 个既有文件只做「追加」，不改既有条目（路由为纯追加的 2 条，其余 3 处各 1–2 行）。
>
> **v1.0 → v1.1 修订（2026-11-17）**：① 预算模型改为**单一总预算 + 任务额度分配**（总预算手动增减、新建任务划拨额度、额度耗尽拦截），管理端 Tab1 由「积分池」重做为「总预算与任务额度」；② **用户端不提供 Web 签到**，签到卡改只读引导；③ 调账不做双人复核（仅 RBAC + 审计）。
>
> **v1.1 → v1.2 修正（2026-11-17）**：**去掉页头的 tab 级主操作按钮**（原「增加预算 / 新建任务 / 新建商品 / 导出 CSV」随 tab 变文字，与各 tab 工具条重复）。确立**操作唯一性**原则：主操作只出现在所属 tab 的工具条内，页头仅保留全局「刷新」；用户积分 tab 的「导出 CSV」下沉到该 tab 工具条。
>
> **v1.2 → v1.3 补充（2026-11-17）**：明确**任务不提供删除**——无「删除」入口、无 `DELETE` 接口，下线（`OFFLINE`）是唯一移除手段且可「重新上线」（须重新划拨额度）；补齐 `POST /tasks/:id/online` 接口、任务三态可用操作表、`task_code` 不可修改不可复用 + 新建即时唯一校验。

---

## 0. 设计目标与边界

### 0.1 目标

| # | 目标 | 说明 |
|---|---|---|
| 1 | 管理员一站式管理 | 总预算与任务额度、任务与规则、兑换品与库存、兑换订单、用户积分与调账，集中在 `/admin/growth` 一个模块内，用 tab 切换 |
| 2 | 管理者可查普通用户 | 用户积分总览 + 单用户抽屉（积分/流水/兑换记录三合一），支持搜索与排序 |
| 3 | 普通用户可感知成长 | `/growth` 一页看完：当前积分、签到状态、可做任务、兑换商品预览、积分获取明细 |
| 4 | 低侵入 | 不修改既有页面、store、组件内部实现；4 处登记式改动全部为追加，不改既有条目 |
| 5 | 零新增依赖 | 复用现有 Vue3 + Tailwind + 自研组件，不引入 UI 库、图表库 |

### 0.2 明确不做

- 不在 sub2api 前端实现「任务完成判定」——判定在 AIWork，Web 端只展示任务与引导（见 §5.3）。
- 不把 AIWork 的 SM2 加密信封逻辑引入浏览器：**SM2 私钥绝不下发前端**。
- 不复刻 AIWork 的成长中心 UI，不与其争抢入口，Web 端定位为「只读概览 + 兑换」——**签到只能在 AIWork 发起**，Web 端只展示签到状态与引导。
- 不做图表报表（趋势图后续可基于 `/admin/growth/stats` 增量加，不在首版）。

### 0.3 页面清单（共 2 个路由、5+3 个视图区）

| 路由 | 页面 | 使用者 | 权限 |
|---|---|---|---|
| `/admin/growth` | 成长计划管理（tab：总预算 / 任务 / 兑换品 / 兑换订单 / 用户积分） | 管理员 | `growth.read`，写操作用 `growth.manage`，调账用 `growth.adjust` |
| `/growth` | 我的积分（tab：总览 / 兑换中心 / 明细） | 登录用户 | `growth.self.read` |

> 只加 2 条路由是刻意的：**tab 化**把 8 个视图区收敛成 2 个路由，路由/菜单/权限矩阵各只需 +1 条管理项 +1 条个人项，显著降低对既有导航体系的侵入。

---

## 1. 接口平面口径（先说清，避免误用契约）

| 平面 | 路径前缀 | 认证 | 调用方 | 是否在冻结契约内 |
|---|---|---|---|---|
| **U 面** | `/api/v1/growth/*` | SM2 数字信封 | AIWork Main | ✅ 契约 v4.0 全文定义，**本次不改一行** |
| **M 面** | `/api/v1/admin/growth/*` | 现有 `AdminIdentityAuth` + RBAC | sub2api 管理前端 | ❌ 契约范围外，本文给页面所需字段 |
| **W 面** | `/api/v1/user/growth/*` | 现有 `jwtAuth`（Bearer / Cookie） | sub2api 用户前端 | ❌ 本次新增 |

三条平面**共用同一套 service / 账务事务**，只是入口认证不同：

```text
AIWork Main ──SM2──► /api/v1/growth/*        ─┐
浏览器(用户) ─JWT──► /api/v1/user/growth/*   ─┼─► GrowthService ─► 账务事务（任务额度→钱包 / 钱包→REDEEMED）
浏览器(管理员)─JWT+RBAC► /api/v1/admin/growth/* ─┘
```

**关键取舍**：M/W 面**不做 SM2**。它们与 sub2api 同源、走既有会话认证，浏览器里放公钥/做国密封装只会增加攻击面且毫无收益；SM2 只解决「AIWork 不是 sub2api 前端、没有会话」的问题。

---

## 2. 信息架构与导航

### 2.1 导航落点（只加 2 个菜单项）

```text
侧边栏
├─ [管理区]（管理员可见）
│   ├─ 仪表盘 / 渠道 / 用户 / 分组 / …
│   ├─ ★ 成长计划            → /admin/growth        ← 新增，放在「用户」之后
│   └─ 系统设置 / …
└─ [我的账户]
    ├─ 密钥 / 用量 / 订阅 / 订单 / 兑换码
    ├─ ★ 我的积分            → /growth             ← 新增，放在「兑换码」之后
    └─ 个人资料
```

- 管理端菜单用一个**分组父项**（`expandOnly`）还是**单条目**？首版用**单条目**：`成长计划` → `/admin/growth`，tab 在页内切换。理由：分组需要 5 个子项，会把侧边栏拉长且与「渠道管理/订单管理」的分组风格重复；单条目 + 页内 tab 更轻。
- 图标复用现有 `GiftIcon`（已在 `AppSidebar.vue` 内定义，无需新增图标组件）；管理端与用户端复用同一图标，符合"同一个功能"的语义。

### 2.2 管理端 tab 结构

```text
/admin/growth?tab=budget|tasks|products|orders|users
├─ 页头：成长计划 · 副标题 · 全局操作（仅「刷新」）
├─ Tab 条：总预算与任务额度 │ 任务管理 │ 兑换品管理 │ 兑换订单 │ 用户积分
└─ 内容区（TablePageLayout：actions / filters / table / pagination 四段式）
```

- **操作唯一性（v1.2 修正）**：**任何操作在页面上只出现一次，且只出现在它所属的 tab 工具条内**。页头**不放**随 tab 变化的主操作按钮——否则「增加预算 / 新建任务 / 新建商品 / 导出 CSV」会与各自 tab 的 `actions` 区重复，用户会问"点哪个"。页头只承担与 tab 无关的全局动作（刷新）。
- tab 状态写入 query（`?tab=tasks`），可分享、可刷新保持、可被侧边栏恢复。
- 非法 tab 回落到 `budget`。
- 首屏只加载当前 tab 的数据（懒请求）；切换 tab 不整页重载。

### 2.3 用户端 tab 结构

```text
/growth?tab=overview|store|ledger
├─ 顶部 Hero：当前可用积分 + 累计获得/消耗 + 本月获得 + 签到卡
├─ Tab：积分总览 │ 兑换中心 │ 积分明细
└─ 内容区
```

---

## 3. 管理端页面设计

### 3.1 Tab 1｜总预算与任务额度 `tab=budget`

**目标**：管理员掌握「一共有多少分、划给任务多少、已经发出去多少、还能给任务分多少」，并能增加/核减总预算、给任务追加或回收额度。

> **模型（v1.1）**：全局只有**一个总预算账户**，与任务解耦。新增任务时从总预算**划拨**一份额度给该任务，发放只消耗任务额度。不再有签到池/任务池/兑换池三个池。

**布局**

```text
┌────────────────┬────────────────┬────────────────┬────────────────┐
│ 总预算 120,000 │ 已分配 78,000  │ 已发放 42,150  │ 未分配 42,000  │  ← 4×StatCard
│ 累计批准       │ 已划给任务      │ 实际到用户钱包  │ 可再划拨       │
│                │ 未消耗部分 35,850│ 发放率 35.1%   │                │
└────────────────┴────────────────┴────────────────┴────────────────┘
[+ 增加预算] [− 核减预算]                                    [预算流水]
┌─ 任务额度分配 ──────────────────────────────── 共 4 个任务 ────────┐
│ 任务          │ 已分配 │ 已发放 │ 剩余  │ 使用率      │ 状态   │ 操作  │
│ 每日签到      │ 20,000 │  8,850 │ 11,150│ ▓▓▓▓░ 44.3% │ 充足   │追加/回收│
│ 首次调用 API  │ 30,000 │ 21,300 │  8,700│ ▓▓▓▓▓ 71.0% │ 预警   │追加/回收│
│ 连续签到 7 天 │ 25,000 │ 11,000 │ 14,000│ ▓▓▓░ 44.0%  │ 充足   │追加/回收│
│ 邀请同事体验  │  3,000 │  1,000 │  2,000│ ▓▓ 33.3%    │ 充足   │追加/回收│
└────────────────────────────────────────────────────────────────────┘
┌─ 预算变动流水（最近 10 条）──────────────────────────── 查看全部 ──┐
│ 11-17 16:20 │ 划拨 │ −3,000 → 邀请同事体验 │ 预算 120,000 / 已分配 78,000 │ 张三 │
│ 11-15 10:00 │ 增加 │ +20,000                │ 预算 120,000 / 已分配 75,000 │ 李四 │
└────────────────────────────────────────────────────────────────────┘
```

**统计卡口径**

| 卡片 | 字段 | 说明 |
|---|---|---|
| 总预算 | `total_amount` | 累计批准（TOPUP − REDUCE），手动增减的净额 |
| 已分配 | `allocated_amount` | 已划给任务、尚未消耗的额度；同时显示「已分配给任务的总额」（含已发放）|
| 已发放 | `distributed_amount` | 已到用户钱包的累计；副标题给发放率 = 已发放 / 总预算 |
| 未分配 | `total − allocated` | **就是新增任务时能划的上限**；为 0 时「新建任务」按钮提示先增加预算 |

**操作与交互**

| 操作 | 弹窗内容 | 校验与提示 |
|---|---|---|
| **增加预算** | 增加积分（正整数）+ 原因 + 备注 | 提交后 `total += N`；toast「总预算已增至 120,000」 |
| **核减预算** | 核减积分 + 原因（必填 ≥5 字） | 前端预校验 `当前总预算 − N ≥ 已分配`；不满足直接禁用提交并提示「已划拨给任务的 78,000 分不能核减，请先回收任务额度」。后端同样返回 `BUDGET_ALLOCATED_LOCKED` |
| **划拨额度**（新建任务时） | 见 §3.2 任务表单：必填「初始额度」 | 上限为「未分配」；超出即时报错 |
| **追加额度** | 追加积分 + 原因 | 上限同上；追加后任务状态可能由「预警/耗尽」回到「充足」 |
| **回收额度** | 展示可回收剩余额度，输入回收数量（默认全部）+ 原因 | 仅 `OFFLINE` / 已结束任务可回收；**已发放部分不可回收**，弹窗内明确写出 |
| 预算流水 | 抽屉：时间、类型、变动数、预算余额、已分配余额、关联任务、原因、操作人 | 只读，可筛选类型 |

- **危险操作统一二次确认**：核减预算、回收额度均需二次确认，红字说明「写入审计日志，不可撤销」。
- **额度状态色阶**：使用率 `<60%` 充足（teal）、`60–85%` 预警（amber）、`>85%` 紧张（rose）、`剩余 = 0` 耗尽（rose 实心 badge，行内提示「该任务已停止发分」）。
- 空态：「还没有设置总预算」+ 主按钮「设置初始预算」+ 说明「总预算是所有任务奖励的唯一来源；新增任务时需要从这里划拨额度」。
- 权限：`growth.read` 可见；增加/核减/划拨/回收需 `growth.manage`。

### 3.2 Tab 2｜任务管理 `tab=tasks`

**目标**：管理任务目录与奖励规则版本，明确"任务有哪些、做一次给多少、一天能做几次"。

**布局**：标准表格页（筛选行 + 表格 + 分页）。

| 列 | 字段 | 说明 |
|---|---|---|
| 任务 | `title` + `code`（副行小字） | 点击进详情抽屉 |
| 类型 | `CHECKIN` / `DAILY` / `ACHIEVEMENT` | badge |
| 奖励积分 | 当前生效规则 `points` | 右侧小字 `v3` |
| 周期 | `DAILY` / `ONCE` / `WEEKLY` | |
| 上限 | `daily_limit` / `per_user_limit` | 形如 `1/日 · 1/人` |
| **任务额度** | `allocated_points` / `distributed_points` | 形如 `20,000 / 已发 8,850`，剩余用进度条；点击跳 Tab1 并高亮该行 |
| 状态 | `DRAFT` / `ACTIVE` / `OFFLINE` | badge；额度耗尽时额外显示「额度耗尽」rose badge |
| 更新时间 | `updated_at` | |
| 操作 | 编辑 / 发布 / 下线 / 重新上线 / 额度 / 规则历史 | 按状态显隐，见下方三态表；**始终无「删除」** |

**关键交互**

1. **新建/编辑任务抽屉**（宽 640）：基础信息（名称、编码、描述、图标）+ **预算划拨（初始额度，必填）** + 奖励规则（积分、周期、每日上限、每人上限、生效时间）+ 前端文案（任务卡片副标题、完成引导语）。
2. **新建时划拨额度**：抽屉顶部显示「未分配预算 42,000」，初始额度输入框实时校验 `0 < N ≤ 未分配`；提交时后端在同一事务内完成「建任务 + ALLOCATE」，**不允许先建任务后补额度**（避免零额度任务上线）。额度不足时禁用提交并给「去增加预算」快捷入口。
3. **编辑不自动改额度**：编辑奖励积分不改变已分配额度，只提示「按当前额度还可发放 X 次」；需要更多额度走「追加额度」。
4. **发布 = 生成新规则版本**：编辑已上线任务的奖励字段时，保存不直接生效，而是产生 `rule_version+1` 的待发布规则；点「发布」后写入生效时间。列表显示「已发布 v3 / 待发布 v4」双标记，避免"改了没生效"的经典困惑。
5. **规则历史抽屉**：时间倒序的版本列表（版本号、积分、上限、生效区间、操作人），可对比相邻版本（仅高亮差异字段，不做 diff 视图）。
6. **下线（唯一的"移除"手段）**：二次确认「下线后用户端不再展示该任务，已发放积分不受影响」；弹窗内提供「同时回收剩余额度（X 分）」勾选项，并必填下线原因。
7. **不提供删除（v1.3 决策）**：任务表**没有删除入口**，后端也**不提供 `DELETE` 接口**。理由：`growth_task_rules` / `growth_completion_requests` / `growth_account_ledger` 都以 `task_id` 反查，物理删除会让历史发分无法追溯，与"账务不可变"冲突；软删又会让用户看过的任务从目录消失、与流水对不上。**下线就是终态入口**。
8. **下线后可重新上线**：下线任务的剩余额度已回收，因此「重新上线」必须重新划拨额度（`POST /admin/growth/tasks/:id/online`，参数 `initial_allocation`，从总预算未分配中划出）；历史已发放记录沿用，不占新额度。
9. **任务编码不可修改、不可复用**：`task_code` 创建后锁定，用于流水反查；下线后编码仍被占用，新建任务不能复用同名编码。因此**新建时对编码做即时唯一性校验**（失焦即查，避免提交后才发现冲突）——这是"不能删除"下最重要的一道防线。
10. **规则历史抽屉**：时间倒序的版本列表（版本号、积分、上限、生效区间、操作人），可对比相邻版本（仅高亮差异字段，不做 diff 视图）。
11. **校验**：积分必须为正整数；提示「按 20,000 额度可发放约 2,000 次」，不做硬性阈值拦截（额度本身就是硬上限）。

**状态与可用操作**

| 状态 | 含义 | 可用操作 |
|---|---|---|
| `DRAFT` | 已建任务、规则未发布 | 编辑 · **发布** · 规则历史 · 额度（追加/回收） |
| `ACTIVE` | 规则已生效、用户端展示 | 编辑（生成待发布规则） · **发布新版本** · **下线** · 额度 · 规则历史 |
| `OFFLINE` | 已下线、用户端不展示 | 编辑 · **重新上线**（须重新划拨额度） · 规则历史 |

> 三态之间**没有"删除"**；`OFFLINE` 是终态入口，可回到 `ACTIVE`（需重新划拨额度），不可删除。

**空态**：`还没有任务` + 「新建任务」+ 提示「新建任务需从总预算划拨额度；任务完成后由 AIWork 发起完成请求，sub2api 只负责算分与发放」。

### 3.3 Tab 3｜兑换品管理 `tab=products`

**目标**：维护可兑换商品、定价版本、库存与限购。

**布局**：表格与卡片双视图切换（右上角 segmented control），默认**表格**（信息密度高），卡片用于快速看视觉。

| 列 | 字段 |
|---|---|
| 商品 | 图标 + `name` + `product_code` |
| 类型 | `TOKEN_QUOTA`（Token 额度）/ `SUBSCRIPTION`（订阅时长）/ `PHYSICAL`（实物/权益） |
| 所需积分 | 当前版本 `points_cost` + `v2` |
| 库存 | `stock`（`-1` 显示「不限」），≤10 标黄，0 标红「已售罄」 |
| 限购 | `per_user_limit`（`0`=不限） |
| 履约方式 | 自动发放 / 人工处理 |
| 状态 | `DRAFT` / `ACTIVE` / `OFFLINE` |
| 操作 | 编辑 / 上下架 / 库存调整 / 版本历史 |

**关键交互**

- **定价即版本**：改价/改权益描述 → 生成新 `product_version`，旧订单仍按旧版本履约。列表 hover 版本号显示「v2 生效于 11-17 14:00」。
- **库存调整**：弹窗输入「增加/减少数量」+ 原因；扣减到 0 后自动展示「已售罄」但仍可上架（用户端置灰）。
- **上架前校验**：必须已绑定履约方式；`TOKEN_QUOTA` 类型必须填「每次发放 token 数量」，否则阻断并提示。
- **删除**：不提供。只能 `OFFLINE`（历史订单需要商品快照）。
- 卡片视图：商品图/图标、名称、积分价、库存条、上架开关，适合运营快速扫。

### 3.4 Tab 4｜兑换订单 `tab=orders`

**目标**：跟踪扣分与履约，处理失败重试与退款。

| 列 | 字段 |
|---|---|
| 订单号 | `order_no`（可复制） |
| 用户 | 用户名/邮箱（点击打开用户抽屉） |
| 商品 | 名称 + `product_version`（旧版本用灰 badge） |
| 扣分 | `points_cost`（负数展示，红色） |
| 状态 | `CREATED` / `FULFILLING` / `SUCCEEDED` / `FAILED` / `REFUNDED` |
| 履约 | 自动/人工 + 重试次数（`0/3`） |
| 创建时间 | 精确到秒 |
| 操作 | 详情 / 重试履约 / 退款 |

- **筛选**：状态（多选 tab 式）、用户关键词、时间范围（复用 `DateRangePicker`）、商品。
- **详情抽屉**：状态时间线（创建→扣分→履约中→成功/失败→退款）+ 关联 `transfer_id` + 失败原因原文 + 人工处理备注输入。
- **重试履约**：仅 `FAILED` 可用；提示「将按原订单重新发起履约，不会重复扣分」。
- **退款**：仅 `FAILED` 且未退款可用；二次确认（红），必填原因；执行 `REDEEMED 账户 → 用户钱包` 反向转账（不占用发放预算），成功后状态 `REFUNDED` 并展示 `refund_transfer_id`。
- **风险提示条**：当页存在 `FAILED` 超过 24h 的订单时，顶部黄条「有 3 笔订单履约失败超过 24 小时，建议处理」+ 「只看失败」快捷筛选。

### 3.5 Tab 5｜用户积分 `tab=users`

**目标**：回答"每个普通用户有多少分、怎么来的、兑换了什么"。

| 列 | 字段 | 交互 |
|---|---|---|
| 用户 | 头像 + 用户名 + 邮箱副行 | 点击打开用户抽屉 |
| 当前积分 | `available_points`（teal 加粗） | 支持列排序 |
| 累计获得 | `total_earned` | 排序 |
| 累计消耗 | `total_spent` | 排序 |
| 最近变动 | `last_changed_at`（相对时间 + tooltip 绝对时间） | |
| 操作 | 查看明细 / 调账 | |

- **筛选**：关键词（用户名/邮箱）、积分区间（0 / 1-100 / 101-1000 / >1000）、排序（余额降序默认）。
- **用户抽屉**（宽 720，三块）：

```text
┌ 用户：alice@corp.com ────────────────────────────────┐
│ 可用 260 │ 累计获得 320 │ 累计消耗 60 │ 签到 12 天      │  ← 概览条
├ 积分流水（游标分页，20/页）───────────────────────────┤
│ 时间 │ 类型 │ 说明 │ 变动 │ 余额快照 │ 关联业务        │
│ 11-17 15:43 │ 签到 │ 每日签到 │ +10 │ 260 │ checkin#66001 │
│ 11-17 14:20 │ 兑换 │ 100万Token │ -50 │ 250 │ order#88001   │
├ 兑换记录（最近 5 条 + 查看全部）──────────────────────┤
│ 时间 │ 商品 │ 扣分 │ 状态 │ 履约                        │
└──────────────────────────────────────────────────────┘
```

- **调账弹窗**（`growth.adjust` 专属）：
  - 方向：增加 / 扣减（segmented）
  - 积分数（正整数）；**扣减时从用户钱包扣回，增加时占用总预算**——后端按「增加 = TOPUP 式补偿（走预算已分配校验）/ 扣减 = 钱包 → REDEEMED」处理，页面只让管理员填数量与原因，不暴露内部账户
  - 原因（必填，≥5 字，会写入审计）
  - 二次确认：**显示调整前后余额对比**，红字「调账会直接改变用户资产，请确认已获得授权」
  - 提交后 toast + 抽屉内流水即时插入首行（不整页刷新）
- **越权保护**：调账接口必须 RBAC `growth.adjust`，前端按钮同样按权限隐藏；后端二次校验（前端隐藏不等于安全边界）。**不设双人复核**，靠权限收敛 + 审计流水 + 每日对账告警。

---

## 4. 用户端页面设计 `/growth`

### 4.1 定位

"我的积分"是用户对成长体系的**唯一可见门面**：一眼看到有多少分、今天签没签、能换什么、分是怎么来的。**不做成任务工作台**——任务的真正完成动作在 AIWork。

### 4.2 Tab 1｜积分总览 `tab=overview`

```text
┌───────────────────────────────────────────────────────────────┐
│  我的积分                                       [去 AIWork 做任务]│
│  260                                        ┌──────────────────┐│
│  可用积分                                     │ 今日已签到 ✓      ││
│  累计获得 320 · 累计消耗 60 · 本月 +45         │ 连续 12 天        ││
│                                              │ 签到请前往 AIWork ││
│                                              └──────────────────┘│
└───────────────────────────────────────────────────────────────┘
┌─ 可做任务 ──────────────────────────────────────────────────────┐
│ [签到图标] 每日签到    +10 分   每天 1 次     今日已完成          │
│ [任务图标] 首次调用 API +20 分   每人 1 次     [去 AIWork 完成]   │
└─────────────────────────────────────────────────────────────────┘
┌─ 兑换好物（预览 4 个）────────────────────── [查看全部 →] ────────┐
│ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐                    │
│ │100万Token│ │ Pro 7天 │ │ 定制周边│ │ …      │                    │
│ │ 50 积分 │ │ 300 积分│ │ 800 积分│ │        │                    │
│ │ 可兑换  │ │ 积分不足│ │ 可兑换  │ │        │                    │
│ └────────┘ └────────┘ └────────┘ └────────┘                    │
└─────────────────────────────────────────────────────────────────┘
┌─ 最近积分明细 ────────────────────────────── [查看全部 →] ───────┐
│ 11-17 15:43  每日签到            +10      余额 260              │
└─────────────────────────────────────────────────────────────────┘
```

**Hero 卡细节**
- 数字用大号（`text-4xl font-bold`）+ 千分位，右侧签到卡用 teal 渐变描边。
- 数据全部来自 `GET /user/growth/overview` 一次返回，避免首屏 4 个请求。
- 余额为 `0` 时数字灰色并显示「完成一次任务即可获得积分」。

**签到卡（v1.1：只读，不允许 Web 签到）**
- **不渲染签到按钮**，只展示今日状态（已签到 / 未签到）与连续天数。
- 未签到时展示引导：「今日还未签到，请前往 AIWork 完成签到」；跳转目标（自定义协议 / 深链）未定前只展示文案，不做跳转。
- 状态来自 `GET /user/growth/overview` 的 `checkin.claimed_today` 与 `checkin.streak_days`；**不轮询、不按本地日期推断**。
- `checkin` 为空（未配置签到任务）时整块卡片隐藏。
- **W 面不提供签到接口**，页面不得调用 `POST /user/growth/checkins`（该路径不存在）；签到与发分只走 AIWork 的 U 面 #8。

### 4.3 Tab 2｜兑换中心 `tab=store`

- 商品网格（响应式 1/2/3/4 列），卡片含：图标、名称、一句话描述、所需积分、库存/限购标记、兑换按钮。
- 按钮状态机：`可兑换`（teal）/ `积分不足 · 还差 40`（灰 + 差值）/ `已售罄`（灰）/ `已达限购`（灰）/ `已下架`（不展示）。
- 点击 → 商品详情抽屉（大图、权益说明、履约时效、兑换须知）→ 「确认兑换」二次确认：「将扣除 50 积分兑换 100万 Token 额度，兑换后不可撤销」→ 提交后展示订单状态。
- 下单成功：积分卡即时刷新（用响应 `current_balance`），订单进入「处理中」，有限轮询 `GET /user/growth/exchanges/{id}`（≤5 次，间隔 2s，页面隐藏即停）。
- 失败/退款：卡片提示 + 跳「我的兑换记录」（同 tab 内切换过滤）。

### 4.4 Tab 3｜积分明细 `tab=ledger`

- 列表（移动端卡片 / 桌面表格）：时间、类型（签到/任务/兑换/调账/退款）、说明、变动、余额快照。
- **颜色口径**：积分是资产而非股票，不使用红涨绿跌——**增加用 teal（+）、减少用 rose（−）**，与全站 `PointsAmount` 组件统一。

> 说明：积分不是股票，红涨绿跌规则不适用；统一用「+teal / −rose」表达资产增减。

- 类型筛选 chips、时间范围、游标分页「加载更多」。
- 空态：「还没有积分记录，去完成第一个任务吧」+ 按钮跳总览。
- 单条可点开展开行内详情：业务关联号（`checkin#66001` / `order#88001` / `transfer#77003`），便于客服排查。

---

## 5. 交互与状态规范

### 5.1 统一页面状态

| 状态 | 表现 |
|---|---|
| 首屏加载 | 骨架屏（表格 8 行 / 卡片 4 个），不用全屏 spinner |
| 空数据 | `EmptyState` + 主操作按钮 + 一句解释（见各 tab） |
| 请求失败 | 内容区错误条 + 「重试」；保留上一次成功数据（不闪空） |
| 无权限 | 页面级 403 卡片「你没有成长计划查看权限，请联系管理员」；按钮级置灰 + tooltip |
| 危险操作 | 二次确认弹窗，红色主按钮，必填原因，文案说明不可撤销 |
| 提交中 | 按钮 loading + 禁用，防重复提交 |

### 5.2 数据刷新策略

- 管理端：手动刷新 + 操作成功后**局部更新**（改一行不重拉整表）。
- 用户端：进入页面拉一次；兑换成功后用响应数据更新局部，不整页重拉。
- 不做自动轮询（除兑换履约的有限轮询），避免后台常驻请求。

### 5.3 与 AIWork 的职责衔接（重要）

| 场景 | Web 端行为 | AIWork 行为 |
|---|---|---|
| 任务完成 | 只展示任务与奖励，按钮为「去 AIWork 完成」，**不提供完成按钮** | 判定完成后调 U 面 `completion-requests` |
| 签到 | **只读展示状态 + 引导去 AIWork**，不提供签到按钮、不调用任何签到接口 | **唯一签到入口**，调 U 面 #8 |
| 兑换 | Web 可兑换（W 面）；AIWork 也可兑换（U 面 #10） | 同左 |
| 积分明细 | W 面只读 | U 面 #4 |

> 兑换两入口并发由服务端唯一键兜底；签到只有 AIWork 一个入口，不存在双写。

### 5.4 响应式

- ≥1024：侧边栏 + 表格；
- 768–1023：表格横向滚动，统计卡 2×2；
- <768：表格降级为卡片列表（`DataTable` 已内建），统计卡单列，商品网格 1–2 列，操作收进「更多」菜单。

---

## 6. 组件规划（复用优先）

### 6.1 直接复用（不改动）

`AppLayout`、`TablePageLayout`、`DataTable`、`Pagination`、`StatCard`、`BaseDialog`、`ConfirmDialog`、`EmptyState`、`LoadingSpinner`、`Skeleton`、`Select`、`Input`、`TextArea`、`Toggle`、`DateRangePicker`、`Icon`、`Toast`、`SearchInput`、`StatusBadge`、`GroupBadge`；全局 `.card` / `.btn` / `.badge-*` / `.stat-*` 样式。

### 6.2 新增组件（全部放 `src/components/growth/`，不外溢）

| 组件 | 用途 | 复用点 |
|---|---|---|
| `GrowthTabs.vue` | 管理端/用户端 tab 条（受控，query 同步） | — |
| `PointsAmount.vue` | 积分数字统一展示（千分位、正负色、可空） | 全站积分展示唯一出口 |
| `BudgetUsageBar.vue` | 任务额度使用率进度条（三档色 + tooltip） | Tab1 / Tab2 |
| `RuleVersionBadge.vue` | 规则/商品版本 badge（已发布/待发布） | Tab2 / Tab3 |
| `BudgetEntryDialog.vue` | 增加 / 核减总预算（含校验与二次确认） | Tab1 |
| `TaskBudgetDialog.vue` | 任务追加 / 回收额度 | Tab1 / Tab2 |
| `BudgetEntryDrawer.vue` | 预算变动流水（四类） | Tab1 |
| `TaskFormDrawer.vue` | 任务新建/编辑 + 规则发布 | Tab2 |
| `TaskRuleHistoryDrawer.vue` | 规则版本历史 | Tab2 |
| `ProductFormDrawer.vue` | 商品新建/编辑 + 版本发布 + 库存 | Tab3 |
| `ExchangeOrderDrawer.vue` | 订单详情时间线 + 重试/退款 | Tab4 |
| `UserPointsDrawer.vue` | 用户积分三合一抽屉 | Tab5 / Tab4 |
| `PointsAdjustDialog.vue` | 手工调账（敏感操作） | Tab5 |
| `ProductCard.vue` | 用户端商品卡（状态机按钮） | 用户端 store |
| `CheckinCard.vue` | 用户端签到状态卡（只读，无按钮，引导去 AIWork） | 用户端 overview |
| `PointsHero.vue` | 用户端积分大卡 | 用户端 overview |

### 6.3 视图文件

```text
src/views/growth/
├─ AdminGrowthView.vue        # 管理端外壳：页头 + GrowthTabs + <component :is>
├─ admin/
│  ├─ BudgetTab.vue           # 总预算统计卡 + 任务额度表 + 预算流水
│  ├─ TasksTab.vue
│  ├─ ProductsTab.vue
│  ├─ OrdersTab.vue
│  └─ UserPointsTab.vue
├─ UserGrowthView.vue         # 用户端外壳
└─ user/
   ├─ OverviewTab.vue
   ├─ StoreTab.vue
   └─ LedgerTab.vue
```

```text
src/api/growth/
├─ admin.ts   # M 面：预算/任务额度/任务/商品/订单/用户/调账
├─ user.ts    # W 面：overview/ledger/tasks/products/exchanges（不含签到）
├─ types.ts   # 本文 §8 的 view model（仅页面所需字段）
└─ index.ts   # barrel
```

---

## 7. 低侵入落点清单（本次唯一的既有文件改动）

| # | 文件 | 改动 | 规模 | 必要性 |
|---|---|---|---|---|
| 1 | `frontend/src/router/index.ts` | 追加 2 条路由（`/admin/growth`、`/growth`），懒加载 | +26 行 | 必须 |
| 2 | `frontend/src/components/layout/AppSidebar.vue` | `adminNavItems` 追加 1 项；`buildSelfNavItems` 追加 1 项 | +2 行 | 必须 |
| 3 | `frontend/src/rbac/permissionMatrix.ts` | 追加 2 条路径→权限映射 | +2 行 | 必须 |
| 4 | `frontend/src/i18n/locales/zh.ts` / `en.ts` | 追加 `growth` 命名空间（含 `adminGrowth` 子命名空间） | 新增 key，不改旧 key | 必须（文案规范） |
| 5 | `frontend/src/api/admin/index.ts` | 追加 `growth: growthAdminAPI` 聚合（若页面直接用 `adminAPI.growth`） | +2 行 | 可选（也可直接 import 模块） |

> 第 5 项可省略：新页面直接 `import growthAdminAPI from '@/api/growth/admin'`，就不必碰聚合文件。**推荐省略**，把侵入压到 4 个文件。

**明确不碰**：`App.vue`、`AppLayout.vue`、`style.css`、现有任何 view、任何 store、`client.ts`、`tailwind.config.js`、`vite.config.ts`。新增页面不使用新的全局 CSS 类，全部用现有语义类 + Tailwind 原子类。

**后端侵入（另一侧，供前端对齐）**：新增 `handler/growth/**`、`service/growth/**`、`repository/growth/**`、`sqlArchiving/18x_growth.sql`；既有文件只改 `server/routes/admin.go` + 新增 `server/routes/growth.go` 并在 `router.go` 调 1 行注册；RBAC `permissions.go` 加 3 个权限码 + 登记 inventory。

---

## 8. 数据依赖（页面字段 → 接口）

> 以下为**页面所需 view model**，M/W 面不在冻结契约内，字段可随页面演进；U 面字段一律以契约 v4.0 为准，不在此重复。

### 8.1 M 面（管理员，`AdminIdentityAuth` + RBAC）

| 方法 | 路径 | 权限 | 页面用途 |
|---|---|---|---|
| GET | `/admin/growth/stats/overview` | `growth.read` | Tab1 顶部 4 个 StatCard（总预算/已分配/已发放/未分配） |
| GET | `/admin/growth/budget` | `growth.read` | 总预算账户（含三项余额与版本号） |
| POST | `/admin/growth/budget/entries` | `growth.manage` | 增加 / 核减总预算（`direction: INCREASE\|DECREASE`，核减校验 `total-N ≥ allocated`） |
| GET | `/admin/growth/budget/entries` | `growth.read` | 预算变动流水（分页 + 类型筛选） |
| GET | `/admin/growth/budget/allocations` | `growth.read` | 任务额度分配表（任务/已分配/已发放/剩余/使用率） |
| POST | `/admin/growth/budget/allocations` | `growth.manage` | 划拨额度（新建任务时由 `POST /tasks` 内部调用，也可单独给任务追加） |
| POST | `/admin/growth/tasks/:id/budget/revoke` | `growth.manage` | 回收任务剩余额度（仅下线/结束任务） |
| GET | `/admin/growth/tasks` | `growth.read` | 任务列表（含规则摘要 + 额度） |
| POST | `/admin/growth/tasks` | `growth.manage` | 新建任务（`initial_allocation` 必填，事务内建任务 + 划拨） |
| PUT | `/admin/growth/tasks/:id` | `growth.manage` | 编辑（生成待发布规则，不改额度） |
| POST | `/admin/growth/tasks/:id/publish` | `growth.manage` | 发布规则版本（写生效时间，旧版本自动结束） |
| POST | `/admin/growth/tasks/:id/offline` | `growth.manage` | 下线（必填原因，可选同时回收剩余额度） |
| POST | `/admin/growth/tasks/:id/online` | `growth.manage` | 重新上线（必填 `initial_allocation`，从总预算未分配划拨） |
| GET | `/admin/growth/tasks/:id/rule-versions` | `growth.read` | 规则历史 |
| GET | `/admin/growth/products` | `growth.read` | 商品列表 |
| POST | `/admin/growth/products` | `growth.manage` | 新建商品 |
| PUT | `/admin/growth/products/:id` | `growth.manage` | 编辑（生成待发布版本） |
| POST | `/admin/growth/products/:id/publish` | `growth.manage` | 发布版本 |
| POST | `/admin/growth/products/:id/stock` | `growth.manage` | 库存增减 |
| POST | `/admin/growth/products/:id/offline` | `growth.manage` | 下架 |
| GET | `/admin/growth/exchanges` | `growth.read` | 订单列表（状态/用户/时间筛选） |
| GET | `/admin/growth/exchanges/:id` | `growth.read` | 订单详情时间线 |
| POST | `/admin/growth/exchanges/:id/retry` | `growth.manage` | 重试履约 |
| POST | `/admin/growth/exchanges/:id/refund` | `growth.manage` | 退款 |
| GET | `/admin/growth/users` | `growth.read` | 用户积分总览（搜索/排序/区间） |
| GET | `/admin/growth/users/:id/ledger` | `growth.read` | 用户流水 |
| GET | `/admin/growth/users/:id/exchanges` | `growth.read` | 用户兑换记录 |
| POST | `/admin/growth/users/:id/adjustments` | `growth.adjust` | 手工调账 |

> **任务无 `DELETE` 接口**（v1.3 决策）：下线是唯一的"移除"手段，`OFFLINE` 可经 `POST /admin/growth/tasks/:id/online` 回到 `ACTIVE`。前端任务表操作列不出现「删除」，后端不注册 `DELETE /admin/growth/tasks/:id`。

统一响应沿用 `response.Response`（`{code, message, data}`），分页 `{items, total, page, page_size}`，列表默认 20/页。

### 8.2 W 面（登录用户，`jwtAuth`）

| 方法 | 路径 | 页面用途 | 主要返回字段 |
|---|---|---|---|
| GET | `/user/growth/overview` | Hero 卡 + 签到状态（只读） | `available_points`、`total_earned`、`total_spent`、`month_earned`、`checkin{claimed_today, streak_days, next_eligible_at}`、`server_time` |
| GET | `/user/growth/tasks` | 任务区 | `items[{task_id,title,description,points,period,daily_limit,per_user_limit,completed_today,status}]` |
| GET | `/user/growth/products` | 兑换中心 | `items[{product_id,name,description,points_cost,stock,per_user_limit,can_exchange,reason}]` |
| GET | `/user/growth/products/:id` | 商品详情抽屉 | 同上 + `benefit_detail`、`fulfillment_note`、`version` |
| POST | `/user/growth/exchanges` | 兑换下单（幂等键必填） | `order_id`、`order_no`、`status`、`points_cost`、`current_balance` |
| GET | `/user/growth/exchanges` | 我的兑换记录 | 分页列表 |
| GET | `/user/growth/exchanges/:id` | 订单状态轮询 | `status`、`fail_reason`、`refunded` |
| GET | `/user/growth/ledger` | 积分明细 | 游标分页 `{items, next_cursor}` |

> **W 面没有签到接口**（v1.1）：签到唯一入口是 AIWork 的 U 面 #8，Web 只读 `overview.checkin` 展示状态。`exchanges` 与 U 面共用 service 与幂等键空间；`username` 不从请求体取，直接用 `AuthSubject.UserID`。
>
> **W 面也不提供任务完成接口**：完成判定与声明在 AIWork。

---

## 9. 权限与 i18n

### 9.1 新增 RBAC 权限码

| 权限码 | 含义 | 绑定角色建议 |
|---|---|---|
| `growth.read` | 查看成长计划全部管理数据 | 管理员、运营只读 |
| `growth.manage` | 总预算增减、任务额度划拨/回收、任务/商品/订单运维 | 运营 |
| `growth.adjust` | 手工调账（高风险） | 仅超级管理员 |

前端：`permissionMatrix` 加 `/admin/growth → growth.read`、`/growth → growth.self.read`；按钮级用 `authStore.can('growth.manage')` 或 `v-permission="'growth.adjust'"`。

### 9.2 i18n key 规划（`zh.ts` / `en.ts` 同步）

```text
growth.common.*            # 通用：积分、加载、重试、确认、取消、原因必填
growth.admin.title / description
growth.admin.tabs.pool|tasks|products|orders|users
growth.admin.pool.*        # 列表列名、新建/调整弹窗、空态
growth.admin.task.*
growth.admin.product.*
growth.admin.order.*
growth.admin.userPoints.*
growth.user.title / hero.* / checkin.* / store.* / ledger.*
```

- 所有文案走 `t()`，禁止硬编码中文（与现有页面一致）。
- 中文为主，英文同步补全；缺 key 时回退 zh。

---

## 10. 任务拆分与验收

### 10.1 前端拆分（建议 6 个 PR）

| # | 任务 | 产出 | 验收 |
|---|---|---|---|
| F0 | 骨架 | 2 条路由、菜单 2 项、权限矩阵 2 条、i18n 命名空间、`api/growth/*` 空实现 + mock | 能进页面，tab 可切换，无权限时 403 |
| F1 | 管理端 Tab1 总预算与任务额度 | 4 统计卡 + 任务额度表 + 增加/核减预算弹窗 + 追加/回收额度 + 预算流水抽屉 | 核减不得低于已分配（前端禁用 + 后端 409）；额度色阶正确；流水四类齐全 |
| F2 | 管理端 Tab2 任务 | 任务表（含额度列）+ 表单抽屉（初始额度必填）+ 发布/下线 + 规则历史 | 新建任务事务内完成划拨；额度不足禁止提交；发布后显示新版本 |
| F3 | 管理端 Tab3 兑换品 | 商品表 + 双视图 + 表单 + 库存 | 改价生成新版本；售罄置灰；上架校验拦截 |
| F4 | 管理端 Tab4/Tab5 | 订单表 + 详情 + 重试/退款；用户表 + 三合一抽屉 + 调账 | 退款需原因；调账显示前后余额；`growth.adjust` 缺失时不可见 |
| F5 | 用户端 `/growth` | Hero + 只读签到卡 + 任务 + 商品 + 明细 | 签到卡无按钮、不调接口；兑换后余额即时刷新；移动端可用 |

### 10.2 验收清单（页面级）

- [ ] 2 条路由懒加载，首屏 bundle 未显著增大（chunk 独立）。
- [ ] 既有 4 个文件的改动均为**追加**，无删除、无重排、无格式化噪声。
- [ ] 所有既有页面回归：侧边栏菜单顺序、权限过滤、简单模式（`isSimpleMode`）下行为正确。
- [ ] 移动端 375px 无横向溢出；表格降级为卡片。
- [ ] 深色模式（`.dark`）下所有新页面颜色正确（使用 `dark:` 语义类，无硬编码色值）。
- [ ] 空态、错误态、无权限态三态齐全。
- [ ] 危险操作（增加/核减预算、回收额度、下线、退款、调账）均二次确认 + 必填原因。
- [ ] **主操作唯一**：`增加预算`/`新建任务`/`新建商品`/`导出 CSV` 在页面上各只出现一次，且都在所属 tab 的工具条内；页头仅「刷新」，无随 tab 变文字的主按钮。
- [ ] **任务无删除**：任务表无「删除」按钮，Network 面板确认前端不调用 `DELETE /admin/growth/tasks/:id`；下线后任务仍可查、可重新上线；`task_code` 重复时新建即时报错、不落库。
- [ ] 预算不变量可见可控：核减超过未分配时前端禁用提交且后端返回 `BUDGET_ALLOCATED_LOCKED`；新建任务额度不得超过未分配；额度耗尽任务显示「已停止发分」。
- [ ] 用户端签到卡无任何按钮，页面加载期间不出现签到请求（Network 面板无 `checkins`）。
- [ ] 无 `any` 泄漏（新增 TS 类型集中在 `api/growth/types.ts`）。
- [ ] i18n 无硬编码中文；`en.ts` 与 `zh.ts` key 结构一致。

---

## 11. 已确认决策与剩余待确认

### 11.1 已确认（2026-11-17）

| # | 决策 | 落地影响 |
|---|---|---|
| 1 | **不允许 Web 签到** | 用户端签到卡只读 + 引导去 AIWork；W 面不提供签到接口；签到唯一入口为 U 面 #8 |
| 2 | **调账不做双人复核** | 仅 RBAC `growth.adjust` + 审计流水 + 每日对账告警 |
| 3 | **总预算与任务解耦** | 单一总预算账户，手动增加/核减；新增任务从总预算划拨额度；额度耗尽即停止发分；`allocated ≤ total` 硬约束 |

### 11.2 仍待确认（不阻塞首版）

| # | 问题 | 默认取值 | 影响 |
|---|---|---|---|
| 1 | 管理员是否允许在 Web 直接兑换（代用户） | **不允许**（首版无此功能） | 后续加「代下单」需审计设计 |
| 2 | 商品类型首版范围 | `TOKEN_QUOTA` + `PHYSICAL` | `SUBSCRIPTION` 依赖订阅域对接 |
| 3 | 是否需要积分趋势图 | **首版不做** | 后续基于 `stats/overview` 扩展 |
| 4 | 总预算是否按年度/季度分账 | **单账户累计，不自动重置** | 影响报表口径与「年度预算」展示 |
