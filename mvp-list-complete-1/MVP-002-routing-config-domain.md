# MVP-002: 定义兼容新旧格式的模型路由领域类型

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 只引入纯领域类型和解析/排序测试，范围可在一次短测试循环内完成。
- Dependencies: `MVP-001`

## Outcome

后端能把旧的账号 ID 数组和新的候选对象统一解析为确定性的路由规则。

## Context

当前 `Group.ModelRouting`、Ent JSON 字段和 DTO 都是 `map[string][]int64`；适合把兼容解析集中到 `backend/internal/domain` 或同等纯逻辑位置。

## In Scope

- 定义候选的 model、account_ids、priority、daily_token_limit 类型。
- 解析旧格式并保留尾部 `*` 匹配行为；新格式按 priority 稳定升序。
- 补纯单元测试覆盖非法、空值和并列优先级。

## Out of Scope

- 修改数据库 schema、调度器或前端。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 旧 JSON 解析后仍得到原账号顺序。
- [x] 新 JSON 得到稳定排序的候选，0/null 限额归一为不限额。
- [x] 非法候选返回可识别校验错误且不 panic。

## Verification Plan

- `cd backend; go test ./internal/domain/...`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/domain/model_routing.go` | 新增兼容旧账号数组与新候选对象的解析、校验、稳定排序和精确/尾部通配匹配。 |
| Tests | `cd backend; go test ./internal/domain/...` | PASS（`ok github.com/Wei-Shaw/sub2api/internal/domain 1.323s`）。 |
| Hygiene | `git diff --check` | PASS；仅报告既有 `PLAN.md` 的 CRLF 提示，无空白错误。 |

## Execution Notes

- 旧格式被归一为单个 `Legacy` 候选；匹配时把实际请求模型填入 `Model`，从而保留旧通配规则的透传语义和账号顺序。
- 新格式候选的相同优先级保持 JSON 输入顺序；不同通配规则同时命中时优先最长前缀，再按规则名排序，避免 Go map 遍历造成不确定结果。
- 空白输入、空路由名、非尾部通配符、空/null 候选、无模型、无账号、非法账号 ID、负优先级和负限额均返回包装 `ErrInvalidModelRouting` 的错误；顶层 JSON `null` 视为空配置。
