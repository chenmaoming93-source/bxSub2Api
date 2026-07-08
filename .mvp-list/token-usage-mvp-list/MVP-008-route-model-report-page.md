# MVP-008：交付模型路由 Token 统计页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-006`

## 预期成果

管理员可通过独立页面查看一个分组路由别名下的候选模型用量。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。路由为 `/admin/token-usage/routes`；未选分组不得加载路由，未选别名不得查询统计。

## 范围内

- 实现 `RouteTokenUsageView.vue` 和专属表格
- 接入分组、路由别名、候选模型级联搜索
- 实现级联清空、页码重置、默认目标和当前配置提示
- 覆盖级联、历史配置缺失和分页测试

## 范围外

- 不实现用户或全局模型页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 首次只查询一个默认路由目标
- [x] 分组和别名变化正确清理下游状态及旧数据
- [x] 历史候选仍显示用量且当前配置缺失可辨识
- [x] 测试和类型检查通过

## 验证计划

- `cd frontend; pnpm test:run`
- `cd frontend; pnpm typecheck`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `npx vue-tsc --noEmit` | 通过，无错误 |
| 视图 | `views/admin/token-usage/RouteTokenUsageView.vue` | 三级联动搜索、级联清空、默认目标、分页、优先级显示 |
| 表格 | `components/admin/token-usage/RouteTokenUsageTable.vue` | 9 列字段、缺失配置项显示 "—" |
| 路由 | `router/index.ts` | `/admin/token-usage/routes` 已注册 |
| 级联逻辑 | 分组→路由→模型三级联动 | 变化时清空下游状态和旧数据 |

## 执行记录

执行时记录偏差、阻塞项、索引或接口决策；当前无执行记录。

