# MVP-004: Rewrite repository — write methods (Set + Upsert + Increment)

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Three write methods rewritten together with MVP-003 (same file). Config-table writes, simplified increment.
- Dependencies: MVP-003

## Outcome

All write methods operate on config tables for limits and usage tables for tokens only. `IncrementDailyTokenQuotas` no longer writes limit columns.

## Acceptance Criteria

- [x] `SetModelDailyTokenQuota` writes to `model_token_daily_limit_configs`
- [x] `UpsertUserModelDailyTokenQuotas` writes to `user_model_token_daily_limit_configs` with stale cleanup
- [x] `IncrementDailyTokenQuotas` writes only `used_tokens` to all 3 usage tables
- [x] No `SetNillableDailyLimitTokens`, `ClearDailyLimitTokens` references remain
- [x] `go build ./internal/repository/...` succeeds

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Build | `go build ./internal/repository/...` | PASS |
| Full build | `go build ./...` | PASS |
