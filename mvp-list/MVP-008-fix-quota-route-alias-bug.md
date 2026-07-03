# MVP-008: Fix Bug 1 — quota recorded to route alias instead of upstream model

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Found the actual bug — accounting defaulted to `routeAlias` when `UpstreamModel` equals `Model`. Fixed with 3-line change.
- Dependencies: MVP-005

## Outcome

Fixed the root cause: when `UsageLog.UpstreamModel` is nil (because `optionalNonEqualStringPtr` returns nil when upstream model equals request model), the accounting code now falls back to `usageLog.Model` instead of `routeAlias`.

## Root Cause

In `daily_token_quota_accounting.go`, the `upstreamModel` variable defaulted to `routeAlias`:
```go
upstreamModel := routeAlias
```
When `UpstreamModel` equals `Model`, `optionalNonEqualStringPtr` returns nil, so the override never triggered. This caused quota to be recorded against the route alias instead of the actual model.

## Fix

Changed default to `usageLog.Model` with `routeAlias` as last-resort fallback:
```go
upstreamModel := strings.TrimSpace(usageLog.Model)
if usageLog.UpstreamModel != nil && strings.TrimSpace(*usageLog.UpstreamModel) != "" {
    upstreamModel = strings.TrimSpace(*usageLog.UpstreamModel)
}
if upstreamModel == "" {
    upstreamModel = routeAlias
}
```

## Acceptance Criteria

- [x] `usageLog.UpstreamModel` correctly used when non-nil
- [x] Falls back to `usageLog.Model` when `UpstreamModel` is nil
- [x] Last-resort fallback to `routeAlias` when Model is also empty
- [x] All quota accounting tests pass

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Test | `go test ./internal/service/... -run "Quota|TokenQuota"` | ALL PASS |
| File | `daily_token_quota_accounting.go:32-38` | Fixed fallback chain |
