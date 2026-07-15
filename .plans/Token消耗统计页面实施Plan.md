# Token 消耗统计页面实施 Plan

**状态：Final — 用户已批准**  
**版本：v1.0**  
**日期：2026-07-03**

## 1. 建设目标

为管理员新增三个独立统计页面：

| 页面 | 路由 |
|---|---|
| 全局模型 Token 统计 | `/admin/token-usage/models` |
| 模型路由 Token 统计 | `/admin/token-usage/routes` |
| 用户模型 Token 统计 | `/admin/token-usage/users` |

目标：查看各实际上游模型、指定分组模型路由候选及指定用户各模型的 Token 消耗；支持日期和业务维度筛选；默认查询今天；首次只加载一个目标；禁止无界查询和渲染；视觉与现有管理员报表一致。

非目标：不修改 Token 计量、配额扣减和网关转发；不重算历史 `usage_logs`；不向普通用户开放；本期不建设通用 BI 查询器和 Excel 导出。

## 2. 统计口径与决策

总 Token 口径：

```text
input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens
```

统计维度：

```text
全局模型：upstream_model
模型路由：group_id + route_alias + upstream_model
用户模型：user_id + upstream_model
```

- 日期边界使用项目全局时区，默认范围为今天。
- 三个维度采用三个独立页面。
- 页面复用布局、筛选卡片、状态和分页组件，分别使用业务专属表格。
- 现有日用量表只保存 `used_tokens`，本期不拆分输入、输出和缓存 Token。
- 历史报表中的限额和优先级来自当前配置，页面必须标注“当前配置”。

## 3. 功能设计

### 3.1 全局模型统计

筛选：模型（必选、服务端搜索）、开始日期、结束日期。

字段：日期、实际上游模型、已用 Token、当前每日限额、使用率、限额状态。未配置限额时显示“不限额”；有限额时使用率为 `used_tokens / daily_limit_tokens`。

### 3.2 模型路由统计

筛选：分组（必选、服务端搜索）、路由别名（必选、按分组加载）、候选上游模型（可选）、开始日期、结束日期。

字段：日期、分组名称及 ID、路由别名、实际上游模型、当前候选优先级、已用 Token、当前每日限额、使用率、限额状态。

联动规则：未选分组时不加载路由别名；分组变化后清空路由别名、候选模型和旧结果；路由别名变化后清空候选模型并重置页码；未指定候选模型时，仅分页查询当前路由别名下的数据。

### 3.3 用户模型统计

筛选：用户（必选，支持邮箱、用户名或 ID 服务端搜索）、模型（可选，限定当前用户）、开始日期、结束日期。

字段：日期、用户名称或邮箱及 ID、实际上游模型、已用 Token、当前用户模型每日限额、使用率、限额状态。软删除用户仍可展示历史用量并标注“已删除”。

## 4. 首次加载与数据量控制

1. 日期设置为今天。
2. 优先恢复 URL 中的筛选目标。
3. URL 无目标时仅请求一个默认目标。
4. 优先选择今天存在用量的目标；无今日用量时从配置表选择一个目标。
5. 没有目标时展示引导空状态。

统一限制：服务端分页，默认每页 20 条、最大 100 条；选项搜索最多返回 20 条；缺少核心目标时禁止统计查询；筛选变化后由用户点击“查询”；搜索采用约 300ms 防抖；新请求取消旧请求，过期响应不得覆盖最新结果；前端只渲染当前页。

## 5. 前端设计

```text
frontend/src/views/admin/token-usage/
├── ModelTokenUsageView.vue
├── RouteTokenUsageView.vue
└── UserModelTokenUsageView.vue

frontend/src/components/admin/token-usage/
├── TokenUsageReportLayout.vue
├── TokenUsageSummary.vue
├── UsageStatusBadge.vue
├── ModelTokenUsageTable.vue
├── RouteTokenUsageTable.vue
└── UserModelTokenUsageTable.vue
```

新增 `frontend/src/api/admin/tokenUsage.ts`。公共能力包括 `AppLayout`、筛选卡片、汇总、加载骨架、空数据、错误与重试、服务端分页、Token 格式化、状态颜色、中英文、深色模式、响应式布局和 URL 状态同步。侧边栏增加“Token 消耗统计”分组及三个独立入口。

## 6. 后端设计

```text
backend/internal/handler/admin/token_usage_report_handler.go
backend/internal/service/token_usage_report_service.go
backend/internal/repository/token_usage_report_repo.go
```

- Handler：管理员鉴权、参数校验、错误转换和响应。
- Service：默认目标选择、配置关联、使用率和状态计算。
- Repository：有界查询、分页、排序、计数和汇总。

接口前缀：`/api/v1/admin/token-usage`。

## 7. API 契约

```http
GET /api/v1/admin/token-usage/models
```

参数：`model`（必选）、`start_date`、`end_date`、`page`、`page_size`、`sort_by=usage_date|used_tokens`、`sort_order=asc|desc`。

```http
GET /api/v1/admin/token-usage/routes
```

参数：`group_id`（必选）、`route_alias`（必选）、`upstream_model`（可选）、日期、分页和排序参数。

```http
GET /api/v1/admin/token-usage/users
```

参数：`user_id`（必选）、`model`（可选）、日期、分页和排序参数。

统一响应：

```json
{
  "items": [],
  "summary": { "used_tokens": 0 },
  "pagination": { "page": 1, "page_size": 20, "total": 0 }
}
```

选项和默认目标接口：

```http
GET /api/v1/admin/token-usage/options/models?q=&limit=20
GET /api/v1/admin/token-usage/options/groups?q=&limit=20
GET /api/v1/admin/token-usage/options/groups/{group_id}/routes?q=&limit=20
GET /api/v1/admin/token-usage/options/users?q=&limit=20
GET /api/v1/admin/token-usage/options/users/{user_id}/models?q=&limit=20
GET /api/v1/admin/token-usage/default-target?dimension=model|route|user&date=YYYY-MM-DD
```

错误：400 参数无效；401 未登录；403 非管理员；404 目标不存在；500 查询失败。GET 查询接口天然幂等。

## 8. 数据模型与索引

复用现有表，不新增业务表：

- 全局：`model_token_daily_usages`、`model_token_daily_limit_configs`
- 路由：`group_candidate_token_daily_usages`、`group_candidate_token_daily_limit_configs`、`groups`
- 用户：`user_model_token_daily_usages`、`user_model_token_daily_limit_configs`、`users`

关联键分别为 `model`、`group_id + route_alias + upstream_model`、`user_id + model`。

实施时通过 `EXPLAIN` 验证查询计划，按需增加：

```text
model_token_daily_usages(usage_date, used_tokens)
group_candidate_token_daily_usages(usage_date, group_id, route_alias)
user_model_token_daily_usages(usage_date, user_id, model)
```

不得在未验证查询计划前增加重复索引。

## 9. 安全与隐私

- 所有接口必须经过管理员权限校验。
- 普通用户不得访问其他用户信息及内部路由配置。
- 用户信息仅在管理员页面展示。
- 日志和指标不得记录认证凭据、完整邮箱或高基数搜索词。
- 统计接口只读，不改变配置或用量。

## 10. 核心流程

```text
页面加载：设置今天 -> 恢复 URL 目标 -> 缺失时请求一个默认目标
         -> 有目标则查询第一页，否则展示引导空状态
```

```text
统计查询：管理员鉴权 -> 参数和日期校验 -> 分页查询明细
         -> 查询总数及汇总 -> 关联当前限额 -> 计算使用率和状态
```

```text
前端请求：取消旧请求 -> 生成请求序号 -> 发起请求
         -> 丢弃过期响应 -> 仅使用最新响应更新页面
```

## 11. 性能、可靠性与运维

目标：常规统计接口 P95 小于 1 秒；搜索接口 P95 小于 500ms；单次响应建议小于 200KB；目标总数增长不导致浏览器内存线性增加。

降级：汇总失败但明细成功时可隐藏汇总；默认目标失败时允许手动选择；搜索失败时保留已选目标；报表故障不得影响网关、计费和 Token 记账。

监控 API 耗时、查询维度、页大小、返回条数、数据库错误、慢查询、默认目标及选项搜索失败数。

## 12. 测试与验收

测试范围：参数校验、使用率和状态、默认目标、筛选联动、请求取消、三类 Repository 查询、时区边界、当前配置关联、无配置和零用量、软删除用户、分页与汇总、权限、中英文、深色模式、移动端和大数据性能。

| ID | 验收标准 |
|---|---|
| AC-01 | 可查看指定全局模型每日 Token 用量 |
| AC-02 | 可查看指定分组和路由别名下候选模型用量 |
| AC-03 | 可查看指定用户各模型用量 |
| AC-04 | 三个页面默认日期均为今天 |
| AC-05 | 页面首次仅加载一个默认目标 |
| AC-06 | 所有明细采用服务端分页 |
| AC-07 | 搜索选项单次不超过 20 条 |
| AC-08 | 页面视觉规范一致，业务列允许不同 |
| AC-09 | 非管理员无法访问页面和接口 |
| AC-10 | 报表故障不影响网关和 Token 记账 |

## 13. 实施顺序

1. 核实现有表结构、数据语义和日期口径。
2. 编写三类查询并执行 `EXPLAIN`，必要时补充索引。
3. 实现 Repository、Service、Handler。
4. 实现搜索选项和默认目标接口。
5. 建设公共前端报表能力。
6. 分别实现三个独立页面。
7. 添加路由、侧边栏和中英文文案。
8. 完成权限、集成、E2E 和性能测试。
9. 发布并验证慢查询及接口指标。

推荐下游拆分：后端查询基础 → 搜索与默认目标 → 公共前端框架 → 三个独立页面 → 集成与性能验证。

## 14. 风险

| 风险 | 缓解措施 |
|---|---|
| 日期默认目标查询缺少合适索引 | `EXPLAIN` 后增加日期前导索引 |
| 当前限额不是历史快照 | 页面标注“当前配置” |
| 软删除用户无法常规加载 | 兼容软删除记录或回退显示用户 ID |
| 历史路由候选已被删除 | 以用量记录为准，配置字段允许为空 |
| 大日期范围查询变慢 | 分页和索引优化；性能测试后确定最大范围 |

最大日期范围在性能测试后确定，候选为 31 天或 90 天。

## 15. 需求追踪

| 目标 | 模块 | 验收标准 |
|---|---|---|
| 全局模型统计 | 全局模型页面及 API | AC-01 |
| 路由候选统计 | 模型路由页面及 API | AC-02 |
| 用户模型统计 | 用户模型页面及 API | AC-03 |
| 日期和目标筛选 | 搜索、日期和 URL 状态 | AC-04、AC-07 |
| 有界加载 | 默认目标、分页、请求取消 | AC-05、AC-06 |
| 统一视觉 | 公共布局及状态组件 | AC-08 |
| 管理员专用 | 权限中间件 | AC-09 |
| 核心链路隔离 | 独立只读统计模块 | AC-10 |

## 16. 评审记录

| 版本 | 状态 | 说明 |
|---|---|---|
| v0.1 | 草案 | 确定三个独立页面、默认查询今天及有界加载策略 |
| v1.0 | 用户批准 | 保留差异化报表结构，统一视觉规范和基础交互 |

该 Plan 已获用户批准，可以用于下游任务或 MVP 拆分。
