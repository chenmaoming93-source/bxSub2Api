# MVP-007：交付场景账号 Token 详情弹窗

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 独立完成一个高频下钻弹窗，包含按需加载、表格和分页。
- Dependencies: `MVP-003, MVP-005, MVP-006`

## 预期成果

点击管理员仪表盘场景排行后，打开居中弹窗显示该场景下账号 Token 数据。

## 背景

现有接口为 `GET /admin/usage/scene-account/daily`，前端函数为 `querySceneAccountDaily()`，返回日期、场景、账号、上游模型和 Token。

## 范围内

- `SceneAccountTokenDialog`。
- 使用 `UiDialog`，不使用抽屉或行内展开。
- 按点击场景传递 `group_name/start_date/end_date`。
- 汇总显示总 Token、账号数、请求数和场景占比。
- 表格显示账号、上游模型、Token、占比和请求数。
- 搜索、刷新、加载、错误、空状态和分页。
- 参考 `assets/final-admin-dashboard-scene-modal.png`。

## 范围外

- 不修改场景排行接口的业务定义。
- 不一次加载全部场景详情。
- 不删除现有场景查询逻辑。

## 实现说明

弹窗打开时按需查询，关闭后清理详情状态；保留当前页面筛选上下文。

## 验收标准

- [x] 点击场景只打开居中弹窗。
- [x] 页面没有账号内嵌展开内容。
- [x] 弹窗显示正确日期范围和场景名称。
- [x] 分页和错误重试可用。
- [x] 弹窗视觉符合指定原型图。

## 验证计划

- `pnpm --dir frontend typecheck`
- 场景点击、取消、错误、分页和重复打开测试。
- 启动项目人工验证弹窗遮罩和滚动锁定。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/features/admin-dashboard/__tests__/scene-account-dialog.test.ts` | 1 个测试通过 |
| Build | `pnpm run build` | 通过 |
| Changed files | `frontend/src/features/admin-dashboard/SceneAccountTokenDialog.vue`; `frontend/src/views/admin/AdminDashboardV2View.vue` | 场景排行点击后按需打开居中详情弹窗 |

## 执行记录

已修正初始加载 watcher，场景详情按 `group_name/start_date/end_date` 请求并在弹窗内分页。