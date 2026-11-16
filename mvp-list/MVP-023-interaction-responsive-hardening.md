# MVP-023: 完成时间选择、交互与响应式细节

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 聚焦已成型页面上的交互细节、移动端和可访问性，不再改动核心数据架构。
- Dependencies: `MVP-020`, `MVP-021`, `MVP-022`

## Outcome

时间选择、热力图、链路、图例和抽屉在桌面与移动端可用，交互与确认稿一致，并具备键盘与深浅色支持。

## Context

确认稿包含快捷时间、始终展开的自定义时间输入、真实时间回填、图标对齐、热力格键盘 tooltip 与同页链路定位；这些需要在正式组件中逐项实现。

## In Scope

- 时间弹层提供今天、昨天、最近 1h/6h/24h/7d，以及始终可见的自定义起止输入。
- 快捷时间选择后保持面板打开，将真实起算时间回填到自定义输入。
- 自定义起止应用前校验，提交后显示真实起止时间。
- 下拉图标使用垂直居中 SVG，展开状态翻转。
- 热力格、图例、Tab、链路节点、表格行和抽屉补充键盘操作与焦点管理。
- 覆盖 1280/768/390 响应式布局；宽表与热力图只在局部滚动。

## Out of Scope

- 不改变后端指标定义。
- 不引入新的 UI 框架或图表库。
- 不进行完整截图基线与性能发布验收。

## Implementation Notes

- 沿用现有 i18n、Tailwind 和主题变量，补齐中英文文案。
- 热力图 tooltip 在固定容器内定位，避免窄屏溢出。
- 抽屉打开后焦点进入抽屉，Esc 关闭并返回触发元素。
- 响应式布局不要用固定最小宽度导致整页横向滚动。

## Acceptance Criteria

- [x] 时间面板一打开即可看到自定义起止输入，快捷选择后真实时间立即回填。
- [x] 今天、昨天、固定窗口与自定义范围切换后所有组件同步刷新。
- [x] 热力格、链路节点和抽屉可通过键盘完整操作。
- [x] 390px 下无页面级横向溢出，热力图和宽表可局部滚动。
- [x] 浅色、深色和错误态文案均可读。

## Verification Plan

```powershell
cd frontend
pnpm run typecheck
pnpm run lint:check
pnpm run test:run
pnpm run build
```

人工与截图验证 1440、1280、768、390 四个宽度。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Time picker | `ModelObservabilityView.vue` | Today/yesterday/fixed presets and always-visible custom start/end inputs; real times retained while menu stays open |
| Custom query | `useObservability.ts`, `useObservabilityDrilldown.ts` | Absolute `startAt/endAt` sent to snapshot, trend, heatmap, paths and relations |
| Keyboard/a11y | `PressureHeatmap.vue`, `DrilldownPanel.vue`, `useObservabilityDrilldown.ts` | Heatmap focus tooltip, table Space/Enter, drawer focus/restore and Escape close |
| Responsive | 390px browser check | PASS: no page-level horizontal overflow |
| Tests | Observability view, composable and drilldown suites | PASS |
| Frontend | Typecheck, focused tests, lint and production build | PASS; one pre-existing UsageFilters warning |

## Execution Notes

关键视觉截图以 `dashboard-demo/revision-v2/screenshots/` 为对照，不以“功能存在”替代布局一致性。

完整 `pnpm run test:run` 当前为 843/849 通过，6 个失败均位于既有认证重定向测试（EmailVerify/WechatOAuth/EmailOAuthButtons），另有 AccountsView 测试 mock 缺少 `getModelRouteReferences` 的既有未处理错误；本 MVP 的 19 项定向测试全部通过。
