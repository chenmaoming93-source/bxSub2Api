# MVP-007：交付全局模型 Token 统计页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-006`

## 预期成果

管理员可通过独立页面筛选并查看一个全局模型的每日 Token 统计。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。页面路由为 `/admin/token-usage/models`；缺少目标时不得调用统计接口。

## 范围内

- 实现 `ModelTokenUsageView.vue` 和专属表格
- 接入模型搜索、默认目标、日期、分页、排序和汇总
- 展示不限额、使用率及状态
- 覆盖首次加载、查询、空错态和竞态测试

## 范围外

- 不添加最终侧边栏分组，不实现另外两个页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 首次仅加载一个默认模型且默认日期今天
- [x] 管理员可搜索模型并获得分页结果
- [x] 表格字段和限额状态符合 Plan
- [x] 测试和类型检查通过

## 验证计划

- `cd frontend; pnpm test:run`
- `cd frontend; pnpm typecheck`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `npx vue-tsc --noEmit` | 通过，无错误 |
| 视图 | `views/admin/token-usage/ModelTokenUsageView.vue` | 搜索、默认目标、日期、分页、汇总、状态完整性 |
| 表格 | `components/admin/token-usage/ModelTokenUsageTable.vue` | 6 列字段、限额与状态渲染 |
| 路由 | `router/index.ts` | `/admin/token-usage/models` 已注册 |
| 竞态保护 | `useTokenUsageReport` composable | 请求序号 AbortController 双重保护 |

## 执行记录

执行时记录偏差、阻塞项、索引或接口决策；当前无执行记录。

