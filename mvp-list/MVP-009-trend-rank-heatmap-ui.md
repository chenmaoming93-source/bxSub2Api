# MVP-009: 实现 Token/质量趋势、压力排行与热力图

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 三类可视化共享桶和筛选语义，需处理响应式、空态、图例与真实数据一致性。
- Dependencies: `MVP-006`, `MVP-008`

## Outcome

管理员可在同一时间桶下观察 Token/并发、耗时/失败率和真实压力热力图，点击对象可驱动后续下钻。

## Context

项目已安装 Chart.js、vue-chartjs、@tanstack/vue-virtual，并已有 `components/charts` 色板。Demo 中的 SVG 与合成曲线只用于视觉参考。

## In Scope

- Token/请求速率趋势：堆叠面积或折线；另叠加真实采样并发。
- 质量趋势：duration、TTFT、最终失败率；SLO 线和样本数提示。
- PressureRanking：按账号/候选有效水位排序，展示当前/上限和共享压力。
- PressureHeatmap：group_by 切换、真实 sample/token_rate/error_rate、tooltip 和点击联动。
- 图例开关、服务端 TopN/其他、空桶、无采样和错误状态。
- 组件测试验证数据到图表序列的转换不做二次平均。

## Out of Scope

- WebSocket/SSE 实时推送。
- 动态自定义图表。
- 数据导出。

## Implementation Notes

- 不引入新图表库；优先封装公共 chart adapter，保持图例、tooltip、颜色和键盘语义一致。
- x 轴直接使用接口 `bucket_starts`，不在前端补桶。
- 热力格同时显示颜色和文本/图标严重度；颜色不是唯一编码。
- 点击排行/热力格只更新共享 filter，不直接拼 URL 参数给后端。

## Acceptance Criteria

- [x] Token、耗时和热力图使用同一轮询/筛选状态，接口响应顺序不会造成错配。
- [x] 服务端返回的 labels 长度与所有 series 对齐，前端不自造时间点。
- [x] 无采样时热力图显示 unavailable 状态，不渲染虚假 0 或系数值。
- [x] 质量趋势的 TTFT 样本数为 0 时不给 0ms 误导。
- [x] 切换 group_by 后 summary 显示值不变。
- [x] 主要组件测试、类型检查和构建通过。

## Verification Plan

```powershell
pnpm --dir frontend run test:run -- src/components/observability
pnpm --dir frontend run typecheck
pnpm --dir frontend run build
```

补充 1280px、768px、375px 三个视口的人工/截图验证。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Token trend | `frontend/src/components/observability/TokenTrendChart.vue` | Token, request and persisted sampled-concurrency overlay on dual axes; server labels only |
| Quality trend | `QualityTrendChart.vue` | P95 duration, failure/business/SLA rates and 2000ms SLO line; TTFT hidden when samples are missing |
| Data adapter | `frontend/src/components/observability/chartData.ts` | PASS: misaligned series are dropped and no interpolation/secondary averaging is performed |
| Pressure ranking | `PressureRanking.vue` | Account/candidate tabs, current/limit/waiting, severity text and progress bars; click emits only canonical account selection |
| Heatmap | `PressureHeatmap.vue` | sampled_concurrency/token_rate/error_rate; numeric text plus color severity; unavailable state when no samples |
| Click linkage | `ModelObservabilityView.vue` | Ranking/heatmap clicks update shared snapshot filters; no direct URL or SQL parameter construction |
| Polling consistency | `useObservability.ts` | One trend cycle fetches timeseries/latency/two heatmaps under the same sequence/AbortController and filter snapshot |
| Group summary | Existing query test + visible summary chips | PASS: entity grouping does not alter total requests/tokens/failure summary |
| Component tests | `pnpm exec vitest run src/components/observability/__tests__ src/composables/__tests__/useObservability.spec.ts src/views/admin/__tests__/ModelObservabilityView.spec.ts` | PASS, 11 tests |
| Typecheck | `pnpm run typecheck` | PASS |
| Production build | `pnpm run build` | PASS; `ModelObservabilityView` chunk generated |
| Visual 1280 | `C:/Users/HP/.codex/visualizations/2026/11/10/01a1c42d-c00a-70c1-8a7a-e80fefedb296/observability-1280-v2.png` | PASS: desktop KPI, charts, ranking, heatmap and TopN |
| Visual 768 | `C:/Users/HP/.codex/visualizations/2026/11/10/01a1c42d-c00a-70c1-8a7a-e80fefedb296/observability-768-v3.png` | PASS: single-column chart/ranking flow, horizontal heatmap scroll |
| Visual 375 | `C:/Users/HP/.codex/visualizations/2026/11/10/01a1c42d-c00a-70c1-8a7a-e80fefedb296/observability-375-v3.png` | PASS: no page-level horizontal overflow; charts and heatmap remain usable |
| Dark mode | `C:/Users/HP/.codex/visualizations/2026/11/10/01a1c42d-c00a-70c1-8a7a-e80fefedb296/observability-dark-1280-v2.png` | PASS after fixing native select contrast |
| Integration fix | `frontend/src/api/admin/observability.ts` | Fixed double-unwrapping caused by global Axios response interceptor; regression test added |

## Execution Notes

性能不足时优先减少 series 数量/增大桶，而不是在前端抽样后让汇总漂移。
