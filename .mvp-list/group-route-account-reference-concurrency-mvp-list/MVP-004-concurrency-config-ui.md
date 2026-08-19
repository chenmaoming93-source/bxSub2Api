# MVP-004：候选最大并发配置界面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 分组候选编辑区增加字段并接入可空配置保存。
- Dependencies: `MVP-001`

## 预期成果

在编辑分组卡片的路由别名候选区域、优先级输入框旁增加最大并发输入框。

## 范围内

- 可空 `max_concurrency` 的 API/DTO/校验。
- 分组候选区域输入、回显和保存。
- 留空保存为 `NULL`，`0` 拒绝保存。
- 保存后刷新 Redis 配置的接口契约（实际 Redis 由 MVP-005 完成）。

## 范围外

- 请求级并发计数。

## 验收标准

- [x] 优先级旁显示最大并发输入框。
- [x] 正整数可以保存和回显。
- [x] 留空保存为 `NULL`。
- [x] `0`、负数和非数字被拒绝。
- [x] 只允许修改真实存在的候选关系。

## 验证计划

- 运行前后端相关测试。
- 手工验证新增、修改、清空和非法输入。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端 API | `PUT /api/v1/admin/groups/:id/model-route-references/concurrency` | 已支持可空并发配置 |
| 前端 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue` | 优先级旁新增最大并发输入框 |
| 验证 | `npm run typecheck`、后端 admin/repository 测试 | 通过 |

## 执行记录

- 2026-08-10：完成候选并发输入、回显和保存接口。
