# AIWork ↔ sub2api 成长计划接口契约 v4.0（冻结）

> 配套设计：[GROWTH_PLAN_DESIGN.md](GROWTH_PLAN_DESIGN.md) ｜ 冻结日期：2026-11-17
> 本文是 **AIWork 与 sub2api 并行开发的唯一接口真相源**。sub2api 后管接口（`/api/v1/admin/growth/*`）不在本文范围。
> 两侧开发设计：[AIWORK_DEV](GROWTH_PLAN_AIWORK_DEV.md) ｜ [SUB2API_DEV](GROWTH_PLAN_SUB2API_DEV.md)

## 0. 冻结与变更规则（并行开发必读）

| 项 | 约定 |
|---|---|
| 版本 | `v4.0`，冻结于 2026-11-17。v2.0 → v3.0 为破坏性变更（对称 → 国密 SM2 数字信封）；v3.0 → v3.1 为非破坏性变更（密钥管理简化）；**v3.1 → v4.0 为破坏性变更**（**任务完成判定职责反转**：由 AIWork 判定完成、sub2api 只按规则计算发分；删除 I 面事件接口；完成请求状态机简化为同步发分）。**同日非破坏性措辞修订**：sub2api 内部预算模型由「多积分池」改为「总预算 + 任务额度」，`POOL_EXHAUSTED` 语义对应「任务额度耗尽 / 预算不足」；**接口路径、字段、枚举、信封均不变，AIWork 无需改动** |
| 唯一来源 | 请求/响应字段、路径、状态枚举、错误码、加密信封**只以本文为准**；两侧文档不一致时以本文为准并提 issue |
| 变更流程 | 字段增删改、算法/信封调整需提变更单 → 双方确认 → 版本升位（`v4.0`…）→ 同步两侧设计；**禁止单边偷改** |
| 兼容策略 | 新增响应字段算兼容；删除/重命名字段、改枚举语义、改信封格式算破坏性变更，需升 minor 并约定切换日 |
| Mock 依据 | AIWork Mock Server 与 sub2api 契约测试均按本文第 3 章 + 第 1.3 节加密规范实现；密钥用测试专用固定值 |
| 验收口径 | 「Mock 通过 → 真实 sub2api 通过」两阶段，字段级 diff 为零才算对齐 |

> **v2.0 → v3.0 变更摘要**：加密算法由双方共享的 AES-256-GCM 对称密钥，替换为**国密 SM2 非对称数字信封**——sub2api 持 SM2 私钥（永不出服务端），AIWork Main 持 SM2 公钥；每次请求随机生成 SM4 会话密钥，SM2 公钥加密会话密钥、SM4 加密 payload、SM3-HMAC 保障完整性。接口路径、方法（全 `POST`）、响应字段与业务码**均不变**，仅请求信封与加解密流程替换。「不做服务间身份校验、不改 AIWork 登录」的边界保持不变。
>
> **v3.0 → v3.1 变更摘要**：SM2 密钥对改为**直接分配、双方写死固定值**——由 sub2api 侧生成一对密钥，私钥写死在服务端、公钥写死在 AIWork Main；**取消 `kid` 轮换、取消公钥 SM3 指纹校验、取消 KMS / 配置密文注入**。信封五字段、SM2/SM4/SM3-HMAC 算法、12 个接口、响应与业务码全部不变；`kid` 字段仍保留（固定为 `growth-sm2-v1`），仅作信封版本标识与将来万一需要换密钥时的扩展位，服务端按固定映射取私钥。
>
> **v3.1 → v4.0 变更摘要（职责反转，破坏性）**：**任务/签到是否完成改由 AIWork 侧判定**；AIWork 提交完成请求时携带完成凭据 `completion_ref`，**sub2api 不再回查 `usage_logs`、不接收 I 面事件、不做证据核验**，收到完成请求即视为「任务已完成」，仅按当前生效规则计算应发积分、执行账务防重并发分。具体影响：① 完成请求 #9 的 `params` 由 `run_id`（证据回查键）改为 `completion_ref`（AIWork 侧完成实例标识，仅用于业务去重/审计）；② 完成请求状态机由「异步核验 `PENDING_VERIFICATION` → 轮询」简化为**同步发分**（`AWARDED` / `ALREADY_COMPLETED` / `NOT_ELIGIBLE`），删除 `REJECTED`（无证据核验即无「证据不可信」态）；③ **删除 I 面 `POST /integrations/growth/task-events`**（不再有可信事件接入），三平面收敛为 U + M 两平面；④ `growth_task_events` 表删除，新增 `growth_completion_requests` 表承接完成请求记录与业务去重；⑤ 签到 #8 语义调整为「AIWork 判定签到完成后提交，sub2api 按服务端业务时区算 `cycle_key` + 规则发分 + 防重复」。**加密信封、SM2 密钥模型、账务/幂等双层、商品/兑换接口、响应封装、错误码体系全部不变。**

---

## 1. 通用约定

### 1.1 地址与调用方

| 项 | 值 |
|---|---|
| U 面 Base URL | `https://<sub2api-host>/api/v1/growth` |
| 调用方 | AIWork Electron **Main**；Renderer 经类型化 `window.aiwork` preload API 间接访问 |
| ~~I 面 Base URL~~ | **v4.0 起删除 I 面**（原 `/api/v1/integrations/growth`），任务完成判定与提交全部收敛到 U 面完成请求接口 |
| 协议 | HTTPS + `Content-Type: application/json`；**所有接口统一 `POST`** |

### 1.2 信任模型：SM2 加密报文互信（无身份校验）

> **决策变更**：内部系统不做用户级认证体系，**不做任何服务间身份校验**，**不改动 AIWork 现有登录功能**。两服务之间只有接口往来；唯一信任根是**国密 SM2 非对称加密报文**——sub2api 用私钥能解开报文即视为可信调用。

| 平面 | 信任方式 | 说明 |
|---|---|---|
| U 面 | 请求体 SM2 数字信封（SM2 封装会话密钥 + SM4 加密 + SM3-HMAC） | 服务端用 SM2 私钥成功解封装且 MAC 校验通过即信任；无 `Authorization` / `X-Growth-User` 头。**任务完成也在此声明** |
| M 面 | 沿用 sub2api `AdminIdentityAuth` + RBAC | **仅 sub2api 后管自用**，不对外、不属两服务往来范围 |

> **v4.0 信任边界变化**：任务是否完成由 **AIWork 判定并在 U 面声明**，sub2api 不再做「完成证据核验」。信任根仍是「私钥解封装成功即可信调用方」——这一原则现在同时覆盖「调用合法性」与「任务完成声明」。这意味着 sub2api 无法独立证明任务真的完成，**这是可信调用方模型的直接推论，属产品明确取舍**（见总体设计第 2.4 节风险书面化）。

**密钥归属与保管（v3.1：直接分配、双方写死）**：

| 密钥 | 持有方 | 保管要求 |
|---|---|---|
| SM2 **公钥** | AIWork Main（`crypto.ts` 中写死的固定常量）、sub2api（配私钥时自带） | 由 sub2api 侧生成密钥对后**直接分配**给 AIWork；公钥可公开，写死在 Main 源码常量中。**不做指纹校验、不做运行时下发**——公钥被替换的风险由「内网边界 + 代码库访问控制」承接，属内部系统明确取舍 |
| SM2 **私钥** | **仅 sub2api 服务端**（写死的固定常量） | **不下发客户端、不进日志、不进任何响应**；直接写死在服务端密钥常量中。**不做轮换、不做 KMS、不做配置密文注入** |

> **一次性分配约定**：密钥对由一方生成后同时固化到两个项目——AIWork 只拿到公钥，sub2api 只保留私钥。任何一方的密钥需要变更时，必须**双方同步改代码并发版**（无热切换窗口），因此上线前务必确认密钥对正确配对。

> **非对称加密只保证「机密性 + 完整性」，不保证「调用方身份」**：SM2 公钥是可公开的，任何拿到公钥的一方都能构造出能通过解密与 MAC 校验的合法报文。这与「不做服务间身份校验」的产品决策一致，是可信调用方模型，不是身份认证模型。

**边界前提（必须书面确认）**：

1. AIWork 侧只持有**公钥**，不存在“私钥被逆向提取”风险。密钥写死意味着**没有热轮换能力**：泄露或配错只能双方改代码发版，故需确认代码库访问受控、构建产物不被篡改、内网/VPN 可达 + IP 白名单。
2. 解密后的 `username` 只做**账务映射**（`GetByEmail` → `users.id`），用于积分记账和幂等隔离；**这不是身份校验**——不验签、不校验账号状态、不核对凭证。映射失败（用户不存在）按数据错误返回，不叫鉴权失败。
3. 请求体不传 `user_id`，只传业务用户名；权威账务标识一律为映射得到的 `users.id`。
4. U 面仅内网 / VPN 可达 + IP 白名单 + HTTPS；若暴露公网或引入不可信调用方，必须重新评审（加密 + 内网边界不再足够）。
5. 双方 NTP 同步，`issued_at` 时间窗（默认 ±5 分钟）才可靠。

### 1.3 加密信封规范（核心，双方必须逐字一致）

**算法组合（国密，固定不可协商）**：

| 用途 | 算法 | 参数固定值 |
|---|---|---|
| 会话密钥封装 | **SM2** 非对称加密（GM/T 0003.4） | 公钥加密 / 私钥解密；密文编码 `C1C3C2` 拼接（tjfoc/gmsm 默认，**不**用 ASN.1 DER） |
| 报文加密 | **SM4-CBC** 对称加密 | 每请求随机 16 字节会话密钥 + 16 字节 IV；PKCS#7 填充 |
| 完整性 | **SM3-HMAC** | key 取会话密钥，覆盖 `kid \|\| encrypted_key \|\| iv \|\| ciphertext`，输出 32 字节 |

> v3.1 起**不再计算公钥 SM3 指纹**（取消指纹校验）；SM3 仅用于报文的 SM3-HMAC 完整性校验，不可混淆。

**为什么是混合信封**：SM2 单次加密明文有长度限制（约 曲线阶相关，远小于业务报文），因此只用 SM2 加密 16 字节 SM4 会话密钥，业务 payload 用 SM4 加密；SM3-HMAC 提供完整性（对应 AES-GCM 的认证标签）。这是国密场景的标准做法。

**请求体（明文 HTTP body，即"加密信封"）**：

```json
{
  "kid": "growth-sm2-v1",
  "encrypted_key": "<base64，SM2 公钥加密的 16 字节 SM4 会话密钥，C1C3C2 拼接>",
  "iv": "<base64，16 字节随机 SM4-CBC IV>",
  "ciphertext": "<base64，SM4-CBC 加密 payload 的密文>",
  "mac": "<base64，SM3-HMAC 32 字节>"
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `kid` | 是 | 信封版本标识，明文；固定为 `growth-sm2-v1`，双方写死（服务端按此固定值取写死的私钥）。**不做轮换**；保留字段仅作扩展位，服务端校验不等于固定值即返回 `CRYPTO_KEY_UNKNOWN` |
| `encrypted_key` | 是 | SM2 公钥加密的 16 字节 SM4 会话密钥，`C1C3C2` 裸拼接后 base64；每请求重新随机生成会话密钥 |
| `iv` | 是 | 每次请求**重新随机生成**的 16 字节 SM4-CBC IV，base64；同一会话密钥下不得复用 |
| `ciphertext` | 是 | SM4-CBC 加密下述「明文 payload」JSON 的密文，base64 |
| `mac` | 是 | SM3-HMAC(会话密钥, `kid‖encrypted_key‖iv‖ciphertext`)，base64；**先校验 MAC 再解密**，失败 `CRYPTO_MAC_INVALID` |

**服务端解封装顺序（固定）**：按 `kid` 取私钥 → SM2 解密 `encrypted_key` 得会话密钥 → SM3-HMAC 校验 `mac`（不一致直接拒绝，**不做后续解密**，防填充/oracle 攻击）→ SM4-CBC 解密 `ciphertext` → 解析 payload。

**待加密的「明文 payload」**（加密前的 JSON 对象，不直接出现在 HTTP 报文里）：

```json
{
  "path": "/growth/checkins",
  "username": "zhangsan@corp.com",
  "params": { },
  "request_id": "trace-9f1c...",
  "issued_at": 1763389980,
  "nonce": "b7e1c2a4-...",
  "idempotency_key": "9f1c2b7e-..."
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `path` | 是 | 目标接口的**路由模板**（Gin 风格，路径参数用 `:`，如 `/growth/tasks/:task_id`；无参接口为 `/growth/me`）。服务端用 `c.FullPath()` 逐字比对，用于防止密文错投到别的接口 |
| `username` | 是 | 当前用户的业务标识（email）；**仅用于映射到 `users.id`，不做身份校验**。AIWork 从既有登录态读取，登录功能零改动 |
| `params` | 是 | 该接口的业务参数对象，见第 3 章各接口的 `params` 定义；无参数传 `{}` |
| `request_id` | 否 | 链路追踪 ID，不作为身份或发奖依据 |
| `issued_at` | 是 | Unix 秒时间戳，用于防重放；超出服务端时间窗（默认 ±5 分钟）拒绝 |
| `nonce` | 是 | 每次请求随机 UUID，配合 `issued_at` 防重放；命令接口可直接复用 `idempotency_key` |
| `idempotency_key` | 命令接口必填 | 一次逻辑请求生成一次；网络重试/超时重放**必须复用同键**；新操作生成新键 |

**加密/解密参考流程**（伪代码，双方实现须等价；国密库建议 Go：`github.com/tjfoc/gmsm`，Node：`sm-crypto`，均需安全团队确认版本）：

```text
加密（AIWork Main，只用公钥）：
  payload     = { path, username, params, request_id, issued_at, nonce, idempotency_key? }
  sm4Key      = randomBytes(16)                  // 每请求新会话密钥，绝不复用
  iv          = randomBytes(16)                  // SM4-CBC IV，每请求新随机
  ciphertext  = SM4-CBC-Encrypt(sm4Key, iv, utf8(json(payload)), PKCS7)
  encryptedKey= SM2-Encrypt(GROWTH_SM2_PUBKEY, sm4Key)  // 写死的公钥常量；C1C3C2 拼接
  mac         = SM3-HMAC(sm4Key, kid||encryptedKey||iv||ciphertext)
  body        = { kid, encrypted_key: b64(encryptedKey), iv: b64(iv),
                  ciphertext: b64(ciphertext), mac: b64(mac) }
  POST <BaseURL><path>  body=json(body)

解密（sub2api 中间件，先于一切业务）：
  body        = parseJSON(requestBody)           // 失败 → CRYPTO_ENVELOPE_INVALID
  if body.kid != "growth-sm2-v1" → CRYPTO_KEY_UNKNOWN   // kid 固定，不匹配即拒绝
  priv        = GROWTH_SM2_PRIVKEY            // 写死的私钥常量
  ek/iv/ct/mac= b64dec(各字段)                    // 长度/格式非法 → CRYPTO_ENVELOPE_INVALID
  sm4Key      = SM2-Decrypt(priv, ek)            // 失败 → CRYPTO_KEY_UNWRAP_FAILED
  expectMac   = SM3-HMAC(sm4Key, kid||ek||iv||ct)
  if !constantTimeEqual(expectMac, mac) → CRYPTO_MAC_INVALID   // 先校验，不做后续解密
  plain       = SM4-CBC-Decrypt(sm4Key, iv, ct)  // 失败 → CRYPTO_DECRYPT_FAILED
  payload     = parseJSON(plain)                  // 失败 → CRYPTO_PAYLOAD_INVALID
  check |issued_at - now| <= window && !seen(nonce)  // 否则 → CRYPTO_REPLAY_REJECTED
  check payload.path == 当前路由 path             // 否则 → CRYPTO_PATH_MISMATCH
  ctx.set("growth_payload", payload)             // 供 handler 读取
```

**实现一致性要点**（联调重点）：
- SM2 密文编码必须双方都是 `C1C3C2` 拼接（32+32+明文长度字节），不得一端 ASN.1 一端裸拼接；tjfoc/gmsm 与 sm-crypto 均需显式确认输出格式。
- SM3-HMAC 拼接顺序 `kid || encrypted_key || iv || ciphertext` 用**base64 字符串**拼接（非解码后的字节），双方一致。
- MAC 校验必须用常量时间比较。
- SM4 会话密钥、`iv`、`nonce` 都每请求新随机；重试时复用 `idempotency_key`，但**重新生成会话密钥/IV 并重新走完整加密**。

**响应**：保持明文（沿用 1.5 两套 envelope）。理由：用户仅要求请求报文加密，响应不含凭证，加密收益有限而双方成本显著上升。如后续要求双向加密，作为 `v3.x` 破坏性变更处理。

### 1.4 请求头

| 头 | 必填 | 说明 |
|---|---|---|
| `Content-Type` | 是 | 固定 `application/json` |
| `X-Request-Id` | 否 | 链路追踪；与 payload `request_id` 二选一即可，**均不作为身份或发奖依据** |
| `X-Growth-Client-Version` | 否 | 客户端版本，兼容与排障 |

> 身份、幂等键、业务参数**全部在加密 payload 内**；明文头不含任何身份信息。SM2 信封字段（`kid`/`encrypted_key`/`iv`/`ciphertext`/`mac`）在 HTTP body 中明文传输。

### 1.5 响应封装（**两套，AIWork 必须都兼容**，与 v1.0 一致）

业务响应（含业务错误）：

```json
{ "code": 0, "message": "success", "data": { } }
{ "code": 409, "message": "INSUFFICIENT_POINTS", "reason": "余额不足", "metadata": { "current_balance": 40, "required_points": 100 } }
```

解密中间件早退（404 关闭 / 415 / 429 等）是另一套 envelope，`code` 为**字符串**：

```json
{ "code": "CRYPTO_DECRYPT_FAILED", "message": "request payload decryption failed" }
```

> 判定规则：`code` 为数字 `0` 视为成功；`code` 非 0（数字或字符串）均视为失败，按 `message` 分支处理。

### 1.6 统一业务码

| HTTP | `message` | 含义 | AIWork 处理 |
|---|---|---|---|
| 400 | `INVALID_REQUEST` | 参数非法 | 提示并停止重试 |
| 400 | `CRYPTO_ENVELOPE_INVALID` | 信封 JSON 结构/字段长度非法 | 不重试；检查客户端加密封装 |
| 400 | `CRYPTO_KEY_UNKNOWN` | `kid` 不等于固定值 `growth-sm2-v1` | 不重试；客户端 `kid` 写死为固定值 |
| 400 | `CRYPTO_PAYLOAD_INVALID` | 解密后 payload 结构非法 | 不重试；检查 payload 构造 |
| 400 | `CRYPTO_PATH_MISMATCH` | payload `path` 与实际接口不符 | 不重试 |
| 400 | `IDEMPOTENCY_KEY_REQUIRED` | 命令接口缺幂等键 | 补键后重新加密发送 |
| 401 | `CRYPTO_KEY_UNWRAP_FAILED` | SM2 私钥解封会话密钥失败（密钥对不匹配） | 不重试；核对双方写死的公私钥是否配对（变更密钥必须双方同步改代码发版） |
| 401 | `CRYPTO_MAC_INVALID` | SM3-HMAC 校验失败（报文被篡改或拼接顺序不一致） | 不重试；核对 MAC 拼接顺序与字段编码 |
| 401 | `CRYPTO_DECRYPT_FAILED` | SM4 解密失败（IV/密文损坏） | 不重试；核对 SM4-CBC 参数与 PKCS#7 填充 |
| 401 | `CRYPTO_REPLAY_REJECTED` | `issued_at` 超窗或 `nonce` 重复 | 校准时钟后重新生成 nonce 重发 |
| 404 | `GROWTH_USER_NOT_FOUND` | payload `username` 未映射到 sub2api 用户 | 提示账号未开通，不重试（**账务映射失败，非身份校验**） |
| 404 | `TASK_NOT_FOUND` / `PRODUCT_NOT_FOUND` / `ORDER_NOT_FOUND` | 资源不存在或已下线 | 刷新列表 |
| 409 | `IDEMPOTENCY_CONFLICT` | 同键不同请求体 | 视为错误，不重试 |
| 409 | `IDEMPOTENCY_IN_PROGRESS` | 同键请求处理中 | 退避后同键重发 |
| 409 | `PRODUCT_VERSION_CHANGED` / `PRODUCT_OFFLINE` / `PRODUCT_SOLD_OUT` | 目录过期/下架/售罄 | 重拉商品详情，不创建订单 |
| 409 | `INSUFFICIENT_POINTS` / `PURCHASE_LIMIT_REACHED` | 余额不足 / 触达限购 | 展示原因，不重试 |
| 422 | `TASK_NOT_ELIGIBLE` / `NOT_ELIGIBLE` | 不满足规则或证据不可信 | 展示「暂不符合条件」 |
| 429 | `RATE_LIMITED` | 限流 | 退避重试，同幂等键重新加密 |
| 503 | `GROWTH_UNAVAILABLE` / `POOL_EXHAUSTED` / `FULFILLMENT_UNAVAILABLE` | 服务 / **任务额度耗尽或预算不足** / 履约暂不可用 | 展示「暂不可用」，允许重试 |

> v1.0 的 `GROWTH_USER_REQUIRED` / `GROWTH_ACCOUNT_DISABLED` 与 v2.0 的 `CRYPTO_DECRYPT_FAILED`（语义已拆分为 `CRYPTO_KEY_UNWRAP_FAILED` / `CRYPTO_MAC_INVALID` / `CRYPTO_DECRYPT_FAILED`）按当前版本处理；不再有身份头、不校验账号状态。

### 1.7 幂等、时区、限流、防重放

- 命令接口幂等记录保留 24h；命中重放返回首次结果并带 `X-Idempotency-Replayed: true`。
- `issued_at` 时间窗默认 ±5 分钟、`nonce` 窗口内去重，拒绝重放；时钟偏移会导致合法请求被拒，双方需 NTP 同步。
- 服务端按配置业务时区（默认 `Asia/Shanghai`）生成 `business_date` / `cycle_key`；客户端不传日期。
- U 面按映射得到的 `users.id` 限流（v4.0 起 I 面已删除，无集成面限流）。
- 所有余额、价格、库存、状态字段以响应为准；客户端不得本地推算权威值。

---

## 2. 接口总表

> 原令牌交换与身份头相关接口/字段已随模型整体取消；**v4.0 起 I 面 `task-events` 接口亦删除**（任务完成判定收敛到 AIWork）。**U 面 12 个接口统一 `POST`，业务参数放加密 payload 的 `params`。**

### 2.1 U 面 12 个接口

| # | 分类 | Path（均 POST） | 用途 | 幂等键 |
|---|---|---|---|---|
| 1 | 读 | `/growth/me` | 积分概览 + 签到状态 + 摘要 | — |
| 2 | 读 | `/growth/tasks` | 任务列表与我的进度 | — |
| 3 | 读 | `/growth/tasks/{task_id}` | 任务详情与可操作入口 | — |
| 4 | 读 | `/growth/ledger` | 积分流水（游标分页） | — |
| 5 | 读 | `/growth/products` | 当前可兑换商品列表 | — |
| 6 | 读 | `/growth/products/{product_id}` | 商品详情与兑换条件 | — |
| 7 | 读 | `/growth/exchanges` | 我的兑换订单列表 | — |
| 8 | 命令 | `/growth/checkins` | 签到（AIWork 判定完成后提交，sub2api 按规则发分） | **必填** |
| 9 | 命令 | `/growth/tasks/{task_id}/completion-requests` | 提交任务完成（AIWork 判定完成并声明，sub2api 按规则发分） | **必填** |
| 10 | 命令 | `/growth/exchanges` | 提交兑换请求 | **必填** |
| 11 | 状态 | `/growth/exchanges/{order_id}` | 查询订单扣分与履约状态 | — |
| 12 | 读 | `/growth/rank` | 当前用户积分排名及参与排名总人数 | — |

> **v4.0 删除的接口**：原 #11 `POST /growth/completion-requests/{request_id}`（完成请求核验状态查询）——完成请求改为**同步发分**，响应即最终结果，无需异步轮询；如需追溯历史完成记录，用 `/growth/ledger` 的 `biz_type=TASK` 流水。原 I 面 `task-events` 一并删除。U 面由 12 个收敛为 11 个。

### 2.2 服务间接口：无

> **v4.0 起 I 面整体删除**。任务是否完成由 AIWork 判定、经 U 面完成请求直接声明，sub2api 不再接收可信事件、不做证据核验。原 `POST /api/v1/integrations/growth/task-events` 作废，不得实现或调用。

---

## 3. 逐接口定义

> **统一说明**：下列接口的请求参数一律放在加密 payload 的 `params` 内（`username` 为当前用户业务标识）。**响应为明文，字段定义与 v1.0 完全一致**。`path` 用第 1.3 节的路由模板格式（含路径参数的接口如 `/growth/tasks/:task_id`）；路径参数值放 URL，`params` 内也要带同名值供服务端校验一致性。例如 `POST /growth/tasks/9002`，payload `path="/growth/tasks/:task_id"`、`params={"task_id":9002}`。

### 3.1 `POST /growth/me`

`params`：空对象 `{}`。响应 `data`（`user_id` 为 `username` 映射结果）：

```json
{
  "user_id": 12345,
  "available_points": 260,
  "earned_total": 300,
  "spent_total": 40,
  "balance_updated_at": "2026-11-17T15:40:12.123+08:00",
  "checkin": { "business_date": "2026-11-17", "timezone": "Asia/Shanghai", "status": "AVAILABLE", "rule_version": 3, "reward_hint": 10 },
  "task_summary": { "in_progress": 2, "completed_today": 1 },
  "server_time": "2026-11-17T15:43:00+08:00"
}
```

`checkin.status`：`AVAILABLE` / `CLAIMED` / `NOT_ELIGIBLE` / `UNAVAILABLE`。签到状态由服务端判定，AIWork 不得按本地日期推断。

### 3.2 `POST /growth/tasks`

`params`：`{ "status"?: string, "page": number, "page_size": number }`。响应 `data.items[]`：

```json
{ "task_id": 9001, "task_code": "daily_checkin", "title": "每日签到", "category": "DAILY",
  "task_type": "CHECKIN", "status": "AVAILABLE", "progress": 0, "target": 1,
  "cycle_key": "2026-11-17", "reward_hint": 10, "reward_pool": "CHECKIN",
  "rule_version": 3, "action": "CHECKIN", "icon": "...", "sort_order": 1 }
```

`status`：`AVAILABLE` / `IN_PROGRESS` / `COMPLETED` / `REWARDED` / `EXPIRED` / `PAUSED`。`action` 由服务端下发，UI 只按值渲染入口。

### 3.3 `POST /growth/tasks/{task_id}`

`params`：`{ "task_id": number }`。在 3.2 字段基础上增加 `description`、`effective_from/to`、`period_type`、`daily_limit`、`per_user_limit`、`my_completed_count`、`completion_request_endpoint`；任务详情可返回 `completion_hint` 说明 AIWork 如何判定该任务完成（如「用户完成一次模型请求」「浏览页面满 30 秒」），但**不对 completion_ref 取值做强约束**——该值由 AIWork 自行生成并声明。

### 3.4 `POST /growth/ledger`

`params`：`{ "before_entry_id"?: number, "limit"?: number, "type"?: "EARN"|"REDEEM"|"REFUND"|"ADJUST" }`（`limit` 默认 20，上限 100）。

```json
{ "code": 0, "data": { "items": [
    { "entry_id": 88001, "transfer_id": 77001, "type": "EARN", "direction": "IN",
      "points": 10, "balance_after": 260, "biz_type": "CHECKIN",
      "title": "每日签到", "created_at": "2026-11-17T09:02:11.123+08:00" }
  ], "next_cursor": "88000", "has_more": true } }
```

只追加流水，不做客户端汇总；权威余额取 `/growth/me`。用 `before_entry_id` 游标避免翻页漂移。

### 3.5 `POST /growth/products`

`params`：`{ "page": number, "page_size": number }`。响应 `data.items[]`：

```json
{ "product_id": 5001, "product_code": "token_1000", "title": "1000 Token 额度包",
  "product_type": "TOKEN_QUOTA", "version": 2, "points_cost": 100,
  "benefit_summary": "1000 Token 额度，有效期 30 天", "stock_hint": "充足",
  "per_user_limit": 5, "my_redeemed_count": 1, "status": "ON_SHELF" }
```

`points_cost`、`benefit_summary`、`stock_hint` 取自当前有效商品版本，仅用于展示。

### 3.6 `POST /growth/products/{product_id}`

`params`：`{ "product_id": number }`。增加 `benefit_detail`、`redeem_window`、`remaining_quota`、`eligible`（bool）与 `ineligible_reason`。`eligible=false` 时 UI 禁用兑换按钮。

### 3.7 `POST /growth/exchanges`（订单列表）

`params`：`{ "page": number, "page_size": number, "status"?: string }`。复用 `response.Paginated`（`items/total/page/page_size/pages`）。`items[]`：`order_id`、`product_title`、`points_cost`、`status`、`created_at`、`fulfilled_at`。

### 3.8 `POST /growth/checkins` — 命令

`params`：`{ "request_id"?: string }`（可选，仅追踪；幂等键在 payload 顶层 `idempotency_key`）。响应 `data`：

```json
{ "result": "AWARDED", "claim_id": 66001, "transfer_id": 77001,
  "business_date": "2026-11-17", "timezone": "Asia/Shanghai",
  "awarded_points": 10, "current_balance": 260, "rule_version": 3,
  "server_time": "2026-11-17T15:43:00+08:00" }
```

`result`：`AWARDED` / `ALREADY_CLAIMED` / `NOT_ELIGIBLE` / `UNAVAILABLE`。**v4.0 语义**：签到动作是否完成由 **AIWork 判定**（本地业务规则），完成后提交本接口；sub2api 按服务端业务时区生成 `cycle_key`、查当前生效 `CHECKIN` 规则计算发分并用 `(user_id, task_id, cycle_key)` 防重复。重复/跨周期重复提交返回 `ALREADY_CLAIMED` 且 `awarded_points=0`，不创建第二笔转账。**禁止**在 `params` 携带 `business_date` / `cycle_key` / `points` 等由服务端生成的账务字段。`current_balance` 可能为 `null`，此时调 `/growth/me` 刷新。

### 3.9 `POST /growth/tasks/{task_id}/completion-requests` — 命令（v4.0 重写）

`params`：`{ "task_id": number, "completion_ref": string, "request_id"?: string }`。

- `completion_ref`（**必填**）：**AIWork 侧判定该任务本次已完成**后生成的完成实例标识（建议 AIWork 业务侧唯一，如任务实例 ID / 本地完成事件 ID）。sub2api **不校验其真实性、不据此回查任何外部证据**，仅用于：① 业务去重（同一 `completion_ref` 不重复发分，配合幂等键）；② 写入完成记录供对账审计。
- **v4.0 变更**：原 `run_id`（供 sub2api 回查 `usage_logs` 证据）**删除**——证据核验职责已移除。
- **禁止**携带 `user_id`、`points`、`awarded_points`、`reward`、`rule_version`、成功标志等自证或账务字段（见第 6 节）。

响应 `data`（**同步发分**，无异步核验态）：

```json
{ "request_id": "cr-88001", "result": "AWARDED",
  "task_id": 9002, "completion_ref": "local-task-12345",
  "transfer_id": 77003, "awarded_points": 2, "current_balance": 262,
  "rule_version": 7, "cycle_key": "2026-11-17", "server_time": "..." }
```

`result`：`AWARDED`（已按规则发分，附带 `transfer_id` / `awarded_points` / `current_balance`）/ `ALREADY_COMPLETED`（同一 `completion_ref` 已处理过，返回原结果，`awarded_points` 按首次结果，可能为 0）/ `NOT_ELIGIBLE`（任务已暂停/每日或每人上限已满/规则不适用，不发分）。**同一任务在同一周期内可以多次发分**：每次真实完成必须使用新的 `completion_ref` 和新的幂等键，服务端按 `daily_limit`、`per_user_limit` 及任务剩余额度裁决；达到任一上限后返回 `NOT_ELIGIBLE`。**v4.0 删除 `PENDING_VERIFICATION` 与 `REJECTED`**：不再异步核验，也不存在「证据不可信」判定。网关任务原本的 `X-Client-Request-ID` 如需保留追溯，可放入 `request_id`，但不作为发奖依据。

### 3.10 `POST /growth/exchanges` — 命令

`params`：`{ "product_id": number, "expected_version"?: number, "request_id"?: string }`。

`expected_version` 仅用于发现目录过期；服务端始终按当前有效版本定价。响应 `data`：

```json
{ "order_id": 44001, "status": "FULFILLING", "product_version": 2,
  "points_cost": 100, "transfer_id": 77002, "current_balance": 160,
  "fulfillment_status": "PENDING", "server_time": "..." }
```

失败业务码见 1.6（版本变化/下架/售罄/余额不足/限购），失败时不创建扣分订单。

### 3.11 ~~`POST /growth/completion-requests/{request_id}`~~ — v4.0 删除

> **删除**：完成请求改为同步发分（见 3.9），响应 `data` 即最终结果，无需异步核验状态查询。历史发分记录通过 `/growth/ledger`（`type=EARN`、`biz_type=TASK`）查询。若 AIWork 已有对接代码调用本接口，v4.0 起收到 404，应移除轮询逻辑。

### 3.12 `POST /growth/exchanges/{order_id}`

`params`：`{ "order_id": number }`。响应：

```json
{ "order_id": 44001, "status": "SUCCEEDED", "product_snapshot": { "title": "1000 Token 额度包", "version": 2, "points_cost": 100, "benefit_detail": {} },
  "points_cost": 100, "transfer_id": 77002, "fulfillment_status": "SUCCEEDED",
  "failure_reason": null, "refund_transfer_id": null,
  "created_at": "...", "fulfilled_at": "..." }
```

`status`：`CREATED` / `FULFILLING` / `SUCCEEDED` / `FAILED` / `REFUNDED`。最终失败时 `refund_transfer_id` 非空。

### 3.13 `POST /growth/rank`

`params`：`{}`。按当前可用积分计算竞争排名；同分同名次，后续名次按超过当前积分的人数加 1。
`admin@example.com` 从排名和总用户数中排除。

```json
{ "user_id": 12345, "available_points": 260, "rank": 3, "total_users": 20, "rank_text": "3/20" }
```

### 3.14 ~~I 面 `POST /api/v1/integrations/growth/task-events`~~ — v4.0 删除

> **删除**：v4.0 起任务完成判定归属 AIWork、经 U 面完成请求直接声明，sub2api 不接收可信任务事件、不做证据核验。原接口的 `source_system` / `source_event_id` / `payload_digest` / 归属校验等语义整体作废，不得实现或调用。

---

## 4. 状态机与 UI 展示规则（v4.0：完成判定在 AIWork，发分同步返回）

| 场景 | 允许展示的状态 | 判定归属 | 何时可更新界面数据 |
|---|---|---|---|
| 签到 | 加载/待确认、已签到（日期+奖励）、不符合条件、暂不可用（可重试） | **AIWork 判定签到动作完成**，周期/发分由 sub2api | **仅在收到 #8 服务端响应后** |
| 任务完成 | 待提交、已发分、已完成过、不符合条件 | **AIWork 判定任务完成**并声明 `completion_ref`；发分由 sub2api | 收到 #9 服务端响应后（同步发分，无轮询） |
| 兑换订单 | 创建中、履约中、成功、失败、已退款 | 用户操作触发，sub2api 裁决 | 收到 #10 / #11 服务端结果后 |

**禁止乐观更新（v4.0 修订）**：AIWork 可以（且应当）在本地判定「任务是否完成」，但**不得在本地直接加分或把余额标记为已到账**——积分数额、周期归属、上限、任务额度仍由 sub2api 按规则裁决，UI 的余额/积分进度只在收到服务端响应后更新。任务卡可以依据 AIWork 本地判定显示「已完成（待发分）」，但「已发分/积分到账」必须以 #9 响应为准。失败时保留上一次服务端确认的余额。

---

## 5. 超时、重试与密钥管理

1. 网络错误 / 429 / 503：指数退避、**有上限**重试；命令接口**复用同一 `idempotency_key`**，但每次重发**重新生成 SM4 会话密钥 / IV / nonce / issued_at 并重新走完整加密**（会话密钥与 IV 绝不复用，密钥本身写死不变）。
2. 明确业务响应（已签到、不符合条件、余额不足、任务额度耗尽、版本变化）：**不按网络错误无限重试**。
3. 超时等结果不确定：不得认定已成功；用原幂等键重新加密发送（完成请求靠 `completion_ref` 业务去重、签到靠服务端 `cycle_key`，不会重复发分），或调 `/growth/me` / `/growth/ledger` 核实。**v4.0 起完成请求同步发分，无 #11 状态轮询接口。**
4. 加密类错误（`CRYPTO_*`）**一律不重试**：先核对 SM2 公私钥配对、`kid`、MAC 拼接顺序、客户端时钟与信封构造；这类错误是配置问题不是瞬时故障。
5. **SM2 密钥写死，无热轮换**：密钥对一次性生成后直接分配到两个项目写死。需要变更密钥（如泄露、配错）时**没有过渡窗口**——必须由 sub2api 侧重新生成密钥对、双方同步修改代码常量并发版；双方发版完成前跨版本请求会因密钥不匹配返回 `CRYPTO_KEY_UNWRAP_FAILED`。因此上线前必须确认密钥对配对正确，且双方发版可同步协调。
6. 切账号：AIWork 从新登录态读取新的 `username` 放入 payload；清 Renderer store 与幂等上下文；不得用旧账号的幂等键重放。

---

## 6. 安全红线

**密钥管理（v3.1：直接分配、双方写死）**：

- **密钥对生成与分配**：由 sub2api 侧生成一对 SM2 密钥，私钥写入 sub2api 服务端常量、公钥写入 AIWork Main 常量；双方各自写死固定值，不做运行时注入、不做下发。
- **SM2 私钥**：仅 sub2api 服务端持有，绝不下发客户端/进文档/进日志/进任何响应。直接写死为服务端代码常量（内部系统取舍，需代码库访问控制）。**不轮换、不进 KMS、不进配置密文**。
- **SM2 公钥**：写死在 AIWork Main 的 `crypto.ts` 常量，只进 Main 不进 Renderer。**不做 SM3 指纹校验**；公钥被替换的风险由内网边界 + 代码库访问控制承接。
- **SM4 会话密钥 / IV / nonce**：每请求随机生成，不持久化、不复用、不进日志（这是报文级随机量，与写死的长期密钥不同）。
- 任何解封失败、MAC 失败、重放、错投都记录审计日志（`kid`、`request_id`、来源 IP；**不记录私钥、不记录会话密钥、不记录明文 payload**）。

**请求体禁止字段**：以下字段不得出现在明文 payload 的 `params` 顶层或作为身份字段（服务端忽略或拒绝）：

```text
user_id, points, awarded_points, balance, current_balance,
token_amount, token_quota, pool_code, pool_balance,
rule_id, rule_version, reward, business_date, cycle_key, success,
price, points_cost, product_version（仅允许 expected_version 作版本探测）
```

> **v4.0 变化**：`is_completed` 从禁止列表移除——完成声明本身由 `completion_ref` 表达（提交完成请求即等于声明完成），无需布尔成功标志；但**账务字段仍全部禁止**。原 `user_id（I 面归属校验除外）` 例外删除（I 面已移除）。

同时禁止：
- 把展示用的 `points_cost` / `benefit_detail` 回传下单接口作为定价依据；
- 用 `username` 之外的方式声明身份，或在明文头里放身份信息（身份相关的只有加密 payload 里的 `username`）；
- 复用 SM4 会话密钥或 CBC `iv`（每请求必须新随机）、让 SM2 私钥出现在客户端任何位置、跳过 SM3-HMAC 校验先解密；
- 用 ASN.1 DER 与 C1C3C2 混用（SM2 密文编码必须双方统一为 C1C3C2 裸拼接）；
- 调用或探测 `/api/v1/admin/growth/*`；
- 在 Renderer 中持久化公钥、积分余额或幂等键。

**v4.0 完成判定边界**：AIWork 在**本地/业务侧**判定任务是否完成后再提交——这是 AIWork 的职责，不违反上表。禁止的是让 AIWork 决定「发多少分、归哪个周期、走哪个预算/任务额度」，这些始终由 sub2api 按当前生效规则裁决。
