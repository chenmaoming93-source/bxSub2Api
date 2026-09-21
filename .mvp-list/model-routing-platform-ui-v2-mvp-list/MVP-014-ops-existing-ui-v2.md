# MVP-014：迁移运维监控既有页面到 UI V2

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 复用现有 Ops 页面和接口，仅完成页面壳层与已有组件视觉迁移。
- Dependencies: `MVP-003, MVP-004`

## 预期成果

在不改变运维业务逻辑的情况下，将 `/admin/ops` 既有能力迁移到 UI V2。

## 背景

现有页面为 `frontend/src/views/admin/ops/OpsDashboard.vue`，组件覆盖并发、吞吐、延迟、错误、告警和系统日志。参考 `assets/final-ops-prototype.png`。

## 范围内

- 迁移 `OpsDashboardHeader`、并发、吞吐、切换率、延迟、错误、告警、日志和详情组件。
- 保留时间、平台、分组、自动刷新、全屏和查询模式。
- 复用既有 Ops API。
- 使用 V2 Card、Toolbar、ChartCard、Badge、Dialog。
- 保持暗色模式和响应式。

## 范围外

- 按模型耗时后端接口。
- 模型耗时新图表。
- 修改运维统计口径。

## 验收标准

- [x] 原 Ops 模块均仍可显示和刷新。
- [x] 自动刷新、全屏、筛选和告警行为不变。
- [x] 请求和错误详情仍使用居中弹窗。
- [x] 页面风格符合运维原型图。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Ops tests | `pnpm run test:run -- src/views/admin/ops` | 19 个测试通过 |
| Route | `frontend/src/router/index.ts` | `/admin/ops` 已指向 `AdminOpsV2View.vue` |
| Legacy preservation | `frontend/src/views/admin/ops/OpsDashboard.vue` | 保留原组件，新增 embedded 模式 |

## 执行记录

新增 V2 运维入口壳层，复用原并发、吞吐、切换率、延迟、错误、告警、日志、自动刷新、全屏和居中详情弹窗能力。
