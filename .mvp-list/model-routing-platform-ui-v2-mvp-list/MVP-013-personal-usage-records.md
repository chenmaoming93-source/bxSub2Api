# MVP-013：交付新版个人使用记录

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 提取现有个人明细模块并完成 UI V2 迁移，不改变搜索需求。
- Dependencies: `MVP-003, MVP-004`

## 预期成果

`/usage` 只显示个人用量明细和错误请求查询。

## 背景

现有 `frontend/src/views/user/UsageView.vue` 同时包含统计内容和记录查询。参考 `assets/final-admin-usage-prototype.png` 的密集记录页风格，并按个人权限使用个人接口。

## 范围内

- 用量明细和错误请求 Tabs。
- 原筛选条件、查询、重置、分页和详情。
- 使用 V2 FilterBar、DataTable、Pagination 和 Dialog。
- 复用 `/usage`、`/usage/stats`、个人错误请求和详情接口。
- 保持 `allow_user_view_error_requests` 开关行为。

## 范围外

- 本轮不优化搜索。
- 不保留顶部统计卡和其他分析图表。
- 不删除旧个人 Usage 文件。

## 验收标准

- [x] 页面只包含记录查询模块。
- [x] 个人只能看到自己的记录。
- [x] 请求详情和错误详情均为居中弹窗。
- [x] 原筛选、分页和错误开关行为不变。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 认证用户、无权限错误请求和详情弹窗人工验证。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Regression tests | `pnpm run test:run -- src/views/user/__tests__/UsageView.spec.ts` | 5 个测试通过 |
| Route | `frontend/src/router/index.ts` | `/usage` 已指向 `UserUsageV2View.vue` |
| Legacy preservation | `frontend/src/views/user/UsageView.vue` | 保留旧文件，新增 compact/embedded 模式 |
| Error detail | `frontend/src/components/user/UserErrorRequestsTable.vue` | 复用居中 `UserErrorDetailModal` |

## 执行记录

个人使用记录 V2 入口仅显示明细与错误请求查询，保留个人权限接口、筛选、日期范围、分页、导出和错误开关。
