# sub2api 成长计划对外接口与调用示例

> 适用范围：AIWork Main 调用 sub2api U 面。  
> 契约版本：`v4.0`，以 [GROWTH_PLAN_AIWORK_API.md](GROWTH_PLAN_AIWORK_API.md) 为唯一字段真相源。  
> 不包含 `/api/v1/admin/growth/*`（管理端）和 `/api/v1/user/growth/*`（sub2api Web 自用）。

## 1. 接入信息

| 项 | 值 |
|---|---|
| Base URL | `https://<sub2api-host>/api/v1` |
| 固定信封标识 | `kid = growth-sm2-v1` |
| HTTP 方法 | 所有 U 面接口均为 `POST` |
| Content-Type | `application/json` |
| 认证 | 无 JWT、无 API Key、无身份 Header；请求体使用 SM2 数字信封 |
| 时间窗 | 默认 `issued_at` 与服务端时间相差不超过 5 分钟 |
| 返回 | 响应为明文 JSON，不加密 |

生产固定 SM2 公钥（可在 AIWork Main 中写死，只允许 Main 使用）：

```text
04 82faea4b9093687dd34416141fd05ac825944de64a55e60be17873e45395249
   31a90619fe65ca78657dabe658ff7278c5a0a5161981c9e3a55ed75115fe6b72d
```

合并后的完整值为：

```text
0482faea4b9093687dd34416141fd05ac825944de64a55e60be17873e4539524931a90619fe65ca78657dabe658ff7278c5a0a5161981c9e3a55ed75115fe6b72d
```

SM2 私钥只存在于 sub2api 服务端，任何情况下都不得下发、记录或写入 AIWork。

## 2. 统一请求信封

HTTP 请求体中只能看到加密信封：

```json
{
  "kid": "growth-sm2-v1",
  "encrypted_key": "<base64>",
  "iv": "<base64>",
  "ciphertext": "<base64>",
  "mac": "<base64>"
}
```

加密前的 payload：

```json
{
  "path": "/growth/me",
  "username": "user@example.com",
  "params": {},
  "request_id": "trace-01J...",
  "issued_at": 1763424000,
  "nonce": "0b7a6f2b-9b4a-45ed-91f7-523713c7687f"
}
```

命令接口额外携带顶层幂等键：

```json
{
  "path": "/growth/checkins",
  "username": "user@example.com",
  "params": {
    "request_id": "trace-checkin-01"
  },
  "request_id": "trace-checkin-01",
  "issued_at": 1763424000,
  "nonce": "d1b60a6a-110d-44e1-8f5d-5cb7f0cc85df",
  "idempotency_key": "a6f1c756-09fb-4ba2-b3f9-bddc856f0f6b"
}
```

固定规则：

1. `path` 使用路由模板；实际 URL 中要带真实路径参数。
2. `params` 始终是对象，无参数时传 `{}`。
3. `nonce` 每次 HTTP 请求都必须新随机，重试也不能复用。
4. 命令重试复用同一个 `idempotency_key`，但必须重新生成 `nonce`、`issued_at`、SM4 会话密钥和 IV。
5. 禁止在 `params` 或顶层传 `user_id`、`points`、`balance`、`rule_version`、`business_date`、`cycle_key`、`price`、`points_cost` 等账务字段。

### 加密组合

```text
SM4 会话密钥：16 字节随机值
SM4 IV：16 字节随机值
业务加密：SM4-CBC + PKCS#7
会话密钥封装：SM2，C1C3C2
完整性：SM3-HMAC(session_key, kid || encrypted_key || iv || ciphertext)
```

MAC 输入中的四个值使用最终 base64 字符串拼接，不先做 base64 解码。服务端先验证 MAC，再解密 payload。

## 3. 对外接口总表

共 **11 个操作**，由 **10 个 URL 路径**承载；`/growth/exchanges` 根据参数区分“订单列表”和“创建兑换”。

| # | 操作 | URL | payload.path | 幂等键 |
|---|---|---|---|---|
| 1 | 我的积分概览 | `POST /api/v1/growth/me` | `/growth/me` | 否 |
| 2 | 任务列表 | `POST /api/v1/growth/tasks` | `/growth/tasks` | 否 |
| 3 | 任务详情 | `POST /api/v1/growth/tasks/{task_id}` | `/growth/tasks/:task_id` | 否 |
| 4 | 积分流水 | `POST /api/v1/growth/ledger` | `/growth/ledger` | 否 |
| 5 | 商品列表 | `POST /api/v1/growth/products` | `/growth/products` | 否 |
| 6 | 商品详情 | `POST /api/v1/growth/products/{product_id}` | `/growth/products/:product_id` | 否 |
| 7 | 我的兑换列表 | `POST /api/v1/growth/exchanges` | `/growth/exchanges` | 否 |
| 8 | 签到发分 | `POST /api/v1/growth/checkins` | `/growth/checkins` | **是** |
| 9 | 任务完成发分 | `POST /api/v1/growth/tasks/{task_id}/completion-requests` | `/growth/tasks/:task_id/completion-requests` | **是** |
| 10 | 创建兑换 | `POST /api/v1/growth/exchanges` | `/growth/exchanges` | **是** |
| 11 | 兑换订单状态 | `POST /api/v1/growth/exchanges/{order_id}` | `/growth/exchanges/:order_id` | 否 |

## 4. TypeScript 通用调用封装

以下代码负责构造 payload、调用加密封装和统一处理两种响应 envelope。`sealGrowthEnvelope` 必须由 AIWork Main 按第 2 节规范实现，且必须用 sub2api 的固定向量做互操作测试。

```ts
type GrowthEnvelope = {
  kid: 'growth-sm2-v1'
  encrypted_key: string
  iv: string
  ciphertext: string
  mac: string
}

type GrowthPayload = {
  path: string
  username: string
  params: Record<string, unknown>
  request_id: string
  issued_at: number
  nonce: string
  idempotency_key?: string
}

type GrowthSuccess<T> = {
  code: 0
  message: string
  data: T
}

type GrowthError = {
  code: number | string
  message: string
  reason?: string
  metadata?: Record<string, unknown>
}

// AIWork Main 中实现：此函数只在 Main 可见。
declare function sealGrowthEnvelope(payload: GrowthPayload): Promise<GrowthEnvelope>

const GROWTH_BASE_URL = 'https://sub2api.example.com/api/v1'

async function callGrowth<T>(input: {
  pathTemplate: string
  requestPath: string
  username: string
  params: Record<string, unknown>
  idempotencyKey?: string
}): Promise<T> {
  const payload: GrowthPayload = {
    path: input.pathTemplate,
    username: input.username,
    params: input.params,
    request_id: crypto.randomUUID(),
    issued_at: Math.floor(Date.now() / 1000),
    nonce: crypto.randomUUID(),
    ...(input.idempotencyKey ? { idempotency_key: input.idempotencyKey } : {}),
  }

  // 每次发送（包括重试）都重新封装，会话密钥、IV、nonce、issued_at 均重新生成。
  const envelope = await sealGrowthEnvelope(payload)
  const response = await fetch(`${GROWTH_BASE_URL}${input.requestPath}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(envelope),
  })

  const body = await response.json() as GrowthSuccess<T> | GrowthError

  // 数字 0 才是成功；CRYPTO_* 错误时 code 是字符串。
  if (body.code !== 0) {
    const error = body as GrowthError
    throw new Error(`${String(error.code)}: ${error.reason ?? error.message}`)
  }

  return body.data
}
```

## 5. 各接口调用示例

以下示例展示解密前的 payload 业务内容；实际 HTTP body 必须替换为加密信封。

### 5.1 我的积分概览

```ts
const me = await callGrowth<MeData>({
  pathTemplate: '/growth/me',
  requestPath: '/growth/me',
  username: 'user@example.com',
  params: {},
})
```

payload：

```json
{
  "path": "/growth/me",
  "username": "user@example.com",
  "params": {},
  "request_id": "trace-me-01",
  "issued_at": 1763424000,
  "nonce": "9acf6745-910b-4c0e-a8a5-ee8f50caa760"
}
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 12345,
    "available_points": 260,
    "earned_total": 320,
    "spent_total": 60,
    "balance_updated_at": "2026-11-18T10:00:00+08:00",
    "checkin": {
      "business_date": "2026-11-18",
      "timezone": "Asia/Shanghai",
      "status": "AVAILABLE",
      "rule_version": 3,
      "reward_hint": 10
    },
    "task_summary": {
      "in_progress": 2,
      "completed_today": 1
    },
    "server_time": "2026-11-18T10:05:00+08:00"
  }
}
```

### 5.2 任务列表

```ts
const tasks = await callGrowth<PageData<TaskItem>>({
  pathTemplate: '/growth/tasks',
  requestPath: '/growth/tasks',
  username: 'user@example.com',
  params: {
    page: 1,
    page_size: 20,
    status: 'AVAILABLE',
  },
})
```

响应核心字段：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "task_id": 9001,
        "task_code": "daily_checkin",
        "title": "每日签到",
        "category": "DAILY",
        "task_type": "CHECKIN",
        "status": "AVAILABLE",
        "progress": 0,
        "target": 1,
        "cycle_key": "2026-11-18",
        "reward_hint": 10,
        "reward_pool": "CHECKIN",
        "rule_version": 3,
        "action": "CHECKIN",
        "sort_order": 1
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20,
    "pages": 1
  }
}
```

### 5.3 任务详情

注意：URL 与 `params.task_id` 必须一致。

```ts
const taskId = 9001
const task = await callGrowth<TaskDetail>({
  pathTemplate: '/growth/tasks/:task_id',
  requestPath: `/growth/tasks/${taskId}`,
  username: 'user@example.com',
  params: { task_id: taskId },
})
```

payload：

```json
{
  "path": "/growth/tasks/:task_id",
  "username": "user@example.com",
  "params": { "task_id": 9001 },
  "request_id": "trace-task-01",
  "issued_at": 1763424000,
  "nonce": "48dc344c-f0ce-4087-a9fc-94b1d799cd17"
}
```

### 5.4 积分流水

```ts
const ledger = await callGrowth<LedgerPage>({
  pathTemplate: '/growth/ledger',
  requestPath: '/growth/ledger',
  username: 'user@example.com',
  params: {
    limit: 20,
    type: 'EARN',
  },
})
```

下一页使用上一响应中的 `next_cursor`：

```ts
const nextLedger = await callGrowth<LedgerPage>({
  pathTemplate: '/growth/ledger',
  requestPath: '/growth/ledger',
  username: 'user@example.com',
  params: {
    before_entry_id: Number(ledger.next_cursor),
    limit: 20,
    type: 'EARN',
  },
})
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "entry_id": 88001,
        "transfer_id": 77001,
        "type": "EARN",
        "direction": "IN",
        "points": 10,
        "balance_after": 260,
        "biz_type": "CHECKIN",
        "title": "每日签到",
        "created_at": "2026-11-18T09:02:11.123+08:00"
      }
    ],
    "next_cursor": "88000",
    "has_more": true
  }
}
```

### 5.5 商品列表

```ts
const products = await callGrowth<PageData<ProductItem>>({
  pathTemplate: '/growth/products',
  requestPath: '/growth/products',
  username: 'user@example.com',
  params: { page: 1, page_size: 20 },
})
```

### 5.6 商品详情

```ts
const productId = 5001
const product = await callGrowth<ProductDetail>({
  pathTemplate: '/growth/products/:product_id',
  requestPath: `/growth/products/${productId}`,
  username: 'user@example.com',
  params: { product_id: productId },
})
```

响应核心字段：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "product_id": 5001,
    "product_code": "token_1m",
    "title": "100万 Token",
    "product_type": "TOKEN_QUOTA",
    "version": 1,
    "points_cost": 50,
    "benefit_summary": "兑换后发放 1,000,000 Token 额度",
    "stock_hint": "100 件",
    "per_user_limit": 3,
    "my_redeemed_count": 0,
    "status": "ON_SHELF",
    "benefit_detail": {},
    "redeem_window": "兑换后进入履约队列",
    "remaining_quota": 100,
    "eligible": true,
    "ineligible_reason": ""
  }
}
```

### 5.7 签到

AIWork 先在本地判定签到完成，再调用该接口；不要在本地直接加积分。

```ts
const idempotencyKey = crypto.randomUUID()

const checkin = await callGrowth<CheckinResult>({
  pathTemplate: '/growth/checkins',
  requestPath: '/growth/checkins',
  username: 'user@example.com',
  params: {
    request_id: `checkin-${new Date().toISOString().slice(0, 10)}`,
  },
  idempotencyKey,
})
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "result": "AWARDED",
    "claim_id": 66001,
    "transfer_id": 77001,
    "business_date": "2026-11-18",
    "timezone": "Asia/Shanghai",
    "awarded_points": 10,
    "current_balance": 270,
    "rule_version": 3,
    "server_time": "2026-11-18T10:10:00+08:00"
  }
}
```

可能结果：`AWARDED`、`ALREADY_CLAIMED`、`NOT_ELIGIBLE`、`UNAVAILABLE`。

### 5.8 任务完成

AIWork 判定任务完成后生成 `completion_ref`，再调用此接口。sub2api 同步计算发分，不轮询。

```ts
const taskId = 9002
const completionRef = 'aiwork-local-task-instance-12345'

const completion = await callGrowth<CompletionResult>({
  pathTemplate: '/growth/tasks/:task_id/completion-requests',
  requestPath: `/growth/tasks/${taskId}/completion-requests`,
  username: 'user@example.com',
  params: {
    task_id: taskId,
    completion_ref: completionRef,
    request_id: 'trace-completion-01',
  },
  idempotencyKey: crypto.randomUUID(),
})
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "request_id": "cr-88001",
    "result": "AWARDED",
    "task_id": 9002,
    "completion_ref": "aiwork-local-task-instance-12345",
    "transfer_id": 77003,
    "awarded_points": 20,
    "current_balance": 290,
    "rule_version": 7,
    "cycle_key": "2026-11-18",
    "server_time": "2026-11-18T10:12:00+08:00"
  }
}
```

可能结果：`AWARDED`、`ALREADY_COMPLETED`、`NOT_ELIGIBLE`。  
`ALREADY_COMPLETED` 仅表示同一 `completion_ref` 已处理；新的完成实例使用新的 `completion_ref` 和幂等键，在每日/每人上限和任务额度未耗尽时仍可再次返回 `AWARDED`。

不得调用已删除的 `/growth/completion-requests/{request_id}` 轮询接口。

### 5.9 创建兑换

`expected_version` 用于发现商品目录过期，不是服务端定价依据。

```ts
const order = await callGrowth<ExchangeCreateResult>({
  pathTemplate: '/growth/exchanges',
  requestPath: '/growth/exchanges',
  username: 'user@example.com',
  params: {
    product_id: 5001,
    expected_version: 1,
    request_id: 'trace-exchange-01',
  },
  idempotencyKey: crypto.randomUUID(),
})
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "order_id": 44001,
    "order_no": "EX20261118101200000001",
    "status": "FULFILLING",
    "product_version": 1,
    "points_cost": 50,
    "transfer_id": 77002,
    "current_balance": 240,
    "fulfillment_status": "PENDING",
    "server_time": "2026-11-18T10:12:00+08:00"
  }
}
```

错误示例：

```json
{
  "code": 409,
  "message": "INSUFFICIENT_POINTS",
  "reason": "余额不足",
  "metadata": {
    "current_balance": 40,
    "required_points": 50
  }
}
```

### 5.10 我的兑换列表

```ts
const orders = await callGrowth<PageData<ExchangeItem>>({
  pathTemplate: '/growth/exchanges',
  requestPath: '/growth/exchanges',
  username: 'user@example.com',
  params: {
    page: 1,
    page_size: 20,
    status: 'FULFILLING',
  },
})
```

### 5.11 查询兑换订单状态

创建兑换后建议最多轮询 5 次、间隔 2 秒；成功、失败、退款或页面隐藏后停止。

```ts
const orderId = 44001
const orderStatus = await callGrowth<ExchangeOrderDetail>({
  pathTemplate: '/growth/exchanges/:order_id',
  requestPath: `/growth/exchanges/${orderId}`,
  username: 'user@example.com',
  params: { order_id: orderId },
})
```

状态：`CREATED`、`FULFILLING`、`SUCCEEDED`、`FAILED`、`REFUNDED`。

## 6. cURL 示例

由于业务参数在密文中，cURL 只能展示信封传输方式。下列文件内容必须由 AIWork Main 的 `sealGrowthEnvelope` 生成：

```bash
curl -X POST 'https://sub2api.example.com/api/v1/growth/me' \
  -H 'Content-Type: application/json' \
  -H 'X-Growth-Client-Version: 1.0.0' \
  --data @growth-envelope.json
```

`growth-envelope.json`：

```json
{
  "kid": "growth-sm2-v1",
  "encrypted_key": "<base64>",
  "iv": "<base64>",
  "ciphertext": "<base64>",
  "mac": "<base64>"
}
```

## 7. 响应与重试规则

业务响应：

```json
{ "code": 0, "message": "success", "data": {} }
```

业务错误：

```json
{ "code": 409, "message": "INSUFFICIENT_POINTS", "reason": "余额不足" }
```

加密中间件早退：

```json
{ "code": "CRYPTO_MAC_INVALID", "message": "growth envelope integrity check failed" }
```

| 类型 | 处理 |
|---|---|
| 网络错误、429、503 | 有上限指数退避；命令复用原 `idempotency_key`，但重新生成信封随机量 |
| `CRYPTO_*` | 不重试；检查固定公钥、C1C3C2、MAC 拼接、时钟和业务时区 |
| `GROWTH_USER_NOT_FOUND` | 不重试；提示账号未映射到 sub2api |
| `INSUFFICIENT_POINTS`、`PRODUCT_SOLD_OUT`、`PURCHASE_LIMIT_REACHED` | 不重试；刷新 `/growth/me` 或商品详情 |
| `IDEMPOTENCY_IN_PROGRESS` | 退避后复用同一幂等键重发 |
| `IDEMPOTENCY_CONFLICT` | 不重试；同键被用于不同请求体 |
| `AWARDED`、`ALREADY_COMPLETED`、`SUCCEEDED`、`REFUNDED` | 终态；停止任务/订单轮询 |

## 8. 联调检查清单

- AIWork Main 持有的公钥与 sub2api 固定私钥配对。
- SM2 密文编码统一为 `C1C3C2`，不是 ASN.1 DER。
- SM4-CBC 使用 PKCS#7 填充。
- SM3-HMAC 输入使用 base64 字符串拼接，顺序固定。
- 每次请求使用新的 SM4 密钥、IV 和 nonce。
- payload 的 `path` 与实际 URL 路由模板一致。
- 路径参数同时出现在 URL 和 `params` 中且数值一致。
- 命令重试复用 `idempotency_key`，但不复用 nonce/会话密钥/IV。
- 不自行计算积分、余额、业务日期或周期。
- 不调用管理端、Web 自用端或已删除的 I 面/完成轮询接口。
