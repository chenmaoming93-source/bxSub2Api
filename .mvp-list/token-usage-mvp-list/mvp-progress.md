# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../Token消耗统计页面实施Plan.md`
- Target effort per MVP: `假设为 1 个专注开发日`
- Update batch size: `1 个已完成 MVP`
- Last updated: `2026-07-06T17:40:00+08:00`
- Overall: `10/10 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 在批量更新之间，记录进度可以落后于已验证工作，但绝不能超前。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-query-contract-and-index-audit.md](./MVP-001-query-contract-and-index-audit.md) | DONE | none | 1 个专注开发日 | 2026-07-06T16:12:00+08:00 | [Repository、EXPLAIN 与索引证据](./MVP-001-query-contract-and-index-audit.md#完成证据) |
| MVP-002 | [MVP-002-global-model-report-api.md](./MVP-002-global-model-report-api.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-06T16:24:00+08:00 | [API、权限与测试证据](./MVP-002-global-model-report-api.md#完成证据) |
| MVP-003 | [MVP-003-route-model-report-api.md](./MVP-003-route-model-report-api.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-06T16:38:00+08:00 | [路由查询与历史兼容证据](./MVP-003-route-model-report-api.md#完成证据) |
| MVP-004 | [MVP-004-user-model-report-api.md](./MVP-004-user-model-report-api.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-06T16:39:00+08:00 | [用户查询与软删除证据](./MVP-004-user-model-report-api.md#完成证据) |
| MVP-005 | [MVP-005-bounded-options-default-targets.md](./MVP-005-bounded-options-default-targets.md) | DONE | MVP-002, MVP-003, MVP-004 | 1 个专注开发日 | 2026-07-06T17:20:00+08:00 | [选项上限、默认目标与测试证据](./MVP-005-bounded-options-default-targets.md#完成证据) |
| MVP-006 | [MVP-006-frontend-report-foundation.md](./MVP-006-frontend-report-foundation.md) | DONE | MVP-002, MVP-003, MVP-004, MVP-005 | 1 个专注开发日 | 2026-07-06T17:27:00+08:00 | [API、Composable、组件与测试证据](./MVP-006-frontend-report-foundation.md#完成证据) |
| MVP-007 | [MVP-007-global-model-report-page.md](./MVP-007-global-model-report-page.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-06T17:33:00+08:00 | [视图、表格与竞态保护证据](./MVP-007-global-model-report-page.md#完成证据) |
| MVP-008 | [MVP-008-route-model-report-page.md](./MVP-008-route-model-report-page.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-06T17:33:00+08:00 | [三级联动与级联清空证据](./MVP-008-route-model-report-page.md#完成证据) |
| MVP-009 | [MVP-009-user-model-report-page.md](./MVP-009-user-model-report-page.md) | DONE | MVP-006 | 1 个专注开发日 | 2026-07-06T17:33:00+08:00 | [用户搜索、软删除与切换保护证据](./MVP-009-user-model-report-page.md#完成证据) |
| MVP-010 | [MVP-010-navigation-performance-regression.md](./MVP-010-navigation-performance-regression.md) | DONE | MVP-007, MVP-008, MVP-009 | 1 个专注开发日 | 2026-07-06T17:40:00+08:00 | [全量测试、导航、i18n与性能审查证据](./MVP-010-navigation-performance-regression.md#完成证据) |
## 依赖说明

- 关键路径：MVP-001 → MVP-002/003/004 → MVP-005 → MVP-006 → MVP-007/008/009 → MVP-010。
- MVP-002、003、004 可并行；MVP-007、008、009 可并行。

## 规划假设

- 每个 MVP 目标工作量为一个专注开发日，允许约 0.5 至 1.5 日浮动。
- 复用现有六张配额/用量表，不新增业务数据表；索引只在 EXPLAIN 证明必要时增加。
- 后端测试命令假设使用 Go 1.26.4；前端使用仓库现有 pnpm 脚本。
- 最大查询日期范围由性能验收决定，不阻塞前序切片。
