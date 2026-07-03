# MVP-015: Preserve Requested And Upstream Model Identity

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: Existing `ForwardResult.UpstreamModel` and usage fields already existed; this MVP only corrects grouped alias flow into upstream forwarding.
- Dependencies: `MVP-012`

## Outcome

Client-facing group model aliases remain the requested/display model, while the selected routed candidate model is carried as the actual upstream model and used in the upstream request payload.

## Context

The implementation checks `ForwardResult`, `ParsedRequest`, Anthropic/Chat Completions forwarding paths, and `optionalNonEqualStringPtr` usage persistence behavior.

## In Scope

- Carry request alias and actual upstream model in the selection result.
- Ensure upstream payloads use the actual model while response/usage display keeps the request alias.
- Add Anthropic/OpenAI-oriented regression coverage.

## Out of Scope

- Quota increment behavior and pricing-rule refactors.

## Implementation Notes

- Added `RequestedModel` and `UpstreamModel` to `AccountSelectionResult`.
- Added `ParsedRequest.UpstreamModel` as an optional upstream model override.
- Routed candidate selection now stamps the selected route alias and candidate model into the selection result.
- Anthropic `Forward` and API-key passthrough use `ParsedRequest.UpstreamModel` for the upstream wire body while preserving `ForwardResult.Model` as the requested alias.
- Chat Completions forwarding now keeps `parsed.Model` as the requested/display model even if the forwarded body has already been rewritten to the upstream model.

## Acceptance Criteria

- [x] The upstream request body `model` is the actual selected candidate model.
- [x] Usage `requested_model` remains the group alias/requested model.
- [x] Usage `upstream_model` is the actual model and still follows existing normalization when it equals the requested model.

## Verification Plan

- `cd backend; go test ./internal/service -run 'RequestedModel|UpstreamModel|GroupedModelIdentity'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Test | `cd backend; go test .\internal\service -run 'RequestedModel|UpstreamModel|GroupedModelIdentity' -v` | PASS; covered selected upstream model payload rewrite, existing requested/upstream usage tests, and OpenAI requested/upstream regressions. |
| Test | `cd backend; go test .\internal\service -run 'QuotaAwareRouting|GroupedModel.*Failover|UpstreamModel|RequestedModel' -v` | PASS; verified model identity together with quota-aware grouped routing and candidate failover. |
| Test | `cd backend; go test .\internal\handler -run 'GatewayModels|BillingErrorDetails_RoutedTokenQuotaExhausted' -v` | PASS; handler regressions still pass after selection identity changes. |
| Compile | `cd backend; go test -c -o .\.gotmp\cmd_server_mvp015_test.exe .\cmd\server` | PASS; server package compiled successfully. |
| Static check | `git diff --check` | PASS; only existing CRLF conversion warnings for MVP markdown/progress files were reported. |

## Execution Notes

- No pricing or quota increment logic was changed.
- The selected upstream model is kept as an override instead of mutating the requested model, so usage display and `optionalNonEqualStringPtr` normalization retain their existing semantics.
