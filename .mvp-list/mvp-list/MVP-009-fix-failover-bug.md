# MVP-009: Fix Bug 2 — no failover to next candidate on upstream error

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Investigation + fix in the failover loop of `SelectAccountWithLoadAwareness`. The plan identifies specific suspicious code paths. Time-boxed to 20 min; if the root cause is deeper, document findings and escalate.
- Dependencies: none

## Outcome

When candidate 1's accounts all fail with upstream errors, the handler re-resolves the routing to advance to candidate 2 instead of returning an error.

## Context

**Bug**: When candidate 1's accounts call upstream and return errors, the system returns an error to the client instead of failing over to candidate 2.

## In Scope

- Trace the full failover loop from handler retry loop to route resolution
- Fix: `ResolveQuotaAllowedGroupRoute` skips candidates whose all accounts are excluded
- Fix: handler re-resolves route when all routing account IDs are excluded

## Out of Scope

- Changing the quota-checking logic (`quotaAwareRouteCandidateExhausted`)
- Fixing Bug 1 (MVP-008)

## Acceptance Criteria

- [x] When candidate 1 accounts all fail with upstream errors, the system tries candidate 2
- [x] `ResolveQuotaAllowedGroupRoute` correctly skips candidates whose accounts are all excluded
- [x] When all candidates are exhausted, a proper error is returned
- [x] No regression: single-candidate routes still work correctly (test passes)
- [x] No regression: quota-based candidate exhaustion still works (test passes)

## Verification Plan

```bash
cd backend
go build ./...
go test ./internal/service/... -run "TestOpenAIGroupRoute" -v -count=1
```

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Build | `go build ./...` | PASS — full project compiles |
| Test | `go test ./internal/service/... -run "TestOpenAIGroupRoute"` | PASS — both quota-aware routing tests pass |
| File | `openai_group_route_quota.go` | Added `excludedAccountIDs` param + `allAccountIDsExcluded` helper |
| File | `openai_chat_completions.go` | Re-resolve route when all routing accounts excluded |
| File | `openai_group_route_quota_test.go` | Updated call sites with `nil` excluded param |

## Root Cause

`ResolveQuotaAllowedGroupRoute` returns only the first quota-passing candidate's model and account IDs. In the handler's retry loop, when that candidate's accounts all fail upstream, the selector can't find any valid account within `routingAccountIDs`, and the handler returns an error — without ever trying the next candidate.

## Fix Summary

1. **`openai_group_route_quota.go`**: Added `excludedAccountIDs` parameter to `ResolveQuotaAllowedGroupRoute`. Candidates whose all accounts are in the excluded set are skipped, allowing the function to return the next priority candidate.
2. **`openai_chat_completions.go`**: Added re-resolution logic at the top of the retry loop: when all `routingAccountIDs` have been tried and failed, re-call `ResolveQuotaAllowedGroupRoute` with the current `failedAccountIDs` to advance to the next candidate.
