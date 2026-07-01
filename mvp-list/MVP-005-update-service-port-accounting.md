# MVP-005: Update service port + accounting layer

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Removed unused struct fields and accounting code. Build passes.

## Outcome

`DailyTokenQuotaIncrement` struct cleaned up. Accounting function no longer passes limits — repo reads from config tables.

## Acceptance Criteria

- [x] `ModelDailyLimitTokens` and `UserModelDailyLimitTokens` removed from struct
- [x] `GroupCandidateDailyLimitTokens` removed (repo no longer uses it)
- [x] `incrementDailyTokenQuotasForUsage` no longer sets these fields
- [x] `go build ./...` succeeds
