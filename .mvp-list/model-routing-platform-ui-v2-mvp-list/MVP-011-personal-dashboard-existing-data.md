# MVP-011：迁移个人仪表盘既有统计能力

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 只迁移已有个人统计、模型数据和统一筛选，新增接口另行拆分。
- Dependencies: `MVP-004`

## 预期成果

个人仪表盘接入 UI V2，并展示现有接口已经支持的个人指标和模型 Token 排行。

## 背景

现有接口在 `frontend/src/api/usage.ts`，包括 `/usage/dashboard/stats`、`/usage/stats`、`/usage/dashboard/models` 和 `/usage/dashboard/trend`。参考 `assets/final-personal-dashboard-prototype-v2.png`。

## 范围内

- 统一日期范围、粒度和刷新。
- 当前用户 API Key、范围请求、范围 Token、累计 Token、平均耗时和累计费用。
- 模型 Token 横向排行榜。
- 复用认证状态、用户信息和现有数据加载骨架。
- 使用 V2 AppShell、StatCard、ChartCard 和 Ranking。

## 范围外

- 个人场景接口。
- 个人耗时趋势和百分位接口。
- 个人使用记录。

## 验收标准

- [x] 既有个人指标显示正确。
- [x] 日期范围正确传递给范围接口。
- [x] 模型排行按 Token 展示。
- [x] 旧个人 Dashboard 文件仍存在。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 使用测试数据验证日期切换和模型排行。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Regression tests | `pnpm run test:run -- src/views/user/__tests__/UsageView.spec.ts src/components/ui-v2/__tests__/shell-charts.test.ts` | 8 个测试通过 |
| Build | `pnpm run build` | 通过；生成 `UserDashboardV2View` chunk |
| Route | `frontend/src/router/index.ts` | `/dashboard` 已指向 `UserDashboardV2View.vue` |
| Legacy preservation | `frontend/src/views/user/DashboardView.vue` | 旧个人 Dashboard 保留 |

## 执行记录

新增个人仪表盘 V2 入口，复用现有个人 stats/trend/models API，统一日期范围、粒度、刷新、指标卡和模型 Token 横向排行。个人场景及耗时趋势/百分位待 MVP-012 接口完成后接入。
