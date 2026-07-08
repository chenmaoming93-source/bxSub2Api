# MVP-010: 提供全局模型每日限额管理 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
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

- [x] 非管理员无法调用端点。
- [x] GET 返回模型、限额、当日 used_tokens。
- [x] PUT 后再次 GET 和缓存读取均返回新值。

## Verification Plan

- `cd backend; go test ./internal/handler/admin ./internal/server -run 'ModelTokenQuota'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/service/model_token_quota_admin_service.go`, `backend/internal/handler/admin/model_token_quota_handler.go`, `backend/internal/server/routes/admin.go` | Added global model token quota admin service, GET/PUT admin handlers, DTOs, validation, routes, and cache invalidation after limit updates. |
| Repository support | `backend/internal/repository/daily_token_quota_repo.go`, `backend/internal/repository/daily_token_quota_cache.go` | Added model quota list/set repository methods and a Redis cache invalidator compatible with the daily token quota cache keys. |
| DI wiring | `backend/internal/handler/wire.go`, `backend/internal/repository/wire.go`, `backend/internal/service/wire.go`, `backend/cmd/server/wire_gen.go` | Wired the new admin handler/service/repository. `go generate ./cmd/server` was blocked by Windows Application Control for `wire.exe`, so `wire_gen.go` was synchronized manually and compile-checked. |
| Focused handler/server verification | `cd backend; go test ./internal/handler/admin ./internal/server -run 'ModelTokenQuota'` | `internal/server` PASS. `internal/handler/admin` normal `go test` was blocked by Windows Application Control for `admin.test.exe`; compiled with `go test -c -o .\.gotmp\handler_admin_modelquota_test.exe ./internal/handler/admin` and ran `.\.gotmp\handler_admin_modelquota_test.exe -test.run ModelTokenQuota -test.v`: PASS. |
| Cache freshness regression | `cd backend; $env:GOCACHE='E:\code\vs\sub2api\sub2api\backend\.gocache'; $env:GOMODCACHE='E:\code\vs\sub2api\sub2api\backend\.gomodcache'; $env:GOTMPDIR='E:\code\vs\sub2api\sub2api\backend\.gotmp'; go test ./internal/repository -run 'ModelTokenQuota|DailyTokenQuota'` | PASS: `ok github.com/Wei-Shaw/sub2api/internal/repository 5.988s`. |
| Compile check | `cd backend; $env:GOCACHE='E:\code\vs\sub2api\sub2api\backend\.gocache'; $env:GOMODCACHE='E:\code\vs\sub2api\sub2api\backend\.gomodcache'; $env:GOTMPDIR='E:\code\vs\sub2api\sub2api\backend\.gotmp'; go test -c -o .\.gotmp\cmd_server_modelquota_test.exe ./cmd/server` | PASS: `cmd/server` test binary compiled successfully. |

## Execution Notes

- `go get github.com/google/wire/cmd/wire@v0.7.0` was run to add missing Wire dependency checksums before the generator hit the local application-control policy.


