# MVP-022: 恢复同页链路下钻与热力图联动

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要重排页面结构、重构链路选择上下文、联动滚动并调整告警顺序。
- Dependencies: `MVP-021`

## Outcome

调用链路、交叉明细和单维排行位于观测台同一页面下方，点击热力格会定位对应分组、路由、账号和时间桶并滚动到链路区，告警区位于链路区之后。

## Context

当前 `DrilldownPanel` 已在同页，但位于排行之后；链路为六列并含 API Key；热力格点击会切换全局筛选，可能导致整张热力图塌缩。设计要求四列链路和独立下钻上下文。

## In Scope

- 页面顺序调整为压力区 → 同页下钻区 → 告警与数据质量。
- 链路列收敛为：用户/部门 → 分组场景 → 路由别名 → 模型账号；API Key 只在交叉明细和抽屉出现。
- 热力格点击写入独立 `HeatmapSelection`，保留上方全局筛选与热力图，同时更新链路选中项、摘要和滚动定位。
- 支持键盘 Enter/空格触发同样定位。
- 告警“查看链路”滚动到同页下钻区，不切换路由。
- 链路节点逐级联动，下一列只显示当前路径可达对象，并始终展示账号唯一绑定模型。

## Out of Scope

- 不新增 `/admin/observability/paths` 路由。
- 不修改路由配置或切流。
- 不改变 CSV 公式注入防护。

## Implementation Notes

- 区分 `globalFilters` 与 `drilldownSelection`；点击热力格不应自动切换全局 account/group 筛选。
- 交叉明细查询使用当前全局筛选映射到服务端路径请求，并额外携带选中时间桶上下文；若现有 API 不支持精确桶，明确降级为窗口级明细。
- API Key 仅作为 detail/matrix 字段，不作为主链路第五列。
- 下钻区标题展示热力图定位来源、时间桶和实际/上限。

## Acceptance Criteria

- [x] 页面只有一个观测台路由，链路区位于压力区下方、告警区上方。
- [x] 点击热力格后页面滚动到链路区，并高亮对应用户/分组/别名/账号。
- [x] 点击热力格不会把上方热力图缩减为单行或清空用户当前全局筛选。
- [x] 主链路固定四列，API Key 只出现在明细和详情。
- [x] 告警按钮只做页内跳转，浏览器地址仍为 `/admin/observability`。

## Verification Plan

```powershell
cd frontend
pnpm run test:run -- src/components/observability/__tests__/DrilldownPanel.spec.ts src/views/admin/__tests__/ModelObservabilityView.spec.ts
pnpm run typecheck
pnpm run build
```

人工验证热力格点击、Enter、告警跳转和四列链路。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Page order | `ModelObservabilityView.vue` | Pressure/capacity → inline drilldown → data quality/alerts |
| Heatmap context | `handleHeatmapSelection` | Stores independent focus context and scrolls to `#observability-drilldown` without mutating global filters |
| Four-column chain | `DrilldownPanel.vue` | User → group → route alias → account; API Key and model remain in matrix/detail |
| Focus highlight | `DrilldownPanel.vue` | Heatmap focus selects matching node and downstream reachability; keyboard node activation supported |
| Alert navigation | `openAlertObject` | Uses same `/admin/observability` route and scrolls to inline drilldown |
| Tests | `DrilldownPanel.spec.ts`, `ModelObservabilityView.spec.ts` | PASS: four columns, focus context and no global filter mutation |

## Execution Notes

当前已存在的单页路由保持不变；本次是信息架构修正，不新增页面。

链路节点点击只更新内部级联选择；交叉明细与单维排行仍可显式驱动全局筛选。
