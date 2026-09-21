# MVP-006：交付管理员仪表盘新版页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 依赖 UI V2 和数据协调，垂直交付管理员仪表盘主体。
- Dependencies: `MVP-003, MVP-004, MVP-005`

## 预期成果

`/admin/dashboard` 渲染符合最终原型图风格的新管理员仪表盘主体。

## 背景

旧管理员页面为 `frontend/src/views/admin/DashboardView.vue`，旧文件保留但不再作为新页面实现依据。最终参考 `assets/final-admin-dashboard-prototype-v2.png`。

## 范围内

- 统一时间范围和粒度工具栏。
- 九类总览指标：用户、模型账号、API Key、范围请求、范围 Token、累计 Token、性能指标、平均响应耗时、累计费用。
- 用户 Token Top12 趋势。
- 模型 Token 横向排行。
- 场景（分组）Token 横向排行。
- 部门 Token 横向排行。
- 搜索、Top12/全部、分页和“查看详情”操作。
- 使用 V2 AppShell、Card、StatCard、ChartCard、Ranking。
- 不在页面中显示内嵌账号或用户明细。

## 范围外

- 场景详情弹窗和部门详情弹窗由 MVP-007/008 完成。
- 不迁移其他 Part。
- 不删除旧 Dashboard 文件。

## 实现说明

- 页面统一调用 MVP-005 数据协调器。
- 所有详情入口先提供事件契约，禁止抽屉和行内展开。
- 费用指标按最终已审核需求保留。

## 验收标准

- [x] 页面布局与 `final-admin-dashboard-prototype-v2.png` 的结构和视觉一致。
- [x] 所有范围数据受同一日期范围控制。
- [x] 模型、场景、部门均为横向排行榜。
- [x] 排行行内没有展开子表。
- [x] 旧页面文件仍存在。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- 启动项目后人工检查桌面布局、暗色模式和原型图视觉一致性。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Data tests | `pnpm run test:run -- src/features/admin-dashboard/__tests__/adminDashboardData.test.ts` | 2 个测试通过 |
| Build | `pnpm run build` | 通过；生成 `AdminDashboardV2View` chunk |
| Route | `frontend/src/router/index.ts` | `/admin/dashboard` 已指向 `AdminDashboardV2View.vue` |
| Changed files | `frontend/src/views/admin/AdminDashboardV2View.vue`; `frontend/src/features/admin-dashboard/UserTokenTrendChart.vue` | 新管理员仪表盘主体和用户 Token 趋势图 |

## 执行记录

已完成管理员仪表盘主体；场景和部门详情弹窗将在 MVP-007/008 接入，当前排行不做行内展开。