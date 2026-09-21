# MVP-010：交付新版管理员使用记录

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 从旧页面提取已有记录模块，保持搜索契约并完成独立 UI 迁移。
- Dependencies: `MVP-003, MVP-004`

## 预期成果

`/admin/usage` 只显示用量明细和错误请求查询，保持原有搜索、分页、导出和详情行为。

## 背景

旧 `frontend/src/views/admin/UsageView.vue` 同时承载统计卡、图表、部门/场景统计和明细。新页面只保留底部查询部分，参考 `assets/final-admin-usage-prototype.png`。

## 范围内

- 用量明细和错误请求 Tabs。
- `UsageFilters`、`UsageTable`、分页、导出、列设置。
- 管理员请求详情和错误详情弹窗。
- 复用 `/admin/usage`、`/admin/ops/request-errors` 等现有接口。
- 使用 V2 FilterBar、DataTable、Dialog 和页面框架。

## 范围外

- 不优化搜索条件。
- 不显示统计卡、趋势图、模型/分组/部门/场景统计。
- 不删除旧 `UsageView.vue`。

## 验收标准

- [x] 页面除明细和错误查询外没有其他统计内容。
- [x] 原搜索参数、分页、导出和列设置行为保持。
- [x] 请求和错误详情均为居中弹窗。
- [x] 旧页面源文件仍存在。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 使用 fixture 验证筛选、分页、导出和详情。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Regression tests | `pnpm run test:run -- src/views/admin/__tests__/UsageView.spec.ts src/components/admin/usage/__tests__/UsageFilters.spec.ts src/components/admin/usage/__tests__/UsageTable.spec.ts` | 20 个测试通过 |
| Build | `pnpm run build` | 通过；生成 `AdminUsageV2View` chunk |
| Route | `frontend/src/router/index.ts` | `/admin/usage` 已指向 `AdminUsageV2View.vue` |
| Legacy preservation | `frontend/src/views/admin/UsageView.vue` | 保留旧页面，新增 `compact/embedded` 模式供 V2 入口复用记录功能 |

## 执行记录

新增精简管理员使用记录入口：仅展示用量明细、错误请求、筛选、分页、导出、列设置及居中详情；隐藏旧页面统计卡、图表、部门和场景统计。
