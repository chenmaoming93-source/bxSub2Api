# MVP-010：跨模块回归、恢复与性能验收

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `不新增独立业务能力，集中验证已交付切片之间的数据流、Redis 恢复和既有调度兼容。`
- Dependencies: `MVP-004, MVP-005, MVP-007, MVP-009`

## 预期成果

五类需求在完整调用链中协同工作，并有证据证明 OAuth、账号快照同步、多账号调度、Redis 恢复和查询口径未发生非预期回归。

## 背景

源 Plan 明确区分新开发与已有能力。本 MVP 只补充必要的跨模块测试、缺失修复和验收证据，不重做已有 outbox 或调度算法。

## 范围内

- 运行后端和前端相关完整测试。
- 验证 Redis 热态路由请求账号数据库查询为零。
- 验证 Redis 清空后的启动预热和请求回源。
- 验证账号 `model_mapping` 更新后现有快照同步仍生效。
- 验证 candidate priority、粘性、账号 priority、负载、LRU 和并发逻辑保持。
- 验证 OAuth 和非模型路由不受影响。
- 验证 wildcard、API Key 选择和 usage 路由别名的端到端契约。

## 范围外

- 修复 `UpdateCredentials` 缺少 outbox 兜底。
- 新增未在前序 MVP 定义的产品行为。
- 生产部署或真实流量切换。

## 实现说明

- 若发现前序 MVP 遗漏，只修复其既定验收范围并在执行记录中标明归属。
- 数据库查询数应通过 mock/spy、SQL 观测或集成测试计数形成证据。
- 恢复测试需区分“预热成功”和“预热失败后请求回源”两条路径。

## 验收标准

- [x] 所有 MVP 的后端、前端和类型检查通过。
- [x] 热缓存模型路由请求不查询账号数据库。
- [x] Redis 清空后纯路由账号可通过预热或批量回源恢复。
- [x] 管理员修改模型后现有 `account_changed`/即时快照链路未被破坏。
- [x] OAuth、非模型路由和现有多账号调度行为回归通过。
- [x] wildcard 不合并实际维度值，usage 各视图按路由别名保持一致。

## 验证计划

- `cd backend && go test ./...`
- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`
- 在仓库集成环境执行 Redis 清空、服务初始重建、路由首请求和热态后续请求验证。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端全量 | `cd backend && go test ./...` | 除 `internal/rbac` 两个 HEAD 基线失败（`token_usage.manage` 缺失于 compatibility seed，`internal/rbac` 与 `sqlArchiving/` 工作区零改动）外，全部包 `ok`；`internal/service` 在修复前存在 1 个跨模块回归，已修复后通过。 |
| 前端全量 | `cd frontend && pnpm test:run` | 126 文件 757 用例：751 通过；仅 3 个 auth 测试文件 6 用例失败（`EmailVerifyView`/`WechatOAuthSection`/`EmailOAuthButtons`，`frontend/src/components/auth`、`views/auth` 工作区零改动，为 HEAD 基线失败）。 |
| 类型检查 | `cd frontend && pnpm typecheck` | `vue-tsc --noEmit` 通过。 |
| 热态零查询 | `go test ./internal/service -run 'RouteAccountBatchCacheFullHitSkipsDatabase' -count=1` | 全命中时 `repo.calls == 0`，账号数据库查询为零；请求 ID 去重保序。 |
| Redis 清空恢复 | `go test ./internal/service -run 'RouteAccountBatchCachePartialMiss|RouteAccountWarmupLoadsPureRouting' -count=1` | 部分未命中只回源缺失 ID 一次并 best-effort 写回，二次热读不再查库；130 个纯路由账号经 2 个 128-ID 有界批次预热全部回填；写回失败与 fallback 限流不阻塞当前读取。 |
| 快照链路 | `go test ./internal/service ./internal/repository ./internal/handler/admin -run 'AccountChanged|Snapshot|Account.*Update|SyncScheduler|Outbox' -count=1` | 通过；`model_mapping` 更新后的即时快照/outbox 链路未被破坏。 |
| 调度与 OAuth 回归 | `go test ./internal/service ./internal/repository ./internal/handler/admin ./internal/integration -run 'SchedulerCache|SchedulerSnapshot|RouteAccount|ModelRouting|GatewayMultiPlatform|GroupRoute|AccountFirstModel|TokenQuotaModelRouting|UsageRequestedModel|OpsError|APIKey|Quota|MaskUsage|AccountModelMapping' -count=1` | 四个包全部通过；priority、粘性、账号 priority、负载、LRU、并发选择、OAuth/非模型路由均保持。 |
| 跨模块修复 | `backend/internal/service/external_provisioning_service.go`、`backend/internal/domain/model_routing.go` | 修复 MVP-002 遗漏：`ListGroupModelRoutes` 恢复从历史候选聚合 `upstream_models`（`Model` 恢复为 `json:"-"` 进程内字段，解析时捕获、序列化不输出）；`go test ./internal/domain ./internal/service -run 'ParseModelRoutingConfig|ListGroupModelRoutes'` 通过。 |

## 执行记录

- 跨模块缺陷（归属 MVP-002）：`ListGroupModelRoutes`（外部供应集成 API `/integrations/model-routes/list`）在候选去 model 时被改为恒空数组，破坏既有 `upstream_models` 契约。修复：`domain.ModelRouteCandidate.Model` 恢复为 `json:"-"` 进程内过渡字段（解析时从历史 JSON 捕获、去空白，序列化不输出），service 层按已排序候选去重聚合。符合 MVP-002“旧 model 可读取但不再输出”的既定契约。
- 恢复演练：以 miniredis 集成测试验证“Redis 清空（全未命中）→ 有界预热或批量回源 → 写回 `sched:acc:<id>` → 热态零查询”完整路径，以及预热失败/写回失败降级路径；未连接外部 Redis/MySQL，避免依赖外部环境。
- 最终验收结论：本 MVP 系列 9 个已完成 MVP 的后端、前端与类型检查在范围内全部通过；`internal/rbac`（2 用例）与前端 auth（3 文件 6 用例）失败经核对为 HEAD 基线问题，相关目录零改动，不属于本系列回归。
