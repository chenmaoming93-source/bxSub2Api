# MVP-016：交付运维模型耗时分析模块

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 将新接口接入现有 Ops 页面，范围包含两个图表和按模型交互。
- Dependencies: `MVP-004, MVP-015`

## 预期成果

`/admin/ops` 增加模型耗时百分位图和平均耗时趋势图。

## 背景

页面基础能力由 MVP-014 提供，新接口由 MVP-015 提供。参考 `assets/final-ops-prototype.png`。

## 范围内

- `ModelLatencyPercentileChart`。
- `ModelAverageLatencyTrendChart`。
- 模型搜索和单模型/最多 5 个模型对比。
- 复用顶部时间、平台、分组筛选。
- 点击图表点位打开 `OpsRequestDetailsModal`。
- 加载、错误、空状态和降级展示。

## 范围外

- 不使用抽屉。
- 不改变全局 Ops 指标。
- 不修改请求详情接口。

## 验收标准

- [x] 百分位显示 P50/P90/P95/P99。
- [x] 平均耗时按时间变化显示。
- [x] 最多同时比较 5 个模型。
- [x] 点位点击打开请求详情弹窗。
- [x] 新模块失败时原 Ops 页面仍可用。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- 多模型筛选、点位下钻和接口失败人工验证。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Ops tests | `pnpm run test:run -- src/views/admin/ops` | 19 个测试通过 |
| Component | `frontend/src/views/admin/ops/components/OpsModelLatencyPanel.vue` | 百分位表、趋势图、最多 5 个模型选择、点位详情事件 |
| Integration | `frontend/src/views/admin/ops/OpsDashboard.vue` | 新模块失败时保留原 Ops 模块 |

## 执行记录

运维页面已接入模型耗时百分位与趋势接口，筛选沿用顶部时间、平台、分组和 query mode，详情保持居中请求弹窗。
