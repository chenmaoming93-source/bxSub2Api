# 场景 Token 用量统计与展示实施方案

> **Final — user approved（正式版，用户已批准）**  
> 版本：v0.1  
> 日期：2026-08-26  
> 变更摘要：为分组增加可重复、可为空的场景名；基于现有动态 Token 统计系统，提供按日、按场景、按模型账号和上游模型统计的管理端与外部接口，并在 `/admin/usage` 展示。

## 1. 背景与目标

当前分组的 `name` 同时承担技术唯一标识和展示名称职责，无法满足业务上的场景展示需求。

本需求目标：

1. 为分组增加 `scene_name` 字段。
2. 使用现有动态 Token 统计系统统计：
   - 每天
   - 每个场景
   - 场景内每个模型账号
   - 每个模型账号对应的 `upstream_model`
3. 同时提供：
   - 管理端接口
   - 外部调用接口
4. 在 `/admin/usage` 页面展示相同统计结果。
5. 当必要的动态统计配置不存在或未启用时，接口直接返回明确错误。

非目标：

- 不替换现有动态 Token 统计系统。
- 不使用 `scene_name` 作为统计维度。
- 不改变现有 `name` 的唯一性和技术用途。
- 不自动创建统计项或自动回填历史 Token 数据。

---

## 2. 已确认的需求与设计决策

| 项目 | 决策 |
|---|---|
| 场景名字段 | `scene_name` |
| 是否必填 | 否 |
| 是否唯一 | 否，允许重复 |
| 技术分组筛选字段 | `groups.name` |
| 日期范围 | 首尾日期均包含 |
| 日期时区 | 使用动态 Token 统计系统配置的时区 |
| 账号粒度 | 按 `account_id + upstream_model` 拆分 |
| Token 口径 | 沿用现有 `UsageLog.TotalTokens()`：输入、输出、缓存创建、缓存读取之和 |
| 统计系统 | 现有动态 Token 统计系统 |
| 管理页面 | `/admin/usage` |
| 外部接口 | 复用现有 `/api/v1/integrations` 鉴权体系 |
| 管理端与外部端逻辑 | 共用同一个后端查询 Service |
| 缺少配置时 | 返回错误，不返回空数据伪装成功 |

---

## 3. 必须配置的自定义统计项

实现完成后，管理员必须在 `/admin/token-statistics` 配置以下统计项：

```text
统计项名称：
场景-模型账号-上游模型日 Token 用量
```

名称本身不是匹配依据，真正要求是：

```text
指标：
- total_tokens（总 Token）

统计周期：
- D（日）

统计维度：
- group_id（分组）
- account_id（模型账号）
- upstream_model（上游模型）

状态：
- ACTIVE
```

配置流程：

```text
保存草稿 → 发布 → 启用
```

同时需要开启全局动态 Token 统计开关。

不需要配置：

```text
scene_name
route_alias
user_id
api_key_id
```

统计项必须按维度签名精确匹配：

```text
account_id,group_id,upstream_model
```

接口不能仅根据统计项名称判断，因为名称允许管理员自定义。

---

## 4. 功能设计

### 4.1 分组场景名

在分组创建和编辑表单中增加：

```text
场景名
```

特性：

- 可为空
- 可重复
- 最大长度建议为 100
- 仅用于展示和报表标签
- 不参与唯一性校验
- 不参与 API Key 绑定和路由匹配

分组列表建议同时展示：

```text
场景名
分组名
```

当场景名为空时显示：

```text
未设置场景名
```

### 4.2 管理端查询

管理端接口：

```text
GET /api/v1/admin/usage/scene-account/daily
```

请求参数：

```text
start_date=2026-07-01
end_date=2026-07-31
group_name=internal-group（可选）
```

`group_name` 始终匹配技术分组名 `groups.name`，不能匹配 `scene_name`。

### 4.3 外部查询

外部接口：

```text
POST /api/v1/integrations/token-usage/query/scene-account/daily
```

鉴权：

- 沿用现有外部 Bearer Token
- 沿用现有外部接口限流和请求加固中间件

请求体：

```json
{
  "start_date": "2026-07-01",
  "end_date": "2026-07-31",
  "group_name": "internal-group"
}
```

`group_name` 可选。不传时查询所有分组。

### 4.4 返回结构

```json
{
  "timezone": "Asia/Shanghai",
  "start_date": "2026-07-01",
  "end_date": "2026-07-31",
  "complete": true,
  "consistency": "mysql_eventual",
  "projection_id": 7,
  "projection_enabled_at": "2026-06-01T00:00:00+08:00",
  "last_synced_at": "2026-07-31T23:59:00+08:00",
  "days": [
    {
      "date": "2026-07-01",
      "scenes": [
        {
          "group_id": 1,
          "group_name": "internal-group",
          "scene_name": "代码助手",
          "total_tokens": 120000,
          "accounts": [
            {
              "account_id": 10,
              "account_name": "账号A",
              "upstream_model": "claude-sonnet-4-5",
              "total_tokens": 70000
            }
          ]
        }
      ]
    }
  ]
}
```

内部聚合规则：

```text
原始统计行：
日期 + group_id + account_id + upstream_model

场景总量：
同一天、同一个 group_id 的所有账号和模型相加

账号展示行：
同一天、同一个 group_id、account_id、upstream_model 一行
```

即使两个分组的 `scene_name` 相同，也不能合并。

---

## 5. 配置错误行为

接口调用时必须执行配置检查。

### 5.1 统计项不存在

```http
409 Conflict
```

```json
{
  "code": "SCENE_USAGE_STATISTICS_NOT_CONFIGURED",
  "message": "缺少必要的动态 Token 统计项：group_id + account_id + upstream_model，指标为 total_tokens，周期为日统计"
}
```

### 5.2 统计项未启用

```http
409 Conflict
```

```json
{
  "code": "SCENE_USAGE_STATISTICS_NOT_ACTIVE",
  "message": "场景 Token 统计项尚未启用，请先发布并启用该统计项"
}
```

### 5.3 全局统计功能关闭

```http
503 Service Unavailable
```

```json
{
  "code": "TOKEN_STATISTICS_DISABLED",
  "message": "动态 Token 统计功能当前未开启"
}
```

### 5.4 无数据

无数据不是配置错误，正常返回：

```json
{
  "complete": true,
  "days": []
}
```

### 5.5 数据尚未完全同步

正常返回数据，同时：

```json
{
  "complete": false,
  "consistency": "mysql_eventual"
}
```

前端显示同步提示。

---

## 6. 技术设计

### 6.1 数据流

```text
API 请求完成
    ↓
生成 UsageLog
    ↓
submitDynamicTokenUsage
    ↓
动态统计系统根据 ACTIVE Projection 生成统计操作
    ↓
Redis 累计
    ↓
同步到 token_stat_aggregates
    ↓
管理端/外部端查询
    ↓
批量补全分组名、场景名、账号名
    ↓
组装日期 → 场景 → 账号/模型结构
```

现有动态统计代码已支持：

- `group_id`
- `account_id`
- `upstream_model`
- `total_tokens`
- 按日聚合
- 同步状态和完整性判断

### 6.2 分组表变更

修改现有 `groups` 表：

| 字段 | 类型 | 可空 | 约束 | 说明 |
|---|---|---:|---|---|
| `scene_name` | `VARCHAR(100)` | 是 | 无唯一约束 | 业务场景展示名称 |

不增加索引，不增加唯一约束。

### 6.3 后端模型变更

需要同步修改：

- `backend/ent/schema/group.go`
- `backend/internal/service/group.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/repository/group_repo.go`
- `backend/internal/repository/api_key_repo.go`
- `backend/internal/handler/dto/types.go`
- `backend/internal/handler/dto/mappers.go`

分组实体映射必须覆盖 `scene_name`，避免不同查询入口返回字段不一致。

### 6.4 动态统计查询 Service

新增共享查询用例，例如：

```text
QuerySceneAccountDailyUsage
```

输入：

```text
start_date
end_date
group_name optional
```

职责：

1. 校验日期范围。
2. 校验动态统计运行状态。
3. 查找精确匹配的 ACTIVE Projection。
4. 将 `group_name` 转换为 `group_id` 过滤条件。
5. 调用现有 `tokenstat.QueryUsage`。
6. 批量加载分组和账号名称。
7. 聚合并生成统一响应结构。

管理端 Handler 和外部 Handler 只负责：

- 鉴权接入
- 参数解析
- 响应格式适配
- 错误映射

核心查询逻辑必须共用。

### 6.5 查询参数限制

建议沿用动态统计系统限制：

- 最大查询范围：366 天
- 日期必须为 `YYYY-MM-DD`
- `end_date` 包含当天
- 查询结束时间内部转换为结束日期次日零点
- 超过行数上限时返回明确错误，提示缩小日期范围或增加分组过滤

---

## 7. 管理端页面设计

修改：

```text
frontend/src/views/admin/UsageView.vue
```

增加“场景 Token 用量”区域或 Tab。

页面控件：

- 开始日期
- 结束日期
- 分组名下拉框
- 查询按钮
- 刷新按钮

分组筛选器：

- 展示可读标签时可使用 `scene_name（name）`
- 实际提交值必须是 `name`
- 不使用 `scene_name` 查询

结果展示：

```text
日期
└── 场景
    ├── 场景总 Token
    └── 模型账号 + upstream_model + Token
```

页面应展示：

- 统计时区
- Projection 启用时间
- 最后同步时间
- 数据是否完整
- 无数据状态
- 缺少配置时的可操作错误提示

同时扩展：

```text
frontend/src/api/admin/usage.ts
```

增加管理端接口调用方法和响应类型。

---

## 8. 数据库迁移与兼容性

新增一个后续编号迁移：

```sql
ALTER TABLE `groups`
ADD COLUMN scene_name VARCHAR(100) NULL;
```

由于字段可空：

- 不需要强制回填历史数据
- 老分组可以保持空值
- 老接口请求不受影响
- 老客户端忽略新字段即可

回滚建议采用前向修复，不建议生产环境直接删除字段。若必须回滚，仅在确认没有使用该字段后执行：

```sql
ALTER TABLE `groups` DROP COLUMN scene_name;
```

同时重新生成 Ent 代码并执行后端测试。

---

## 9. 安全与可靠性

### 安全

- 管理端使用现有 `usage.admin.read` 权限。
- 外部端使用现有外部 Bearer Token。
- 不返回 API Key 明文。
- 只接受有限日期范围。
- `group_name` 使用参数绑定，禁止拼接 SQL。
- 外部查询继续使用现有限流和请求加固机制。

### 可靠性

- 动态统计同步是最终一致的。
- 查询结果必须返回 `complete` 和 `last_synced_at`。
- Projection 未启用时不可默认为空数据。
- Projection 启用前的历史数据不承诺自动补采。
- 分组和账号名称查询使用批量加载，避免 N+1 查询。

### 运营

建议记录结构化日志：

```text
scene_usage_query
- source: admin/integration
- group_name
- date_range
- projection_id
- result_count
- complete
- error_code
```

---

## 10. 关键伪代码

```text
function querySceneAccountDaily(actor, input):
    validate start_date and end_date
    validate end_date - start_date <= 366 days

    authorize actor

    if dynamic token statistics is disabled:
        return TOKEN_STATISTICS_DISABLED

    projection = find ACTIVE projection where:
        metric = total_tokens
        period = day
        dimensions = group_id + account_id + upstream_model

    if projection not found:
        return SCENE_USAGE_STATISTICS_NOT_CONFIGURED

    group_id = null
    if input.group_name exists:
        group = find group by groups.name
        if group not found:
            return GROUP_NOT_FOUND
        group_id = group.id

    rows = tokenstat.QueryUsage(
        projection_id = projection.id,
        period_type = day,
        start = start_date at configured timezone,
        end = day_after(end_date) at configured timezone,
        filters = group_id if present,
        group_by = [group_id, account_id, upstream_model]
    )

    load all referenced groups and accounts in batch

    for each row:
        resolve scene_name
        resolve group_name
        resolve account_name
        append to date + group + account + upstream_model bucket

    calculate scene totals

    return nested result with:
        complete
        consistency
        projection metadata
        days
```

---

## 11. 验证策略

### 后端单元测试

- `scene_name` 可为空。
- `scene_name` 可重复。
- 分组创建和编辑正确保存字段。
- 分组 DTO 正确返回字段。
- 自定义统计项精确匹配逻辑。
- 缺少统计项返回配置错误。
- 统计项未启用返回状态错误。
- 全局统计关闭返回错误。
- 日期范围包含首尾日期。
- 按 `group_id` 正确聚合场景总量。
- 按 `account_id + upstream_model` 正确拆分。
- 同名场景不会被错误合并。

### 后端集成测试

- Projection 写入动态统计系统。
- 同步到 `token_stat_aggregates` 后能正确查询。
- 管理端接口鉴权和权限校验。
- 外部接口 Bearer Token 校验。
- 缺少配置时两个接口返回一致的业务错误。
- `group_name` 过滤只匹配技术分组名。

### 前端测试

- 场景名创建和编辑。
- 空场景名显示“未设置场景名”。
- `/admin/usage` 日期和分组筛选。
- 嵌套场景/账号/模型展示。
- 同步不完整提示。
- 配置缺失错误提示。

### 数据库迁移测试

- 新迁移可重复执行或符合项目迁移机制。
- 老数据不受影响。
- Ent schema 与数据库字段一致。

---

## 12. 实施顺序

### 阶段一：分组字段

完成：

- 数据库迁移
- Ent schema 和生成代码
- Service、Repository、DTO
- 创建/编辑接口
- 前端分组页面

验收：分组可创建、编辑和展示空值/重复场景名。

### 阶段二：共享统计查询 Service

完成：

- Projection 查找和配置校验
- 动态统计查询
- 分组、账号名称批量补全
- 日期/分组过滤
- 嵌套响应模型

验收：使用已启用统计项可以返回正确统计结果。

### 阶段三：管理端接口与页面

完成：

- `/admin/usage/scene-account/daily`
- `UsageView.vue` 展示
- 管理员权限与错误提示

验收：管理员可以按日期和技术分组名查询。

### 阶段四：外部接口

完成：

- `/integrations/token-usage/query/scene-account/daily`
- 外部鉴权
- 接口契约测试
- 限流和日志

验收：外部系统与管理端返回相同统计结果。

### 阶段五：回归验证

完成：

- 动态 Token 统计回归
- 现有 `/admin/token-statistics` 回归
- 现有分组、用量、外部接口回归
- 完整测试和构建验证

---

## 13. 风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| Projection 未及时启用 | 查询无数据 | 接口返回明确配置错误 |
| 动态统计最终一致 | 页面数据短暂滞后 | 返回 `complete` 和同步时间 |
| Projection 只从启用后开始采集 | 无法查询启用前历史 | 页面和接口明确显示统计起点 |
| 场景名重复 | 可能错误合并 | 内部始终按 `group_id` 聚合 |
| 分组/账号被软删除 | 名称查询异常 | 报表按 ID 统计，名称查询保留历史实体 |
| 组合维度过多导致结果行超限 | 查询失败 | 限制 366 天并提示缩小范围或指定分组 |

---

## 14. 验收标准

- **AC-01**：分组支持保存空的 `scene_name`。
- **AC-02**：多个分组可以使用相同的 `scene_name`。
- **AC-03**：`name` 仍保持唯一并继续用于分组过滤。
- **AC-04**：动态统计项配置为 `group_id + account_id + upstream_model`、`total_tokens`、日统计并启用后，接口可返回数据。
- **AC-05**：返回结果包含 `account_name` 和 `upstream_model`。
- **AC-06**：同一天同一场景的总 Token 等于其账号/模型明细之和。
- **AC-07**：管理端和外部接口使用相同后端逻辑，结果一致。
- **AC-08**：`group_name` 缺省时查询所有分组。
- **AC-09**：缺少必要统计项时返回明确的配置错误。
- **AC-10**：统计项未启用或全局统计关闭时返回明确原因。
- **AC-11**：`/admin/usage` 可以按日期和技术分组名查看统计。
- **AC-12**：页面展示同步状态和数据完整性。
- **AC-13**：日期范围首尾日期均被统计。
- **AC-14**：现有分组、动态统计和用量功能不回归。

---

## 15. 需求追踪矩阵

| 需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| 增加场景名 | 分组管理 | groups 表、Ent、DTO、GroupsView | AC-01～AC-03 |
| 按日统计 Token | 动态统计查询 | Projection、tokenstat QueryUsage | AC-04、AC-13 |
| 按场景汇总 | 报表 Service | group_id 聚合 | AC-06 |
| 返回账号和模型 | 报表 Service | account repository、upstream_model | AC-05 |
| 外部查询 | 外部接口 | integrations route/handler | AC-07～AC-10 |
| 页面展示 | 管理端用量页 | UsageView、admin usage API | AC-11～AC-12 |
| 配置缺失报错 | 配置校验 | Projection 状态检查 | AC-09～AC-10 |

---

## 16. 审核记录

### v0.1

用户已批准以下内容：

- `scene_name` 可为空、可重复。
- 使用动态 Token 统计系统。
- 必须配置 `group_id + account_id + upstream_model` 日统计项。
- 管理端与外部端共用相同后端逻辑。
- 管理页面放在 `/admin/usage`。
- 返回 `account_name` 和 `upstream_model`。
- 缺少配置时直接返回明确错误。
