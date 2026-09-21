# MVP-004：实现 UI V2 页面框架与图表外壳

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦页面壳层和图表统一主题，为管理员仪表盘提供完整视觉基础。
- Dependencies: `MVP-002`

## 预期成果

新增与原型图风格一致的应用壳层、侧栏、页面标题、卡片和统一 Chart.js 图表外壳。

## 背景

现有布局位于 `frontend/src/components/layout/`，图表位于 `frontend/src/components/charts/`。新组件不能直接替换旧组件实现。

## 范围内

- `UiAppShell`、`UiSidebar`、`UiTopbar`、`UiPageHeader`、`UiPageContainer`。
- `UiSection`、`UiToolbar`、`UiResponsiveGrid`。
- `UiCard`、`UiStatCard`、`UiChartCard`。
- `UiLineChart`、`UiMultiLineChart`、`UiAreaChart`。
- `UiHorizontalBarChart`、`UiPercentileChart`、`UiHistogram`。
- 统一 Chart.js 色板、字体、Tooltip、图例、网格和空状态。
- 暗色模式和常用桌面宽度适配。

## 范围外

- 不迁移具体业务页面。
- 不修改旧布局和旧图表组件。
- 不实现业务 API 调用。

## 实现说明

- 视觉参考六张最终原型图，重点参考管理员仪表盘、运维监控和个人仪表盘。
- 图表组件只接收数据和展示参数，不直接调用接口。

## 验收标准

- [x] V2 页面壳层能渲染侧栏、顶部栏和内容区。
- [x] 图表外壳统一处理标题、操作区、加载、空数据和错误状态。
- [x] 图表支持暗色模式和响应式宽度。
- [x] 旧布局和图表源文件未删除。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 建立静态示例页与六张原型图进行人工视觉对照。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/components/ui-v2/__tests__/shell-charts.test.ts` | 2 个测试通过 |
| Build | `pnpm run build` | 通过；保留既有 chunk size 和 dynamic import 警告 |
| Changed files | `frontend/src/components/ui-v2/layout/`、`frontend/src/components/ui-v2/charts/`、`frontend/src/components/ui-v2/styles.css` | 新增 V2 页面框架、图表外壳和主题 |

## 执行记录

已完成 UI V2 壳层和图表外壳；未修改旧布局与旧图表实现。