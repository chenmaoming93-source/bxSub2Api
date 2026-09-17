# MVP-007：管理员错误请求显示“上游服务断开”

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.5 个聚焦开发日`
- Estimate rationale: 后端接口已通过 `ops_error_logs` 提供数据，本 MVP 只补充错误类型显示、筛选和详情回归，范围小于标准 MVP。
- Dependencies: `MVP-002`

## 预期成果

`/admin/usage` 的“错误请求”菜单能够将 `error_type=client_disconnected` 显示为“上游服务断开”，支持查询、分页和详情查看，且不改变现有接口返回格式。

## 背景

当前 `frontend/src/views/admin/UsageView.vue` 的错误 Tab 使用 `listErrorLogs` 调用 `/admin/ops/errors`，数据来自 `ops_error_logs`。错误表格和详情组件位于 `frontend/src/views/admin/ops/components/`。

## 范围内

- 增加 `client_disconnected` 的显示标签；
- 在错误类型展示或筛选逻辑中支持该类型；
- 保持错误请求分页、详情弹窗和现有字段兼容；
- 补充组件或映射单元测试；
- 验证 `/admin/usage` 错误请求 Tab 能展示该记录。

## 范围外

- 不新增管理员接口；
- 不修改 `ops_error_logs` schema；
- 不把该记录加入“用量明细”；
- 不修改错误记录的后端存储逻辑；
- 不修改其他错误类型的显示语义。

## 实现说明

- 重点检查 `frontend/src/views/admin/UsageView.vue`、`frontend/src/api/admin/ops.ts`、`OpsErrorLogTable.vue`、`OpsErrorDetailModal.vue` 和现有 i18n 文案；
- 后端接口继续使用 `/api/v1/admin/ops/errors`；
- 显示标签可使用稳定 error type 映射，不改变 API 返回字段；
- 保留 `request_type` 对 sync/stream/ws_v2 的原有展示。

## 验收标准

- [x] `client_disconnected` 显示为“上游服务断开”；
- [x] 错误请求列表能够展示该类型记录；
- [x] 现有分页、筛选和错误详情弹窗正常；
- [x] 该记录不会出现在用量明细列表；
- [x] 其他 Ops 错误类型的显示和筛选不回归。

## 验证计划

- `pnpm test:run -- src/components/admin/usage src/views/admin`
- `pnpm typecheck`
- 如测试目录不匹配，使用 `pnpm test:run` 验证相关组件测试，并人工访问 `/admin/usage` 的“错误请求”Tab。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 组件专项测试 | `pnpm test:run -- src/views/admin/ops/components/__tests__/OpsErrorLogTable.spec.ts` | 通过，5 tests；覆盖 `client_disconnected` 标签、用户/API Key/账号列及 i18n key。 |
| 类型检查 | `pnpm typecheck` | 通过。 |
| 实现 | `OpsErrorLogTable.vue`、`zh.ts`、`en.ts` | `log.type === client_disconnected` 显示“上游服务断开”/英文对应文案；原有类型映射保持不变。 |
| 用量隔离 | `UsageView.vue` / `listErrorLogs` 现有错误 Tab | 未新增用量接口或字段，记录仍只由错误请求列表消费。 |

## 执行记录

- 2026-09-16：增加 client_disconnected 稳定类型标签与中英文文案，并补充组件回归测试。

