# MVP-020: 实现双列趋势与质量工作区

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 两张趋势图、排行和 TOP8 需要共享时间桶、切换状态、图例和空态，并与 Demo 双列布局对齐。
- Dependencies: `MVP-019`

## Outcome

左侧显示 Token/并发趋势和纵向紧接的平均耗时/平均首 Token/失败率趋势，右侧显示压力排行和分时段质量 TOP8，二者形成 Demo 风格双列工作区。

## Context

当前趋势和质量图是水平并排的基础图卡，质量图主指标仍是 P95。后端 `latency-timeseries` 已返回 `avg_duration_ms` 与 `avg_first_token_ms`，但前端未作为主口径展示。

## In Scope

- 重组主工作区为左趋势列、右排行/明细列。
- Token 图支持账号/别名/分组/用户拆分、Token 分段、图例开关、真实采样并发和 tooltip。
- 质量图主曲线改为平均请求耗时和平均首 Token，失败率独立右轴。
- P95 仅保留为详情或可选扩展，不占用主 KPI/主趋势口径。
- 两张图共用时间桶、筛选和 x 轴时间标签。
- 压力排行和分时段质量 TOP8 使用真实接口摘要，空态与缺样本独立处理。

## Out of Scope

- 不在本阶段实现热力图和容量卡。
- 不新增任意指标组合编辑器。
- 不在前端重新平均桶均值。

## Implementation Notes

- 图表继续使用 Chart.js；统一 tooltip、颜色、图例和空桶断点。
- 质量图必须读取 `avg_duration_ms`、`avg_first_token_ms`，不得把 P95 换名。
- 服务端 labels 与 series 长度不一致时丢弃失配序列并显示数据质量提示。
- 排行点击只设置待下钻上下文，不立即让上方热力图坍缩成单行，除非用户明确修改全局筛选。

## Acceptance Criteria

- [x] 两张趋势图均以时间为横轴且关键刻度对齐。
- [x] 主趋势明确展示平均请求耗时和首 Token 平均耗时，P95 不作为主曲线。
- [x] 图例、指标切换和维度拆分不会造成 summary 漂移。
- [x] 质量图无 TTFT 样本时不绘制假 0 线。
- [x] 排行和 TOP8 的 loading、empty、error、partial 状态相互独立。

## Verification Plan

```powershell
cd frontend
pnpm run test:run -- src/components/observability/__tests__/chartData.spec.ts src/views/admin/__tests__/ModelObservabilityView.spec.ts
pnpm run typecheck
pnpm run build
```

人工核对 1h/6h/24h/7d 下的时间标签和双列布局。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Workbench layout | `frontend/src/views/admin/ModelObservabilityView.vue` | Left column stacks Token and quality trends; right column stacks pressure ranking and quality TOP8 |
| Token series | `backend/internal/service/observability_query.go` | Timeseries exposes total/input/output/cache Token series from server aggregates |
| Average quality trend | `frontend/src/components/observability/chartData.ts` | Primary latency series switched from P95 to `avg_duration_ms` and `avg_first_token_ms` |
| Quality TOP8 | `QualityTopPeriods.vue` | Uses server bucket averages, sorts without re-averaging, shows missing TTFT as `无采样` |
| Chart tests | `QualityTopPeriods.spec.ts`, `chartData.spec.ts` | PASS, 7 tests: no interpolation, no fake TTFT, P95 excluded from main trend, TOP8 states |
| View test | `ModelObservabilityView.spec.ts` | PASS |
| Frontend typecheck/build | `pnpm run typecheck`; `pnpm run build` | PASS |
| Backend focused tests | Observability service, admin handler and routes | PASS |

## Execution Notes

图表只消费服务端桶；缺桶时断点或明示，不做前端插值。

后端构建在该 MVP 并行执行前后端构建时曾因 Windows `VirtualAlloc` 临时 OOM 失败；源码编译相关单测已通过，最终发布检查会在 `MVP-024` 单独串行重建后端。
