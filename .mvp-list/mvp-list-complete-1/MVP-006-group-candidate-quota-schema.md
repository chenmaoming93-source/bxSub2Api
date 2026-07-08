# MVP-006: 建立分组候选每日用量持久化

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 按 MVP-001 决策只实现一个候选用量存储切片。
- Dependencies: `MVP-001`

## Outcome

分组路由候选的 daily_token_limit 有不会与其他分组或别名串用的每日计数。

## Context

具体表名和唯一键以 MVP-001 记录的决策为准；需关联 group 并标识路由别名和实际模型。

## In Scope

- 新增候选用量 Ent schema、migration 和唯一索引。
- 实现级联删除或清理策略。
- 补结构和唯一性测试。

## Out of Scope

- 候选选择、用量写入和 UI。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 两个分组使用同一实际模型时用量互不影响。
- [x] 同分组不同路由别名的计数符合 MVP-001 决策。
- [x] migration 与 Ent schema 一致。

## Verification Plan

- `cd backend; go generate ./ent; go test ./migrations ./ent/...`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Ent generation | `cd backend; $env:GOPROXY='https://goproxy.cn,direct'; go generate ./ent` | PASS；生成候选日用量 entity 与 Group 关系代码。 |
| Focused migration test | `cd backend; go test ./migrations -run GroupCandidateTokenDailyUsages` | PASS（1.402s）。 |
| Focused schema/compile tests | `cd backend; go test ./ent/schema -run GroupCandidateTokenDailyUsage; go test ./ent/groupcandidatetokendailyusage ./ent -run GroupCandidateTokenDailyUsage` | PASS（schema 5.761s；生成包编译通过）。 |
| Broad verification limitation | `cd backend; go test ./migrations ./ent/...` | 延续 MVP-004 已记录的 4 个既有 migration 断言失败；本切片聚焦测试通过。 |
| Hygiene | `git diff --check` | PASS；仅有既有行尾提示。 |

## Execution Notes

- 唯一身份严格采用 MVP-001 的 `(group_id, route_alias, upstream_model, usage_date)`；因此不同分组、不同别名即使实际模型相同也不会串用。
- `group_id` 外键和 Ent 反向边使用级联删除；删除分组会清理候选日用量。
- 另建 `group_id` 查询索引；用量默认 0，限额可空且负数非法。
