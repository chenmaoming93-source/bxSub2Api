# MVP-003: 让分组持久化与 Admin API 往返新旧路由配置

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 限定在 Group schema、repository、DTO/handler 的 JSON 往返，不进入调度。
- Dependencies: `MVP-002`

## Outcome

管理员读取和保存分组时，新格式候选不丢字段，旧格式仍可读取。

## Context

涉及 `backend/ent/schema/group.go`、`backend/internal/repository/group_repo.go`、`backend/internal/handler/dto/types.go` 和 `backend/internal/handler/admin/group_handler.go`。

## In Scope

- 将 model_routing JSON 的 Go 类型改为兼容载体。
- 更新 mapper、创建/更新请求校验与 repository 往返。
- 补 group repository/handler 定向测试。

## Out of Scope

- 实现候选调度或管理端编辑器。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] Admin Group GET 对新旧数据均返回语义等价 JSON。
- [x] PUT 后 model、account_ids、priority、daily_token_limit 均不丢失。
- [x] 普通用户 Group DTO 不暴露内部路由配置。

## Verification Plan

- `cd backend; go test ./internal/repository ./internal/handler/admin -run 'Group|ModelRouting'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Ent generation | `cd backend; $env:GOPROXY='https://goproxy.cn,direct'; go generate ./ent` | PASS；首次官方代理超时，切换代理后生成成功；仅保留 model_routing 类型相关生成差异。 |
| Focused tests | `cd backend; go test ./internal/repository ./internal/handler/admin -run 'Group|ModelRouting'` | PASS（repository 4.614s，handler/admin 4.876s）。 |
| Broader relevant tests | `cd backend; go test ./internal/domain ./internal/service ./internal/handler/dto ./internal/handler/admin ./internal/repository -run 'Group|ModelRouting'` | PASS（五个相关包全部通过）。 |
| Hygiene | `git diff --check` | PASS；仅有既有 `PLAN.md` 与生成依赖 `go.sum` 的行尾提示。 |

## Execution Notes

- `groups.model_routing` 的 SQL 类型仍为 JSON；Go 侧改用 `domain.ModelRoutingJSON` 透明载体，避免把旧账号数组强制改写成新候选对象，也不会丢弃新对象字段。
- service/auth cache 使用兼容载体，Admin create/update 请求先通过 MVP-002 解析器校验再保存；repository 在写 Ent 前统一编码，读取时保留原 JSON 语义。
- `dto.AdminGroup` 返回路由配置；普通 `dto.Group` 结构仍没有 `model_routing`。定向测试同时覆盖旧数组、新候选字段往返和普通 DTO 不泄露。
- repository 测试最初被既有 `usage_log_repo_request_type_test.go` 对已不存在的 `usageLogInsertArgTypes` 引用阻断；已将该断言改为依据现行 `usageLogInsertColumns` 计算列数，恢复原测试意图后指定命令通过。
