# MVP-011: 提供用户模型每日限额管理 API

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 复用现有用户 platform quota 的路由与 DTO 习惯，限定两个端点。
- Dependencies: `MVP-005`, `MVP-007`, `MVP-009`

## Outcome

管理员可按用户读取和替换模型每日 Token 限额，且不改动其他用户。

## Context

参考 `backend/internal/handler/admin/user_handler.go` 和 `/admin/users/:id/platform-quotas`。

## In Scope

- 新增 GET/PUT user model quota routes、handler 和 service 方法。
- 实现全量替换或明确的 upsert 语义。
- 失效目标用户/模型缓存并加权限、隔离测试。

## Out of Scope

- 普通用户自助配置、前端弹窗。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] GET 只返回目标用户记录。
- [ ] PUT 不影响未指定用户。
- [ ] 非法用户、模型或限额得到稳定 4xx。

## Verification Plan

- `cd backend; go test ./internal/handler/admin ./internal/server -run 'UserModelTokenQuota'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


