# MVP-011：交付管理员通用多维查询页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `动态查询表单、汇总、趋势、排行榜、明细和导出可作为独立用户可见成果完成。`
- Dependencies: `MVP-009, MVP-010`

## 预期成果

管理员可以在同一页面选择任意已发布投影，动态筛选和分组数据，并查看汇总、趋势、排行榜、分页明细和CSV。

## 背景

页面只能呈现已经采集的投影和指标，必须清楚显示数据完整性与同步时间。

## 范围内

- 多维查询页签。
- 投影和指标选择。
- 动态维度筛选和group_by。
- 日期范围与日/周/月粒度。
- 汇总卡片、趋势图、排行榜、分页表。
- CSV导出。
- 空状态、错误状态、同步延迟和数据不完整提示。
- 前端测试。

## 范围外

- 用户自助查询页面。
- 跨投影数据拼接。

## 实现说明

- 复用项目公共组件时不得修改其通用语义。
- 查询条件应支持URL或本地状态恢复，但不得暴露敏感值。

## 验收标准

- [x] 一个页面覆盖所有已发布投影。
- [x] 筛选和分组项随投影动态变化。
- [x] 不允许选择投影外维度或指标。
- [x] 汇总、趋势、排行榜和明细正确渲染。
- [x] 显示统计开始时间、完整性和最后同步时间。
- [x] 仅管理员且RBAC路由保护正确。
- [x] 页面测试、类型检查和构建通过。

## 验证计划

- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 通用查询页签 | `frontend/src/views/admin/TokenStatisticsView.vue` | 同一页面覆盖所有 ACTIVE/DISABLED 已采集投影；维度筛选和分组由所选投影动态生成，并展示汇总、趋势、排行榜、分页明细、空/错/不完整状态 |
| 查询 API 与导出 | `frontend/src/api/admin/dynamicTokenStatistics.ts` | 新增强类型 query 和 CSV blob 下载，均调用通用 `/admin/token-statistics/query` |
| 状态恢复与权限 | `TokenStatisticsView.vue`、`frontend/src/rbac/permissionMatrix.ts` | 仅在 localStorage 保存非敏感投影 ID；路由要求 `token_usage.read`，不提供用户入口 |
| 定向测试 | `cd frontend && pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts src/rbac/permissionMatrix.spec.ts` | 3 个测试文件、9 项测试全部通过 |
| 类型与构建 | `cd frontend && pnpm typecheck && pnpm build` | 均通过；仅有仓库既有 chunk size/dynamic import 警告 |
| 全量测试说明 | `cd frontend && pnpm test:run` | 已于 MVP-010 执行；本功能新增用例通过，6 项既有 OAuth/EmailVerify 跳转断言失败，与查询页面无关 |

## 执行记录

2026-07-30：完成同页通用多维查询。前端选项来自后端注册表和投影，后端再次执行白名单校验，不支持跨投影拼接。
