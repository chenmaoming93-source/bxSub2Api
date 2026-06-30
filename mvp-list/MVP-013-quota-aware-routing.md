# MVP-013: Quota-Aware Candidate Routing

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: Add a unified preflight check to the existing candidate loop with focused stub tests.
- Dependencies: `MVP-007`, `MVP-009`, `MVP-012`

## Outcome

When a group candidate, global model, or current user's model daily token quota is exhausted, routing skips the current candidate and tries the next candidate. Infrastructure errors from the quota repository are returned as real errors and are not disguised as quota exhaustion.

## Context

Quota exhaustion is used for internal downgrade decisions. It must not directly return to the client while another candidate remains available.

## In Scope

- Inject the daily token quota repository into `GatewayService` and Wire.
- Run preflight in candidate, global model, and user model order before routed account selection.
- Cover the three exhaustion types and non-exhaustion repository errors with stub tests.

## Out of Scope

- Cross-model failover after an upstream account request fails.
- Usage accumulation after a completed request.

## Implementation Notes

- Added `GatewayService.dailyTokenQuotaRepo` and wired it from `repository.NewDailyTokenQuotaRepository`.
- Added `selectQuotaAllowedRouteCandidate` and `quotaAwareRouteCandidateExhausted` to keep candidate selection and quota classification testable.
- If all route candidates are quota-exhausted, the selection path returns `ErrNoAvailableAccounts` instead of falling back to ordinary account selection and bypassing quotas.

## Acceptance Criteria

- [x] All three exhaustion types skip the current candidate.
- [x] User A exhaustion does not affect user B.
- [x] Non-exhaustion infrastructure errors are not disguised as quota exhaustion.

## Verification Plan

- `cd backend; go test ./internal/service -run 'QuotaAwareRouting'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/service/gateway_service.go`; `backend/cmd/server/wire_gen.go` | Added daily token quota preflight to routed candidate selection and wired the repository into `GatewayService`. |
| Tests | `cd backend; go test .\internal\service -run QuotaAwareRouting -v` | PASS: group candidate, global model, and user model exhaustion skip current candidate; user-specific exhaustion is scoped; infrastructure errors remain ordinary errors. |
| Tests | `cd backend; go test .\internal\service -run "QuotaAwareRouting|ModelRouting|GroupedModelCandidate" -v` | PASS: quota-aware routing plus existing model routing compatibility. |
| Tests | `cd backend; go test -tags unit .\internal\service -run "QuotaAwareRouting|GroupedModelCandidate" -v` | PASS: unit-tag gateway candidate routing and quota-aware tests. |
| Tests | `cd backend; go test .\internal\handler -run GatewayModels -v` | PASS: constructor signature changes did not break gateway models handler tests. |
| Compile | `cd backend; go test -c -o .\.gotmp\cmd_server_quotaaware_test.exe .\cmd\server` | PASS: manual Wire graph update compiles. |

## Execution Notes

- `go generate ./cmd/server` remains impractical in this environment because prior runs of `wire.exe` were blocked by Windows Application Control, so `wire_gen.go` was updated manually and compile-checked.
