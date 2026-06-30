# MVP-012: Priority-Based Group Model Candidate Routing

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: Connect the already parsed candidate routing shape to the selection entry point without adding token quota checks.
- Dependencies: `MVP-002`, `MVP-003`

## Outcome

When a request model matches the new group alias format, the scheduler uses the lowest-priority candidate's actual model and account ID set during account filtering, while legacy wildcard/account-ID routing remains compatible.

## Context

The entry point is `GatewayService.SelectAccountWithLoadAwareness`; legacy behavior continues through `GetRoutingAccountIDs`.

## In Scope

- Recognize exact group aliases and construct ordered model route candidates.
- Pass the candidate actual model and account IDs into the existing account filtering path.
- Preserve legacy wildcard routing behavior and add unit tests.

## Out of Scope

- Token quota checks and usage writes.

## Implementation Notes

- Added `Group.GetRoutingCandidates(requestedModel)` to expose the ordered candidate objects parsed by the domain routing parser.
- Kept `Group.GetRoutingAccountIDs(requestedModel)` as a compatibility wrapper returning the first matched candidate's account IDs.
- Updated the load-aware gateway routing layer to use the selected candidate model for model support and model schedulability checks.

## Acceptance Criteria

- [x] New aliases select the lowest-priority candidate first.
- [x] Routing only considers schedulable accounts from the selected candidate's `account_ids`.
- [x] Legacy routing tests continue to pass.

## Verification Plan

- `cd backend; go test ./internal/service -run 'ModelRouting|GroupedModelCandidate'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/service/group.go`; `backend/internal/service/gateway_service.go` | Added ordered candidate access and routed selection with candidate actual model filtering while preserving `GetRoutingAccountIDs` compatibility. |
| Tests | `cd backend; go test .\internal\service -run "ModelRouting|GroupedModelCandidate" -v` | PASS: lowest-priority candidate selection and legacy wildcard account ID compatibility. |
| Tests | `cd backend; go test -tags unit .\internal\service -run GroupedModelCandidate -v` | PASS: gateway selection uses the candidate actual model for account model filtering. |

## Execution Notes

- Token quota checks are intentionally deferred to MVP-013.
