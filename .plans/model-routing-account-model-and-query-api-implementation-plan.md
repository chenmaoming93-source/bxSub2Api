# 模型路由账号模型联动与分组路由查询接口实施 Plan

> **状态：最终版 — 用户已批准**  
> **版本：v1.0**  
> **日期：2026-07-16**  
> **变更摘要：新增模型路由“先选账号、再选上游模型”交互，并新增按分组名查询路由别名及上游模型的 integrations 接口。**

## 1. 项目背景与目标

### 1.1 背景

当前管理员配置分组模型路由时：

- 上游模型名由管理员手动填写。
- 路由候选可以绑定一个或多个账号。
- 手动填写无法保证所填模型属于已选账号支持的模型范围，容易产生无效配置。

同时，外部可信系统需要在不登录管理后台的情况下，通过固定 Authorization Token 查询指定分组下的模型路由信息。

### 1.2 目标

- `G-01`：模型路由候选必须先选择账号，再从账号支持的模型中选择上游模型。
- `G-02`：选择多个账号时，只允许选择所有账号共同支持的模型，即模型白名单交集。
- `G-03`：新增只读接口，根据分组名返回该分组的全部路由别名及其上游模型名。
- `G-04`：新接口复用 `/api/v1/integrations/api-keys/getOrCreate` 的鉴权和安全加固逻辑。
- `G-05`：不修改数据库表，不改变现有模型路由持久化结构，兼容已有配置。

### 1.3 使用者

- 模型路由配置功能：系统管理员。
- 分组路由查询接口：持有 integrations Access Token 的可信内部系统。

### 1.4 成功标准

- 未选择账号时不能选择或填写上游模型。
- 选择账号后，上游模型以下拉框形式展示。
- 多账号情况下只能选择模型白名单交集。
- 保存的数据仍符合现有 `model_routing` 结构。
- 外部系统可通过分组名查询路由别名及上游模型。
- 新接口不要求登录，但必须通过固定 Bearer Token 鉴权。
- 不产生数据库迁移。

## 2. 范围与非目标

### 2.1 本期范围

- 调整模型路由编辑器的账号和上游模型选择顺序。
- 复用现有账号模型查询接口：`GET /api/v1/admin/accounts/:id/models`。
- 计算多个账号支持模型的交集。
- 处理账号变更后的模型重新校验。
- 新增 integrations 分组模型路由查询接口。
- 增加前后端自动化测试。
- 增加必要的中英文界面文案。

### 2.2 非目标

- 不修改数据库表或 Ent Schema。
- 不修改现有 `model_routing` JSON 持久化格式。
- 不将 `account_ids` 数组改成单账号字段。
- 不修改模型路由运行时调度逻辑。
- 不调整账号模型白名单本身的配置方式。
- 不为需求 2 增加登录态或管理员 JWT 鉴权。
- 不返回账号 ID、账号名称、优先级、Token 限额等内部路由细节。
- 不新增分页；单个分组的路由配置预计规模有限。

## 3. 假设与已确认决策

| 编号 | 类型 | 内容 |
|---|---|---|
| `D-01` | 已确认 | 一个路由候选继续支持多个账号。 |
| `D-02` | 已确认 | 多账号的可选模型取各账号模型列表的交集。 |
| `D-03` | 已确认 | 不修改数据库表和现有模型路由持久化结构。 |
| `D-04` | 已确认 | 需求 2 复用 `getOrCreate` 的固定 Bearer Token 鉴权。 |
| `D-05` | 已确认 | 分组名作为需求 2 的请求参数。 |
| `A-01` | 保守默认 | 需求 2 使用 POST 和 JSON 请求体，以便与 `getOrCreate` 的调用风格保持接近。 |
| `A-02` | 保守默认 | 新接口路径采用 `/api/v1/integrations/model-routes/list`。 |
| `A-03` | 保守默认 | 即使模型路由开关关闭，接口仍返回已经保存的路由配置；接口查询的是配置内容，而不是当前是否生效。 |
| `A-04` | 保守默认 | 同一路由别名中的重复上游模型去重后返回。 |
| `A-05` | 保守默认 | 路由别名和模型名按照稳定的字典顺序返回，避免 Go Map 顺序不稳定。 |
| `A-06` | 保守默认 | 账号模型查询接口返回的模型 `id` 是路由编辑器的标准候选值。 |
| `A-07` | 保守默认 | 历史配置中的模型不在当前交集内时，界面展示为失效值并阻止保存，要求管理员重新选择。 |

## 4. 功能设计

### 4.1 模块一：模型路由账号与模型联动

#### 4.1.1 功能要求

- `FR-01`：路由候选区域先展示账号选择，再展示上游模型选择。
- `FR-02`：没有选择账号时，上游模型下拉框保持禁用。
- `FR-03`：选择账号后，系统查询该账号的可用模型。
- `FR-04`：选择多个账号时，系统计算模型 ID 交集。
- `FR-05`：上游模型只能从交集下拉框中选择，不允许自由输入。
- `FR-06`：账号集合发生变化时，系统重新计算模型交集。
- `FR-07`：已选模型不属于新交集时，清空该模型并提示重新选择。
- `FR-08`：账号模型加载失败时，禁用模型选择并展示加载失败提示。
- `FR-09`：保存前校验账号不为空、模型有效、优先级及限额合法。
- `FR-10`：继续保存现有候选结构，不改变接口和持久化字段。

#### 4.1.2 用户操作流程

1. 管理员新增一个路由候选。
2. 上游模型下拉框处于禁用状态。
3. 管理员搜索并选择第一个账号。
4. 前端请求该账号的可用模型。
5. 模型加载成功后解锁上游模型下拉框。
6. 如果继续选择其他账号，前端加载新增账号的模型列表。
7. 前端计算所有已选账号模型 ID 的交集。
8. 管理员从交集下拉框中选择上游模型。
9. 管理员保存分组。
10. 后端按照现有模型路由结构完成校验和保存。

#### 4.1.3 多账号交集规则

假设三个账号模型列表分别为：

```text
账号 A：model-1、model-2、model-3
账号 B：model-2、model-3、model-4
账号 C：model-3、model-5
```

最终下拉框只展示 `model-3`。交集判断以模型 `id` 精确匹配，不根据展示名称匹配。

#### 4.1.4 边界行为

- 账号列表为空：禁用模型选择。
- 任意账号模型加载中：模型选择显示加载状态。
- 任意账号模型加载失败：不使用不完整结果计算交集。
- 多账号无共同模型：显示“所选账号没有共同支持的模型”，不能保存当前候选。
- 删除账号后：重新计算剩余账号模型交集。
- 更换账号后：重新校验已选模型。
- 多个账号返回重复模型：单个模型只展示一次。
- 历史模型仍在交集中：保留原值。
- 历史模型不在交集中：显示失效提示并阻止保存。

### 4.2 模块二：分组路由查询接口

#### 4.2.1 功能要求

- `FR-11`：支持按清理后的分组名精确查找分组。
- `FR-12`：返回该分组全部路由别名。
- `FR-13`：返回每个路由别名下去重后的上游模型名。
- `FR-14`：不返回账号、优先级和限额信息。
- `FR-15`：无路由配置时返回空数组。
- `FR-16`：接口复用 integrations 固定 Token 鉴权。
- `FR-17`：接口复用 integrations 安全加固和限流中间件。
- `FR-18`：响应使用项目统一的 `code/message/data` 格式。

## 5. 详细技术设计

### 5.1 前端组件调整

主要调整：

```text
frontend/src/components/admin/group/GroupModelRoutingEditor.vue
frontend/src/components/admin/group/groupModelRoutingEditor.ts
```

职责划分：

- `GroupModelRoutingEditor.vue`：账号选择、模型下拉框、加载状态和错误展示。
- `groupModelRoutingEditor.ts`：候选数据结构、模型交集计算和有效性判断等纯逻辑。
- Admin Account API：调用现有 `GET /api/v1/admin/accounts/:id/models`。

#### 5.1.1 前端临时状态

模型列表和加载状态属于编辑器临时状态，不写入模型路由配置：

```ts
interface CandidateModelState {
  accountModels: Record<number, AccountModelOption[]>
  availableModels: AccountModelOption[]
  loadingAccountIds: number[]
  failedAccountIds: number[]
}

interface AccountModelOption {
  id: string
  display_name?: string
}
```

继续以稳定的候选对象 Key 管理状态，避免使用数组下标导致删除候选后状态错位。

#### 5.1.2 请求策略

- 账号被选中时按账号 ID 查询模型。
- 同一编辑会话中缓存账号模型结果，避免同一账号被重复请求。
- 多个新增账号的模型查询可并行执行。
- 请求失败可通过重新选择账号或“重试”按钮再次请求。
- 组件卸载时取消未完成请求，避免更新已销毁组件。

#### 5.1.3 交集算法

```text
第一个账号的模型 ID 集合作为初始集合
逐个与其他账号的模型 ID 集合求交集
按模型 ID 排序
使用模型 ID 去重
```

展示名称优先采用接口返回的 `display_name`；不存在时使用模型 `id`。

### 5.2 现有模型路由结构

现有结构保持不变：

```json
{
  "coding-fast": [
    {
      "model": "claude-sonnet-4-6",
      "account_ids": [101, 102],
      "priority": 0
    }
  ]
}
```

本次不新增持久化字段。每日 Token 限额如果当前由独立配置保存，也继续使用现有机制，不因本需求变更。

### 5.3 新增接口契约

#### `API-01` 查询分组模型路由

```http
POST /api/v1/integrations/model-routes/list
Authorization: Bearer <external_api_key_provisioning.access_token>
Content-Type: application/json
```

请求体：

```json
{
  "group_name": "Claude生产分组"
}
```

请求 DTO：

```go
type GetGroupModelRoutesRequest struct {
    GroupName string `json:"group_name" binding:"required"`
}
```

校验规则：

- 必须是合法 JSON。
- `group_name` 必填。
- 去除首尾空格后不能为空。
- 使用清理后的名称精确查找未删除分组。

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_name": "Claude生产分组",
    "routes": [
      {
        "route_alias": "coding-fast",
        "upstream_models": [
          "claude-sonnet-4-5",
          "claude-sonnet-4-6"
        ]
      },
      {
        "route_alias": "coding-powerful",
        "upstream_models": [
          "claude-opus-4-6"
        ]
      }
    ]
  }
}
```

响应 DTO：

```go
type GroupModelRouteItem struct {
    RouteAlias     string   `json:"route_alias"`
    UpstreamModels []string `json:"upstream_models"`
}

type GetGroupModelRoutesResponse struct {
    GroupName string                `json:"group_name"`
    Routes    []GroupModelRouteItem `json:"routes"`
}
```

空配置响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_name": "Claude生产分组",
    "routes": []
  }
}
```

错误响应：

| HTTP 状态 | 错误码 | 场景 |
|---:|---|---|
| `400` | `INVALID_REQUEST` | JSON 错误、缺少分组名或分组名为空 |
| `401` | `INVALID_ACCESS_TOKEN` | Authorization 缺失、格式错误或 Token 不匹配 |
| `404` | `NOT_FOUND` | integrations 功能未启用或 Access Token 未配置 |
| `404` | `GROUP_NOT_FOUND` | 指定分组不存在 |
| `500` | `INTERNAL_ERROR` | 查询或解析路由配置发生内部错误 |

#### 鉴权

将新路由注册到现有 integrations 路由组：

```text
/api/v1/integrations
  ├── POST /api-keys/getOrCreate
  └── POST /model-routes/list
```

两者共同经过：

```text
ExternalProvisioningAuth
→ ProvisioningHardening
→ Handler
```

Bearer Token 使用常量时间比较，避免时序侧信道风险。

#### 幂等性

接口为纯查询操作，不写数据库；相同请求和相同数据库状态下返回相同结果，不需要幂等键或事务。

### 5.4 后端职责

建议扩展现有 External Provisioning 相关分层：

#### Handler

- 绑定和清理 `group_name`。
- 调用 Service。
- 映射错误码。
- 输出统一响应。
- 写入查询成功或失败的安全审计日志。
- 不记录 Authorization Token。

#### Service

- 根据分组名精确查询分组。
- 读取并解析现有 `model_routing`。
- 遍历全部路由别名和候选。
- 提取候选中的 `model`。
- 去除空模型和重复模型。
- 对路由及模型排序。
- 构造只读响应。

#### Repository

优先复用现有 `GetByNameExact(ctx, groupName)`，不新增专用 SQL，不新增数据表。

### 5.5 数据与迁移

#### 数据库

本需求不涉及数据库结构变更。状态继续保存在：

- 现有分组记录的 `model_routing` JSON 字段。
- 现有账号 `credentials.model_mapping` 等模型限制配置。
- 需求 1 的模型查询结果只存在于前端内存中。

#### 迁移

- 不需要数据库 Migration。
- 不需要历史数据回填。
- 不需要修改已有路由配置。
- 不需要改变 `account_ids` 数组结构。
- 不需要数据回滚脚本。

### 5.6 安全与隐私

- `NFR-01`：新接口必须经过 integrations 固定 Token 鉴权。
- `NFR-02`：响应不得暴露账号 ID、账号名称、账号凭证或优先级。
- `NFR-03`：日志不得记录 Access Token 或完整 Authorization Header。
- `NFR-04`：分组名必须去除首尾空格并精确查询。
- `NFR-05`：接口沿用现有 integrations 限流和请求加固策略。
- `NFR-06`：管理员账号模型接口继续使用管理员 JWT，不对外开放。

### 5.7 可靠性与可观测性

- 需求 1 的账号模型请求失败时，不允许使用不完整交集。
- 多账号查询可并行，但单个候选的并发请求数量受已选账号数量限制。
- 新接口增加结构化审计日志：事件名、分组名、来源 IP、查询结果、路由数量和失败原因。
- 不记录上游账号信息和鉴权 Token。
- 接口只读取单个分组配置，不需要额外缓存。

### 5.8 配置、发布与回滚

本需求默认不新增配置项，复用：

```yaml
external_api_key_provisioning:
  enabled: true
  access_token: "..."
  rate_limit_biz_per_minute: ...
  rate_limit_auth_per_minute: ...
```

因此默认不需要修改配置文件。

发布方式：

1. 先发布后端，使新接口可用。
2. 再发布前端模型路由交互。
3. 验证管理员编辑流程和 integrations 查询流程。

回滚方式：

- 前端回滚后恢复手填模型，不影响已有数据。
- 后端回滚后新查询接口消失，不影响模型路由运行。
- 无数据库变更，因此不存在 Schema 回滚风险。

## 6. 关键流程伪代码

### 6.1 账号变更与模型交集

```text
function onCandidateAccountsChanged(candidate):
  selectedIds = unique(candidate.accounts.map(account => account.id))

  if selectedIds is empty:
    candidate.model = ""
    disable model selector
    return

  results = load model lists for all selectedIds in parallel

  if any result failed:
    mark candidate model options unavailable
    disable model selector
    show loading error
    return

  intersection = model IDs from first account

  for each remaining account:
    intersection = intersection ∩ model IDs from account

  sort intersection by model ID
  candidate.availableModels = intersection

  if candidate.model is not in intersection:
    candidate.model = ""
    show model reselection hint

  enable model selector only when intersection is not empty
```

### 6.2 保存前校验

```text
function validateCandidate(candidate):
  require at least one selected account
  require upstream model is not empty
  require upstream model exists in current account model intersection
  require priority is a non-negative integer

  if daily token limit is provided:
    require it is a non-negative integer

  return validation result
```

### 6.3 查询分组路由

```text
function getGroupModelRoutes(request):
  authenticate integrations Bearer token
  apply integrations hardening and rate limit

  groupName = trim(request.group_name)
  if groupName is empty:
    return 400 INVALID_REQUEST

  group = repository.getByNameExact(groupName)
  if group does not exist:
    return 404 GROUP_NOT_FOUND
  if repository failed:
    return 500 INTERNAL_ERROR

  config = parse existing group.model_routing
  if config is empty:
    return success(groupName, [])

  routes = []

  for routeAlias in sorted config keys:
    modelSet = empty set

    for candidate in config[routeAlias]:
      model = trim(candidate.model)
      if model is not empty:
        add model to modelSet

    routes.append({
      route_alias: routeAlias,
      upstream_models: sorted modelSet
    })

  audit successful query without token or account data
  return success(groupName, routes)
```

## 7. 验证策略

### 7.1 前端单元测试

- `AC-01`：未选择账号时模型下拉框禁用。
- `AC-02`：选择一个账号后展示该账号接口返回的模型。
- `AC-03`：选择多个账号后只展示模型交集。
- `AC-04`：账号无共同模型时展示空状态并阻止保存。
- `AC-05`：删除账号后重新计算交集。
- `AC-06`：账号变化导致原模型失效时清空原模型。
- `AC-07`：模型请求失败时禁用模型下拉框并展示错误。
- `AC-08`：相同账号在同一编辑会话中不重复加载。
- `AC-09`：提交数据仍符合现有 `model_routing` 结构。
- `AC-10`：历史模型有效时能够正常回显。

### 7.2 后端单元测试

- `AC-11`：请求体缺少 `group_name` 返回 `400`。
- `AC-12`：空白分组名返回 `400`。
- `AC-13`：分组不存在返回 `404 GROUP_NOT_FOUND`。
- `AC-14`：无路由配置返回空数组。
- `AC-15`：正常配置返回所有路由别名和模型。
- `AC-16`：同一路由下重复模型被去重。
- `AC-17`：路由别名和模型名稳定排序。
- `AC-18`：返回体不包含账号 ID、优先级和限额。

### 7.3 鉴权与集成测试

- `AC-19`：缺少 Authorization 返回 `401`。
- `AC-20`：错误 Token 返回 `401`。
- `AC-21`：正确 Bearer Token 可以调用。
- `AC-22`：功能未启用时返回 `404`。
- `AC-23`：接口经过现有 integrations 限流和安全加固。
- `AC-24`：管理员 JWT 不能替代 integrations Access Token，除非值恰好与固定 Token 一致。
- `AC-25`：完整请求响应符合统一 `code/message/data` 契约。

### 7.4 数据与兼容性验证

- `AC-26`：不生成数据库迁移文件。
- `AC-27`：已有模型路由配置不需要转换即可继续读取。
- `AC-28`：保存前后 `model_routing` 字段结构保持一致。
- `AC-29`：现有路由调度测试保持通过。
- `AC-30`：`getOrCreate` 接口行为和鉴权保持不变。

性能专项测试不作为本期强制要求；接口只读取一个分组，前端请求数量与所选账号数线性相关。

## 8. 实施顺序与后续拆分建议

### 阶段一：模型交集基础逻辑

- 补充账号模型 API 的前端封装。
- 实现模型列表标准化与交集函数。
- 完成纯函数单元测试。

产物：可复用的模型交集逻辑。

### 阶段二：模型路由编辑器改造

- 调整账号和模型的界面顺序。
- 增加模型加载、缓存、错误和空交集状态。
- 增加历史配置校验。
- 验证提交结构未变化。

依赖：阶段一。

### 阶段三：分组路由查询 Service

- 定义内部查询输入和输出。
- 复用分组精确查询。
- 解析和整理现有模型路由配置。
- 完成排序、去重和单元测试。

可与阶段一、阶段二并行。

### 阶段四：接口与鉴权接入

- 新增 Handler DTO。
- 注册 integrations 路由。
- 复用鉴权和安全加固中间件。
- 增加接口契约、鉴权和错误测试。

依赖：阶段三。

### 阶段五：集成验证

- 验证管理员配置流程。
- 验证多账号交集。
- 验证新接口成功和失败响应。
- 回归 `getOrCreate`、分组管理和模型路由调度。
- 确认没有数据库 Migration 和配置结构变更。

### 推荐 MVP 拆分边界

1. 模型路由多账号模型交集选择。
2. integrations 分组模型路由查询接口。

两个 MVP 不共享持久化变更，可以独立开发和验证。

## 9. 风险与开放项

| 编号 | 风险 | 可能性/影响 | 应对措施 |
|---|---|---|---|
| `R-01` | 多账号数量较多时触发较多模型查询 | 中/低 | 编辑会话内按账号 ID 缓存，并行加载并取消失效请求 |
| `R-02` | 历史模型不在当前账号模型列表中 | 中/中 | 明确展示失效状态并要求重新选择 |
| `R-03` | 任意账号模型查询失败导致交集不完整 | 中/中 | 不使用部分结果，禁用模型选择并支持重试 |
| `R-04` | Go Map 遍历导致接口返回顺序不稳定 | 高/低 | 对路由别名和模型名显式排序 |
| `R-05` | 新接口意外暴露内部账号信息 | 低/高 | 使用专用响应 DTO，只输出别名和模型 |
| `R-06` | 现有模型接口的平台差异导致返回字段不完全一致 | 中/中 | 前端统一只依赖模型 `id`，展示名作为可选字段 |
| `R-07` | integrations Token 权限较大 | 中/中 | 保持固定 Token 鉴权、限流、安全日志及最小化响应 |

非阻塞开放项：接口路径 `/api/v1/integrations/model-routes/list` 为已批准的当前实施路径；若后续需要变更，应同步更新接口契约与测试。

## 10. 需求追踪矩阵

| 目标/需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| `G-01`、`FR-01`～`FR-03` | 账号模型联动 | 路由编辑器、账号模型 API | `AC-01`、`AC-02` |
| `G-02`、`FR-04`～`FR-07` | 多账号模型交集 | 交集函数、前端临时状态 | `AC-03`～`AC-06` |
| `FR-08`～`FR-10` | 编辑可靠性与兼容性 | 错误状态、提交校验 | `AC-07`～`AC-10` |
| `G-03`、`FR-11`～`FR-15` | 分组路由查询 | Repository、Service、Handler | `AC-11`～`AC-18` |
| `G-04`、`FR-16`～`FR-18` | integrations 鉴权 | 路由组、中间件 | `AC-19`～`AC-25` |
| `G-05`、`D-03` | 数据兼容 | 现有 `model_routing` | `AC-26`～`AC-30` |
| `NFR-01`～`NFR-06` | 安全与隐私 | 鉴权、DTO、日志、限流 | `AC-18`～`AC-25` |

## 11. 审核记录

| 版本 | 状态 | 说明 |
|---|---|---|
| `v0.1` | 草案 | 根据两个需求生成；确认多账号模型取交集；保持数据库和现有持久化结构不变。 |
| `v1.0` | 最终版，用户已批准 | 用户于 2026-07-16 明确批准，进入最终文件交付。 |
