# MVP-025: 修复趋势维度切换后的拆分趋势展示

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 需要同时收敛后端分组序列契约、前端多序列渲染、TopN 对账和交互回归。
- Dependencies: `MVP-020`, `MVP-024`

## Outcome

“趋势维度”选择用户、分组、路由别名或模型账号后，Token/请求趋势会真正按所选对象拆分展示，而不是继续只显示整体汇总曲线。

## Context

当前 `groupBy` 已传给 `timeseries` API，后端也会追加分组序列，例如 `tokens:<dimension-key>`。但 `TokenTrendChart` 当前只消费 `total_tokens`、`total_requests` 和采样并发，因此切换“趋势维度”后视觉上几乎没有变化。

同时，后端分组序列目前只完整提供总 Token、请求和错误计数；输入、输出、缓存 Token 的维度拆分尚未返回。A 环境当前无 DDL 权限，本 MVP 不新增数据库表或字段。

## In Scope

- 明确 timeseries 分组序列契约，覆盖总 Token、输入 Token、输出 Token 和缓存 Token。
- 前端根据当前 Token 指标和趋势维度选择对应分组序列。
- 使用快照筛选项解析用户、分组、账号和路由别名的人类可读名称；缺失时回退使用 ID。
- 默认展示当前窗口内排名前 8 的对象，其余合并为“其他”，并保证各序列之和与整体值一致。
- 保留整体汇总、请求数和采样并发序列，不改变全局筛选、KPI、热力图或下钻范围。
- 多序列图例、隐藏/显示、空态、文字替代摘要和移动端可读性保持可用。
- 增加后端序列契约测试、前端图表数据测试和页面切换行为测试。

## Out of Scope

- 不改变时间窗口、时间桶选择或查询范围。
- 不改变热力图行维度和容量计算。
- 不新增数据库表、字段、索引或回填任务。
- 不引入任意指标组合编辑器或用户自定义 TopN。

## Implementation Notes

- 扩展 `groupSeriesByBucket`，同时生成 `tokens:`、`input_tokens:`、`output_tokens:`、`cache_tokens:`、`requests:` 等稳定前缀序列。
- 前端在 `chartData.ts` 中按 `tokenMetric + groupBy` 选择序列，不能把 `:other` 或未知前缀误画成整体指标。
- 维度展示名从 `snapshot.top_users/top_groups/top_aliases/top_accounts` 解析，避免直接显示数字 ID。
- “其他”使用整体桶值减去已展示对象桶值计算，结果不得为负。
- 趋势维度只触发 timeseries 刷新，不得重复刷新 snapshot、latency、heatmap 或 drilldown。
- 图表继续使用服务端时间桶；空桶不插值，序列长度不匹配时丢弃该序列并给出数据质量提示。

## Acceptance Criteria

- [x] 选择用户、分组、路由别名或模型账号后，趋势图序列和名称实际发生变化。
- [x] 总 Token、输入、输出、缓存四种 Token 指标都能按当前趋势维度拆分。
- [x] 前 8 个对象之外的剩余量进入“其他”，所有序列按桶求和等于整体 Token 值。
- [x] 有名称数据时显示名称，无名称时才显示 ID；图例可控制序列显隐。
- [x] 切换趋势维度不会改变 KPI、热力图、链路或全局筛选。
- [x] 后端序列契约、前端图表数据、页面联动行为测试和类型检查通过。

## Verification Plan

```powershell
cd backend
go test -p 1 -tags=unit ./internal/service -run 'Observability.*Timeseries|Observability.*Series'

cd ..\frontend
pnpm exec vitest run src/components/observability/__tests__/chartData.spec.ts src/views/admin/__tests__/ModelObservabilityView.behavior.spec.ts
pnpm run typecheck
pnpm run lint:check
pnpm run build
```

人工验证：

- 在相同时间窗口依次切换四个趋势维度。
- 对比整体 Token 与各拆分序列的桶级求和。
- 在浅/深色及 1440/390 视口确认图例、tooltip 和文字摘要。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Backend grouped contract | `go test -p 1 -tags=unit ./internal/handler/admin -run 'Observability.*GroupBy|Observability.*Timeseries|Observability.*Heatmap'` | PASS |
| Backend build | `go build -p 1 -o server.exe ./cmd/server` | PASS |
| Live timeseries | `group_by=account_id` | PASS: 12 labels, 32 series; `tokens:`, `input_tokens:`, `output_tokens:`, `cache_tokens:` and `requests:` each contain 3 dimension series |
| Chart adapter | `chartData.spec.ts` | PASS: 4 Token metrics mapped by dimension, Top8 plus Other reconciles to overall value |
| View behavior | `ModelObservabilityView.behavior.spec.ts` | PASS: selected groupBy and human-readable dimension labels reach Token trend |
| Refresh isolation | `useObservability.spec.ts` | PASS: dimension change refreshes only timeseries; snapshot/latency/heatmap untouched |
| Frontend tests | Focused Vitest | PASS: 20 tests |
| Frontend typecheck | `pnpm run typecheck` | PASS |
| Frontend lint | `pnpm run lint:check` | PASS: 0 errors; 1 pre-existing warning |
| Frontend build | `pnpm run build` | PASS |

## Execution Notes

已补齐总 Token、输入、输出、缓存 Token 的维度序列，并在前端按当前指标展示 Top8 对象、其他和整体 Token。该修复不涉及配置和数据库迁移。
