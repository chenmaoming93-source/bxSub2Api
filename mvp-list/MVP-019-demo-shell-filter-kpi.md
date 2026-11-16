# MVP-019: 重建 Demo 风格页头、对象筛选与 7 KPI

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要重构页面容器、响应式布局、筛选交互和快照展示，但不包含趋势与下钻细节。
- Dependencies: `MVP-016`, `MVP-017`

## Outcome

管理员进入 `/admin/observability` 后，先看到与 Demo 一致的 Hero 状态、时间工具栏、左右工作台式对象筛选、范围摘要、容量关注和 7 个 KPI。

## Context

当前视图顶部是普通筛选表单和 4 列 checkbox，没有 Hero、范围摘要、容量关注和 Demo 风格 KPI。快照当前只返回 P95，没有平均请求耗时和平均首 Token KPI 所需值。

## In Scope

- 增加快照平均耗时与平均首 Token 指标，按总分子/总样本计算。
- 重建 Hero：整体负载、活跃账号、待关注告警、数据延迟。
- 顶部工具栏接入 MVP-016 的快捷/绝对范围状态、自动刷新、手动刷新和更新时间。
- 对象筛选改为四 Tab 工作台：搜索、全选、清空、对象卡片、chips、无交集提示。
- 右侧固定范围摘要展示 7 个指标，不继续向下平铺。
- 增加容量关注卡和单行 7 卡 KPI，窄屏按响应式规则换行。
- 每个区块独立 loading/empty/error/partial 状态。

## Out of Scope

- 不重做趋势图、热力图、容量详情、下钻或告警列表。
- 不新增写操作。
- 不在本阶段完成全部键盘和截图回归。

## Implementation Notes

- 保留 `useObservability` 的 AbortController、序列校验和 ETag 缓存机制。
- 筛选切维度只改变聚合视图，不能变成 WHERE；对象选择才改变筛选。
- 范围摘要必须沿用 snapshot 服务端汇总，不能对当前分组结果再次求和。
- 删除无后端契约的环境选择器。

## Acceptance Criteria

- [x] 首屏区块顺序与 Demo 一致，Hero、筛选工作台、容量关注、7 KPI 均可见。
- [x] 四个对象维度支持搜索、多选、清空和 chips，慢请求不会覆盖新筛选。
- [x] 单账号筛选时账号摘要和 KPI 不出现重复累加。
- [x] 平均请求耗时和平均首 Token 使用样本加权口径。
- [x] 桌面、窄屏、深浅色下没有页面级横向溢出。

## Verification Plan

```powershell
cd frontend
pnpm run typecheck
pnpm run lint:check
pnpm run test:run -- src/views/admin/__tests__/ModelObservabilityView.spec.ts src/composables/__tests__/useObservability.spec.ts
pnpm run build
```

人工对照 `dashboard-demo/revision-v2/screenshots/overview-light-wide.png` 检查首屏结构。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Demo shell | `frontend/src/views/admin/ModelObservabilityView.vue` | Hero status, time toolbar, four-tab selector workbench, right scope summary, capacity attention and 7-card KPI restored |
| Average metrics | `backend/internal/service/observability_snapshot.go` | Snapshot emits `avg_duration_ms` and `avg_first_token_ms` from accumulated sums and sample counts |
| Filter interaction | `ModelObservabilityView.vue` | Current-dimension search, select-all, clear-dimension, object cards, chips, high-pressure toggle and canonical filter updates |
| Polling control | `frontend/src/composables/useObservability.ts` | Auto refresh can pause/resume; resume performs an immediate refresh |
| Responsive fix | `PressureRanking.vue`, `PressureHeatmap.vue` | Grid items use `min-w-0`; narrow viewport reports `scrollWidth=382`, `clientWidth=382` |
| Unit tests | `pnpm run test:run -- ModelObservabilityView.spec.ts useObservability.spec.ts PressureHeatmap.spec.ts` | PASS, 10 tests |
| Typecheck | `pnpm run typecheck` | PASS |
| Lint | `pnpm run lint:check` | PASS with one pre-existing `UsageFilters.spec.ts` unused-variable warning |
| Production build | `pnpm run build` | PASS |
| Backend focused | Observability/admin handler/route tests | PASS |
| Visual verification | 1440 dark and 390 narrow screenshots | PASS: Demo-style Hero/workbench/7 KPI; no page-level horizontal overflow |

## Execution Notes

首页信息量较大时优先使用左右工作台，不得退回单列上下平铺。

当前测试数据无请求流量，因此视觉检查中指标为 0，并显示 `request_rollup_unavailable` / `wait_duration_unavailable` 的真实降级状态；布局和交互仍可验证。时间快捷项与自定义时间弹层仍由后续 `MVP-023` 完成。
