# MVP-001: 固化观测接口契约与访问权限

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 主要是跨后端 DTO、RBAC 目录、兼容 seed 和前端权限映射的收敛，范围清晰且可独立验证。
- Dependencies: `none`

## Outcome

项目拥有唯一只读观测看板接口契约和专用读权限，后续数据/API/UI 不再使用不存在的权限名或 Demo 参数。

## Context

当前 RBAC 只有 `token_usage.read` 与 `token_usage.manage`，没有设计稿中的 `token_usage.view`。详情入口为 `backend/internal/rbac/permissions.go`、前端 `usePermission`、MySQL RBAC compatibility seed。设计契约已在 `dashboard-demo/DESIGN.md` v1.7 第 6、13 节明确。

## In Scope

- 在 Go 中建立 `internal/service/observability` 公共 DTO：时间窗口、桶、canonical 维度、筛选、数据质量 meta、快照/时序/路径响应。
- 新增唯一的 `observability.read` 权限并加入 RBAC 目录、兼容 seed 和前端权限元数据。
- 首期只授予超级管理员；预留运维只读角色授权流程，验证一天后再开放。
- Token/配额面板额外检查 `token_usage.read`，不把 `token_usage.manage` 作为只读看板权限。
- 为参数构造、时间 URL 解码、比率 `0..1`、`ETag` 响应头和分页上限建立共享校验。
- 添加契约 golden test 与 RBAC catalog/seed 一致性测试。
- 路由可先只注册权限守卫；真实 handler 由后续 MVP 填充。

## Out of Scope

- 查询真实业务数据。
- 新建前端页面。
- 自动把新权限授予任意自定义角色；只有系统 admin 与明确迁移策略获得默认授权。
- 任何观测写权限；看板只读。

## Implementation Notes

- 维度 code 使用 `user_id / api_key_id / group_id / route_alias / account_id / department`；上游模型作为 `account.bound_model` 属性返回。
- 公共参数使用 `window=1h|6h|24h|7d`、`bucket=5m|15m|1h|6h` 和重复 ID 参数；服务端校验合法组合。
- 统一 success envelope，`data.meta` 是所有读接口必填字段。
- 首期由系统 admin 通过通配权限访问；`observability.read` 先完成目录和授权 API，不自动批量授予普通用户。
- 旧浏览器或意外参数返回 400/422，不接受空字符串静默变成全量。

## Acceptance Criteria

- [x] RBAC catalog、SQL seed、前端权限列表三处均存在 `observability.read` 且编码完全一致。
- [x] 全仓搜索不存在 `token_usage.view` 或 `observability.manage`。
- [x] 契约测试覆盖合法/非法 window-bucket 组合、重复筛选参数、超过 50 个 ID、page_size 上限和比率单位。
- [x] 未授权用户访问观测路由得到 403；所有观测路由均为 GET。
- [x] 首期只有超级管理员能访问，运维只读角色授权可通过配置单独启用。
- [x] 后端单测和 RBAC 契约测试通过。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./...
cd ..
pnpm --dir frontend run test:run -- src/views/admin/__tests__/SettingsView.spec.ts
pnpm --dir frontend run typecheck
```

若项目测试标签/前端权限测试路径变化，以实现时仓库实际命令为准，但必须保留 RBAC seed 一致性与路由授权测试。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Backend unit | `go test -tags=unit ./internal/service/observability ./internal/rbac ./internal/server/routes` | PASS |
| Handler contract | `go test -tags=unit ./internal/handler/admin -run '^TestObservabilityCapabilitiesContractAndETag$'` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Frontend typecheck | `pnpm run typecheck` in `frontend` | PASS |
| Frontend permissions | `pnpm exec vitest run src/rbac/permissionMatrix.spec.ts` | PASS, 5 tests |
| Source audit | `rg -n 'token_usage\\.view|observability\\.manage' backend frontend --glob '!**/testdata/**'` | No implementation matches |
| Changed paths | `backend/internal/service/observability/*`, `backend/internal/handler/admin/observability_capabilities_handler*.go`, `backend/internal/rbac/permissions.go`, `backend/internal/server/routes/admin.go`, `backend/sqlArchiving/163_seed_rbac_compatibility.sql`, `frontend/src/rbac/*` | Implemented |
| Known limitation | Full `go test -tags=unit ./internal/handler/admin` | Pre-existing `TestGroupRequestsAcceptLegacyAndCandidateModelRouting` failure; focused MVP tests pass |

## Execution Notes

后续 handler 必须复用这里的 DTO 和校验，禁止各自定义 `range/scene/account` 等别名参数。全量 `handler/admin` 包存在与本 MVP 无关的既存模型路由测试失败；本次以聚焦包测试、路由权限测试和串行构建作为验收证据。
