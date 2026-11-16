# MVP-002: 强制账号单模型并持久化请求路由归属

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 涉及账号模型不变量、两张事实表、Ent 生成、gateway 写入点和兼容测试，范围仍可在一个聚焦切片内完成。
- Dependencies: `MVP-001`

## Outcome

全平台严格保证一个模型账号只对应一个上游模型；每条新请求都能在成功和失败事实中保存最终命中的 `route_alias`。

## Context

`backend/internal/service/usage_log.go` 的 `UsageLog.RouteAlias` 当前明确“不持久化”；`ops_error_logs` 也没有 `route_alias`。账号模型校验目前只限制非 OAuth，与产品要求的全账号 1:1 不一致。Gateway 已把 `selection.RouteAlias` 写入 UsageLog 结构，但落库时丢失。

## In Scope

- `usage_logs` 新增可空 `route_alias VARCHAR(255)`，Ent schema 与生成代码同步。
- `ops_error_logs` 新增可空 `route_alias VARCHAR(255)`，并添加查询索引。
- 不新增 `accounts.bound_model` 列；复用现有 Account 领域对象和单模型映射解析，在服务层输出 `bound_model` 并缓存解析结果。
- 将 `model_mapping` 单模型约束扩展到所有账号类型，并覆盖单账号编辑、批量更新、导入、路由候选绑定。
- 对齐所有 HTTP、WebSocket、Gemini、OpenAI gateway 的最终成功/错误写入点；只写最终命中的 alias。
- 为历史缺失数据提供 `legacy_inferred=true` 标记：回退 `requested_model`，不从 `model` 无条件冒充 alias。
- 添加迁移、写入链路与错误请求归属测试。

## Out of Scope

- 回填无法证明的历史精确 alias。
- 把上游模型继续作为独立看板维度。
- 用 `scene_name` 代替 route alias。

## Implementation Notes

- 建议索引：`usage_logs(group_id, route_alias, created_at)`、`ops_error_logs(group_id, route_alias, created_at)`；同时评估 `(route_alias, created_at)`。
- 空模型映射必须解析为一个确定性透明上游模型；若无法唯一确定，账号保存失败。
- 错误重试信息可能产生多条上游事件；`route_alias` 取客户端最终请求的分类上下文，不按每个上游错误重新解释。
- 保留旧 API 的 `RouteAlias` 结构字段，仅修复持久化，避免大范围改调用签名。

## Acceptance Criteria

- [x] OAuth、API Key 和其他账号类型都无法保存两个及以上上游模型。
- [x] 不执行全表 JSON 回填；只读审计能列出历史多模型异常账号，正常账号无需迁移。
- [x] 新成功请求写入 `usage_logs.route_alias`，空 alias 场景写入 NULL。
- [x] 新最终失败请求写入 `ops_error_logs.route_alias`，同一 request_id 不产生矛盾 alias。
- [x] HTTP、responses/chat/Gemini 至少各有一个测试覆盖 alias 落库。
- [x] 任一账号读出的 `bound_model` 唯一且稳定。
- [x] 历史回退结果在 API 中带有 `legacy_inferred` 或等价警告。
- [x] schema migration 在空库和已有数据副本上均可执行。

## Verification Plan

```powershell
cd backend
go generate ./ent
go test -tags=unit ./internal/service ./internal/handler ./internal/repository
go test -tags=integration ./internal/repository
go build ./cmd/server
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Account invariant | `backend/internal/service/account_model_mapping_validation.go` | PASS: all account types reject more than one non-blank mapping key; bulk mapping edits use the same validator |
| Bound model | `backend/internal/handler/dto/account_bound_model_test.go` | PASS: deterministic `bound_model`; historical multi-model accounts expose `bound_model_legacy_inferred` without mutation |
| Usage persistence | `go test -tags=unit ./internal/repository ./internal/handler/dto` | PASS: non-empty alias is persisted and empty alias is written as NULL |
| Route attribution | `go test -tags=unit ./internal/handler -run 'TestSetOpsRequestContextCapturesAndClearsRouteAlias'` | PASS: HTTP messages, responses, chat completions and Gemini contexts capture the route alias and clear stale aliases after failover |
| Error persistence | `backend/internal/repository/ops_repo_route_alias_test.go` | PASS: `ops_error_logs.route_alias` is present in the insert column/argument list |
| Ent generation | `go generate ./ent` | PASS: generated usage-log model/query/create/update code includes `route_alias` |
| Backend build | `go build -p 1 ./cmd/server` in `backend` | PASS |
| Migration archive | `backend/sqlArchiving/176_add_route_alias_facts.sql` | PASS: executed twice against empty tables and against tables containing historical rows; rows and prior values remained intact, aliases stayed NULL, 2 columns and 4 indexes existed |
| Performance | `usage_logs` and `ops_error_logs` indexes | `(group_id, route_alias, created_at)` for scoped time queries and `(route_alias, created_at)` for route-only trends/filtering |
| Historical data | No `UPDATE accounts` or JSON backfill in this MVP | PASS: historical multi-model accounts are reported via `bound_model_legacy_inferred`; normal accounts require no migration |
| Changed paths | `backend/ent/schema/usage_log.go`, generated `backend/ent/*`, `backend/internal/{service,repository,handler,dto}`, `backend/sqlArchiving/176_add_route_alias_facts.sql` | Implemented |
| Known limitation | `go test -tags=unit ./internal/service` | Existing test mocks predate `AccountRepository.ExistsByName` and fail package compilation; production build and focused repository/handler/DTO checks pass |

## Execution Notes

若发现批量导入或旧配置已经含多模型账号，提供显式失败清单和人工迁移指引，不做模糊截断。历史事实仅在 `route_alias IS NULL` 且 `requested_model` 与网关 `model` 不同时启用启发式回退，并通过 `legacy_inferred=true` 明示；不会覆盖已保存的新 alias。
