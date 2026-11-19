# MVP-027: 补充失败率指标口径说明

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.5 day`
- Estimate rationale: 只需要统一前后端指标口径文案、扩展质量图表例交互并补回归测试。
- Dependencies: `MVP-020`, `MVP-026`

## Outcome

用户悬停最终失败率、业务限制率和 SLA 错误率图例时能看到准确口径；热力图错误率也明确说明与最终失败率一致，避免误认为三个率可以相加。

## Context

三个率共用总请求数作为分母，且业务限制率和 SLA 错误率都属于最终失败请求的子集。热力图错误率使用对应对象和时间桶的 `error_count / request_count`，不是 SLA 错误率。

## In Scope

- 质量趋势图使用可交互的外部图例，保留曲线显隐能力。
- 三个率图例提供悬浮说明和无障碍标签。
- 热力图错误率展示口径说明。
- 中英文文案和组件测试同步更新。

## Out of Scope

- 不改变任何聚合口径、分子、分母或告警阈值。
- 不新增数据库字段或配置。

## Acceptance Criteria

- [x] 三个率图例可悬浮查看中文或英文口径。
- [x] 图例点击仍能隐藏和恢复曲线，并更新 `aria-pressed`。
- [x] 热力图错误率可悬浮查看“最终失败请求数 / 总请求数”的说明。
- [x] 聚焦测试、类型检查、lint 和前端构建通过。

## Verification Plan

```powershell
cd frontend
pnpm exec vitest run src/components/observability/__tests__/QualityTrendChart.spec.ts src/components/observability/__tests__/PressureHeatmap.spec.ts src/components/observability/__tests__/observabilityLocale.spec.ts
pnpm run typecheck
pnpm run lint:check
pnpm run build
```

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Quality legend tests | `QualityTrendChart.spec.ts` | PASS: rate descriptions exist and toggling updates `aria-pressed` |
| Heatmap explanation | `PressureHeatmap.spec.ts` | PASS: error-rate help is exposed in title |
| Locale regression | `observabilityLocale.spec.ts` | PASS |
| Focused tests | Vitest | PASS: 11 tests |
| Frontend typecheck | `pnpm run typecheck` | PASS |
| Frontend lint | `pnpm run lint:check` | PASS: 0 errors; 1 pre-existing warning |
| Frontend build | `pnpm run build` | PASS |

## Execution Notes

三个率存在包含关系，不能相加。热力图错误率与最终失败率同口径，仅增加对象和时间桶维度。
