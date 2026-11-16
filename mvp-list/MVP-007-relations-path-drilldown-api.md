# MVP-007: 提供关系链路与路径明细 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 复用 `group_model_route_accounts` 和分钟事实即可完成关系与分页，无需改动路由算法。
- Dependencies: `MVP-002`, `MVP-003`, `MVP-005`

## Outcome

调用方可以查询用户→分组→路由→候选账号的命中关系和完整路径明细，模型账号同时携带唯一绑定模型，支持逐级筛选、分页和空闲关系。

## Context

`group_model_route_accounts` 是现有 JSON 路由的规范化投影，但 `scene_name` 可重复、路由键可有通配符。产品口径规定账号严格对应一个上游模型，路径关系不能靠名称猜测。

## In Scope

- 新增 `GET /relations`，返回 user/api_key/group/route_alias/account 节点和有方向边；account 节点携带 `bound_model`。
- 新增 `GET /paths`，按完整维度分页返回 Token、请求、占比、压力、P95、错误率和状态。
- 支持 `include_idle=true`，空闲关系与有流量关系明确分离。
- 用 `group_model_route_accounts` 展开配置关系；实际路由 alias 用请求事实匹配通配模式。
- 实现服务端排序、状态筛选、搜索和分页。

## Out of Scope

- 修改或保存路由配置。
- 跨分组流量权重。
- 用户敏感信息脱敏策略的最终产品决策。

## Implementation Notes

- 所有写操作标识使用 ID；显示名单独返回。
- 账号并发只作为共享属性，不在 `/paths` 的 Token/请求 sum 中重复相加。
- 账号模型应唯一；缺失时返回 `model_attribution=unknown` 并要求上游数据修复，不得自行推断成多个模型。
- 历史 alias 推断值带 `legacy_inferred=true`，前端可筛选是否包含。
- 服务端先对请求事实做有界 aggregation，再做分页，避免前端拼接。

## Acceptance Criteria

- [x] `/relations` 正确显示 API Key 到分组、分组到路由、路由到账号、账号到上游模型关系。
- [x] 通配路由 `gpt-*` 命中 `gpt-5` 时，请求 alias 保留 `gpt-5` 并可追溯到模式。
- [x] 任一账号只返回一个 `bound_model`；发现多模型数据时接口显式报数据质量错误。
- [x] `include_idle=false` 不返回零流量配置边；`true` 时 idle 数据不影响 usage 汇总。
- [x] 分页 total、page_size、排序稳定，限定筛选下结果确定。
- [x] `window=7d` 的用户路径明细只返回 TopN/100，不开放深分页全量扫描，并返回 `coverage=top_n`。
- [x] 账号并发不会在路径行求和时重复累计。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/handler ./internal/service ./internal/repository
go test -tags=integration ./internal/repository
go build ./cmd/server
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| API endpoints | `GET /observability/relations`, `GET /observability/paths` | Implemented under `observability.read` |
| Relation graph | `backend/internal/service/observability_paths.go` | user -> api_key -> group -> route_alias -> account -> bound_model nodes/edges; account model is a property node |
| Wildcard trace | `TestObservabilityPathsAndRelationsIdleAndWildcard` | PASS: persisted request alias `gpt-5`, configured pattern `gpt-*`, traffic edge is matched and non-idle |
| Idle separation | Same handler test | PASS: `include_idle=false` removes idle key/route nodes and edges; usage rows never include idle configuration |
| Bound model quality | Account `BoundModel` + `ModelBindingLegacyMulti` | One deterministic model per account; historical multi-model accounts produce `multiple_bound_models:account:<id>` in `data_quality_errors` |
| Path aggregation | `backend/internal/repository/observability_paths_repo.go` | Groups by user/API key/group/alias/account before pagination; total uses the same grouped count |
| Pagination/sort | Handler + SQL tests | Supports tokens/requests/error rate/avg duration stable ordering with deterministic tie-breakers |
| 7d coverage | `TestObservabilityPathsSevenDayCoverageIsTopN` | PASS: page forced to 1, size capped at 100, meta coverage=`top_n` |
| Concurrency sharing | Path service pressure map keyed by `account_id` | Current/limit are displayed on path rows but never added into token/request sums |
| Local SQL verification | Temporary MariaDB verifier | PASS: 2 paths, total=2, stable request sort, first key=11, actual alias=`gpt-5`, requests=5 |
| Unit tests | `go test -tags=unit ./internal/handler/admin ./internal/server/routes ./internal/repository` | PASS |
| Integration compile | `go test -c -tags=integration ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Known limitation | Docker integration execution | Docker daemon is unavailable; SQL paths were executed against local MariaDB and integration tests compile |

## Execution Notes

如果现有 `group_model_route_accounts` 由于迁移未上线而缺失，迁移必须自动重建该投影并记录失败状态，不能静默返回空关系。
