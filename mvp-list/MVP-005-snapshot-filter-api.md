# MVP-005: 提供首屏快照与统一筛选选项 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 聚焦一个可用的首屏接口及筛选选项，复用已有仓储并增加短缓存，边界清晰。
- Dependencies: `MVP-002`, `MVP-003`, `MVP-004`

## Outcome

管理员可通过一个接口获得 KPI、TopN、当前压力、数据质量和受控筛选选项，首屏不再并发拉取大量互不一致的数据。

## Context

现有 Ops/Dashboard 接口各有时间过滤和响应结构，但不能组合 route alias 等完整账号路径维度，也不返回完整 meta。前端需要稳定的单一首屏快照。

## In Scope

- 新增 `GET /api/v1/admin/observability/snapshot`。
- 返回当前 Token 累计/速率、请求与最终失败率、P95 duration/TTFT、账号与候选压力、Top 用户/分组/别名/账号/模型。
- 用户对象返回 `user_id / email / department`；管理员和运维只读角色可查看，不做邮箱脱敏，但不返回 API Key secret、请求正文或凭据。
- 返回 filter options：有权限的 users、groups、aliases、accounts，支持服务端搜索和数量上限；账号项附带唯一绑定模型。
- 汇总 tokenstat 当前值、分钟聚合、Redis 实时并发，并执行对账。
- 实现 `observability.read` 权限、5 秒短缓存、`ETag` 和 data-quality meta。

## Out of Scope

- 长趋势、热力图和完整路径明细。
- 写操作。
- 未受控的任意 SQL 查询或导出。

## Implementation Notes

- 采用 service/repository/port 分层，避免 handler 直接拼 SQL。
- Token 当前累计以 tokenstat 投影为准；未启用投影时返回明确 warning，不用 usage_logs 静默替换配额口径。
- 并发按 account_id 去重；TopN 的顺序明确为压力、Token 或请求数。
- filter options 只返回当前角色可见且当前窗口存在或配置有效的对象。
- 缓存键包含 canonical filters、permission scope、schema version；不得让不同 RBAC scope 共用敏感缓存。

## Acceptance Criteria

- [x] 无筛选请求一次返回 KPI、TopN、压力摘要、filter options 和 meta。
- [x] 相同参数在缓存有效期内返回 ETag，`If-None-Match` 命中时返回 304。
- [x] tokenstat 未配置/未启用、分钟聚合滞后、并发采样缺失分别产生稳定 warning code。
- [x] 账号并发在 TopN 和多路径摘要中不重复累加。
- [x] 无权限用户 403；只读用户不能访问写接口。
- [x] 用户对象包含 ID、邮箱和部门；敏感凭据字段不会出现在响应。
- [x] P95 常规缓存命中小于 200ms 的测试/压测证据。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/handler ./internal/service ./internal/repository
go test -tags=integration ./internal/repository
go build ./cmd/server
```

使用 1000 账号、10000 路径的固定夹具验证响应大小、TopN 和耗时。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Snapshot contract | `GET /api/v1/admin/observability/snapshot` | Returns metrics, five TopN groups, account/candidate pressure, filter options and meta in one success envelope |
| ETag/cache | `internal/handler/admin/observability_snapshot_handler_test.go` | PASS: first request emits ETag; `If-None-Match` returns 304 from 5s cache with no rollup reload |
| Cache latency | Same handler test | PASS: cached 304 measured below the asserted 200ms ceiling |
| Warning codes | `backend/internal/service/observability_snapshot.go` | `tokenstat_projection_unavailable`, `request_rollup_lagging`, `concurrency_snapshot_unavailable`, `request_rollup_unavailable` |
| Tokenstat accounting | `ProjectionAdminService.ResolveUsageProjection` + daily `QueryUsage` | Current token total is added only when a valid projection exists; otherwise the dashboard keeps request-window tokens and an explicit warning |
| TopN semantics | Local MariaDB snapshot verifier | PASS: 6 requests / 600 tokens; user 2=500, group 7=400, alias coding=400, account 102=500; duration P95=2000ms |
| Account dedupe | SQL groups account TopN by `account_id`; pressure rows dedupe account IDs | PASS: aliases do not multiply account concurrency totals |
| Filters | Contract parser + repository `IN` filters | Supports user/API key/group/alias/account/department; missing filter values are ignored and malformed/oversized arrays are rejected |
| Filter options | Snapshot service | Users include ID/email/department; accounts include unique derived `bound_model`; search and max 50 enforced |
| Sensitive fields | Handler test | PASS: response contains user email/department and bound model, with no API key secret, credential, or request body field |
| RBAC | Route + permission tests | PASS: snapshot uses `observability.read`; denied scope returns 403; no observability write route exists |
| Scope-safe cache | Snapshot cache key includes canonical request plus authenticated user scope | Implemented |
| Unit tests | `go test -tags=unit ./internal/handler/admin ./internal/server/routes ./internal/repository ./internal/service/observability` | PASS |
| Integration compile | `go test -c -tags=integration ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Known limitation | Full 1000-account/10000-path load fixture | Docker daemon is unavailable; functional SQL and cache paths were verified, while high-cardinality load remains for MVP-012 performance gates |

## Execution Notes

如果 provider/platform 仍是现有重要维度，可在 snapshot meta 中保留，但核心 canonical 四维不得改变。
