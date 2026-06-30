# MVP-015: 贯通 requested model 与 upstream model 身份

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 现有 `ForwardResult.UpstreamModel` 和 usage 字段已存在，本任务仅校正分组别名流转。
- Dependencies: `MVP-012`

## Outcome

客户端分组模型名保持为 requested/display model，实际候选模型进入 UpstreamModel 并用于上游请求。

## Context

重点检查 `ForwardResult`、ParsedRequest、Claude/OpenAI 转发路径以及 `optionalNonEqualStringPtr`。

## In Scope

- 在选择结果中携带请求别名和实际模型。
- 确保上游 payload 使用实际模型，响应兼容展示使用请求别名。
- 补 Anthropic/OpenAI 定向测试。

## Out of Scope

- 配额累加、价格规则重构。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] upstream 请求体中的 model 是实际候选。
- [ ] usage requested_model 是分组别名。
- [ ] usage upstream_model 是实际模型且相等时遵循现有归一规则。

## Verification Plan

- `cd backend; go test ./internal/service -run 'RequestedModel|UpstreamModel|GroupedModelIdentity'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


