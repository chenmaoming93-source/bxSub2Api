# MVP-009：usage 现有搜索框重构为路由别名

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `复用原筛选 UI，调整文案、候选来源及所有列表/图表参数转发，并完成页面级测试。`
- Dependencies: `MVP-008`

## 预期成果

`admin/usage` 原“模型”搜索框原位改为“路由别名”，候选来自 requested-model 统计，且明细、错误和图表一致响应同一筛选值。

## 背景

页面位于 `frontend/src/views/admin/UsageView.vue`，筛选组件位于 `frontend/src/components/admin/usage/UsageFilters.vue`。当前模型候选来源和错误页参数转发已有测试。

## 范围内

- 修改标签、placeholder 和辅助文案。
- 保留现有筛选字段位置和 `model` 请求参数。
- 候选来源改为 requested-model 数据。
- 确保明细、错误、汇总、趋势及图表均转发同一筛选值。
- 更新 i18n 和页面测试。

## 范围外

- 新增第二个路由别名输入框。
- 提供 upstream-model 搜索切换。
- 修改后端参数名。

## 实现说明

- 不要把 upstream/mapping 模型分布数据混入路由别名候选。
- 保持现有刷新、分页和错误页签状态管理。
- 页面加载过程中避免旧统计结果产生错误候选。

## 验收标准

- [x] 原搜索框显示为“路由别名”，没有新增第二个字段。
- [x] 候选只来自 requested-model 口径。
- [x] 同一筛选值被转发到用量明细、错误请求和相关图表。
- [x] 重置筛选可清除路由别名。
- [x] 前端类型检查和 UsageView 测试通过。

## 验证计划

- `cd frontend && pnpm test:run src/views/admin/__tests__/UsageView.spec.ts`
- `cd frontend && pnpm typecheck`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 文案 | `frontend/src/components/admin/usage/UsageFilters.vue` | 原“模型”搜索框原位改为“路由别名”标签；Select 增加 `search-placeholder`（`admin.usage.searchRouteAliasPlaceholder`）；空选项文案改为“全部路由别名”（`admin.usage.allRouteAliases`），未新增第二个字段。 |
| i18n | `frontend/src/i18n/locales/zh.ts`、`en.ts` | `admin.usage` 新增 `routeAlias` / `searchRouteAliasPlaceholder` / `allRouteAliases` 三个 key（中英双语）。 |
| 候选来源 | `frontend/src/views/admin/UsageView.vue` `modelNameOptions` | 仅由 `requestedModelStats`（requested-model 口径）生成，未混入 upstream/mapping 分布数据；`getModelStats` 以 `model_source: 'requested'` 请求。 |
| 转发核对 | `UsageView.vue` `applyFilters`/`loadLogs`/`loadStats`/`loadModelStats`/`loadChartData`/`loadAdminErrors` | 同一 `filters.model`（路由别名）值转发到用量明细 `list`、汇总 `getStats`、模型分布 `getModelStats`、趋势/分组 `getSnapshotV2` 及错误请求 `listErrorLogs`；`resetFilters` 重置后 `filters.model` 清空。 |
| 测试 | `cd frontend && pnpm test:run src/views/admin/__tests__/UsageView.spec.ts src/components/admin/usage/__tests__/UsageFilters.spec.ts` | 2 个测试文件、8 个用例全部通过；新增转发一致性与重置清除断言。 |
| 类型检查 | `cd frontend && pnpm typecheck` | `vue-tsc --noEmit` 通过。 |

## 执行记录

- 文案 key：`admin.usage.routeAlias`（标签）、`admin.usage.searchRouteAliasPlaceholder`（搜索占位）、`admin.usage.allRouteAliases`（空选项）。
- 候选来源：`UsageView.vue` 的 `modelNameOptions` 仅来自 `requestedModelStats`（`getModelStats` + `model_source: 'requested'`），杜绝 upstream/mapping 混入。
- 被核对的请求调用：`adminAPI.usage.list`（明细）、`adminAPI.usage.getStats`（汇总）、`adminAPI.dashboard.getModelStats`（模型分布图表）、`adminAPI.dashboard.getSnapshotV2`（趋势/分组图表）、`listErrorLogs`（错误请求）均携带同一 `model` 筛选值；`resetFilters` 清除该值。
