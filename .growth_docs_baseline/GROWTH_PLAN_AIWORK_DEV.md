# AIWork 侧开发设计｜成长计划（用户入口 + 完成判定方）

> 日期：2026-11-17 ｜ 对接契约：[GROWTH_PLAN_AIWORK_API.md](GROWTH_PLAN_AIWORK_API.md) **v4.0（冻结，唯一接口真相源）**
> 总体设计：[GROWTH_PLAN_DESIGN.md](GROWTH_PLAN_DESIGN.md) ｜ 对端开发设计：[GROWTH_PLAN_SUB2API_DEV.md](GROWTH_PLAN_SUB2API_DEV.md)
>
> 本文是 **AIWork 团队内部开发设计**。所有请求/响应字段、路径、状态枚举、错误码**不在本文重复定义**，一律引用冻结契约 v4.0；本文只描述 AIWork 侧如何组织代码、管理状态、处理异常与拆分任务。
>
> **v4.0 职责反转**：任务/签到是否完成改由 **AIWork 侧判定**，完成后提交完成请求（带 `completion_ref`），sub2api 只按规则发分、不再核验。AIWork 从「只提交业务引用、不声明完成」变为「判定完成并声明」。

---

## 0. 范围与不可越界事项

### 0.1 AIWork 做什么

| 能力 | 说明 |
|---|---|
| 展示积分概览 | 可用积分、累计获得/消耗、服务端更新时间 |
| 展示积分流水 | 分页列表，游标 `before_entry_id` |
| 展示任务列表/详情 | 服务端下发的文案、进度、周期、奖励说明、操作入口 |
| **判定任务/签到是否完成**（v4.0 新增） | 按本地业务规则判定完成；为每次完成生成 `completion_ref`（完成实例标识） |
| 发起签到 | 判定签到完成后提交请求，展示服务端发分结果 |
| 发起任务完成请求 | 判定完成后提交 `completion_ref` + 幂等键，展示同步发分结果 |
| 展示商品 / 发起兑换 | 展示服务端价格与权益，只提交 `product_id` + 幂等键 |
| 展示订单 | 扣分与履约状态、失败/退款原因 |

### 0.2 AIWork 不做什么（红线）

- **v4.0 修订**：**判定任务是否完成是 AIWork 的职责**（见 0.1），但**不计算积分、不决定奖励数额/预算与任务额度/周期/上限/签到日期**——这些是 sub2api 裁决的账务字段。
- 不本地加分、不缓存权威余额；余额与「积分已到账」只认服务端响应。任务卡可依本地判定显示「已完成（待发分）」，但不得显示「已发分」。
- 不提供任何运营管理界面，不调用 `/api/v1/admin/growth/*`。
- 不构造 `user_id` / `points` / `awarded_points` / `business_date` / `cycle_key` 等账务字段（契约第 6 节）；`username` 仅由 Main 从既有登录态读取后放入加密报文。
- Renderer 不直接接触 sub2api、不接触密钥、不持久化幂等键。
- **不改动现有登录功能**：不新增登录步骤、不引入令牌；只复用既有登录态里的用户名。
- **只持有 SM2 公钥**（私钥永不在客户端）：公钥由 sub2api 侧生成后直接分配，**写死为 Main 代码常量**（无构建期注入、无指纹校验、无运行时下发）；公钥只进 Main、不进 Renderer、不进日志。
- 使用国密库前需安全团队确认版本与合规（建议 Node `sm-crypto`）。

---

## 1. 技术栈假设（需 AIWork 团队确认）

> AIWork 项目不在 sub2api 仓库内，本节基于「Electron Main + Renderer + 类型化 preload」的形态假设。**若实际栈不同（如 Tauri、纯 Web），只需替换第 2 章的进程边界实现，第 3～6 章的状态与流程设计不变。**

| 项 | 假设 | 确认项 |
|---|---|---|
| 桌面框架 | Electron（Main 进程 + Renderer 进程） | 是否 Electron / 版本 |
| Renderer 框架 | React 或 Vue 3 + TypeScript | 具体框架与状态库（Pinia / Redux / Zustand） |
| 进程通信 | `contextBridge` 暴露类型化 `window.aiwork` API | 现有 preload API 的组织方式 |
| HTTP 客户端 | Main 进程内 Node `fetch` / `axios` | 现有 Main 的网络封装与代理配置 |
| 登录态 | AIWork 自有登录，持有当前用户 email | **email 从哪个 store / 会话对象读取**（关键，见 2.2）；登录功能不改 |
| 加密 | Node 国密库 `sm-crypto`（SM2/SM3/SM4）或同等合规库 | 安全团队确认库版本；SM2 公钥**写死常量**（由 sub2api 侧分配）、`kid` 固定为 `growth-sm2-v1` |
| 构建 | Vite / electron-builder | 与现有工程一致即可 |

> 需 AIWork 团队回复的 3 件事：① 上面 7 项确认；② 当前登录用户的 email 从哪里取；③ 成长页挂载位置。SM2 密钥对由 sub2api 侧生成后**直接分配**：公钥写死进 AIWork Main、私钥写死进 sub2api，双方不做注入/指纹校验/轮换；双方约定同一对测试密钥（公钥给 AIWork、私钥给 sub2api Mock）用于联调。

---

## 2. 进程边界与目录结构

### 2.1 分层

```text
Renderer (展示层)
  growth/ views + components + store        ← 只展示、只调 preload API
      │  window.aiwork.growth.*
      ▼
preload (contextBridge，类型化 API)
  growthBridge.ts                            ← 仅转发 IPC，不含业务逻辑
      │  ipcRenderer.invoke('growth:*')
      ▼
Main (进程)
  growth/                                    ← 唯一发 HTTP 的地方
    ├── client.ts        HTTP 客户端（baseURL、重试、错误归一化）
    ├── service.ts       业务编排（查询聚合、命令结果落状态）
    ├── crypto.ts        SM2 公钥读取 + 数字信封封装（唯一接触公钥处）
    ├── ipc.ts           IPC handler 注册
    └── mock/            本地 Mock Server（P0/P1 阶段用，内置测试公钥与解封装校验）
```

**边界铁律**：Renderer 只能通过 `window.aiwork.growth.*` 调用；SM2 公钥只在 Main 的 `crypto.ts` 读取（私钥永不在客户端），`X-Growth-*` 身份头已取消；用户名由 Main 从既有登录态读取后放进加密 payload；Renderer 传入的参数只能是契约允许的业务字段。

### 2.2 Main 侧加密封装（关键，登录零改动）

```ts
// Main: growth/crypto.ts —— 唯一接触 SM2 公钥的模块（私钥永不在客户端）
import { sm2, sm4, sm3 } from 'sm-crypto';        // 库需安全团队确认
import { randomBytes } from 'node:crypto';

// 公钥由 sub2api 侧生成后直接分配，写死为常量（hex: 04||X||Y）。
// 无指纹校验、无运行时下发；变更须双方同步改代码发版（见契约 5.5）。
const GROWTH_SM2_PUBKEY = '04....';               // TODO(sub2api 分配): 填入固定公钥
const KID = 'growth-sm2-v1';                      // 固定值，不轮换

export function seal(path: string, params: object, opts: {
  idempotencyKey?: string;
}): GrowthEnvelope {
  const kid = KID;
  const sm4Key = randomBytes(16).toString('hex'); // 每请求新会话密钥，绝不复用
  const iv = randomBytes(16);                     // SM4-CBC IV，每请求新随机
  const payload = {
    path,
    username: currentUserEmail(),                 // 从既有登录态读，不改登录
    params,
    request_id: makeTraceId(),
    issued_at: Math.floor(Date.now() / 1000),     // 双方 NTP 同步
    nonce: opts.idempotencyKey ?? randomUUID(),
    ...(opts.idempotencyKey ? { idempotency_key: opts.idempotencyKey } : {}),
  };
  const plaintext = JSON.stringify(payload);
  const ciphertext = sm4.encrypt(plaintext, sm4Key, { mode: 'cbc', iv: iv.toString('hex'), padding: 'pkcs7' });
  const encryptedKey = sm2.doEncrypt(sm4Key, GROWTH_SM2_PUBKEY, 1); // cipherMode=1 → C1C3C2
  // MAC 拼接用 base64 字符串，顺序固定；双方一致（见契约 1.3）
  const b64 = (s: string | Buffer) => Buffer.from(s).toString('base64');
  const macInput = [kid, b64(encryptedKey), b64(iv), b64(ciphertext)].join('');
  const mac = sm3(`${sm4Key}${macInput}`);        // 等价于 SM3-HMAC(会话密钥, ...) 的库实现，按库 API 校准
  return { kid, encrypted_key: b64(encryptedKey), iv: b64(iv),
           ciphertext: b64(ciphertext), mac: b64(mac) };
}

// Main: growth/client.ts —— 所有接口统一 POST + SM2 数字信封
async function request<T>(path: string, params: object, idempotencyKey?: string): Promise<T> {
  const body = seal(path, params, { idempotencyKey });
  // fetch POST <BaseURL><path> body=json(body)，Content-Type: application/json
  // 重试：复用 idempotencyKey，但每次 seal() 重新生成会话密钥/IV 并重新加密（见 5.3）
}
```

> 注意：上面是**结构示意**；SM3-HMAC 的具体调用按 `sm-crypto`/所选库 API 校准，务必与 sub2api 用同一测试向量做互操作测试（契约 1.3 实现一致性要点）。若库不直接提供 HMAC，用 SM3 按 HMAC 标准构造。

**切账号**：不缓存用户名，每次 `seal()` 现取登录态；账号切换清 Renderer store，且不允许用旧幂等键重放。**公钥与登录态解耦**：切账号不换公钥，只换 payload 里的 `username`。

### 2.3 建议目录（具体命名按现有工程风格调整）

```text
src/
  main/growth/
    client.ts            # HTTP + 重试 + 错误归一化
    crypto.ts            # SM2 公钥读取 + 数字信封封装（唯一接触公钥处）
    service.ts           # 命令编排、查询聚合
    ipc.ts               # ipcMain.handle 注册
    errors.ts            # 错误码映射（契约 1.5）
    mock/server.ts       # 本地 mock（按契约第 3 章）
  preload/growthBridge.ts
  renderer/growth/
    views/               # GrowthPage / CheckinCard / TaskList / Ledger / Products / Orders
    store/growthStore.ts # 服务端快照 + 加载态
    types/contract.ts    # 契约 TypeScript 类型（从契约第 3 章手工或用工具生成）
```

---

## 3. 数据模型（Renderer store）

Renderer 只存**服务端快照**，不存权威推导值。

```ts
interface GrowthState {
  me: MeResponse | null;              // 积分概览 + 签到状态（契约 3.1）
  tasks: TaskItem[];                  // 契约 3.2
  ledger: { items: LedgerItem[]; nextCursor: string | null; hasMore: boolean };
  products: ProductItem[];            // 契约 3.5
  orders: ExchangeItem[];             // 契约 3.7
  loading: Record<string, boolean>;   // 逐操作 loading，避免全局遮罩
  serverTime: string | null;
}
```

**更新规则**（防止乐观更新；v4.0 完成判定在 AIWork）：

| 操作 | 何时更新 store |
|---|---|
| 任务完成判定 | AIWork 本地判定完成后，任务卡可标记「已完成（待发分）」并生成 `completion_ref`；**此时不改余额** |
| 签到 | 收到契约 3.8 响应后，用返回的 `current_balance` / `business_date` 更新；为空则重拉 `/me` |
| 完成请求 | 收到 3.9 同步响应（`AWARDED` / `ALREADY_COMPLETED` / `NOT_ELIGIBLE`）后，按 `awarded_points` / `current_balance` 更新任务卡与余额；**v4.0 无轮询、无 3.11** |
| 兑换 | 收到 3.10 响应后更新订单列表与余额；履约状态轮询 3.12 |
| 任何失败 | **保留上次服务端确认的余额与状态**，只展示错误提示；任务完成已判定的本地状态可保留，供下次重试发分 |

---

## 4. 关键流程设计

### 4.1 进入成长页（只读加载）

```text
Renderer mount
  → preload.growth.getMe()          ─┐
  → preload.growth.listTasks()       ├─ Main 并行发 3~4 个 POST，各自 seal() 加密信封
  → preload.growth.listProducts()   ─┘
  → 结果写入 store，按 server_time 标记快照时间
```

加载失败按 5.1 归一化错误展示；加密类错误（`CRYPTO_*`）提示配置/密钥/时钟问题，不重试；`GROWTH_USER_NOT_FOUND` 提示账号未开通（账务映射失败），不做重试。

### 4.2 签到

```text
用户点签到
  → AIWork 本地判定签到动作完成（如首次打开/每日首次点击，业务侧规则）
  → Main 生成 Idempotency-Key（本次逻辑请求唯一）
  → POST /growth/checkins（params 不含 business_date/cycle_key，由服务端算）
  → 结果 AWARDED / ALREADY_CLAIMED → 更新余额与签到卡（ALREADY_CLAIMED 显示今日已完成，不加分）
  → NOT_ELIGIBLE / UNAVAILABLE     → 展示原因，不重试
  → 网络/429/503                   → 有上限退避，复用同一幂等键（见 5.3）
  → 超时                           → 用原幂等键重放，或重拉 /me + 签到状态
```

UI 状态：`加载/待确认` → `已签到（日期+奖励）` / `今日已完成` / `不符合条件` / `暂不可用（可重试）`。**只有收到服务端响应才显示"签到成功"**；周期与日期一律服务端下发，本地不按日期推断 `cycle_key`。

### 4.3 任务完成请求（v4.0 重写：AIWork 判定完成并声明）

```text
AIWork 按本任务业务规则判定「已完成」（判定逻辑在 AIWork 侧，不入 sub2api）
  → 生成本次完成实例标识 completion_ref（建议任务实例 ID / 本地完成事件 ID，AIWork 内唯一）
  → Main 生成 Idempotency-Key；同一 completion_ref 重试复用同一幂等键
  → POST /growth/tasks/{id}/completion-requests
       payload.params = { task_id, completion_ref, request_id? } + 幂等键，整体 seal() 加密
  → AWARDED           → 用 awarded_points / current_balance 更新任务卡与余额
  → ALREADY_COMPLETED → 幂等重放，按首次结果展示，不重复加分
  → NOT_ELIGIBLE      → 展示"暂不符合条件"（任务暂停/上限已满），不重试
  → 网络/429/503/超时 → 有上限退避，复用同一 completion_ref 与幂等键重发（服务端按 completion_ref 去重）
```

**要点（v4.0）**：
- **完成判定在 AIWork**：由 AIWork 决定任务何时算完成，不再依赖网关 `run_id` / `usage_logs` 回查，也不再有 I 面 `task-events` 上报。
- **`completion_ref` 必须稳定**：同一任务完成实例多次重试要用同一 `completion_ref`（配合幂等键双重防重）；用户重新完成一次任务则生成新的 `completion_ref`。
- **同步发分**：#9 响应即最终结果，无 `PENDING_VERIFICATION`、无轮询（原 3.11 接口已删除）。
- **不可越界**：`params` 只放 `task_id` / `completion_ref` / `request_id?`，不得放积分、周期、成功标志等字段。

### 4.4 兑换

```text
用户点兑换（先展示二次确认：商品 + 服务端价格 + 权益）
  → POST /growth/exchanges  payload.params 只含 { product_id, expected_version?, request_id? } + 幂等键，整体 seal() 加密
  → 成功 → 更新订单与余额，展示履约中
  → PRODUCT_VERSION_CHANGED / OFFLINE / SOLD_OUT → 重拉商品详情，提示目录已更新，不自动重试
  → INSUFFICIENT_POINTS / PURCHASE_LIMIT_REACHED → 展示原因，不重试
  → 履约状态轮询 3.12（有限次数）→ SUCCEEDED / FAILED / REFUNDED
```

**禁止**：把展示用的 `points_cost` / `benefit_detail` 回传下单接口；服务端始终按当前有效版本定价。

---

## 5. 横切关注点

### 5.1 两套响应 envelope 兼容（契约 1.4）

```ts
function normalize(raw: any): { ok: boolean; data?: any; code: string | number; message: string } {
  const code = raw?.code;
  if (code === 0) return { ok: true, data: raw.data, code: 0, message: raw.message ?? '' };
  // 数字 code（业务错误）与字符串 code（中间件早退）都走失败分支
  return { ok: false, code, message: raw?.message ?? 'UNKNOWN_ERROR', data: raw?.metadata };
}
```

判定：数字 `0` 为成功；其余（数字或字符串）均为失败，按契约 1.5 的 `message` 分支处理。

### 5.2 幂等键管理（契约 1.3 / 第 5 章）

- 生成时机：每次**逻辑用户操作**生成一次（UUID v4），存在该操作的请求上下文对象里。
- 复用规则：网络重试、超时重放、429/503 退避重试**复用同一键**；用户主动再次点击生成新键。
- 生命周期：仅内存，不持久化到磁盘；渲染进程不接触；账号切换清空。
- 与身份绑定：幂等键与当时 payload 里的 `username` 关联，切账号后禁止用旧键重放。
- **重发必重加密**：网络重试/超时重放复用同一幂等键，但每次都重新调用 `seal()` 生成新 SM4 会话密钥 / `iv` / `nonce` / `issued_at`；会话密钥与 IV 绝不复用。

### 5.3 重试策略

| 错误类型 | 策略 |
|---|---|
| 网络错误 / 429 / 503 | 指数退避（如 300ms → 600ms → 1.2s），最多 3 次，**复用同一幂等键** |
| 业务明确响应（已签到、不符合条件、余额不足、版本变化、任务额度耗尽） | **不重试**，展示原因 |
| 超时（结果未知） | 不生成新请求、不认定成功；用原 `completion_ref` + 原幂等键重放（完成请求靠 `completion_ref` 去重、签到靠服务端 `cycle_key`），或调 `/growth/me` / `/growth/ledger` 核实。v4.0 完成请求无 3.11 状态轮询接口；仅订单履约（3.12）有状态查询 |
| 加密类（`CRYPTO_*`：信封/解封/MAC/解密/重放/错投） | **不重试**；核对双方写死的 SM2 公私钥是否配对、`kid`、MAC 拼接顺序、客户端时钟，属配置问题非瞬时故障 |
| 映射类（`GROWTH_USER_NOT_FOUND`） | 不重试；提示账号未开通（账务映射失败，非身份校验） |

### 5.4 轮询（v4.0：完成请求不再轮询）

- **完成请求（3.9）同步发分**，响应即最终结果，无需轮询；原 3.11 状态查询接口已删除。
- 仅订单履约（3.12）仍需轮询，按响应 `poll_after_ms`。
- 有限次数 + 页面卸载/切后台即停止；超限展示"处理中，请稍后刷新"。

### 5.5 开发/离线模式

- Mock 模式（P0/P1）返回契约样例数据；默认不发真实积分。
- 真实积分发放完全由对端测试环境策略控制，AIWork 不模拟发奖。

---

## 6. Mock Server 与并行联调

### 6.1 阶段策略

| 阶段 | sub2api 状态 | AIWork 对策 |
|---|---|---|
| P0 契约冻结 | 未开发 | AIWork 用本地 Mock（按契约第 3 章逐接口实现样例响应），UI 与流程完整可跑 |
| P1 读接口就绪 | `/me` `/tasks` `/ledger` `/products` 可返回 | 读接口切真，写接口仍 Mock |
| P2 命令就绪 | 签到/完成请求/兑换可返回 | 全量切真，保留 Mock 开关用于回归 |
| P3 联调 | 全链路 | 字段级 diff 校验，状态机/错误码逐条走查 |

### 6.2 Mock 实现要求

- 按契约第 3 章每个接口至少实现：1 个成功样例 + 1 个业务错误样例（如签到 `ALREADY_CLAIMED`、兑换 `INSUFFICIENT_POINTS`）。
- Mock Server 内置**测试专用 SM2 私钥**（公钥给 AIWork），按契约 1.3 实现解封装校验（`kid`/SM2 解封/SM3-HMAC/SM4 解密/时间窗/nonce/`path`）；也支持"跳过解密"开关以便纯 UI 调试。
- Mock 也走同样的响应 envelope（数字 0 / 字符串 code），否则归一化逻辑测不到。
- Mock 保留请求日志，用于核对 AIWork 是否正确构造 SM2 数字信封（合法 `kid`/会话密钥与 IV 非复用/MAC 正确/`path` 匹配）、payload 是否含禁止字段。
- Mock 数据中的 `user_id` 固定为一个测试值，便于断言"客户端不信任请求体 user_id"。

### 6.3 联调检查清单

- [ ] Renderer 无法直接访问 sub2api（preload 只暴露白名单 API）
- [ ] 只有 SM2 公钥在 Main 的 `crypto.ts` 读取（无私钥），Renderer 与日志中均无明文 payload
- [ ] 公钥为写死常量（`GROWTH_SM2_PUBKEY`），与 sub2api 私钥配对；每个请求都是 SM2 数字信封：`kid` 固定为 `growth-sm2-v1`、每次会话密钥/IV 随机不复用、MAC 正确、`path` 与实际接口一致
- [ ] 命令接口都有 payload `idempotency_key`，重试复用、新操作换新，且重发重新加密
- [ ] 解密后的 payload 无契约第 6 节禁止字段，`username` 来自既有登录态（登录功能未改）
- [ ] 两套 envelope 都能正确解析；业务码分支提示正确
- [ ] 签到/兑换/完成请求无乐观更新；失败保留上次余额
- [ ] 切账号后 store 清空、旧幂等键不再使用
- [ ] 任务完成由 AIWork 判定并生成稳定 `completion_ref`；重试复用、重新完成任务换新值
- [ ] 完成请求同步发分、无 3.11 轮询；仅订单履约轮询有限次数、页面卸载停止
- [ ] 超时重放不产生第二笔积分（用 Mock 幂等日志核对 `completion_ref` 去重）

---

## 7. AIWork 侧任务拆分

| 阶段 | 任务 | 产出 | 验收 |
|---|---|---|---|
| A0 契约 | 确认第 1 章技术栈假设与 3 个回复项 + sub2api 分配 SM2 密钥对（公钥写死进 Main、私钥写死进 sub2api）、国密库选型审批 | 确认清单 | 双方签字 |
| A1 骨架 | preload bridge + Main client/crypto SM2 封装 + 错误归一化 + Mock Server | 本地可跑 Mock | SM2 数字信封与契约 1.3 互操作测试通过、两套 envelope 归一化通过 |
| A2 只读 UI | 积分概览、任务列表/详情、流水、商品、订单页 | 只读页面 | 全部数据来自服务端快照，无本地计算 |
| A3 签到 | 签到请求 + 状态机 + 重试/幂等 | 签到可用 | 无乐观更新，重放不加分 |
| A4 完成判定与请求 | AIWork 侧任务完成判定逻辑 + `completion_ref` 生成 + 同步提交 #9 | 任务奖励闭环 | 完成判定正确、`completion_ref` 稳定去重、同步结果落任务卡 |
| A5 兑换 | 二次确认 + 下单 + 履约轮询 | 兑换闭环 | 版本变化/余额不足提示正确，失败保留余额 |
| A6 联调 | 切真 + 字段级 diff + 6.3 检查清单 | 联调通过 | 检查清单全绿 |

---

## 8. 对 sub2api 的依赖与阻塞

| 依赖 | 需要 sub2api 提供 | 阻塞 AIWork 阶段 |
|---|---|---|
| U 面 11 接口 | 契约 v4.0 第 2.1 节全部（全 POST + 加密信封；删除原完成请求状态查询接口） | A2～A5 |
| 加密配置 | sub2api 分配的测试 SM2 密钥对（公钥写死进 Main 常量、私钥写死进 sub2api 常量）、`kid` 固定 `growth-sm2-v1`、SM2/SM4/SM3 规范实现一致 | A1 起 |
| 网络边界 | 内网/VPN 可访问地址 + IP 白名单已放行 AIWork、双方 NTP 同步 | A2 起 |
| 测试环境 | 测试账号 email、测试总预算与任务额度、Mock/真实开关 | A6 |
| 任务完成判定规则 | 每类任务「何时算完成」的业务规则由 AIWork 定义；`completion_ref` 生成策略（设计文档 3.1） | A4 |

> 任一依赖未就绪时，AIWork 用 Mock 继续推进 UI 与流程，不阻塞在本地开发。
