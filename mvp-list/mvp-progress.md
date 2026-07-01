# MVP Progress

- Protocol: `mvp-list/v1`
- Source plan: `配额改造与Bug修复计划.md`
- Target effort per MVP: `~20 minutes (one focused task)`
- Update batch size: `2 completed MVPs`
- Last updated: `2026-07-01T15:30:00+08:00`
- Overall: `10/10 (100%)`

## Status Rules

- `PENDING`: not yet recorded as verified complete
- `BLOCKED`: cannot proceed and is not counted as complete
- `DONE`: implemented, acceptance criteria checked, tests run, and evidence recorded
- Progress may lag verified work between batch updates; it must never lead verified work.

## MVP List

| ID | MVP document | Status | Dependencies | Estimate | Completed at | Evidence |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-new-config-ent-schemas.md](./MVP-001-new-config-ent-schemas.md) | DONE | none | ~20 min | 2026-07-01T15:10 | 3 config schemas + User/Group edges |
| MVP-002 | [MVP-002-remove-limit-from-usage-schemas.md](./MVP-002-remove-limit-from-usage-schemas.md) | DONE | MVP-001 | ~20 min | 2026-07-01T15:12 | Field removed, ent regenerated, 3 new config pkgs |
| MVP-003 | [MVP-003-rewrite-repo-queries.md](./MVP-003-rewrite-repo-queries.md) | DONE | MVP-002 | ~20 min | 2026-07-01T15:18 | Config+usage join, latestXxxLimit removed |
| MVP-004 | [MVP-004-rewrite-repo-writes.md](./MVP-004-rewrite-repo-writes.md) | DONE | MVP-003 | ~20 min | 2026-07-01T15:18 | Config-table writes, simplified increment |
| MVP-005 | [MVP-005-update-service-port-accounting.md](./MVP-005-update-service-port-accounting.md) | DONE | MVP-004 | ~20 min | 2026-07-01T15:20 | Struct cleaned, GroupCandidateLimit removed |
| MVP-006 | [MVP-006-update-admin-services.md](./MVP-006-update-admin-services.md) | DONE | MVP-005 | ~20 min | 2026-07-01T15:22 | No changes needed — adapts naturally |
| MVP-007 | [MVP-007-update-cache-layer.md](./MVP-007-update-cache-layer.md) | DONE | MVP-005 | ~20 min | 2026-07-01T15:22 | No changes needed — transparent proxy |
| MVP-008 | [MVP-008-fix-quota-route-alias-bug.md](./MVP-008-fix-quota-route-alias-bug.md) | DONE | MVP-005 | ~20 min | 2026-07-01T15:27 | Fixed fallback chain: Model → UpstreamModel → routeAlias |
| MVP-009 | [MVP-009-fix-failover-bug.md](./MVP-009-fix-failover-bug.md) | DONE | none | ~20 min | 2026-07-01T15:15 | Re-resolve route when all accounts excluded |
| MVP-010 | [MVP-010-build-and-test.md](./MVP-010-build-and-test.md) | DONE | all | ~20 min | 2026-07-01T15:30 | Full build + all quota tests pass |

## Files Changed

| File | MVPs | Summary |
|---|---|---|
| `ent/schema/model_token_daily_limit_config.go` | MVP-001 | New config schema |
| `ent/schema/user_model_token_daily_limit_config.go` | MVP-001 | New config schema |
| `ent/schema/group_candidate_token_daily_limit_config.go` | MVP-001 | New config schema |
| `ent/schema/user.go` | MVP-001 | Added reverse edge for config |
| `ent/schema/group.go` | MVP-001 | Added reverse edge for config |
| `ent/schema/model_token_daily_usage.go` | MVP-002 | Removed `daily_limit_tokens` field |
| `ent/schema/user_model_token_daily_usage.go` | MVP-002 | Removed `daily_limit_tokens` field |
| `ent/schema/group_candidate_token_daily_usage.go` | MVP-002 | Removed `daily_limit_tokens` field |
| `ent/*.go` (generated) | MVP-002 | Ent regeneration |
| `internal/repository/daily_token_quota_repo.go` | MVP-003,004 | Full rewrite: config+usage |
| `internal/repository/daily_token_quota_repo_test.go` | MVP-010 | Updated for new architecture |
| `internal/service/daily_token_quota_port.go` | MVP-005 | Cleaned `DailyTokenQuotaIncrement` |
| `internal/service/daily_token_quota_accounting.go` | MVP-005,008 | Removed limit passing + fix Bug 1 fallback |
| `internal/service/daily_token_quota_accounting_test.go` | MVP-005 | Removed removed-field assertions |
| `internal/service/openai_group_route_quota.go` | MVP-009 | Added `excludedAccountIDs` param |
| `internal/handler/openai_chat_completions.go` | MVP-009 | Added route re-resolution logic |
| `internal/service/openai_group_route_quota_test.go` | MVP-009 | Updated call sites |
