# MVP-002: Remove daily_limit_tokens from usage schemas + ent code generation

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `~20 min`
- Estimate rationale: Three one-line deletions plus `go generate` run. Had to add reverse edges in User/Group schemas for new config tables.
- Dependencies: MVP-001

## Outcome

Three usage Ent schemas no longer have `daily_limit_tokens` field. Ent code regenerated with new config table packages. Compilation errors isolated to `daily_token_quota_repo.go` only.

## In Scope

- Delete `daily_limit_tokens` field from all 3 usage schema files ✅
- Run `go generate ./ent` ✅
- Add reverse edges in User/Group for new config schemas ✅

## Acceptance Criteria

- [x] `daily_limit_tokens` field removed from all 3 usage schema files
- [x] `go generate ./ent` completes without error
- [x] Generated Ent code no longer has `DailyLimitTokens` field on usage types (verified by build errors)
- [x] New config ent packages generated: modeltokendailylimitconfig, usermodeltokendailylimitconfig, groupcandidatetokendailylimitconfig
- [x] Known compilation errors catalogued: 10+ errors in daily_token_quota_repo.go only

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Schema build | `go build ./ent/schema/...` | PASS |
| Ent generate | `go generate ./ent` | PASS |
| Config packages | `ls ent/*dailylimitconfig/` | 3 packages generated |
| Full build | `go build ./...` | FAIL — errors only in daily_token_quota_repo.go (expected, fixed in MVP-003/004) |

## Execution Notes

- Had to add `edge.To("user_model_token_daily_limit_configs", ...)` in `user.go` and `edge.To("group_candidate_token_daily_limit_configs", ...)` in `group.go` — ent requires reverse edges when using `edge.From` with `Ref`.
