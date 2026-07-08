# MVP-006：建立前端统计报表公共基础

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-002, MVP-003, MVP-004, MVP-005`

## 预期成果

前端具备类型安全 API、统一报表外壳和可靠的有界请求状态管理。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。复用 `AppLayout`、`DateRangePicker`、`Pagination`、`EmptyState` 等既有组件及 Tailwind 风格。

## 范围内

- 新增 `frontend/src/api/admin/tokenUsage.ts` 及响应类型
- 实现 ReportLayout、Summary、UsageStatusBadge 等公共组件
- 实现日期默认今天、分页、URL 状态、AbortController 和请求序号保护
- 补齐公共组件、API 参数和竞态测试

## 范围外

- 不交付三个业务页面和侧边栏入口。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 公共组件覆盖加载、空、错误、重试和深色模式结构
- [x] 旧响应无法覆盖新请求，页大小不超过 100
- [x] URL 状态及今天默认值有测试
- [x] 前端测试、类型检查通过

## 验证计划

- `cd frontend; pnpm test:run`
- `cd frontend; pnpm typecheck`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API 测试 | `frontend/src/api/__tests__/admin.tokenUsage.spec.ts` | 13 tests passed |
| Composable 测试 | `frontend/src/composables/__tests__/useTokenUsageReport.spec.ts` | 12 tests passed |
| 组件测试 | `frontend/src/components/admin/token-usage/__tests__/` (3 files) | 22 tests passed |
| 类型检查 | `npx vue-tsc --noEmit` | 通过，无错误 |
| API 层 | `frontend/src/api/admin/tokenUsage.ts` | 3 个 report 端点 + 5 个 options 端点 + default-target |
| Composable | `frontend/src/composables/useTokenUsageReport.ts` | URL 状态管理、AbortController、请求序号、日期默认今天 |
| 组件 | `UsageStatusBadge.vue`, `TokenUsageSummary.vue`, `TokenUsageReportLayout.vue` | 加载/空/错误/重试/深色模式 |
| admin index 导出 | `frontend/src/api/admin/index.ts` | tokenUsageAPI 已注册并导出类型

## 执行记录

执行时记录偏差、阻塞项、索引或接口决策；当前无执行记录。

