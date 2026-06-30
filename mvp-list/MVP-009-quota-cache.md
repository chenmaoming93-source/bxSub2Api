# MVP-009: 增加每日配额 Redis 快速判断与失效机制

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 沿用现有 BillingCache 形状，实现键、TTL、miss 回源和写后同步的最小闭环。
- Dependencies: `MVP-007`, `MVP-008`

## Outcome

路由预检可优先读取 Redis，缓存缺失回源 DB，写入后不会继续使用旧额度。

## Context

参考 `backend/internal/repository/billing_cache.go` 和 user platform quota 的 cache/dirty-key 模式；TTL 应覆盖到下一个 StartOfDay 后的小缓冲。

## In Scope

- 定义三类缓存 key 与 entry。
- 实现 hit/miss、回源填充、配置更新失效和用量写后同步。
- 用 miniredis 覆盖过期和隔离测试。

## Out of Scope

- 路由接入、后台管理 API。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] cache hit 不访问 DB。
- [x] cache miss 回源一次并写入合理 TTL。
- [x] 不同用户/模型/分组候选的 key 不碰撞。

## Verification Plan

- `cd backend; go test ./internal/repository ./internal/service -run 'TokenQuota.*Cache'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/repository/daily_token_quota_cache.go` | Added Redis-backed cached daily token quota repository wrapper with model, user-model, and group-candidate keys; miss fallback; TTL through next daily boundary plus buffer; write-after-increment cache sync for existing entries; and explicit per-key invalidation methods for configuration updates. |
| Tests | `backend/internal/repository/daily_token_quota_cache_test.go` | Added miniredis tests for cache hit without DB, miss fallback with TTL, key isolation, increment synchronization, and invalidation. |
| Verification | `cd backend; $env:GOCACHE='E:\code\vs\sub2api\sub2api\backend\.gocache'; $env:GOMODCACHE='E:\code\vs\sub2api\sub2api\backend\.gomodcache'; $env:GOTMPDIR='E:\code\vs\sub2api\sub2api\backend\.gotmp'; go test ./internal/repository ./internal/service -run 'TokenQuota.*Cache'` | PASS: `ok github.com/Wei-Shaw/sub2api/internal/repository 6.199s`; `ok github.com/Wei-Shaw/sub2api/internal/service (cached) [no tests to run]`. |
| Regression check | `cd backend; $env:GOCACHE='E:\code\vs\sub2api\sub2api\backend\.gocache'; $env:GOMODCACHE='E:\code\vs\sub2api\sub2api\backend\.gomodcache'; $env:GOTMPDIR='E:\code\vs\sub2api\sub2api\backend\.gotmp'; go test ./internal/repository -run 'DailyTokenQuota'` | PASS: `ok github.com/Wei-Shaw/sub2api/internal/repository 2.818s`. |

## Execution Notes

- The first cache test run needed to download `github.com/alicebob/miniredis/v2`; after approval, the focused verification passed.


