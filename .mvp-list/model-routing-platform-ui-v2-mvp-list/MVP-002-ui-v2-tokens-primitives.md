# MVP-002：实现 UI V2 设计变量与基础组件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦设计变量、按钮、输入、选择器和开关等基础控件，范围适合一个独立垂直切片。
- Dependencies: `MVP-001`

## 预期成果

新增可独立使用的 UI V2 基础控件，并与指定原型图的颜色、圆角、间距和暗色模式一致。

## 背景

旧组件位于 `frontend/src/components/common/`，现有 Tailwind 主题位于 `frontend/tailwind.config.js`。旧组件必须保留，新组件放入 `frontend/src/components/ui-v2/`。

## 范围内

- Design Tokens、颜色、字体、圆角、阴影和间距变量。
- `UiButton`、`UiIconButton`。
- `UiInput`、`UiSearchInput`、`UiTextArea`。
- `UiSelect`、`UiCheckbox`、`UiRadio`、`UiSwitch`。
- `UiDatePicker`、`UiDateRangePicker`、`UiFormField`、`UiTabs`。
- 兼容旧组件主要 props、events 和 v-model 约定。
- 亮色、暗色和禁用/加载状态。

## 范围外

- 不删除或改写旧组件。
- 不迁移业务页面。
- 不实现图表、表格和弹窗。

## 实现说明

- 新组件不直接请求 API。
- 视觉基线参考 `assets/final-admin-dashboard-prototype-v2.png` 和 `assets/final-personal-dashboard-prototype-v2.png`。

## 验收标准

- [x] 基础组件均在 `components/ui-v2/` 下可导入。
- [x] 组件具备正常、禁用、加载和暗色状态。
- [x] DateRange、Select、Switch 的核心事件契约可用于后续页面。
- [x] 旧组件文件内容未删除。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 编写并运行基础组件单元测试。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/components/ui-v2/__tests__/primitives.test.ts` | 3 个测试通过 |
| Build | `pnpm run build` | 通过；首次受 esbuild spawn 边界和超时影响，延长超时复核后通过 |
| Changed files | `frontend/src/components/ui-v2/`、`frontend/src/main.ts` | 新增 V2 基础组件和样式入口；旧组件未修改 |

## 执行记录

已完成 UI V2 基础控件、样式变量、导出入口和基础单元测试。