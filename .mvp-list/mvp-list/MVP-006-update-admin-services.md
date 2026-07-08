# MVP-006: Update admin services (model + user quota admin)

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Thin wrappers around repo. No code changes needed — interfaces stable, delegate to repo correctly.
- Dependencies: MVP-005

## Outcome

Admin services work correctly with new config-table repo. No code changes required.

## Acceptance Criteria

- [x] `ModelTokenQuotaAdminService.List` correctly returns models from config + today's usage
- [x] `ModelTokenQuotaAdminService.Set` correctly writes limits to config table
- [x] `UserModelTokenQuotaAdminService.List` correctly returns user models from config + today's usage
- [x] `UserModelTokenQuotaAdminService.Upsert` correctly writes limits to config table
- [x] `go build ./...` succeeds — no changes needed
