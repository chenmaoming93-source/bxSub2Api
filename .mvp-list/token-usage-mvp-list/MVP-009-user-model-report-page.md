# MVP-009：交付用户模型 Token 统计页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-006`

## 预期成果

管理员可通过独立页面查看一个用户在各模型上的每日 Token 用量。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。路由为 `/admin/token-usage/users`；切换用户后必须清空模型、页码和旧结果。

## 范围内

- 实现 `UserModelTokenUsageView.vue` 和专属表格
- 接入用户搜索、用户内模型搜索、默认目标和分页汇总
- 展示软删除用户和当前用户模型限额
- 覆盖用户切换、模型清空、空错态和分页测试

## 范围外

- 不实现另外两个业务页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 首次只查询一个默认用户并使用今天
- [x] 用户切换不会显示前一用户数据
- [x] 软删除用户状态和历史用量正确展示
- [x] 测试和类型检查通过

## 验证计划

- `cd frontend; pnpm test:run`
- `cd frontend; pnpm typecheck`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `npx vue-tsc --noEmit` | 通过，无错误 |
| 视图 | `views/admin/token-usage/UserModelTokenUsageView.vue` | 用户搜索+模型过滤、默认目标、分页、软删除标记 |
| 表格 | `components/admin/token-usage/UserModelTokenUsageTable.vue` | 7 列字段、软删除用户标记 "(Deleted)" |
| 路由 | `router/index.ts` | `/admin/token-usage/users` 已注册 |
| 用户切换 | watch targetId 变化 | 清空模型、页码和旧结果 |

## 执行记录

执行时记录偏差、阻塞项、索引或接口决策；当前无执行记录。

