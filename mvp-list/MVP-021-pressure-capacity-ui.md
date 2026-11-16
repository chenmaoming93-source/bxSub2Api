# MVP-021: 重建压力热力图与并发水位容量卡

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要同时落地四维热力图、单账号口径、tooltip、四层水位和三个底部容量指标。
- Dependencies: `MVP-017`, `MVP-018`, `MVP-019`

## Outcome

压力区左侧可按用户、分组、别名、账号切换热力图及指标；右侧展示四层并发水位和排队、等待、今日限流指标，并与当前筛选保持一致。

## Context

当前热力图只有 metric select，缺少 Demo 的独立行维度切换和可解释 tooltip；容量卡尚未实现。

## In Scope

- 热力图独立控制行维度（用户/分组/别名/账号）和值类型（采样并发/Token 速率/错误率）。
- 采样并发热力格使用 MVP-018 契约，悬浮和键盘聚焦显示对象、时间桶、实际并发、并发上限、采样时刻。
- 单账号筛选时提示文案使用“实际并发 / 账号并发上限”。
- 实现“并发水位与容量”卡：全局、分组用户级、账号级、路由级。
- 底部固定三项：当前排队请求、平均等待、今日限流次数，并显示各自可靠性和时间口径。
- 无采样或部分缺数时独立降级，不把卡片整体清空。

## Out of Scope

- 不在本阶段实现热力格点击后的链路滚动。
- 不提供容量调整或手工释放排队。
- 不用百分比替代实际并发/上限值。

## Implementation Notes

- 着色使用服务端实际/上限比率，tooltip 展示原始值。
- 移动端热力图允许容器内横向滚动，页面本身不得横向溢出。
- 今日限流次数标注“自然日累计”，避免被误读为当前时间窗口。
- 热力格点击事件先只暴露稳定的 `HeatmapSelection` 数据，供 MVP-022 使用。

## Acceptance Criteria

- [x] 热力图行维度与指标可独立切换，切换维度不改变全局筛选。
- [x] 单账号筛选时每个方块分母均为该账号生效并发上限。
- [x] tooltip 的分子、分母、采样时刻来自同一采样点。
- [x] 容量卡四层水位和三个底部指标全部可见且口径文字明确。
- [x] 无等待样本时显示“无近 5 分钟采样”，不显示 0ms。
- [x] 深浅色和窄屏布局通过视觉检查。

## Verification Plan

```powershell
cd frontend
pnpm run test:run -- src/components/observability/__tests__/PressureHeatmap.spec.ts src/views/admin/__tests__/ModelObservabilityView.spec.ts
pnpm run typecheck
pnpm run lint:check
pnpm run build
```

人工对照 `screenshots/heat-account-concurrency-tooltip-wide.png` 与 `screenshots/concurrency-capacity-card-wide.png`。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Heatmap controls | `ModelObservabilityView.vue` | User/group/alias/account row controls are independent from global trend grouping |
| Tooltip | `PressureHeatmap.vue` | Hover/focus shows paired actual concurrency, limit and sample timestamp |
| Capacity card | `ConcurrencyCapacityCard.vue` | Four levels plus queue depth, average wait and daily rate-limit metrics |
| Tests | `PressureHeatmap.spec.ts`, `ConcurrencyCapacityCard.spec.ts` | PASS, 5 focused tests including unavailable wait and tooltip pairing |
| Responsive | 390px browser check | PASS: `scrollWidth=382`, `clientWidth=382`, capacity card present |

## Execution Notes

热力图行维度属于 GROUP BY；全局观测对象属于 WHERE，二者不能复用同一个状态字段。

用户行维度的 Token/错误率可查询；现有并发采样不包含用户归属，因此用户 + 采样并发返回独立的 `user_concurrency_samples_unavailable` 不可用状态，不伪造用户并发值。
