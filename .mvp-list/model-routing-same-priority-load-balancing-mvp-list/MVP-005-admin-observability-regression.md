# MVP-005：管理端校验、可观测性与回归验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Dependencies: `MVP-003`、`MVP-004`

## 目标

让管理端能够配置同一 priority 下的多个候选，并在非法跨 priority 复用账号时给出明确提示；每个候选条目仍只允许一个账号。同时让 Gateway 调试日志能够定位 priority 层尝试、层耗尽、账号和 LoadRate。

## 实现范围

- 管理端允许同一路由别名下存在多个相同 priority 的候选；每个候选条目仍只允许一个账号。
- 管理端不再要求历史候选 `model` 字段；保留旧数据读取/序列化兼容，真实上游模型由账号 mapping 决定。
- 管理端校验同一路由中账号是否跨不同 priority 重复，并保留空账号、单候选多账号和非法账号校验。
- `max_concurrency` 为空时，管理端提示按所选账号全局并发计算。
- Gateway 调试日志增加 priority 层尝试/耗尽、路由账号和选中账号 LoadRate；不记录 token 或凭据。
- 完成 backend、frontend 相关回归验证。

## 验收标准

- [x] 跨 priority 重复账号在保存前或后端校验时得到明确错误。
- [x] 同 priority 的多个候选可以保存，单个候选仍只能保存一个账号。
- [x] 编辑候选 priority 后按 priority 升序整理，相同 priority 保持原相对顺序。
- [x] `max_concurrency` 为空时管理端不报非法配置，并提示使用账号全局并发。
- [x] 调试日志包含 group、route alias、priority、account 和 LoadRate 等调度诊断字段。
- [x] 关键调度路径有成功、层降级/耗尽和失败可观测记录。
- [x] backend 相关测试通过。
- [x] frontend 类型检查和相关组件测试通过。

## 验证命令与结果

| 类型 | 命令 | 结果 |
|---|---|---|
| backend 回归 | `cd backend; go test ./internal/domain ./internal/service ./internal/repository` | PASS：domain、service、repository 均通过 |
| frontend 类型检查 | `cd frontend; npm run typecheck` | PASS |
| frontend 路由测试 | `cd frontend; npm run test:run -- src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | PASS：2 个文件、11 个用例 |
| 变更格式检查 | `git diff --check` | PASS |

## 已知验证环境问题

仓库整体 `internal/handler` 测试当前存在与本 MVP 无关的既有构造函数参数数量不匹配问题（`NewGatewayService`、`NewOpenAIGatewayService`）。本 MVP 使用的 backend 核心包回归和 frontend 相关测试均已独立通过。

## 变更证据

- `backend/internal/service/gateway_service.go`
- `frontend/src/views/admin/groupsModelRouting.ts`
- `frontend/src/components/admin/group/GroupModelRoutingEditor.vue`
- `frontend/src/views/admin/GroupsView.vue`
- `frontend/src/views/admin/__tests__/groupsModelRouting.spec.ts`
- `frontend/src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts`

## 执行记录

- 2026-08-17：允许同 priority 多候选，保留候选单账号限制；增加 priority 修改后的排序、跨 priority 账号冲突提示和 priority/LoadRate 调试日志；完成前端回归验证。
