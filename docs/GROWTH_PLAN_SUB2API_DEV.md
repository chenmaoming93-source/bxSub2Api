# sub2api 侧开发设计｜成长计划（账务与规则权威 + 后管）

> 日期：2026-11-17 ｜ 对接契约：[GROWTH_PLAN_AIWORK_API.md](GROWTH_PLAN_AIWORK_API.md) **v4.0（冻结，唯一接口真相源）**
> 总体设计：[GROWTH_PLAN_DESIGN.md](GROWTH_PLAN_DESIGN.md) ｜ 对端开发设计：[GROWTH_PLAN_AIWORK_DEV.md](GROWTH_PLAN_AIWORK_DEV.md)
>
> 本文是 **sub2api 团队内部开发设计**。所有对外请求/响应字段、路径、状态枚举、错误码**不在本文重复定义**，一律引用冻结契约 v4.0；本文描述 sub2api 侧数据模型、分层、事务、幂等、工程落点与任务拆分。
>
> **v4.0 职责反转**：任务/签到是否完成由 **AIWork 判定并声明**（`completion_ref`），sub2api **不回查 `usage_logs`、不接 I 面事件、不做证据核验**，收到完成请求即视为「已完成」，仅按当前规则算分、防重、双边入账。I 面与 `growth_task_events` 表删除。
>
> **v4.1 预算模型变更（非破坏性）**：删除 `growth_point_pools` 多池模型，改为 **`growth_budgets` 单账户总预算 + 任务额度两列**。总预算由运营手动增减；新建任务时从总预算划拨额度；发分扣减任务额度，耗尽即拦截。新增 `growth_budget_entries` 记录 TOPUP / REDUCE / ALLOCATE / REVOKE 四类变动。U 面接口与字段不变。
>
> **v4.2 任务生命周期补齐（非破坏性）**：明确**任务不提供删除**（不注册 `DELETE`、`growth_tasks` 不加 `deleted_at`），补齐发布 / 下线 / **重新上线**三个 M 面接口的服务端规则（新增 §4.6），`task_code` 创建后不可修改不可复用。U 面与契约不变。

---

## 0. 范围与原则

### 0.1 sub2api 是账务与规则权威

**总预算与任务额度**、钱包与双边总账、任务与规则版本、商品与版本定价、库存/Token 预算、兑换订单与履约、运营后管、审计与对账，**所有账事实只在 sub2api 产生**。发分规则、周期上限、奖励数额、任务额度由 sub2api 裁决；**任务是否完成由 AIWork 判定并声明**（v4.0），sub2api 不再核验完成证据。

### 0.2 信任（内部系统，国密 SM2 加密报文，无身份校验）

- U 面所有请求为 `POST` + **SM2 数字信封**（`kid`/`encrypted_key`/`iv`/`ciphertext`/`mac`）；`GrowthPayloadDecryptor` 中间件按契约 1.3 顺序 SM2 解封会话密钥 → 校验 SM3-HMAC → SM4 解密 → 防重放 → `path` 绑定，解封装成功即可信。
- **密钥角色不对等、双方写死**：密钥对由 sub2api 侧生成后直接分配——SM2 **私钥写死在 sub2api 服务端常量**（不进日志/响应、不轮换、不进 KMS / 配置密文），AIWork 只把公钥写死在 Main 常量；非对称加密只保证机密性/完整性，不保证来源身份（公钥可公开）——与「不做身份校验」决策一致。
- **不做任何身份校验、不校验账号状态、不改 AIWork 登录**：解封出的 payload `username` 只做 `GetByEmail → users.id` 的**账务映射**（非鉴权）；用户不存在返回 `GROWTH_USER_NOT_FOUND`，不发奖。
- **必须写 `AuthSubject`**，否则幂等组件 `ActorScope` 退化为 `user:0`，跨用户踩键。
- 密钥写死无热轮换（换密钥须双方同步改代码发版）；私钥保密靠代码库访问控制，内网/VPN + IP 白名单 + 双方 NTP 同步（上线阻塞项，见总体设计 2.4）。

### 0.3 工程约束（来自 PROJECT_CONVENTIONS.md）

| 项 | 约束 |
|---|---|
| 栈 | Go + Gin + Ent + MySQL 8 / GoldenDB（MySQL 兼容模式） |
| 主键 | `BIGINT AUTO_INCREMENT`（Ent `--idtype int64`），不引入 UUID/雪花 |
| DDL | 自动路径 `backend/migrations/<下一号>_*.sql` + 手工路径 `backend/sqlArchiving/growth/ddl/182_*.sql`，二者一致；**禁止**改历史迁移 |
| 配置 | 新增 `growth.*` 同步 `backend/config/config.yaml`（顶层已有 `timezone: "Asia/Shanghai"`，可复用作为业务时区） |
| 响应 | 业务响应用 `internal/pkg/response`（`Code int`，成功 0）；中间件早退用 string-code envelope |
| RBAC | 新路由须登记，否则 `ValidateRouteCoverage` 启动 panic（见 5.4） |

---

## 1. 分层架构

```text
routes/growth.go              U 面路由（内网信任边界，全 POST）  /  M 面路由（RBAC 受控）
      │  中间件：GrowthPayloadDecryptor（U，SM2 解封装即信任）/ AdminIdentityAuth+RBAC（M）
      ▼
handler/growth/*.go           参数校验、调用 service、幂等包裹、响应封装
      │
      ▼
service/growth/*.go           业务裁决：规则、进度、转账、订单、履约编排
      │
      ▼
repository (Ent)              growth_* 表 CRUD + 事务（FOR UPDATE / upsert）
      │
      ▼
GoldenDB / MySQL 8            双边账本（不可变流水）+ 幂等记录
```

**事务边界铁律**：钱包余额、转账凭证、两侧分录、领取/订单记录在同一 DB 事务提交或回滚；**不持事务做外部 HTTP**（Token 发放走 Outbox，见 4.5）。

---

## 2. 数据模型（表 → Ent schema → DDL）

### 2.1 表清单与用途

| 表（前缀 `growth_`） | 用途 | 关键约束 |
|---|---|---|
| `growth_budgets`（v4.1） | **总预算单账户**（单行） | 主键固定 `1`；`total_amount / allocated_amount / distributed_amount`；不变量 `0 ≤ allocated_amount ≤ total_amount`；`version` 乐观锁 |
| `growth_budget_entries`（v4.1） | 预算变动流水 | 四类 `TOPUP/REDUCE/ALLOCATE/REVOKE`；`budget_after / allocated_after`；`task_id` 可空；`operator_id`、原因、时间；只追加 |
| ~~`growth_point_pools`~~ | ~~来源积分池与预算~~ | **v4.1 删除**：多池模型由总预算 + 任务额度替代 |
| `growth_wallets` | 个人积分账户 | `user_id` 唯一；`available_points / earned_total / spent_total`；余额不可负 |
| `growth_transfers` | 转账凭证头 | `(biz_type, idempotency_key)` 唯一；转出/转入账户、点数、状态、时间 |
| `growth_account_ledger` | 账户分录（转账两侧） | 每个成功 `transfer_id` 恰好两条分录；索引 `(user_id, created_at, entry_id)`、`(transfer_id)` |
| `growth_tasks` | 任务目录 + **任务额度** | `task_code` 唯一且**创建后不可修改**；状态 `DRAFT/ACTIVE/OFFLINE`（**无删除、无软删**）；排序、生效区间；`allocated_points / distributed_points`（额度不足即停止发分） |
| `growth_task_rules` | 规则版本 | `(task_id, version)` 唯一；触发事件、条件 JSON、目标、奖励、周期、上限；已发布不可覆盖。**不含额度**（额度属于任务，跨版本延续） |
| `growth_user_tasks` | 用户任务进度 | `(user_id, task_id, rule_id, cycle_key)` 唯一 |
| `growth_completion_requests`（v4.0 新增） | 完成请求记录 + 完成实例去重 | `(user_id, task_id, completion_ref)` 唯一；记录 `request_id`、发分结果、关联 `transfer_id`；替代旧 `growth_task_events` |
| `growth_reward_claims` | 奖励领取凭证 + 周期内多次发分 | `(user_id, idempotency_key)` 唯一；`(user_id, task_id, cycle_key, sequence)` 唯一 |
| ~~`growth_task_events`~~ | ~~可信事件收件/去重~~ | **v4.0 删除**：任务完成判定在 AIWork、无外部事件接入 |
| `growth_products` | 商品主档 | `product_code` 唯一；类型、状态、排序 |
| `growth_product_versions` | 商品配置版本 | `(product_id, version)` 唯一；积分价格、权益 JSON、限购；被订单引用后不可覆盖 |
| `growth_product_inventory` | 库存 / Token 预算 | `lock_version` 乐观锁；`available/reserved/sold`、`token_quota_*` |
| `growth_exchange_orders` | 兑换订单 + 快照 | `(user_id, idempotency_key)` 唯一；关联 `transfer_id`、履约单号 |
| `growth_fulfillment_jobs` | 履约 Outbox | `order_id` 唯一；平台幂等键、状态、重试次数、下次重试时间 |
| `growth_inventory_ledger` | 库存变更凭证 | 业务操作唯一；入库/预占/售出/释放/调整 |
| `growth_admin_audit` | 运营/调账审计 | 操作人、资源、变更前后摘要、原因、request_id |

> 完整字段见总体设计第 4 章。**本设计只约定工程落点**，不重复字段清单。
>
> GoldenDB 兼容约定：数据库不使用 `status`、`description`、`action` 作为列名，分别使用 `*_status`、`remark`、`action_type`。`conditions`、`benefit_detail`、`product_snapshot`、`before_json`、`after_json` 使用原生可空 `JSON`；空对象由应用层写为 `{}`。

### 2.2 Ent schema 落点

在 `backend/ent/schema/` 新增（文件 → 结构体，参考 `idempotency_record.go:14-50`、`payment_order.go:186-198`）：

```text
growth_budget.go            → GrowthBudget            # v4.1 总预算单账户（单行）
growth_budget_entry.go      → GrowthBudgetEntry       # v4.1 预算变动流水（四类）
growth_wallet.go            → GrowthWallet
growth_transfer.go          → GrowthTransfer
growth_account_ledger.go    → GrowthAccountLedger
growth_task.go              → GrowthTask
growth_task_rule.go         → GrowthTaskRule
growth_user_task.go         → GrowthUserTask
growth_completion_request.go → GrowthCompletionRequest   # v4.0 新增（完成请求记录/去重）
growth_reward_claim.go      → GrowthRewardClaim
# 注：旧 growth_task_event.go 已删除（v4.0 无 I 面事件）
growth_product.go           → GrowthProduct
growth_product_version.go   → GrowthProductVersion
growth_product_inventory.go → GrowthProductInventory
growth_exchange_order.go    → GrowthExchangeOrder
growth_fulfillment_job.go   → GrowthFulfillmentJob
growth_inventory_ledger.go  → GrowthInventoryLedger
growth_admin_audit.go       → GrowthAdminAudit
```

要求：
- 表名注解 `entsql.Annotation{Table: "growth_xxx"}`；Mixin 用 `mixins.TimeMixin{}`（`created_at/updated_at`）。
- 金额/积分字段 `decimal(20,8)` 用 `SchemaType(map[string]string{dialect.MySQL: "decimal(20,8)"})`（仿 `user.go:49-51`）；积分也可用 `BIGINT`，但需与 DDL 一致。
- 唯一约束用字段级 `.Unique()` 或 `index.Fields(...).Unique()`；以上唯一键必须落到 Ent 索引，不能只靠 DB 手工 DDL。
- JSON 语义字段统一使用 Ent `field.JSON` 映射为可空原生 `JSON`（**禁止执行管理界面写入的任意脚本**）。
- 在 `User.Edges()`（`ent/schema/user.go:122-146`）追加 `edge.To("growth_wallet", GrowthWallet.Type)` 等必要关联。
- 生成：`cd backend && make generate`（`go generate ./ent` + `go generate ./cmd/server`）。`ent/generate.go:5` 已开启 `sql/upsert, sql/lock, --idtype int64`，正好满足幂等插入与 `FOR UPDATE`。

### 2.3 DDL 落点

- 自动路径：`backend/migrations/161_create_growth_tables.sql`（当前最大 160）—— 走 `ApplyMigrationsWithOptions`（`repository/ent.go:30-38`，开关 `cfg.Database.RunMigrations`）。
- 手工路径：`backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql`（当前最大 181）—— 无 DDL 权限时执行；二者 SQL 保持一致。
- 风格：开头 `-- MySQL 8 / GoldenDB（MySQL 兼容模式）`；`BIGINT NOT NULL AUTO_INCREMENT` + `PRIMARY KEY` + `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`；建表内联 `UNIQUE KEY / KEY`；结尾 `-- Verification` + `SHOW COLUMNS`。

---

## 3. 中间件与路由

### 3.1 `GrowthPayloadDecryptor`（U 面必挂，SM2 解封装即信任）

新文件 `backend/internal/server/middleware/growth_payload.go`。职责：**只做 SM2 解封装 + 完整性 + 防重放 + `path` 绑定 + 用户名映射，不做身份校验**。国密库建议 `github.com/tjfoc/gmsm`（SM2/SM3/SM4 纯 Go 实现），需安全团队确认版本。

```go
type GrowthPayload struct {
    Path           string          `json:"path"`
    Username       string          `json:"username"`
    Params         json.RawMessage `json:"params"`
    RequestID      string          `json:"request_id"`
    IssuedAt       int64           `json:"issued_at"`
    Nonce          string          `json:"nonce"`
    IdempotencyKey string          `json:"idempotency_key"`
}
type Envelope struct {                          // 对应契约 1.3
    Kid          string `json:"kid"`
    EncryptedKey string `json:"encrypted_key"`  // SM2 公钥加密的 16B SM4 会话密钥（C1C3C2）
    IV           string `json:"iv"`             // 16B SM4-CBC IV
    Ciphertext   string `json:"ciphertext"`
    MAC          string `json:"mac"`            // SM3-HMAC(会话密钥, kid||ek||iv||ct)
}
type GrowthPayloadDecryptor gin.HandlerFunc

// 私钥由 sub2api 侧生成后写死为常量（hex D 值或 PEM）；公钥分配给 AIWork 写死。
// 不进配置、不进 KMS、无轮换；变更须双方同步改代码发版（见契约 5.5）。
const growthKid = "growth-sm2-v1"
var growthSM2PrivKey = mustParseSM2PrivKey("....")  // TODO(sub2api 生成): 固定私钥常量

func NewGrowthPayloadDecryptor(users service.UserRepository,
    window time.Duration, nonces NonceCache) GrowthPayloadDecryptor {
    return GrowthPayloadDecryptor(func(c *gin.Context) {
        var env Envelope
        if err := c.ShouldBindJSON(&env); err != nil { abort(400,"CRYPTO_ENVELOPE_INVALID"); return }
        if env.Kid != growthKid { abort(400,"CRYPTO_KEY_UNKNOWN"); return }  // kid 固定，不匹配即拒绝
        priv := growthSM2PrivKey   // 写死的 SM2 私钥，仅服务端
        ek, e1 := base64.StdEncoding.DecodeString(env.EncryptedKey)
        iv, e2 := base64.StdEncoding.DecodeString(env.IV)
        ct, e3 := base64.StdEncoding.DecodeString(env.Ciphertext)
        mac, e4 := base64.StdEncoding.DecodeString(env.MAC)
        if e1!=nil||e2!=nil||e3!=nil||e4!=nil || len(iv)!=16 || len(mac)!=32 || len(ct)==0 {
            abort(400,"CRYPTO_ENVELOPE_INVALID"); return
        }
        sm4Key, err := sm2.Decrypt(priv, ek, sm2.C1C3C2)  // 解封会话密钥
        if err != nil || len(sm4Key) != 16 { abort(401,"CRYPTO_KEY_UNWRAP_FAILED"); return }
        // MAC 拼接用 base64 字符串，顺序与客户端一致；常量时间比较
        expect := sm3hmac(sm4Key, []byte(env.Kid+env.EncryptedKey+env.IV+env.Ciphertext))
        if !hmac.Equal(expect, mac) { abort(401,"CRYPTO_MAC_INVALID"); return }  // 先验 MAC 再解密
        plain, err := sm4DecryptCBC(sm4Key, iv, ct)        // SM4-CBC + PKCS#7 去填充
        if err != nil { abort(401,"CRYPTO_DECRYPT_FAILED"); return }
        var p GrowthPayload
        if json.Unmarshal(plain, &p) != nil { abort(400,"CRYPTO_PAYLOAD_INVALID"); return }
        if abs(time.Now().Unix()-p.IssuedAt) > int64(window.Seconds()) || !nonces.Take(p.Nonce) {
            abort(401,"CRYPTO_REPLAY_REJECTED"); return
        }
        if p.Path != c.FullPath() { abort(400,"CRYPTO_PATH_MISMATCH"); return }  // Gin 路由模板比对
        user, err := users.GetByEmail(c.Request.Context(), strings.ToLower(strings.TrimSpace(p.Username)))
        if err != nil || user == nil { abort(404,"GROWTH_USER_NOT_FOUND"); return } // 账务映射，非身份校验
        // 注意：不调用 IsActive()，不做账号状态校验（内部系统决策）
        c.Set(string(ContextKeyGrowthPayload), &p)
        c.Set(string(ContextKeyUser), AuthSubject{UserID: user.ID, Concurrency: user.Concurrency, SuperAdmin: user.IsSuperAdmin()})
        c.Set(string(ContextKeyUserRole), user.Role)
        c.Next()
    })
}
```

- **私钥写死**：`growthSM2PrivKey` 为服务端代码常量（`...` 处填入 sub2api 生成的固定私钥），由 `mustParseSM2PrivKey` 启动时解析，解析失败直接 panic fail-fast。**不进配置、不进 KMS、无 `kid` 轮换**；私钥不打印、不进日志、不进任何响应。公钥由 sub2api 侧生成后分配给 AIWork 写死。
- `kid` 固定为 `growth-sm2-v1`，中间件只做等值校验（不等即 `CRYPTO_KEY_UNKNOWN`），不再按 `kid` 查密钥表。
- `sm3hmac` / `sm4DecryptCBC`：基于 gmsm 的 `sm3`/`sm4` 封装；SM2 解封固定 `sm2.C1C3C2` 编码，与 AIWork 侧一致。
- `NonceCache`：窗口内 nonce 去重（Redis 或进程内 TTL Set）；与幂等记录分开，专防重放。
- `c.FullPath()` 取 Gin 路由模板（如 `/api/v1/growth/tasks/:task_id`），与 payload `path`（契约约定的模板形式）逐字比对。
- 依赖 `service.UserRepository` 已在 `repository/wire.go:85` 注册；新增 `NonceCache` provider 到 `middleware/wire.go:22-27`（**无需 GrowthKeyStore provider**）；新增 gmsm 依赖到 `go.mod`。
- 顺序：`GrowthPayloadDecryptor` → handler。**不要挂 `BackendModeUserGuard`**（会把非 admin 用户 403）。
- **v4.0 删除 I 面**：不再有 `task-events` 复用本中间件的场景；完成请求全部走 U 面 `completion-requests`，payload 只带 `completion_ref`（无 `user_id` 归属校验需求）。

### 3.2 U 面路由

新文件 `backend/internal/server/routes/growth.go`，风格仿 `integrations.go:9-38`（内部系统信任边界，非 RBAC 受控）：

```go
func RegisterGrowthRoutes(v1 *gin.RouterGroup, h *handler.Handlers, dec middleware.GrowthPayloadDecryptor) {
    g := v1.Group("/growth")
    g.Use(gin.HandlerFunc(dec))                    // 解密即信任，所有接口统一 POST
    // 读（统一 POST，业务参数在加密 payload.params 内）
    g.POST("/me", h.Growth.GetMe)
    g.POST("/tasks", h.Growth.ListTasks)
    g.POST("/tasks/:task_id", h.Growth.GetTask)
    g.POST("/ledger", h.Growth.ListLedger)
    g.POST("/products", h.Growth.ListProducts)
    g.POST("/products/:product_id", h.Growth.GetProduct)
    g.POST("/exchanges", h.Growth.ListExchanges)
    // 命令（幂等包裹见 4.1；完成请求 v4.0 同步发分，无证据核验）
    g.POST("/checkins", h.Growth.Checkin)
    g.POST("/tasks/:task_id/completion-requests", h.Growth.CreateCompletionRequest)
    g.POST("/exchanges", h.Growth.CreateExchange)
    // 状态（v4.0 删除 /completion-requests/:request_id：完成请求同步返回最终结果）
    g.POST("/exchanges/:order_id", h.Growth.GetExchange)
}
```

在 `router.go:125-135` 注册链旁加一行 `routes.RegisterGrowthRoutes(v1, h, dec)`。M 面（`/api/v1/admin/growth/*`）走 `rbacRoutes`，**不**走此函数。

### 3.3 I 面：v4.0 删除

**无 I 面**。原 `POST /api/v1/integrations/growth/task-events` 及其 `GrowthTaskEvent` schema / `growth_task_events` 表 / 归属校验逻辑整体作废，不实现、不挂载。任务完成判定归属 AIWork、经 U 面 `completion-requests` 直接声明。页面路由装配仅保留 U 面（§3.2）与 M 面（走 `rbacRoutes`）。

---

## 4. 核心服务设计

### 4.1 幂等（两层，缺一不可）

**L1 请求幂等**：复用 `executeUserIdempotentJSON`（`handler/idempotency_helper.go:17-65`）：

```go
executeUserIdempotentJSON(c, "growth.checkin", req, service.DefaultWriteIdempotencyTTL(),
    func(ctx context.Context) (any, error) { return h.growth.Checkin(ctx, subject.UserID, req) })
```

- `Scope` 分别为 `growth.checkin` / `growth.task-completion` / `growth.exchange`。
- `Idempotency-Key` 取自**解密后的 payload `idempotency_key`**（不再取 header）；`ActorScope` 来自 `AuthSubject.UserID`（3.1 解密映射已保证写入）。
- handler 从 `c.Get(ContextKeyGrowthPayload)` 读取 `Params`（`json.RawMessage`）反序列化为各接口请求体。
- 重放回 `X-Idempotency-Replayed: true`（`idempotency_helper.go:61-63`）；记录表唯一键 `(scope, idempotency_key_hash)`（`ent/schema/idempotency_record.go:30-49`），TTL 24h + 清理服务（`idempotency_cleanup_service.go:12-91`）。

**L2 账务幂等**：DB 唯一约束兜底（L1 被绕过/并发/内部重放）：

- `growth_transfers`：`(biz_type, idempotency_key)` 唯一。
- 签到：`(user_id, task_id, cycle_key)` 唯一（规则版本作审计字段，避免换版本重复领取）。
- `growth_reward_claims`：`(user_id, idempotency_key)` 与 `(user_id, task_id, cycle_key, sequence)` 唯一；签到固定 `sequence=1`，普通任务在发分事务内递增序号。`growth_exchange_orders`：`(user_id, idempotency_key)`。
- **`growth_completion_requests`（v4.0）**：`(user_id, task_id, completion_ref)` 唯一——同一完成实例重复申报（含超时重放、换幂等键但同 `completion_ref`）不重复发分。
- 唯一键冲突时**读取既有结果返回**，不依赖「先查后写」。

### 4.2 总预算与任务额度（v4.1）

**服务方法（`GrowthBudgetService`）**

```go
// 总预算：单行账户，id 固定为 1
type Budget struct {
    TotalAmount       int64 // 累计批准预算（TOPUP − REDUCE）
    AllocatedAmount   int64 // 已划拨未消耗（= Σ 任务剩余额度）
    DistributedAmount int64 // 已实际发放
    Version           int64
}

// 四个操作，全部先写流水再更新账户（同一事务，流水是审计真相）
TopUp(ctx, points int64, reason string, operatorID int64) (Budget, error)
Reduce(ctx, points int64, reason string, operatorID int64) (Budget, error)  // 需 total-points >= allocated
Allocate(ctx, taskID, points int64, reason string, operatorID int64) (Budget, error)
Revoke(ctx, taskID int64, reason string, operatorID int64) (Budget, error)  // 仅回收剩余
```

**关键裁决**

| 场景 | 条件 | 结果 |
|---|---|---|
| 增加预算 | `points > 0` | `total_amount += points` |
| 核减预算 | `total_amount - points >= allocated_amount` | 成功；否则 `409 BUDGET_ALLOCATED_LOCKED`「请先回收任务额度」 |
| 新建任务划拨 | `total_amount - allocated_amount >= points` | `allocated_amount += points`；任务 `allocated_points += points`；**新建任务必填初始额度且 > 0** |
| 追加额度 | 同上 | 同 ALLOCATE，`reason` 必填 |
| 回收额度 | 任务 `OFFLINE` / 已结束 | 回收 `allocated_points - distributed_points`，任务额度与 `allocated_amount` 同减 |
| 编辑任务规则 | —— | **不改额度**；额度属于任务，跨规则版本延续 |

**并发控制**

- 预算行是全局热点：所有预算操作与发分事务都会更新它。用 `SELECT ... FOR UPDATE` 锁单行 + `version` 乐观锁双保险；发分走 `UPDATE growth_budgets SET distributed_amount = distributed_amount + ?`（原子自增，避免读改写）。
- 任务额度扣减必须有条件谓词，依赖影响行数判断，不做「先查余额再扣」。

### 4.3 签到原子事务（v4.1：信任 AIWork 已判定完成，扣任务额度）

1. 解封装映射用户 + L1 幂等 → 按业务时区（复用 `timezone` 配置，默认 `Asia/Shanghai`）生成 `cycle_key`（**服务端生成，客户端传的 `business_date`/`cycle_key` 忽略**）。
2. 查内置签到任务（`task_code = daily_checkin`）的当前生效 `CHECKIN` 规则（规则版本、奖励积分、上限）算分。
3. 同一 GoldenDB 事务内：
   - **条件扣减任务额度**：`UPDATE growth_tasks SET distributed_points = distributed_points + :p WHERE id = :task AND allocated_points - distributed_points >= :p`；影响行数 0 → 额度耗尽，返回 `UNAVAILABLE` 并回滚；
   - 增加 `growth_wallets` 余额（`available_points >= 0` 约束）；
   - `UPDATE growth_budgets SET distributed_amount = distributed_amount + :p`；
   - 写 `growth_user_tasks` / `growth_reward_claims` / `growth_transfers`；
   - 写**恰好两条** `growth_account_ledger` 分录（任务额度出、钱包入，共用 `transfer_id`）。
4. 规则暂停/超限/额度不足 → 整体回滚，不留半笔账。
5. 唯一键冲突 → 读既有领取记录返回 `ALREADY_CLAIMED`，`awarded_points=0`。

### 4.4 任务完成发分（v4.1：无证据核验，按规则 + 任务额度同步发分）

- **职责边界**：任务是否完成由 AIWork 判定并声明 `completion_ref`；sub2api **不回查 `usage_logs`、不接收 I 面事件、不判断任务是否真的完成**，收到完成请求即视为「已完成」。
- **处理流程**：
  1. 解封装映射用户 + L1 幂等（`growth.task-completion`）；解析 `params`：`task_id` + `completion_ref`（+ 可选 `request_id`）。
  2. 查当前生效任务规则（规则版本、奖励积分、周期、每日/总量上限）+ **该任务剩余额度**。
  3. **防重/限额/额度三重裁决**：
     - `growth_completion_requests` 按 `(user_id, task_id, completion_ref)` 唯一去重——命中则读既有结果返回 `ALREADY_COMPLETED`；
     - 在发分事务内按每日/总量上限判断，并为当前周期生成下一个 `sequence`；未超限时允许同一周期多次发分，超限返回 `NOT_ELIGIBLE`（整笔发分回滚）；
     - 任务已暂停/规则不适用返回 `NOT_ELIGIBLE`；**任务额度不足返回 503 `POOL_EXHAUSTED`（对外「暂不可用」）**，不发分。
  4. 同一 GoldenDB 事务内执行 `任务额度 → 钱包` 双边入账（同 4.3 事务结构）：条件扣减任务额度、增加钱包、累加 `distributed_amount`、写 `growth_completion_requests` / `growth_user_tasks` / `growth_reward_claims` / `growth_transfers` / 恰好两条 `growth_account_ledger` 分录。
  5. **同步返回** `AWARDED` + `transfer_id` / `awarded_points` / `current_balance` / `rule_version` / `cycle_key`。
- **删除**：原 `PENDING_VERIFICATION` 异步核验、`REJECTED`（证据不可信）、`usage_logs` 回查 SQL、I 面 `task-events` 上报、完成请求状态轮询接口——全部不实现。
- **防作弊取舍**：sub2api 不独立证明任务完成（可信调用方模型），靠周期/每日上限 + **任务额度硬上限** + `completion_ref` 去重 + 审计流水 + 对账告警兜底；风险书面化见总体设计 §3.4。

### 4.5 兑换与履约

1. 校验商品上架/有效期、`expected_version` 探测、限购、余额、库存/Token 预算；失败返回契约 1.5 业务码，**不建扣分订单**。
2. 同一事务：预占库存（`lock_version` 乐观锁，有限库存商品）→ 扣钱包转 `REDEEMED`（双边分录）→ 建订单（商品版本快照）→ 建 `growth_fulfillment_jobs`（Outbox）。
3. 事务提交后由内部 Worker 按 `order_id` 幂等触发履约：
   - Token 额度：复用 `user_platform_quota_port.go` 的 `UpsertForUser` / `BulkInsertInitial`（已幂等）；或 `redeem_service.go:378` 的兑换码路径。**不引用已废弃的 `/model-token-quotas`**（`model_token_quota_route_test.go:10-18`）。
   - 按业务键幂等参考 `payment_fulfillment.go:249-300`（`doBalance` 用 `RechargeCode` 决定 create/redeem/skip）。
4. 履约成功 → 订单 `SUCCEEDED` + 确认库存售出；明确失败 → 重试；最终失败 → 释放预占库存 + 反向转账退款（`refund_transfer_id`）。
5. **不持 DB 事务调用任何外部 HTTP**。

### 4.6 任务发布 / 下线 / 重新上线（v4.2：无删除）

```text
DRAFT ──publish──► ACTIVE ──offline──► OFFLINE
  ▲                                       │
  └───────────── publish ◄── online ◄─────┘
                 （online 必须先重新划拨额度）
```

**发布 `POST /admin/growth/tasks/:id/publish`**

1. 取该任务最新**未发布**规则（`published_at IS NULL`）；没有则 `TASK_RULE_MISSING`（404）。
2. 同一事务内：把当前生效版本 `effective_to = now`；新版本 `published_at = now`、`effective_from = now` 或指定时间；任务 `task_status = ACTIVE`。
3. 幂等：重复发布同一版本直接返回当前状态（`ALREADY_PUBLISHED`），不产生第二个版本、不改生效区间。
4. 已发布版本**无任何 UPDATE 奖励字段的代码路径**——改价只能 `INSERT` 新版本。
5. **发布不改额度**：`allocated_points` 原样保留，额度跨规则版本延续。

**下线 `POST /admin/growth/tasks/:id/offline`**

1. 仅 `ACTIVE` 可下线，其余状态 → `TASK_STATE_INVALID`（409）。
2. 事务内：`task_status = OFFLINE`；当前生效规则 `effective_to = now`（立即停止继续发分）；写 `growth_admin_audit`（原因必填）。
3. 传 `revoke_remaining = true` 时同事务执行 REVOKE：`growth_tasks.allocated_points` 与 `growth_budgets.allocated_amount` **同减** `allocated_points − distributed_points`，写 `growth_budget_entries(REVOKE)`。
4. `distributed_points` 只增不减——**已发放永不回收**。

**重新上线 `POST /admin/growth/tasks/:id/online`**

1. 仅 `OFFLINE` 可调用；`initial_allocation > 0` 必填。
2. 事务内：`total_amount − allocated_amount >= N` 否则 `BUDGET_ALLOCATED_LOCKED`（409）→ `allocated_points += N`、`allocated_amount += N` → 写 `ALLOCATE` 流水 → `task_status = ACTIVE`。
3. 若该任务所有规则版本均已结束（无生效规则），拒绝并提示先发布新版本（`TASK_RULE_MISSING`）。
4. 历史 `distributed_points` 保留，不占新额度。

**不提供删除**

- 不注册 `DELETE /admin/growth/tasks/:id`；`growth_tasks` **不加 `deleted_at`**（与 user/api_key 等主数据的软删惯例不同，此处刻意不软删——软删会让任务从目录消失、与历史流水对不上）。
- 原因：`growth_task_rules` / `growth_completion_requests` / `growth_account_ledger` 均以 `task_id` 反查，删除即断链。
- `task_code` 创建后**不可修改、不可复用**；创建时靠唯一索引捕获 1062 → `TASK_CODE_DUPLICATED`（409）。
- 以上 `TASK_*` 均为 **M 面内部业务码**，不进冻结契约（U 面对手不会看到）。

### 4.7 后管（M 面）

能力清单见总体设计第 6 章。落点：`handler/admin/growth_*.go`、`service/growth/admin_*.go`，走 `rbacRoutes` + 新增权限点（`rbac/permissions.go`）+ `RBAC_ENDPOINT_INVENTORY.md` 登记 + `sqlArchiving` seed（仿 181 的 INSERT 风格）。**预算增减与任务额度划拨/回收必须同时写 `growth_budget_entries` 与 `growth_admin_audit`（含操作人、原因、变更前后值）**；人工调账必须生成正式转账 + 分录 + 审计；库存调整写 `growth_inventory_ledger`。

---

## 5. 横切关注点

### 5.1 响应封装

- 业务响应：`internal/pkg/response`（`Response{Code int,...}`，成功 `response.Success`；错误 `response.ErrorFrom(c, err)`，`ErrorResponse` 映射契约 1.5 业务码）。
- 解密中间件早退：`middleware.AbortWithError`（string-code envelope，契约 1.5 第二套，`CRYPTO_*` 错误码）。
- 分页：`response.Paginated`（`items/total/page/page_size/pages`）用于 tasks/products/exchanges；流水用轻量游标 `before_entry_id + limit`（append-only，避免 offset 漂移），响应含 `next_cursor / has_more`。

### 5.2 业务时区与 server_time

- 复用顶层 `timezone: "Asia/Shanghai"`；签到 `cycle_key` 与 `business_date` 一律服务端生成，客户端传日期字段一律忽略/拒绝。
- 所有响应带 `server_time`、`timezone`（契约 1.6）。

### 5.3 限流

- U 面按映射出的 `users.id` 限流（解密映射后）；v4.0 I 面已删除，无集成面限流。

### 5.4 RBAC coverage（否则启动 panic）

`router.go:156-161` 启动即校验 `ValidateRouteCoverage`。U 面是内部系统信任边界，在 `rbac/coverage.go:61-90` 的 `exclusionReason` 增加（v4.0 I 面已删除，仅保留 U 面前缀）：

```go
case strings.HasPrefix(path, "/api/v1/growth/"):
    return "internal growth trust boundary (SM2 encrypted payload, no identity auth)"
```

M 面逐路由走 `rbacRoutes` 并登记 `RBAC_ENDPOINT_INVENTORY.md`。

### 5.5 Token 发放幂等能力确认（阻塞项）

对接 `user_platform_quota_port.go` 前需确认：按 `order_id` 重试是否幂等、是否有额度对账查询。这是总体设计第 7 节第 4 条，阻塞兑换上线。

---

## 6. 工程落点一页清单

| 关注点 | 落点 |
|---|---|
| Ent schema | `backend/ent/schema/growth_*.go`（17 个，见 2.2） |
| 代码生成 | `ent/generate.go:5`、`backend/Makefile:9-11`（`make generate`） |
| DDL | `backend/migrations/161_create_growth_tables.sql` + `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql` |
| 解封装中间件 | `internal/server/middleware/growth_payload.go`（新）：写死私钥 + 固定 `kid` 校验 + SM2 解封 + SM3-HMAC + SM4 解密 + 防重放 + `path` 绑定 + 用户名映射；注入 `middleware/wire.go:22-27` |
| 密钥/Nonce | 私钥写死在 `growth_payload.go` 常量（无 `crypto_keys.go` / `GrowthKeyStore` / 配置加载）、`internal/service/growth/nonce_cache.go`（窗口去重）、`sm3hmac`/`sm4DecryptCBC` 封装；配置仅 `growth.crypto.time_window`；`go.mod` 增 `github.com/tjfoc/gmsm` |
| 路由 | `internal/server/routes/growth.go`（新，全 POST）；装配 `router.go:125-135`、`http.go:26+` |
| Handler | `internal/handler/growth/*.go`（新）；幂等 `internal/handler/idempotency_helper.go:17-65`（幂等键从 payload 取） |
| Service | `internal/service/growth/*.go`（新） |
| 履约 | Outbox `growth_fulfillment_jobs`；参考 `payment_fulfillment.go:221-300`、`user_platform_quota_port.go` |
| 响应 | `internal/pkg/response/response.go:14-167` |
| 测试 | 单测模板 `available_channel_handler_test.go:17-28`；解密注入 `c.Set(ContextKeyGrowthPayload, &p)` + `c.Set(ContextKeyUser, AuthSubject{UserID:31})` |
| RBAC | `rbac/coverage.go:61-90`、`rbac/permissions.go`、`RBAC_ENDPOINT_INVENTORY.md` |
| 配置 | `backend/config/config.yaml` 仅增 `growth.crypto.time_window`；SM2 密钥对写死为代码常量（私钥服务端、公钥 AIWork Main），**不进配置**；时区复用顶层 `timezone` |

---

## 7. 测试策略

| 层 | 内容 |
|---|---|
| 单测（`-tags=unit`） | 解封装中间件：合法报文解出 payload、`kid` 未知/SM2 解封失败/MAC 失败/SM4 解密失败/超时窗/nonce 重放/`path` 错投各返回对应 `CRYPTO_*`；用户名不存在返回 `GROWTH_USER_NOT_FOUND`；两套 envelope、错误码映射；幂等 helper 用 stub repo（`idempotency_helper_test.go:20-55`） |
| 加解密互操作测试 | 用 AIWork 侧同款加密样例（固定测试 SM2 密钥对、`kid`、会话密钥/IV）解出预期 payload，确保双方信封逐字节兼容；**重点核对 SM2 C1C3C2 编码、SM3-HMAC 拼接顺序、SM4-CBC 填充** |
| 服务层 | 规则匹配、cycle_key 生成、上限判断、`completion_ref` 去重（**v4.0 无 `usage_logs` 证据核验**）；**v4.1 预算裁决：核减不得低于已分配、划拨不得超未分配余额、额度耗尽拦截、回收只回收剩余** |
| 集成（`-tags=integration`） | 双边入账事务：余额=分录之和；并发签到/兑换不重复记账、不超卖；唯一键冲突返回既有结果 |
| 契约测试 | 按冻结契约逐接口校验响应字段名/类型/枚举/状态机；与 AIWork Mock 对齐 |
| 对账脚本 | 每日校验：`total = Σ TOPUP − Σ REDUCE`、`allocated = Σ 任务剩余额度`、`distributed = Σ 发分 ledger`、钱包余额 = Σ 分录；异常告警不自动改账 |

必测场景：双方写死的 SM2 公私钥不匹配（`CRYPTO_KEY_UNWRAP_FAILED`）、MAC 失败（篡改/拼接顺序错）、SM4 密文损坏、`kid` 不等于固定值（`CRYPTO_KEY_UNKNOWN`）、会话密钥/IV 复用、超时窗/nonce 重放、`path` 错投、用户名不存在（映射失败）、并发同幂等键、切账号旧键重放、**同 `completion_ref` 重复申报不重复发分（含换幂等键场景）、周期/每日上限满返回 `NOT_ELIGIBLE`**、**核减预算低于已分配返回 `BUDGET_ALLOCATED_LOCKED`**、**任务额度耗尽时发分返回 `POOL_EXHAUSTED` 且不产生任何分录**、**并发发分不突破任务额度上限（条件更新影响行数裁决）**、Token 接口超时按 `order_id` 重试、退款反向转账。

---

## 8. sub2api 侧任务拆分

| 阶段 | 任务 | 产出 | 验收 |
|---|---|---|---|
| S0 契约 | 冻结契约签字 + sub2api 生成并分配 SM2 密钥对（私钥写死服务端、公钥写死 AIWork，确认配对）+ 国密库选型安全审批 + 网络边界/IP 白名单/NTP 确认 + 确认「完成判定在 AIWork、发分在 sub2api」边界与风险接受 | 契约 v4.0 | 双方确认 |
| S1 内核 | Ent schema + `make generate` + 161/182 DDL + **预算/额度服务（TOPUP/REDUCE/ALLOCATE/REVOKE）** + 对账脚本 | 17 张表 | 并发与重放不重复记账，余额=分录之和，`allocated = Σ 任务剩余额度` |
| S2 解封装与读 | `GrowthPayloadDecryptor`（SM2 解封 + SM3-HMAC + SM4）+ 密钥/Nonce 组件 + U 面路由 + `/me` `/tasks` `/ledger` `/products` | 读接口可用 | 密文正确解封装、MAC/防重放与 `path` 绑定生效，用户名映射 `users.id`，`ActorScope` 非 `user:0`；加解密互操作测试通过 |
| S3 命令与发分 | 签到 + 完成请求（**AIWork 判定完成、sub2api 同步发分，无证据核验**）+ 任务额度扣减/耗尽拦截 + `growth_completion_requests` 去重 | 签到/任务奖励闭环 | 同步返回发分结果，`completion_ref` 去重、周期/上限/额度三重防超发，超时重放幂等 |
| S4 兑换与后管 | 商品版本/库存预算/订单/Outbox 履约 + M 面（含总预算增减、任务额度划拨/回收）+ 后管 UI | 兑换闭环 | 不超卖，失败可退款，审计完整 |
| S5 联调 | 切真 + 契约测试 + 字段级 diff + 7. 必测场景 | 联调通过 | 与 AIWork 检查清单全绿 |

按项目规约，进入开发需另建 `.mvp-list/growth-plan-mvp-list/` 分阶段 MVP 与 `mvp-progress.md`；本文只定义实现设计，不替代 MVP 拆分。

---

## 9. 对 AIWork 的依赖与阻塞

| 依赖 | 需要 AIWork 提供 | 阻塞 sub2api 阶段 |
|---|---|---|
| 加密信封 | 每个 U 面请求为 SM2 数字信封（SM2 封装 + SM4 + SM3-HMAC），双方写死的公私钥配对、`kid` 固定 `growth-sm2-v1`、C1C3C2/MAC 拼接顺序一致、互操作通过（v4.0 无 I 面） | S0 约定，S2 起联调 |
| 密钥写死分发 | sub2api 生成密钥对 → 私钥写死服务端常量、公钥写死 AIWork Main 常量；无注入/指纹校验/轮换；变更须双方同步发版 | S0（上线阻塞） |
| 幂等键 | 命令接口 payload 带 `idempotency_key`，重试复用但重发重新生成会话密钥/IV | S3 起 |
| 完成声明 | 完成任务提交 `completion_ref`（AIWork 判定的完成实例标识，sub2api 不核验、只去重） | S3 |
| 网络边界 | AIWork 出口 IP 清单 + 双方 NTP 同步 | S0（上线阻塞） |

> SM2 密钥对由 sub2api 生成后双方写死（私钥仅服务端、公钥进 AIWork Main；只保证机密性不保证来源），且**无热轮换**——变更须双方同步改代码发版。U 面必须仅内网/VPN + IP 白名单，代码库访问受控、私钥不进客户端/日志/响应。若无法满足，S2 起不得上线，须重新评审信任方案。
