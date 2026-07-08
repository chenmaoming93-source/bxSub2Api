# MVP-004: 建立全局模型每日配额持久化

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 单表 Ent schema、migration 与结构测试是一个小而完整的持久化切片。
- Dependencies: `MVP-001`

## Outcome

数据库可保存每个实际上游模型的每日 Token 上限与原子累加所需字段。

## Context

新增 `model_token_daily_usages`；日期遵循项目全局 `timezone.StartOfDay` 口径。

## In Scope

- 新增 Ent schema 和 SQL migration。
- 建立 model + usage_date 唯一约束、used_tokens 默认值及 daily_limit_tokens 可空字段。
- 生成 Ent 代码并加 migration/schema 回归测试。

## Out of Scope

- repository、缓存、管理 API。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] migration 可重复执行并创建预期字段和唯一索引。
- [x] `used_tokens` 默认 0，限额可表达不限额。
- [x] Ent 生成物与 schema 一致。

## Verification Plan

- `cd backend; go generate ./ent; go test ./migrations ./ent/...`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Ent generation | `cd backend; $env:GOPROXY='https://goproxy.cn,direct'; go generate ./ent` | PASS；生成 `ModelTokenDailyUsage` entity、builder、query、mutation 与 schema 元数据。 |
| Focused migration test | `cd backend; go test ./migrations -run ModelTokenDailyUsages` | PASS（1.279s）；验证重复建表、字段、默认值及内联唯一约束。 |
| Focused schema/compile tests | `cd backend; go test ./ent/schema -run ModelTokenDailyUsage; go test ./ent/modeltokendailyusage ./ent -run ModelTokenDailyUsage` | PASS（schema 3.096s；生成包编译通过）。 |
| Broad verification | `cd backend; go test ./migrations ./ent/...` | 新增 Ent/schema 测试通过；migrations 包被既有 112、119、134、151 SQL 与断言不一致阻断，均与 migration 154 无关。 |
| Hygiene | `git diff --check` | PASS；仅有既有 `PLAN.md` 与生成依赖 `go.sum` 行尾提示。 |

## Execution Notes

- 新表键为 `(model, usage_date)`，其中 `model` 是实际上游模型；唯一约束内联在 `CREATE TABLE IF NOT EXISTS`，避免重复执行时单独创建索引失败。
- `used_tokens BIGINT NOT NULL DEFAULT 0`；`daily_limit_tokens BIGINT NULL`，NULL/0 均遵循 MVP-001 的不限额语义，并对负值增加 DB/Ent 校验。
- `usage_date` 使用 `DATE`，调用层后续负责从项目 `timezone.StartOfDay` 生成对应日历日期。
