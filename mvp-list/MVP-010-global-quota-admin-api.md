# MVP-010: 提供全局模型每日限额管理 API

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 只新增列表与全量替换/更新端点及定向 handler 测试。
- Dependencies: `MVP-004`, `MVP-007`, `MVP-009`

## Outcome

管理员可查询和设置实际上游模型的每日 Token 限额，更新后缓存立即失效。

## Context

路由注册位于 `backend/internal/server/routes/admin.go`；handler 风格参考 platform-quotas 管理接口。

## In Scope

- 新增 service 方法、admin handler、DTO 和 routes。
- 校验模型名、非负整数及 0/null 语义。
- 配置更新后清理对应缓存。

## Out of Scope

- 前端弹窗和用量展示图表。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 非管理员无法调用端点。
- [ ] GET 返回模型、限额、当日 used_tokens。
- [ ] PUT 后再次 GET 和缓存读取均返回新值。

## Verification Plan

- `cd backend; go test ./internal/handler/admin ./internal/server -run 'ModelTokenQuota'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


