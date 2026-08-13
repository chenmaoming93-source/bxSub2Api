# 模型账户「模型基本属性」实施计划（Implementation Plan）

## 1. 文档状态与元数据

| 项 | 值 |
|---|---|
| 标题 | 模型账户「模型基本属性」（Model Attributes）实施计划 |
| 状态 | **定稿 — 用户已批准** |
| 版本 | v1.0 |
| 日期 | 2026-08-05 |
| 项目 | bxSub2Api（backend: Go + ent v0.14.5；frontend: Vue 3 + TypeScript；DB: MySQL 8 / GoldenDB） |
| 变更摘要 | 2026-08-05 用户审核通过，正式定稿。方案要点：属性以 JSON map 存储（属性名做 key，description + value 做值）；预置清单仅放前端；后端信任前端原样存储；本期仅覆盖管理端创建/编辑账号的存取。 |

## 2. 引言

### 2.1 背景与问题

系统为「AI API 账户」管理网关（数据库表 `accounts`）。当前账户只有凭证、调度、限额等字段，**缺少对账户所绑定模型"基本能力属性"的描述**（如上下文窗口大小、是否支持推理、是否支持图片输入）。这些信息是模型的客观属性，管理员需要能在管理端维护它们，未来可能对外提供查询。

### 2.2 目标与成功标准

| 编号 | 目标 | 成功标准 |
|---|---|---|
| G-01 | 管理端可维护账户的模型基本属性 | 在 `/admin/accounts` 创建、编辑账户弹窗中，可添加/删除/修改模型属性行（含预置快捷项与自定义项），保存后重新打开可见（回显一致） |
| G-02 | 属性以 JSON map 持久化 | `accounts.model_attributes` 列按 `{属性名: {description, value}}` 结构存取，空配置（未填任何属性）合法 |
| G-03 | 后端信任前端 | 属性值类型（数字/布尔/字符串/数组等）由前端解析，后端原样存储，不做类型推断或改写 |

### 2.3 范围（本期）

- `accounts` 表新增 `model_attributes` JSON 列（SQL 归档文件 `168_add_account_model_attributes.sql`）。
- 后端全链路：domain 类型 → ent schema（含重新生成）→ service → repository → handler → dto。
- 前端：类型定义、`ModelAttributesSection` 组件、i18n（zh/en）、`CreateAccountModal` 与 `EditAccountModal` 接入。
- 管理端列表/详情响应中携带 `model_attributes`（供编辑回显）。

### 2.4 非目标（明确不做）

- **网关/调度不消费**这些属性：本期系统内除「创建/编辑账号时存取、编辑回显查询」外，无任何其他使用方。
- **不做外部查询 API**：未来需要时再单独设计，本期不预留接口。
- **不做属性值的后端类型解析/枚举校验**：完全信任前端。
- **预置清单不落后端**：仅前端常量 + i18n，作为添加属性时的快捷提示。
- **不动批量编辑（BulkEditAccountModal）**、不做属性级搜索/筛选、不做审计日志。

## 3. 假设与决策

### 3.1 已确认决策（用户明确）

| 编号 | 决策 | 说明 |
|---|---|---|
| D-01 | 数据结构为 JSON map | `{ "context_window": { "description": "...", "value": 200000 }, ... }`，属性名（英文）为 key，`description`（中文描述）与 `value`（动态类型）为值 |
| D-02 | 后端信任前端，原样存储 | 后端不做 value 类型解析、不做枚举校验、不做内容改写；仅做最小防御（见 D-05） |
| D-03 | 预置清单只放前端 | 18 条常用属性作为「添加属性」下拉的快捷选项（含 zh/en 描述），后端不感知 |
| D-04 | 数据库为 accounts 加 JSON 列，不建新表 | 与账户 1:1；属性集不定长、key 可自定义；沿用 `credentials`/`extra` JSON 列先例 |
| D-05 | 后端最小防御规整 | 保存前仅丢弃「key 去首尾空白后为空」的条目；description 与 value 原样保留（不截断、不解析）。其余数据完整性由前端保证 |
| D-06 | ent 采用全量 `go generate ./ent` | 已实测仓库生成代码与 ent v0.14.5 完全匹配（干净状态全量生成零 diff）；本次预期 diff 仅落在 account 实体相关 ~10 个文件 |
| D-07 | 本期范围封闭 | 除创建/编辑账号存取外无其他消费方；不讨论未来用途 |

### 3.2 保守默认（非阻塞，用户可随时否决）

- **A-01**：已添加的属性行若「属性值」输入为空，提交时整行不纳入 map（避免无意义数据）。
- **A-02**：前端提交前检测重复 key（仅可能来自自定义行手输），检测到则提示并阻止提交，防止 map 静默覆盖。
- **A-03**：不设属性条数/长度硬限制（信任前端 + 管理端管理员场景）。
- **A-04**：`description` 为空也允许（值仍可保存）；`value` 为空字符串 `""` 是合法值（与 A-01 的"输入框为空"区分——输入框留空视为未填，显式输入 `""` 或引号内容按解析结果保存）。

### 3.3 剩余未决项

无阻塞性未决项。见 §9 风险与未决项。

## 4. 概念功能设计

### 4.1 功能模块

**FM-01 创建账户时维护模型属性**
- 参与者：管理员。
- 输入：创建账户弹窗（`CreateAccountModal`）中「模型基本属性（可选）」卡片区；「添加属性」下拉选预置项或自定义行，逐行填 key / description / value。
- 可见行为：卡片位于「模型限制（可选）」区块正下方；预置项自动带出 key 与中文描述；已添加的预置项置灰；每行可删除；value 输入框提示"可填数字、true/false、文本或 JSON 数组"。
- 输出：提交时打包为 JSON map 放入请求体 `model_attributes`；一行不填则传 `{}`。
- 边界：重复 key 提交前提示并阻止；value 留空的行不提交。
- 验收信号：创建成功后可再次打开该账户，属性与创建时填写一致（值类型不变）。

**FM-02 编辑账户时修改/查询模型属性**
- 参与者：管理员。
- 输入：编辑账户弹窗（`EditAccountModal`）中同一卡片区；回显已存属性。
- 可见行为：打开弹窗即回显现有属性（map → 行，key 保持原有顺序）；可增删改；保存后回写。
- 输出：`model_attributes` 随更新请求提交；未配置属性的账户显示空卡片。
- 边界：未配置（DB 为 NULL）按空配置处理；保存空 map 视为显式清空全部属性。
- 验收信号：修改保存后重新打开，变更生效；清空保存后卡片为空。

**FM-03 查询/回显**
- 参与者：管理员（同一弹窗）、未来外部调用方（本期不实现）。
- 输入：账户列表/详情响应。
- 可见行为：管理端各接口响应 `Account` 对象携带 `model_attributes`（map 或 null）。
- 验收信号：列表与详情接口均可读到已存属性，且与库内一致。

### 4.2 跨模块旅程（主流程）

```
创建/编辑弹窗打开
  → 回显：后端 model_attributes(map) → 行数组（value 转可编辑文本）
  → 用户增删改行（预置快捷项 / 自定义）
  → 提交：行数组 → value 智能解析 → 构建 map → 请求体 model_attributes
  → 后端：最小规整 → 写入 accounts.model_attributes
  → 下次打开：库内 JSON → 响应 → 回显（一致性闭环）
```

## 5. 详细技术设计

### 5.1 系统上下文与组件职责

```
[前端 Vue3]
  ModelAttributesSection.vue（新组件，v-model: map）
    ├─ modelAttributePresets.ts（新，预置常量 + i18n key）
    ├─ CreateAccountModal.vue / EditAccountModal.vue（插入组件，打包/回显）
    └─ types/index.ts（ModelAttributeItem / ModelAttributes）
        │  HTTP JSON
        ▼
[后端 Go]
  handler/admin/account_handler.go（请求体 + 透传）→ dto（响应映射）
  service/admin_service.go（Create/UpdateAccountInput、应用逻辑）
  domain/model_attributes.go（类型 + 最小规整）
  repository/account_repo.go（ent client 读写）
  ent/（schema + 生成代码）
        ▼
[MySQL 8 / GoldenDB]
  accounts.model_attributes JSON NULL（DDL 归档于 sqlArchiving/168）
```

### 5.2 技术选型与理由

- JSON 列（非拆列/非新表）：属性集不定长、key 可自定义、1:1 归属账户；项目已有 `credentials`/`extra` JSON 列先例，ent `field.JSON` 支持成熟。
- Go 强类型 map（`map[string]ModelAttributeItem`）而非 `map[string]any`：对已知子结构（description/value）提供类型安全，同时保留 value 动态类型。
- 全量 `go generate ./ent`：与历史提交做法一致；已实测零基线 diff（见 D-06）。

### 5.3 数据流与状态

- 写：前端 map → `POST /api/v1/admin/accounts` 或 `PUT /api/v1/admin/accounts/:id` → service（Create 直接写入 / Update 在 `input.ModelAttributes != nil` 时覆盖，nil 不动）→ repo `SetNillableModelAttributes` → 列。
- 读：ent 查询 → `service.Account.ModelAttributes` → dto `Account.model_attributes` → 前端回显。
- 状态：无新增状态机；NULL = 未配置，`{}` = 显式空配置（读侧两者等价，写侧 Update 语义区分）。

### 5.4 API / 接口契约

**API-01 创建账户（扩展）** `POST /api/v1/admin/accounts`
- 认证/权限：现有管理端鉴权（admin 角色），无变更。
- 请求体新增字段：

```json
"model_attributes": {
  "context_window": { "description": "上下文窗口总大小（token）", "value": 200000 },
  "supports_vision": { "description": "支持图片输入", "value": true }
}
```
- 语义：可选；缺省或 `null` = 未配置；`{}` = 显式空。
- 错误：非对象结构 → 400（由 Go 类型绑定拒绝）；仅此一种后端校验。

**API-02 更新账户（扩展）** `PUT /api/v1/admin/accounts/:id`
- 请求体新增字段同上；`model_attributes` 为 `map` 类型，天然区分「缺省(nil) = 不改动」与「`{}` = 清空」。
- 响应：更新后的 `Account`（含 `model_attributes`）。

**API-03 账户响应扩展**（列表 `GET /api/v1/admin/accounts`、详情、更新响应共用 dto `Account`）
- 新增 `model_attributes` 字段：`Record<string, { description?: string; value?: unknown }> | null`，`omitempty`（未配置时不返回或返回 null，二选一，倾向 omitempty 不输出——实现时确认前端回显对 undefined/null 的容错）。

**注意**：列表 ETag 计算已包含响应体（`lite` 仅影响 ETag 计算路径，不裁剪字段），加字段后 ETag 自然变化，无需额外处理。

### 5.5 数据模型

**表：`accounts`（新增 1 列）**

| 列 | 类型 | 可空 | 默认 | 约束 | 说明 |
|---|---|---|---|---|---|
| `model_attributes` | JSON | 是 | NULL | 无 | 模型基本属性 map：`{属性名: {description, value}}`；NULL = 未配置 |

- 唯一/外键/索引：无需新增。
- 生命周期：随账户行；无独立清理逻辑。
- 敏感度：非敏感（不涉及凭证/密钥），可进入管理端列表响应。
- 迁移/回滚：DDL 通过 `backend/sqlArchiving/168_add_account_model_attributes.sql` 人工/部署执行（项目规约：结构变更不得进 `migrations/`，归档文件不参与运行时自动迁移）；回滚 = 执行 `ALTER TABLE accounts DROP COLUMN model_attributes;`（代码先回退再删列）。
- 存量数据：全部为 NULL，无需回填；前端按空卡片处理。

**Go 类型**（`backend/internal/domain/model_attributes.go` 新增）：

```go
type ModelAttributeItem struct {
    Description string `json:"description,omitempty"`
    Value       any    `json:"value"`
}

type ModelAttributes map[string]ModelAttributeItem

// Normalize 最小防御规整：丢弃 key 去空白后为空的条目；其余原样。
func (m ModelAttributes) Normalize() ModelAttributes
```

**ent schema**（`backend/ent/schema/account.go`）：

```go
field.JSON("model_attributes", domain.ModelAttributes{}).
    Optional().
    Comment("Model basic attributes map: {attrName: {description, value}}.").
    SchemaType(map[string]string{dialect.MySQL: "json"}),
```

生成代码预期受影响文件（参照 `59e415a` 先例，全量生成后核对）：`ent/account.go`、`ent/account/account.go`、`ent/account/where.go`、`ent/account_create.go`、`ent/account_update.go`、`ent/account_query.go`、`ent/mutation.go`、`ent/migrate/schema.go`、`ent/runtime/runtime.go`、`ent/account/` 其他（以实际 `git status` 为准，应仅 account 相关 + 少量共享文件）。

### 5.6 安全与隐私

- 信任边界：管理端接口已有 RBAC 保护，本次无新端点、无鉴权变更。
- 输入校验：仅结构校验（Go map 类型绑定）+ 最小规整（空 key 丢弃）；不引入注入面（JSON 列无 SQL 拼接）。
- 数据暴露：属性进入管理端列表响应；非敏感，符合现有 `extra`/`credentials`（脱敏后）暴露策略。

### 5.7 可靠性与运维

- 失败模式：DDL 未执行时应用读写该列会报错——部署顺序必须「先 SQL 后代码」，或代码对缺列容错（本期按「先 SQL 后代码」约定，不写兼容层）。
- 可观测性：无新增日志/指标需求。
- 性能：单 JSON 列读写，无额外查询路径，无性能风险；列表接口载荷略增（每账户一个 map，量级极小）。

### 5.8 兼容性、配置、发布与回滚

- 无新增配置项。
- 发布：SQL 归档先执行 → 后端/前端代码发布。
- 回滚：代码回退 + `DROP COLUMN`（数据丢失可接受，属性可重新录入）。

## 6. 伪代码与操作逻辑

**前端：构建提交 map**

```text
function buildModelAttributes(rows):
  map = {}
  seenKeys = {}
  for row in rows:
    key = trim(row.key)
    if key == "" or row.valueText is blank:   # A-01：留空的行不提交
      continue
    if seenKeys[key]:                          # A-02：重复 key 阻止提交
      raise DuplicateKeyError(key)
    seenKeys[key] = true
    map[key] = { description: trim(row.description) or omit, value: parseValue(row.valueText) }
  return map

function parseValue(text):
  t = trim(text)
  if t == "": return ""                          # 显式空字符串（合法）
  try: return JSON.parse(t)                      # "200000"→200000, "true"→true, "[\"a\"]"→["a"]
  catch: return t                                # 其余按字符串
```

**前端：回显 map → 行**

```text
function rowsFromModelAttributes(attrs):
  if attrs is null/undefined: return []
  return for each key in attrs:
    { key, description: attrs[key].description ?? "", valueText: displayText(attrs[key].value) }
    # displayText: string 原样；其他类型 JSON.stringify
```

**后端：保存（service UpdateAccount）**

```text
function updateAccount(id, input):
  account = repo.getByID(id)
  ...
  if input.ModelAttributes != nil:                # map 类型：nil=缺省不动，{} = 清空
    account.ModelAttributes = input.ModelAttributes.Normalize()
  ...
  repo.update(account)
```

**后端：最小规整**

```text
function normalize(attrs):
  if attrs == nil: return nil
  out = {}
  for key, item in attrs:
    k = trim(key)
    if k == "": continue                          # 丢弃空 key 条目
    out[k] = item                                 # description/value 原样
  return out
```

## 7. 验证策略

| 层 | 验证项 | 手段 |
|---|---|---|
| 生成代码 | 全量 `go generate ./ent` 后 diff 范围可控、编译通过 | `git status` 核对 + `go build ./ent/...` + `go build ./...` |
| 单元（后端） | Normalize：空 key 丢弃、去空白、nil 安全、原样保留 value/description | `domain` 包新增单测 |
| 单元（后端） | Update：提供 map 覆盖/清空；缺省不动原有值；Create 落库 | `admin_service` 相关单测（沿用现有 stub 模式） |
| 契约 | 请求体/响应体字段绑定与序列化 | handler 层测试（现有 stubAdminService 模式） |
| 前端 | 组件：预置添加、自定义行、重复 key 阻止、value 解析（数字/布尔/数组/字符串）、空 map 提交、回显往返 | `ModelAttributesSection.spec.ts`（Vitest） |
| 前端 | 弹窗集成：创建/编辑提交载荷包含 model_attributes | 现有 `CreateAccountModal`/`EditAccountModal` 测试补充或手工冒烟 |
| 静态 | 前后端类型 | `go vet`；前端 `vue-tsc` typecheck |
| SQL | 归档文件可独立执行、方言正确 | 按项目规约在 MySQL 8 测试库执行（含幂等性检查，声明可重复执行则执行两次） |

**验收标准映射（AC）**

| AC | 对应 | 验证 |
|---|---|---|
| AC-01 创建时可添加预置/自定义属性并保存成功 | G-01/FM-01 | 组件测试 + 后端单测 + 冒烟 |
| AC-02 编辑时回显一致（含值类型）且可改/清空 | G-01/FM-02 | 组件回显测试 + 冒烟 |
| AC-03 库内为 map 结构 JSON，空配置合法 | G-02/DB | SQL 执行 + 后端单测 |
| AC-04 后端不做 value 类型改写（字符串 "true" 不转布尔） | G-03 | 后端单测断言原样存储 |

## 8. 实施顺序与分解指南

依赖边界：后端数据链路 → 前端组件 → 弹窗集成 → 全量验证。

| 阶段 | 内容 | 产物/完成标准 |
|---|---|---|
| P1 后端数据层 | domain 类型 + 单测；ent schema 加字段 + 全量 `go generate ./ent` + diff/编译核对；`sqlArchiving/168` SQL 文件 | `go build ./...` 通过；生成 diff 仅 account 相关 |
| P2 后端业务层 | service（Input/Account/应用逻辑）+ repository（读写映射）+ handler/dto（请求/响应） | 后端全链路单测通过 |
| P3 前端基础 | `types/index.ts` 类型；`ModelAttributesSection.vue` + 预置常量 + i18n（zh/en）+ 组件单测 | `vue-tsc` 通过；组件测试绿 |
| P4 弹窗集成 | `CreateAccountModal.vue` / `EditAccountModal.vue` 每处「模型限制（可选）」下方插入组件 + 回显/打包/提交 | 手工冒烟：创建→回显→编辑→清空 |
| P5 全量验证 | 后端 `go build`/测试、前端 typecheck/测试、SQL 方言核验 | §7 全部通过 |

推荐任务分解边界：P1+P2 为「后端任务」，P3+P4 为「前端任务」，P5 为「验证任务」；P2 依赖 P1，P4 依赖 P3，前后端可并行（P1/P2 与 P3 无相互依赖，P4 的载荷契约以 API-01/02 为准）。

## 9. 风险与未决项

| 风险/未决项 | 影响 | 缓解 | 触发/应急 |
|---|---|---|---|
| R-01 ent 全量生成受 Windows 文件锁中断，留下损坏生成文件（此前发生过） | 高 | 生成前确认无残留 `go.exe`/gopls 进程；生成后立即 `git status` + `go build ./ent/...` 核对 | 发现损坏 → `git checkout -- backend/ent/` 还原后重试 |
| R-02 生成 diff 超出预期（波及非 account 文件） | 中 | 生成后先核对 `git status`；如确属模板漂移，单独评估是否顺带提交对齐 | 与用户确认是否接受 |
| R-03 DDL 未先于代码执行导致运行期报错 | 中 | 发布顺序固定「先 SQL 后代码」，写入部署说明 | 回滚代码或补执行 SQL |
| O-01 value 智能解析规则（JSON.parse 优先）与用户预期不符 | 低 | 已在 §6 伪代码固化；测试覆盖典型输入 | 用户反馈后调整解析规则 |
| O-02 列表接口载荷增加（每账户一个 map） | 低 | 量级极小；不另做裁剪 | 若后续列表过大再评估 lite 裁剪 |

## 10. 可追溯性矩阵

| 目标/需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| G-01 管理端维护属性 | FM-01 / FM-02 | ModelAttributesSection、Create/EditAccountModal、service/repo/handler | AC-01 / AC-02 |
| G-02 map 持久化 | FM-03 | accounts.model_attributes 列、domain.ModelAttributes、ent | AC-03 |
| G-03 信任前端 | FM-01/FM-02（提交链路） | 后端最小规整（不解析 value） | AC-04 |

## 11. 评审记录

| 版本 | 日期 | 变更 | 状态 |
|---|---|---|---|
| v1.0 | 2026-08-05 | 首次正式输出；整合前期讨论结论（v1 固定表单 → v2 动态数组 → v3 map + 前端预置），并确认：后端信任前端原样存储；本期范围仅创建/编辑账号存取 | 草稿 — 等待审核 |
| v1.0 | 2026-08-05 | 用户明确回复「批准」，定稿 | **定稿 — 用户已批准** |
