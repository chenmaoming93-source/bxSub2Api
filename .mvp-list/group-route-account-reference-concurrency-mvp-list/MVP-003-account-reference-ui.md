# MVP-003：账号引用关系展示

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 一个查询接口接入账号列表列和编辑卡片底部展示。
- Dependencies: `MVP-001`

## 预期成果

在 `admin/accounts` 表格新增引用关系列，并在编辑账号卡片最下方展示完整引用关系。

## 范围内

- 账号引用查询 API。
- 列表摘要、总数和无引用状态。
- 编辑账号卡片只读详情区域。
- 分组名、路由别名、候选模型、最大并发展示。

## 范围外

- 在账号页面修改并发配置。

## 验收标准

- [x] 列表显示引用摘要或“未被引用”。
- [x] 编辑卡片底部显示完整引用关系。
- [x] 查询直接使用关系表并正确显示 `NULL` 为“不限制”。
- [x] API 和前端类型/组件检查通过。

## 验证计划

- 运行前端 lint/type/test 及后端 API 测试（以仓库命令为准）。
- 手工检查账号有、无、多条引用三种状态。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端 API | `GET /api/v1/admin/accounts/:id/model-route-references` | 已接入账号引用查询 |
| 前端 | `frontend/src/views/admin/AccountsView.vue` | 新增引用列和摘要 |
| 前端 | `frontend/src/components/account/EditAccountModal.vue` | 卡片底部新增只读引用区域 |
| 验证 | `npm run typecheck` | 通过 |

## 执行记录

- 2026-08-10：完成账号列表摘要、编辑账号详情展示和关系查询 API。
