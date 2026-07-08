# MVP-007: Update cache layer

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Transparent proxy. Cache keys, snapshot fields unchanged. Base repo assembly works correctly.
- Dependencies: MVP-005

## Outcome

Cache layer works correctly with new config-table base repo. No code changes required.

## Acceptance Criteria

- [x] Cache miss → base repo → snapshot with config+usage works correctly
- [x] `IncrementDailyTokenQuotas` via cache correctly increments base + cache
- [x] Cache invalidation targets unchanged and correct
- [x] No stale ent field references
- [x] `go build ./...` succeeds — no changes needed
