# 模型路由平台页面重整与 UI V2 全站改造实施方案

**状态：Final — user approved**  
**版本：v1.1**  
**日期：本次会话生成**  
**变更摘要：** 将最终确认的十个 Part、旧页面隐藏策略、接口及组件复用、统一弹窗交互、UI V2 全站改造、原型图参考和首个 Part 完成后的暂停节点整合为实施基线。

---

# 1. 目标与范围

本次改造包含两条主线：

1. 重整管理员统计、使用记录、运维监控、个人仪表盘和个人使用记录；
2. 新增 UI V2 组件体系，逐步替换当前仍可访问页面的旧视觉组件。

目标：

- 建设新的管理员仪表盘和个人仪表盘；
- 将管理员、个人使用记录精简为明细查询页面；
- 保留账号、分组、角色权限、系统设置、可配置 Token 统计的业务能力；
- 在运维监控增加按模型耗时百分位和平均耗时趋势；
- 所有下钻详情统一使用居中弹窗，不使用抽屉或页面内展开；
- 最大限度复用已有接口、组件和数据处理逻辑；
- 保留全部旧页面和旧组件代码；
- 完成 UI V2 基础组件和 Part 1 管理员仪表盘后暂停开发，等待用户检查。

非目标：

- 删除或覆盖旧页面、旧组件；
- 本轮优化使用记录搜索逻辑；
- 修改计费规则、账号、分组、权限、系统设置或 Token 统计业务规则；
- 首个管理员仪表盘验收前继续开发后续 Part；
- 将停用页面迁移为 UI V2 页面。

成功标准：

- 十个 Part 有明确页面、路由、权限和接口复用范围；
- 停用页面无法通过 URL 访问；
- 统计页面统一使用页面级时间范围；
- 所有详情使用居中 Dialog；
- 可复用业务逻辑不重复实现；
- UI V2 风格统一；
- 旧源代码仍存在；
- Part 1 完成后开发暂停。

---

# 2. 原型图参考基线

新版 UI 开发必须参考以下图片的视觉、信息层级和交互形式：

| 原型 | 文件 |
|---|---|
| 管理员仪表盘默认状态 | `assets/final-admin-dashboard-prototype-v2.png` |
| 场景账号详情弹窗 | `assets/final-admin-dashboard-scene-modal.png` |
| 部门用户详情弹窗 | `assets/final-admin-dashboard-department-modal.png` |
| 管理员使用记录 | `assets/final-admin-usage-prototype.png` |
| 运维监控 | `assets/final-ops-prototype.png` |
| 个人仪表盘 | `assets/final-personal-dashboard-prototype-v2.png` |

原型图作为布局、颜色、间距、圆角、阴影、信息密度、图表方向和弹窗交互的视觉验收基线，不要求逐像素复制。

视觉特征：深色左侧导航、浅灰页面背景、白色卡片、蓝色主色、轻量边框、统一日期工具栏、密集但克制的数据表格和居中大尺寸弹窗。

---

# 3. 已确认决策

| 编号 | 决策 |
|---|---|
| ADR-01 | 旧页面和旧组件代码全部保留，不删除、不覆盖 |
| ADR-02 | 新 UI 组件放入独立 `frontend/src/components/ui-v2/` |
| ADR-03 | 所有详情使用居中弹窗 |
| ADR-04 | 不使用右侧抽屉、页面内展开、手风琴详情 |
| ADR-05 | 分组与场景是同一概念，统一称为“场景（分组）” |
| ADR-06 | 高数量场景使用排行、搜索、Top 12/全部、分页 |
| ADR-07 | 模型、场景、部门排行不使用饼图或高基数堆叠图 |
| ADR-08 | 管理员和个人仪表盘按最终需求保留费用指标 |
| ADR-09 | 停用页面移除菜单、路由和页面权限映射，但保留源代码 |
| ADR-10 | 完成 UI V2 基础组件和管理员仪表盘后暂停 |
| ADR-11 | 场景排行需跨日期聚合，不能直接假设现有接口已经满足 |

---

# 4. 最终页面和停用路由

## 4.1 管理员页面

| Part | 页面 | 路径 | 处理 |
|---|---|---|---|
| 1 | 仪表盘 | `/admin/dashboard` | 新页面，旧组件保留 |
| 2 | 使用记录 | `/admin/usage` | 只保留明细和错误请求 |
| 3 | 账号管理 | `/admin/accounts` | 业务整体保留，迁移 UI |
| 4 | 分组管理 | `/admin/groups` | 业务整体保留，迁移 UI |
| 5 | 角色权限 | `/admin/roles` | 业务整体保留，迁移 UI |
| 6 | 运维监控 | `/admin/ops` | 原页面保留，增加模型耗时 |
| 7 | 系统设置 | `/admin/settings` | 业务整体保留，迁移 UI |
| 8 | 可配置 Token 统计 | `/admin/token-statistics` | 业务整体保留，迁移 UI |

## 4.2 我的账户

| Part | 页面 | 路径 | 处理 |
|---|---|---|---|
| 9 | 个人仪表盘 | `/dashboard` | 新页面，旧组件保留 |
| 10 | 个人使用记录 | `/usage` | 只保留明细和错误请求 |
| — | API Key | `/keys` | 业务保留，后续迁移 UI |
| — | 个人资料 | `/profile` | 业务保留，后续迁移 UI |

## 4.3 停用页面

```text
/admin/default-group-routing
/admin/channels
/admin/channels/pricing
/admin/channels/monitor
/admin/subscriptions
/admin/announcements
/admin/proxies
/admin/redeem
/admin/promo-codes
```

需要同步处理：

- `frontend/src/components/layout/AppSidebar.vue`；
- `frontend/src/router/index.ts`；
- `frontend/src/rbac/permissionMatrix.ts`；
- `frontend/src/views/admin/SettingsView.vue` 中的失效链接；
- `/admin/channels` 父级重定向。

分组管理仍使用 `GroupModelRoutingEditor`，停用默认路由页面不等于删除该组件。

---

# 5. 统一交互规范

所有下钻使用 `UiDialog`：

- 居中显示；
- 页面遮罩；
- ESC 和关闭按钮关闭；
- 打开时锁定页面滚动；
- 关闭后恢复焦点；
- 统计表格使用 `extra-wide`；
- 复杂请求详情可使用 `full`。

禁止右侧抽屉、内嵌子表、手风琴详情和点击后改变当前卡片高度。

详情映射：

| 触发对象 | 弹窗 |
|---|---|
| 场景（分组）排行 | 场景下账号/模型 Token 用量 |
| 部门排行 | 部门下用户 Token 用量 |
| 模型排行 | 模型调用汇总或明细 |
| 使用记录 | 请求详情 |
| 错误请求 | 错误详情 |
| 运维图表点位 | 对应请求列表 |
| 告警事件 | 告警详情 |

---

# 6. Part 1：管理员仪表盘

顶部使用一套 `DateRangePicker`、粒度选择和刷新按钮，所有范围数据共用 `start_date/end_date/granularity`。

## 6.1 总览指标

| 指标 | 时间控制 | 现有接口/字段 | 处理 |
|---|---:|---|---|
| 系统总用户数 | 否 | `/admin/dashboard/stats` → `total_users` | 直接复用 |
| 系统总模型账号数 | 否 | `/admin/dashboard/stats` → `total_accounts` | 直接复用 |
| 系统总 API Key 数 | 否 | `/admin/dashboard/stats` → `total_api_keys` | 直接复用 |
| 时间范围总请求数 | 是 | `/admin/usage/stats` → `total_requests` | 直接复用 |
| 时间范围总 Token | 是 | `/admin/usage/stats` → `total_tokens` | 直接复用 |
| 系统累计 Token | 否 | `/admin/dashboard/stats` → `total_tokens` | 直接复用 |
| 性能指标 | 是 | 当前 RPM/TPM 为近 5 分钟值 | 新增范围口径 |
| 平均响应耗时 | 是 | `/admin/usage/stats` → `average_duration_ms` | 直接复用 |
| 系统累计费用消耗 | 否 | `/admin/dashboard/stats` → `total_actual_cost` | 直接复用 |

建议新增：

```text
average_rpm = 范围请求数 / 范围分钟数
average_tpm = 范围 Token / 范围分钟数
```

## 6.2 图表

| 图表 | 接口/组件复用 | 新展示 |
|---|---|---|
| 用户 Token Top12 趋势 | `GET /admin/dashboard/users-trend`、`getUserUsageTrend()` | V2 多折线图 |
| 模型 Token 分布 | `GET /admin/dashboard/models`、`getModelStats()` | 横向柱状排行 |
| 场景（分组）Token 分布 | `GET /admin/dashboard/groups`、`getGroupStats()` | 聚合后横向排行 |
| 部门 Token 分布 | `GET /admin/usage/department-stats`、`queryDepartmentStats()` | 横向排行 |

场景数据当前主要按日期和 `group_id` 组织，前端需跨日期聚合；如数据量或性能不满足，再新增后端汇总接口。

场景详情复用：

```text
GET /admin/usage/scene-account/daily
aquerySceneAccountDaily()
```

部门详情复用：

```text
GET /admin/usage/department-stats/users
queryDepartmentUsers()
DepartmentUserUsageChart
```

所有详情均在 `UiDialog` 中按需加载。

---

# 7. Part 2：管理员使用记录

只保留：

- 用量明细；
- 错误请求；
- 原筛选条件；
- 查询、重置、导出、列设置；
- 表格、分页、请求详情和错误详情。

移除所有统计卡、模型/分组/接口分布、Token 趋势、部门统计和场景统计。

可整体复用：

```text
UsageFilters
UsageTable
OpsErrorLogTable
OpsErrorDetailModal
UsageExportProgress
UsageCleanupDialog
```

可复用接口：

```text
GET /admin/usage
GET /admin/usage/stats
GET /admin/usage/search-users
GET /admin/usage/search-api-keys
GET /admin/ops/request-errors
GET /admin/ops/request-errors/:id
GET /admin/ops/request-errors/:id/upstream-errors
/admin/usage/cleanup-tasks
```

---

# 8. Part 3～5

## Part 3：账号管理

页面业务整体保留：列表、筛选、CRUD、批量操作、测试、OAuth、统计、配额、错误恢复、模型同步、导入导出和定时测试。

可整体复用：

```text
AccountTableFilters
AccountTableActions
AccountActionMenu
AccountBulkActionsBar
AccountStatsModal
AccountTestModal
ImportDataModal
ReAuthAccountModal
BulkEditAccountModal
QuotaLimitCard
AccountStatusIndicator
UsageProgressBar
ScheduledTestsPanel
```

复用接口：`/admin/accounts` 及其详情、统计、用量、测试、恢复、模型同步和导入导出接口。

## Part 4：分组管理

页面业务整体保留：分组 CRUD、模型范围、关联、倍率、RPM、模型路由、并发、安全检查和排序。

可整体复用：

```text
DefaultGroupSettingsCard
GroupRateMultipliersModal
GroupRPMOverridesModal
GroupModelRoutingEditor
ModelRouteConcurrencyScheduleEditor
GroupConcurrencyViewModal
GroupSecurityCheckModal
SecurityCheckLogsModal
```

复用接口：`/admin/groups` 及其 all、详情、API Key、模型路由、并发、安全检查接口。

## Part 5：角色权限

页面整体保留：角色、权限、角色权限分配、用户角色分配、编辑和删除。

复用接口：

```text
GET/POST/PUT/DELETE /admin/rbac/roles
GET/POST/PUT/DELETE /admin/rbac/permissions
GET/PUT /admin/rbac/roles/:id/permissions
GET/PUT /admin/users/:id/roles
```

以上三个 Part 不需要新增核心业务接口，只做 UI V2 迁移。

---

# 9. Part 6：运维监控

## 9.1 原能力整体保留

保留 `OpsDashboardHeader`、并发、吞吐、切换率、延迟、错误、告警、系统日志、请求详情和错误详情组件。

复用接口：

```text
GET /admin/ops/dashboard/overview
GET /admin/ops/dashboard/snapshot-v2
GET /admin/ops/dashboard/throughput-trend
GET /admin/ops/dashboard/latency-histogram
GET /admin/ops/dashboard/error-trend
GET /admin/ops/dashboard/error-distribution
GET /admin/ops/requests
GET /admin/ops/request-errors
GET /admin/ops/upstream-errors
GET /admin/ops/alert-events
GET /admin/ops/system-logs
```

## 9.2 新增模型耗时

```text
GET /admin/ops/dashboard/model-latency-percentiles
GET /admin/ops/dashboard/model-latency-trend
```

支持时间、平台、分组和模型筛选，最多比较 5 个模型。

返回 P50/P90/P95/P99、平均值、请求数和按时间桶平均耗时。点击图表点位复用 `OpsRequestDetailsModal`。

---

# 10. Part 7：系统设置

系统设置业务整体保留，只迁移 UI V2，并清理已停用页面链接。

复用：

```text
GET/PUT /admin/settings
GET/PUT /admin/settings/default-group
邮件模板、管理员 API Key、超时、限流、整流、Beta、Web Search、备份和数据管理接口
```

不修改设置模型和保存逻辑。

---

# 11. Part 8：可配置 Token 统计

页面业务整体保留：维度、指标、统计项、发布/启停、配额、重置、运行状态、同步状态、查询和 CSV 导出。

复用 `frontend/src/api/admin/dynamicTokenStatistics.ts` 中的全部接口：

```text
/admin/token-statistics/dimensions
/admin/token-statistics/metrics
/admin/token-statistics/projections
/admin/token-statistics/quotas
/admin/token-statistics/quota-usage/reset
/admin/token-statistics/status
/admin/token-statistics/runtime
/admin/token-statistics/query
```

只替换页面布局、表单、表格、Tabs、按钮和弹窗样式。

---

# 12. Part 9：个人仪表盘

## 12.1 现有能力

复用：

```text
GET /usage/dashboard/stats
GET /usage/stats
GET /usage/dashboard/models
GET /usage/dashboard/trend
```

可以直接复用：API Key 数、范围请求、范围 Token、累计 Token、平均耗时、累计费用和模型 Token 数据。

## 12.2 新增能力

```text
GET /usage/dashboard/groups
GET /usage/dashboard/latency-trend
GET /usage/dashboard/latency-percentiles
```

个人接口必须依据认证用户确定数据范围，不允许前端传入任意 `user_id`。

页面包含：

- 场景（分组）Token 横向排行；
- 模型 Token 横向排行；
- 请求平均耗时折线图；
- 请求耗时 P50/P90/P95/P99 图。

---

# 13. Part 10：个人使用记录

只保留个人用量明细和错误请求模块。

复用：

```text
GET /usage
GET /usage/stats
GET /usage/:id
GET /usage/error-requests
GET /usage/error-requests/:id
GET /user/api-keys/:id/usage/daily
```

复用 `UsageFilters`、`UsageTable`、个人错误请求表格和错误详情弹窗，仅迁移 UI V2。

---

# 14. UI V2 组件体系

目录：

```text
frontend/src/components/ui-v2/
├── primitives/
├── forms/
├── data/
├── feedback/
├── layout/
├── charts/
└── domain/
```

基础组件：

```text
UiButton
UiIconButton
UiInput
UiSearchInput
UiTextArea
UiSelect
UiCheckbox
UiRadio
UiSwitch
UiDatePicker
UiDateRangePicker
UiFormField
UiTabs
```

展示组件：

```text
UiCard
UiStatCard
UiStatusBadge
UiTag
UiAvatar
UiProgress
UiTooltip
UiMetricValue
UiDescriptionList
UiCodeBlock
```

数据组件：

```text
UiDataTable
UiTableToolbar
UiFilterBar
UiPagination
UiColumnSettings
UiBulkActions
UiHorizontalRanking
UiKeyValueTable
UiEmptyState
UiTableSkeleton
```

反馈和弹窗：

```text
UiDialog
UiConfirmDialog
UiDetailDialog
UiToast
UiAlert
UiLoadingSpinner
UiSkeleton
UiProgressDialog
UiResultState
```

布局和图表：

```text
UiAppShell
UiSidebar
UiTopbar
UiPageHeader
UiPageContainer
UiDataPage
UiSettingsLayout
UiSection
UiToolbar
UiResponsiveGrid
UiChartCard
UiLineChart
UiMultiLineChart
UiAreaChart
UiHorizontalBarChart
UiPercentileChart
UiHistogram
UiChartLegend
chartThemeV2
```

底层继续使用 Chart.js。

旧组件继续存在，不做全局覆盖。V2 组件尽量保持旧组件 props、events 和 slots 契约，例如：

- `Input` → `UiInput`；
- `Select` → `UiSelect`；
- `Toggle` → `UiSwitch`；
- `DateRangePicker` → `UiDateRangePicker`；
- `BaseDialog` → `UiDialog`；
- `Pagination` → `UiPagination`；
- `DataTable` → `UiDataTable`。

---

# 15. 技术架构、数据和安全

数据流：

```text
页面筛选状态
  → 页面数据协调器
  → 现有/新增 API
  → 数据适配层
  → V2 图表、排行、表格
  → UiDialog 按需加载详情
```

约束：

- V2 UI 组件不直接请求业务 API；
- API 请求集中于 API 模块或页面协调器；
- 详情按需加载；
- 本次原则上不新增数据库表；
- 个人接口从认证上下文确定用户；
- 保持现有权限、脱敏和审计规则；
- 不记录完整 API Key、Token 或敏感请求体。

可靠性：

- 并行请求；
- `AbortController` 取消过期查询；
- 只接受最后一次筛选结果；
- 高基数列表分页；
- 用户趋势最多 12 条线；
- 模型耗时趋势最多 5 个模型；
- 单图表失败不影响其他模块；
- 新接口失败时保留已有统计能力。

---

# 16. 实施顺序和暂停节点

## 阶段 0：基线保护

产出路由清单、菜单清单、接口复用表、旧组件依赖关系和测试基线。

## 阶段 1：UI V2 基础组件

完成 Design Tokens、表单、卡片、指标卡、Dialog、DataTable、Pagination、AppShell、Sidebar、ChartCard、HorizontalRanking、暗色模式和响应式基础。

## 阶段 2：Part 1 管理员仪表盘

完成统一时间范围、总览指标、用户 Top12、模型排行、场景排行、部门排行、场景账号弹窗和部门用户弹窗。

## 阶段 3：强制暂停

管理员仪表盘完成并通过自动测试后：

1. 停止开发；
2. 不继续开发 Part 2～10；
3. 等待用户启动项目检查 UI 风格；
4. 用户确认后再继续。

## 阶段 4：后续页面

按顺序迁移管理员使用记录、个人仪表盘、个人使用记录、运维新增能力，再迁移 Part 3～8 和其他仍启用页面。

停用页面不进入 UI V2 迁移范围。

---

# 17. 验收标准

| 编号 | 标准 |
|---|---|
| AC-01 | 管理员仪表盘所有范围数据由顶部日期控制 |
| AC-02 | 模型、场景、部门使用横向排行榜 |
| AC-03 | 所有详情使用居中弹窗 |
| AC-04 | 使用记录只保留明细和错误查询 |
| AC-05 | Part 3、4、5、7、8 业务行为不变 |
| AC-06 | 运维原有能力不退化 |
| AC-07 | 运维支持按模型耗时百分位和平均趋势 |
| AC-08 | 个人仪表盘满足六项指标和四类图表 |
| AC-09 | 旧页面和旧组件源代码未删除 |
| AC-10 | 停用页面不可通过 URL 访问 |
| AC-11 | UI V2 遵循指定原型图风格 |
| AC-12 | 完成 Part 1 后开发暂停 |

测试包括组件测试、页面测试、接口契约测试、路由测试、权限测试、响应式检查、暗色模式检查和视觉验收。

---

# 18. 风险与应对

| 风险 | 应对 |
|---|---|
| 覆盖旧全局样式造成未迁移页面错乱 | 新增 `ui-v2`，不覆盖旧样式 |
| 场景跨日期聚合口径不一致 | 明确按 `group_id` 聚合并增加测试 |
| 详情数据过大 | 弹窗内分页、按需查询 |
| DataTable slot 不兼容 | 保持旧 slot 契约 |
| 模型耗时查询压力大 | 优先复用预聚合，限制模型数和时间范围 |
| 停用页面仍可访问 | 菜单、路由、权限映射和重定向一起清理 |
| UI V2 影响既有测试 | 新旧组件并行、逐页回归 |
| 原型图与实现差异过大 | 以上述六张原型图作为视觉验收基线 |
| Part 1 后误继续开发 | 将暂停节点作为独立交付门 |

---

# 19. 后续 MVP 分解边界

建议拆分为：

1. UI V2 Design Tokens；
2. UI V2 基础表单组件；
3. UI V2 Dialog；
4. UI V2 DataTable 和 Pagination；
5. UI V2 AppShell 和 Sidebar；
6. UI V2 ChartCard 和横向排行；
7. 管理员仪表盘数据协调；
8. 场景排行聚合；
9. 场景账号详情弹窗；
10. 部门用户详情弹窗；
11. 管理员仪表盘测试；
12. 管理员仪表盘暂停验收；
13. 管理员使用记录；
14. 个人仪表盘既有接口迁移；
15. 个人新增接口；
16. 个人使用记录；
17. 运维模型耗时接口；
18. 运维页面迁移；
19. Part 3～8 UI 迁移；
20. 停用菜单和路由清理；
21. 全站回归和视觉验收。

---

# 20. 评审记录

| 版本 | 内容 | 状态 |
|---|---|---|
| v1.0 | 页面重整、接口复用、UI V2、弹窗规范和暂停节点 | 草案 |
| v1.1 | 增加原型图参考、视觉验收基线、场景聚合修正 | 已审核通过 |
