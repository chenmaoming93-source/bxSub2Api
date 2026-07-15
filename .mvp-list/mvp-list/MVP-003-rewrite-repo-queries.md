# MVP-003: Rewrite repository — query methods (Get + List)

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Six query methods rewritten together with MVP-004 (same file). Config+usage join pattern, helper removal.
- Dependencies: MVP-002

## Outcome

All read-side methods source `DailyLimitTokens` from independent config tables. `latestModelLimit` and `latestUserModelLimit` helpers removed.

## Acceptance Criteria

- [x] `GetModelDailyTokenQuota` returns limit from config + used_tokens from usage
- [x] `GetUserModelDailyTokenQuota` returns limit from config + used_tokens from usage
- [x] `GetGroupCandidateDailyTokenQuota` returns limit from config + used_tokens from usage
- [x] `ListModelDailyTokenQuotas` returns all models with configs + today's usage
- [x] `ListUserModelDailyTokenQuotas` returns all user models with configs + today's usage
- [x] `latestModelLimit` and `latestUserModelLimit` helpers removed
- [x] No references to removed `DailyLimitTokens` field on usage types

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Build | `go build ./internal/repository/...` | PASS |
| Full build | `go build ./...` | PASS |
