# MVP-004：报表兼容性与统一回归验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.5 个专注开发日`
- Estimate rationale: `不新增功能，集中验证三个已完成查询维度的 API 契约、前端兼容性和回归测试。`
- Dependencies: `MVP-002, MVP-003`

## 预期成果

三个 Token 用量报表在零用量补全后保持现有 API 与前端兼容，相关后端测试完整通过，并确认无数据库结构和入库链路变更。

## 背景

补零改变了报表结果集的行数、排序和分页行为，需要统一确认 Service、Handler、默认目标及前端表格能够处理 `used_tokens=0` 的数据。

## 范围内

- 执行 Repository、Service、Admin Handler 相关测试。
- 检查 API 响应结构、分页元数据和汇总字段没有变化。
- 执行前端类型检查及 Token 用量页面相关测试。
- 审查最终差异，确认不存在 Schema、迁移、入库、计费和限额拦截改动。

## 范围外

- 新增报表功能或 UI 重设计。
- 性能压测平台建设。
- 配置历史追踪。

## 实现说明

- 如果回归测试暴露本次查询改造造成的问题，可在相关 Repository、Service、Handler 测试或前端兼容层内修复。
- 不得借此 MVP 扩大到数据入库或数据库结构改造。

## 验收标准

- [x] 三个报表接口请求和响应字段保持兼容。
- [x] 后端相关 Repository、Service、Handler 测试通过。
- [x] 前端类型检查及 Token 用量页面相关测试通过。
- [x] 差异检查确认没有新增表、字段、迁移或用量入库修改。
- [x] 最终实现覆盖 Plan 中 AC-01 至 AC-06。

## 验证计划

- `cd backend; go test ./internal/repository/... ./internal/service/... ./internal/handler/admin/...`
- `cd frontend; pnpm typecheck`
- `cd frontend; pnpm exec vitest run src/views/admin/token-usage/__tests__ src/components/admin/token-usage/__tests__ src/composables/__tests__/useTokenUsageReport.spec.ts src/api/__tests__/admin.tokenUsage.spec.ts`
- `git diff --check`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端回归 | `cd backend; go test ./internal/repository/... ./internal/service/... ./internal/handler/admin/...` | 通过：Repository、Service（含 `openai_ws_v2`）和 Admin Handler 全部成功。 |
| 前端类型 | `cd frontend; pnpm typecheck` | 通过：`vue-tsc --noEmit` 退出码 0。 |
| 前端测试 | `cd frontend; pnpm exec vitest run src/views/admin/token-usage/__tests__ src/components/admin/token-usage/__tests__ src/composables/__tests__/useTokenUsageReport.spec.ts src/api/__tests__/admin.tokenUsage.spec.ts` | 通过：6 个测试文件、51 项测试全部成功；存在既有 Vue 警告但无失败。 |
| 契约检查 | `backend/internal/service/token_usage_report_service.go` 及上述回归测试 | 三个响应继续使用 `Items`、`UsedTokens`、`Page`、`PageSize`、`Total`，请求/响应结构未改。 |
| 差异检查 | `git diff --check` 及执行路径审查 | 通过；本次执行仅修改 Repository、Repository 测试和 MVP 文档，没有修改 Schema、迁移、用量入库、计费或限额拦截。工作树中原有迁移相关改动保持不动。 |

## 执行记录

- `2026-07-07T15:10:30+08:00`：完成三个报表维度及前后端统一回归，确认 AC-01 至 AC-06 均有实现和测试证据。
