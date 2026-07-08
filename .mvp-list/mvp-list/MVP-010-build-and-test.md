# MVP-010: Build verification and run tests

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Full build + test suite. Fixed test files for new architecture. All quota-related tests pass.
- Dependencies: MVP-001 through MVP-009

## Outcome

Full build passes. All quota-related tests pass. 4 new repo tests added for config+usage architecture. Bug 1 definitively fixed with corrected fallback chain.

## Acceptance Criteria

- [x] `go build ./...` succeeds
- [x] `go vet ./...` passes for changed packages
- [x] All quota/token tests pass (30+ tests)
- [x] New config table ent packages generated correctly
- [x] No stale references to removed `daily_limit_tokens` column
- [x] Bug 1 root cause found and fixed

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Build | `go build ./...` | PASS |
| Repo tests | `go test ./internal/repository/... -run "TestGetModel\|TestIncrement"` | PASS (4 tests) |
| Service tests | `go test ./internal/service/... -run "Quota\|TokenQuota"` | ALL PASS |
| Route tests | `go test ./internal/service/... -run "TestOpenAIGroupRoute"` | PASS (2 tests) |
