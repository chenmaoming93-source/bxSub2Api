# MVP-014: 让超级管理员统一豁免用户限额

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 需要统一超级管理员判定并覆盖 HTTP、WebSocket、Gemini、图片等多入口，不能只改一个 handler。
- Dependencies: `none`

## Outcome

所有超级管理员无论在用户并发、用户 RPM、用户平台配额还是用户级 Token 限额中都不被拦截；`admin@example.com` 作为验收样本，而不是写死在业务逻辑中的邮箱。

## Context

当前网关存在多条用户限制链路：

- `handler.AcquireUserSlotWithWait` 控制用户并发；
- `BillingCacheService.checkRPM` 控制用户级 RPM；
- `checkUserPlatformQuotaEligibility` 控制用户 × 平台配额；
- `checkDynamicQuotaBeforeScheduling` / `checkDynamicQuotaAfterSelection` 控制 Token 统计限额。

`middleware.AuthSubject` 当前只有用户 ID 和并发值，没有超级管理员标记。管理员角色和 RBAC `*` 权限已有判定，但网关热路径不应每次查数据库/RBAC。

## In Scope

- 定义统一的 `isSuperAdminUser` 判定：系统 admin 角色或有效权限包含 `*`。
- 将 `SuperAdmin` 布尔值写入认证缓存快照和 `AuthSubject`，由中间件一次解析，热路径只读布尔值。
- 用户级并发、用户 RPM、用户 × 平台配额、用户级 Token 限额均跳过超级管理员。
- 用户额度列表、用户限额选择和用户统计报表排除超级管理员账号。
- 保留分组、账号、路由候选等共享容量限制，不因用户身份被绕过。

## Out of Scope

- 绕过账号池、分组或上游提供商的容量保护。
- 修改历史用量统计；超级管理员的用量仍然正常记录。
- 硬编码 `admin@example.com` 或其它邮箱。

## Implementation Notes

- 在认证快照中携带 `IsSuperAdmin`，缓存失效与用户角色/权限变更联动。
- 对用户配额规则，仅跳过包含该 `user_id` 的规则；与账号/分组共享容量相关的规则仍执行。
- 并发路径在调用 `AcquireUserSlotWithWait` 前直接跳过用户槽位，但仍获取账号/路由槽位。
- 用户限额页面用角色/权限过滤，而不是邮箱过滤；`admin@example.com` 仅作为回归夹具。
- 记录 debug 级豁免日志，避免高流量下刷日志。

## Acceptance Criteria

- [x] `admin@example.com` 和无邮箱关联的任意超级管理员均不受用户级限额影响。
- [x] 普通管理员但无超级管理员权限时不自动豁免。
- [x] 用户并发、RPM、平台配额、Token 限额四条路径均有正向和反向测试。
- [x] HTTP、chat、responses、Gemini、图片和 WebSocket 入口使用同一豁免判定。
- [x] 账号并发、路由并发和上游配额仍然生效。
- [x] 用户限额列表和选择器中不出现超级管理员用户。
- [x] 热路径不新增每请求 RBAC/数据库查询。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/handler ./internal/service ./internal/server/middleware
go test -tags=integration ./internal/repository
go build ./cmd/server
```

增加角色变更后的缓存失效测试，确认权限更新后豁免立即生效。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unified identity | `service.User.IsSuperAdmin()` | PASS: true for system `admin` role or an active wildcard RBAC grant; no email comparison or hard-coded account |
| Auth snapshot | `APIKeyAuthUserSnapshot.SuperAdmin`, snapshot version 14 | PASS: effective flag is resolved only while rebuilding a missing auth-cache snapshot and then travels through `APIKey`/`AuthSubject` |
| Hot-path performance | auth middleware + `AuthSubject.SuperAdmin` | PASS: gateway requests read only the cached boolean and request context marker; no RBAC or database lookup added per request |
| User concurrency | `ConcurrencyHelper` tests | PASS: super admin gets a no-op user release without calling the user cache; account slot acquisition still calls the shared account limiter |
| User RPM | external service exemption test | PASS: system admin and wildcard admin skip user RPM; group `rpm_limit` still returns `GROUP_RPM_EXCEEDED`; normal users still return `USER_RPM_EXCEEDED` |
| Platform quota | `CheckUserPlatformQuotaEligibilityForUser` test | PASS: exhausted daily platform quota is bypassed for super admin without a repository read; normal users receive `USER_PLATFORM_DAILY_QUOTA_EXHAUSTED` |
| User Token quota | `CheckDynamicTokenRouteCandidate` test | PASS: `user_id` is omitted from quota matching for super admin, so user-only rules no longer match; group-scoped rules still enforce |
| Shared capacity | handler/service tests and unchanged account/route paths | PASS: account concurrency, group RPM, route candidates and upstream account/group/model limits remain active |
| Gateway entry coverage | source contract test + focused helper tests | PASS: messages, chat, responses, Gemini, shared OpenAI/image path and three WebSocket user-slot calls use the same super-admin-aware helper |
| User list filtering | admin `List` + `exclude_super_admins=true` | PASS: one SQL predicate excludes legacy `users.role=admin` and active wildcard RBAC users; pagination totals use the filtered query |
| Quota selectors | `TokenStatisticsView.vue`, `GroupRPMOverridesModal.vue` | PASS: both request `exclude_super_admins=true`; admin@example.com is absent from user quota and statistics selectors |
| User page quota controls | `UsersView.vue` + `is_super_admin` batch marker | PASS: super-admin rows show the unlimited label and no platform-quota menu/modal action |
| Cache invalidation | `RBACRoleService.ReplaceUserRoles`, `ReplacePermissions`, role status/delete | PASS: affected users' API Key snapshots and effective-permission cache entries are invalidated; external test records invalidation for assigned users |
| RBAC batch query | `RBACRepository.ListSuperAdminUserIDs` | PASS: unit SQL test covers legacy admin + wildcard union and requested user-ID scoping |
| Backend tests | full handler/admin/repository/RBAC/middleware/routes plus `limit_exemption_test` | PASS |
| Frontend checks | `pnpm run typecheck`, `pnpm run lint:check`, focused Vitest, `pnpm run build` | PASS; lint has one pre-existing warning and zero errors |
| Backend build | `go build -p 1 ./cmd/server`; repository integration test binary compile | PASS |
| Known existing limitation | Full `./internal/service` package tests | Pre-existing account repository test mocks still omit `ExistsByName`; production build and the dedicated external exemption test package pass |

## Execution Notes

“超级管理员”以统一权限判定为准，不能通过邮箱判断；若系统存在 admin 角色与 RBAC 通配权限两套来源，需要由同一 helper 合并处理。
