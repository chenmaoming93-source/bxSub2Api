# MVP-006：在管理员页面提供维度化限额重置界面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: `需要扩展 TypeScript API 契约、动态维度输入、增强选择器复用、确认交互、四种结果展示和前端测试，略高于一个开发日但可形成完整用户功能。`
- Dependencies: `MVP-004`

## 预期成果

具有 `token_quota.update` 权限的管理员能够在 `/admin/token-statistics` 页面选择周期、指标及任意具体维度值，确认后执行重置，并清楚看到完全成功、部分成功、无限额或无用量结果。

## 背景

- API 客户端位于 `frontend/src/api/admin/dynamicTokenStatistics.ts`。
- 页面位于 `frontend/src/views/admin/TokenStatisticsView.vue`，已经具备动态维度列表、限额表单及部分增强维度选择器。
- 页面测试位于 `frontend/src/views/admin/__tests__/TokenStatisticsView.spec.ts`，API 测试位于 `frontend/src/api/admin/__tests__/dynamicTokenStatistics.spec.ts`。
- 入口仅对 `token_quota.update` 显示，服务端权限仍是最终边界。

## 范围内

- 增加 `QuotaResetInput`、`QuotaResetResult` 和状态类型。
- 增加管理员 reset API 客户端方法。
- 在 Token Statistics 页面增加清晰的“重置限额用量”区域或弹窗。
- 从 `/dimensions` 与 `/metrics` 动态生成可选项；指标只展示 `allow_quota=true` 项。
- 为 int64/string 维度提供正确输入，尽可能复用用户、分组、模型等现有增强选择器。
- 请求中只发送被选择的具体维度，不允许 wildcard 或空值。
- 增加二次确认，明确“真实统计不会清零，只重置额度判定起点”。
- 展示成功数、失败数、无用量数，以及 `RESET`、`PARTIAL_RESET`、`NO_QUOTA`、`NO_USAGE` 文案。
- 增加 API 和页面测试，并保持移动端/暗色模式基本可用。

## 范围外

- 不提供外部 integrations 接口 UI。
- 不展示或编辑 Redis Key、projection hash、shard、baseline。
- 不增加复杂审批流程或持久化操作历史页面。
- 不改变现有统计查询结果，使其展示 effective usage。

## 实现说明

- 可以在现有 quotas tab 增加独立卡片或操作弹窗，但不得把重置误设计为删除/修改限额规则。
- 页面应明确区分“真实累计用量”和“重置后的限额判定用量”。
- 部分成功不是全局错误；应展示已成功数量并提示可再次操作。
- 表单提交期间禁用重复点击即可，不引入业务幂等。

## 验收标准

- [x] 只有具有 `token_quota.update` 权限的用户能看到并操作重置入口。
- [x] 维度和指标选项来自后端注册表，不硬编码未来扩展项。
- [x] 至少选择一个维度，所有已选维度必须填写具体合法值。
- [x] 日、周、月可以选择，默认指标为 `total_tokens`。
- [x] 确认提示明确说明真实统计和报表不会被清零。
- [x] API 请求路径和 payload 与 MVP-004 契约一致。
- [x] 四种业务状态及失败计数均有明确中文展示。
- [x] 部分成功提示允许管理员再次点击重置。
- [x] 现有投影、限额 CRUD 和统计查询页面测试继续通过。

## 验证计划

- `cd frontend && pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- 人工验证桌面端、窄屏和暗色模式下的表单、确认弹窗及四种结果展示。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API | `frontend/src/api/admin/dynamicTokenStatistics.ts` | 新增 reset 输入/结果/状态类型及 `/quota-usage/reset` 方法。 |
| 页面 | `frontend/src/views/admin/TokenStatisticsView.vue` | 权限门控、注册表驱动维度/指标、D/W/M、具体值校验、确认和四状态中文汇总。 |
| 测试 | `pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts` | 2 files、15 tests 全部通过。 |
| 类型 | `pnpm typecheck` | 通过。 |
| 构建 | `pnpm build` | 通过（仅既有 chunk/dynamic import/Browserslist 警告）。 |
| UI 检查 | 响应式 `sm/lg/xl` grid 与 `dark:` 状态类、禁用态和部分成功重试文案 | 代码检查及生产构建通过；未启动替代服务器。 |

## 执行记录

2026-09-09 完成。重置区域复用现有卡片、输入和暗色样式；所有注册维度均有与 `value_type` 对应的通用输入，不提供 wildcard。提交成功后保留选择和值，尤其在 `PARTIAL_RESET` 时允许管理员直接再次执行。未修改用户已有的 DepartmentUserUsageChart 相关改动。
