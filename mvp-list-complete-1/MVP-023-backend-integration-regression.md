# MVP-023: 完成每日配额与分组路由后端集成回归

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 只补计划列出的跨组件高风险用例并运行聚焦套件，功能实现已在前置 MVP 完成。
- Dependencies: `MVP-016`

## Outcome

通过数据库与 gateway 集成测试证明并发不丢增量、跨日切换正确、身份字段正确且旧路由不回归。

## Context

优先扩展 `backend/internal/integration`、migration tests 和现有 gateway record usage tests。

## In Scope

- 覆盖 migration 默认值/唯一索引、并发累加与跨日窗口。
- 覆盖新旧 model_routing、配额降级和 requested/upstream model。
- 记录可重复的测试命令与结果到 Completion Evidence。

## Out of Scope

- 全仓不相关 flaky test 修复和性能压测。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 计划中的后端单元与集成场景均有自动化覆盖。
- [x] 旧 model_routing 行为不回归。
- [x] 聚焦 integration/service/repository 测试全部通过。

## Verification Plan

- `cd backend; go test ./internal/repository ./internal/service ./internal/integration ./migrations -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Fix | `backend/internal/repository/usage_log_repo_request_type_test.go` | Fixed `TestBuildUsageLogBestEffortInsertQuery_IncludesRequestedModelColumn` — the test checked for specific tab formatting (`\t\t\t`) that no longer matched the production column list (which uses single-tab `usageLogInsertColumns`). Replaced with content-based assertions (`"requested_model, upstream_model,"` and `"model, requested_model, upstream_model,"`) that correctly verify column presence. |
| Repository tests | `cd backend; go test ./internal/repository -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (4.638s). Covered: cache hit/miss/TTL/isolation, admin invalidation fresh read, start-of-day/unlimited semantics, user isolation, concurrent increment, rollover, rollback, legacy routing shape preservation, and best-effort/exec insert column checks. |
| Service tests | `cd backend; go test ./internal/service -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (4.905s). Covered: content moderation with requested model, Claude/OpenAI exactly-once token accounting, failed/duplicate persistence skip, simple-mode write requirement, increment failure policy, boundary/unlimited semantics, user/group error contexts, Anthropic passthrough upstream model identity, Gemini requested/upstream model preservation, OpenAI compact-only mapping, OAuth passthrough mapping, billing with requested model fallback, quota degradation regression (group/global/user/repository-failure subtests), model routing candidate/legacy regression (priority/legacy/same-candidate/next-candidate-failover/unschedulable subtests), and upstream model ID extraction. |
| Integration tests | `cd backend; go test ./internal/integration -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (1.702s). `TestTokenQuotaModelRoutingIntegrationContract` covers cross-component wiring. |
| Migration tests | `cd backend; go test ./migrations -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (0.863s). `TestTokenQuotaMigrationsDefaultsAndUniqueIdentities` covers global model, user model, and group candidate table idempotency, default values (used_tokens DEFAULT 0), unique composite keys, and foreign key cascades. |
| Handler tests | `cd backend; go test ./internal/handler -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (4.400s). `TestBillingErrorDetails_RoutedTokenQuotaExhaustedReturns429`. |
| Server tests | `cd backend; go test ./internal/server -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel' -count=1` | PASS (5.020s). |

## Execution Notes

- The only fix required was in `usage_log_repo_request_type_test.go` where two assertions checked for deprecated column formatting (`\n\t\t\t` prefix). Changed to content-based checks that verify the `requested_model` and `upstream_model` columns appear in the INSERT query regardless of whitespace.
- The unrelated `TestOpsServiceRecordErrorBatch_SanitizesAndBatches` failure in the full `./internal/service` suite is a pre-existing issue in ops batch URL sanitization — out of scope for this MVP.
- All existing quota/routing/model-identity tests from MVPs 004–016 continue to pass; no new regression tests were needed — the existing coverage already satisfies the acceptance criteria.

