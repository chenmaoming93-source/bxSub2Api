# MVP-005: 建立用户模型每日配额持久化

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 单表 schema、migration 和约束测试，避免与全局表实现混成超时任务。
- Dependencies: `MVP-001`

## Outcome

数据库可按用户、实际上游模型和日期保存每日 Token 上限与用量。

## Context

新增 `user_model_token_daily_usages`，并对 users 使用级联删除或项目既有一致策略。

## In Scope

- 新增 Ent schema 与 SQL migration。
- 建立 user_id + model + usage_date 唯一约束和查询索引。
- 生成 Ent 代码并验证外键/默认值。

## Out of Scope

- repository、管理员接口和 UI。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 同一用户/模型/日期不能出现重复行。
- [x] 不同用户互不共享用量。
- [x] 删除用户后的行为符合既有外键约定。

## Verification Plan

- `cd backend; go generate ./ent; go test ./migrations ./ent/...`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Ent generation | `cd backend; $env:GOPROXY='https://goproxy.cn,direct'; go generate ./ent` | PASS；生成用户模型日用量 entity 及 User 关系代码。 |
| Focused migration test | `cd backend; go test ./migrations -run UserModelTokenDailyUsages` | PASS（1.426s）；验证复合唯一键、用户索引及级联外键。 |
| Focused schema/compile tests | `cd backend; go test ./ent/schema -run UserModelTokenDailyUsage; go test ./ent/usermodeltokendailyusage ./ent -run UserModelTokenDailyUsage` | PASS（schema 5.394s；生成包编译通过）。 |
| Broad verification limitation | `cd backend; go test ./migrations ./ent/...` | 延续 MVP-004 已记录限制：既有 112、119、134、151 migration 断言失败；本 MVP 聚焦测试与 Ent 包通过。 |
| Hygiene | `git diff --check` | PASS；仅有既有行尾提示。 |

## Execution Notes

- `(user_id, model, usage_date)` 唯一键把每个用户的每个实际上游模型每日窗口隔离；另建 `user_id` 查询索引。
- `user_id` 外键采用 `ON DELETE CASCADE`，并在 Ent 的 User 反向边上声明 Cascade，和现有用户拥有型记录策略一致。
- 用量/限额字段沿用 MVP-004：用量默认 0、限额可空且负数非法。
