# MVP-016: 成功记录 usage 后累加三类 Token 用量

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 复用统一增量接口，只改成功 usage 的后置路径和定向测试。
- Dependencies: `MVP-008`, `MVP-009`, `MVP-014`, `MVP-015`

## Outcome

一次成功计费记录以实际模型和 `UsageLog.TotalTokens()` 同步增加候选、全局模型、用户模型用量，失败请求不增加。

## Context

Claude 与 OpenAI 有各自 RecordUsage 路径；simple mode 与正常计费需要明确一致行为。

## In Scope

- 在 usage 持久化成功点调用 quota 增量服务。
- 传入 group/别名/实际模型/user/总 token。
- 定义增量失败的日志、重试或返回策略并测试。

## Out of Scope

- CountTokens 计入配额、历史回填。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] input+output+cache_creation+cache_read 全部计入。
- [ ] 上游失败或 usage 写入失败不累加。
- [ ] Claude 与 OpenAI 成功路径各恰好累加一次。

## Verification Plan

- `cd backend; go test ./internal/service -run 'RecordUsage.*TokenQuota|TokenQuotaAccounting'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


