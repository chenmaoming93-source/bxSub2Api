# MVP-014: Candidate Account And Cross-Model Failover

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: Focus on selection loop exit conditions and error priority without changing upstream forwarding or usage accounting.
- Dependencies: `MVP-013`

## Outcome

Candidate account failure or exclusion is handled inside the routed candidate boundary first. When one routed model candidate has no selectable accounts, routing tries the next candidate by priority. If every routed candidate is blocked only by daily token quota, the selection error maps to HTTP 429; account schedulability exhaustion remains the existing no-available-account 503 path.

## Context

The implementation preserves existing concurrency, RPM, window cost, model mapping, platform, account health, sticky session, and wait-plan checks.

## In Scope

- Make `excludedIDs` cooperate with routed candidate boundaries.
- Distinguish all quota exhausted from all accounts unschedulable.
- Add mixed failure reason and multi-candidate tests.

## Out of Scope

- Refactoring the retry strategy after an upstream request has already been sent.

## Implementation Notes

- Added `ErrRoutedTokenQuotaExhausted` for all routed candidates skipped by daily token quota.
- Changed Anthropic routed selection to iterate priority-ordered `ModelRouteCandidate` entries and attempt account selection per candidate.
- Extracted routed account filtering, sticky selection, load-aware acquisition, and wait-plan selection into `trySelectRouteCandidateAccounts`.
- Mapped `ErrRoutedTokenQuotaExhausted` to HTTP 429/rate-limit responses in Anthropic Messages and Chat Completions selection failures.

## Acceptance Criteria

- [x] While the same candidate still has an available account, selection does not cross to the next model prematurely.
- [x] After a candidate is exhausted, selection tries the next model by priority.
- [x] All quota exhausted maps to 429, while unschedulable accounts keep the existing 503 no-available path.

## Verification Plan

- `cd backend; go test ./internal/service -run 'GroupedModel.*Failover|QuotaExhausted'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Test | `cd backend; go test .\internal\service -run 'GroupedModel.*Failover|QuotaExhausted' -v` | PASS; covered same-candidate account failover, next-candidate failover, and unschedulable no-selection behavior. |
| Test | `cd backend; go test .\internal\service -run 'QuotaAwareRouting|GroupedModel.*Failover|QuotaExhausted' -v` | PASS; verified quota-aware routing regressions plus MVP-014 failover tests. |
| Test | `cd backend; go test .\internal\handler -run 'BillingErrorDetails_RoutedTokenQuotaExhausted|BillingErrorDetails_T10|BillingErrorDetails_Maps' -v` | PASS; verified routed token quota exhaustion maps to HTTP 429 and existing quota/RPM mappings still pass. |
| Compile | `cd backend; go test -c -o .\.gotmp\cmd_server_mvp014_test.exe .\cmd\server` | PASS; server package compiled successfully. |
| Static check | `git diff --check` | PASS; only existing CRLF conversion warnings for MVP markdown/progress files were reported. |

## Execution Notes

- `go generate ./cmd/server` was not needed for this MVP; generated wiring was unchanged by these edits.
- The selection implementation intentionally returns no-available-account when routed candidates exist but none has schedulable accounts, preserving the existing 503 semantics for capacity/schedulability exhaustion.
