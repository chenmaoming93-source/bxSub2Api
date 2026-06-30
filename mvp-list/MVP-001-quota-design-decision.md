# MVP-001: 确定每日 Token 配额的数据模型与错误语义

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 限时核对计划中的持久化缺口，输出一份可直接编码的决策，不涉及实现。
- Dependencies: none

## Outcome

在 `PLAN.md` 的约束下，明确分组候选用量键、跨日额度继承方式、0/null 语义以及全候选耗尽时的 HTTP 错误映射。

## Context

计划只列出全局模型和用户模型两张每日用量表，但还要求分组候选限额；`user_platform_quotas` 现有实现把配置与窗口用量放在同一活跃记录中。

## In Scope

- 检查 `backend/ent/schema/user_platform_quota.go`、`backend/internal/service/gateway_service.go` 与 migration 约定。
- 把结论记录到本文件 Execution Notes，并列出最终表键与错误优先级。

## Out of Scope

- 编写 migration、repository 或 UI。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 分组候选额度可唯一标识 group、路由别名和实际上游模型。
- [x] 明确新的一天如何继承额度以及 `0`/`null` 的统一含义。
- [x] 明确候选全耗尽时 429 与不可调度时 503 的映射。

## Verification Plan

- 人工复核本文件 Execution Notes 与 `PLAN.md` 的三类限额及 failover 要求逐项一致。

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Repository inspection | `backend/ent/schema/user_platform_quota.go`、`backend/ent/schema/group.go`、`backend/internal/service/gateway_service.go` | 已核对现有配额窗口、旧路由 JSON 类型和调度错误路径；确认新 Token 配额必须单独固化语义。 |
| Migration convention | `backend/migrations/README.md`、`backend/migrations/142_user_platform_quotas.sql` | 已确认使用 forward-only 新 migration、每日行唯一键与项目时区日期口径。 |
| Manual verification | 本文件 Execution Notes 对照 `PLAN.md` Summary、Key Changes、Assumptions | 三类限额、候选 failover、`0`/`null` 和 429/503 规则逐项一致。 |

## Execution Notes

### 最终数据键

- 全局模型每日用量行：`(model, usage_date)` 唯一；`model` 是实际上游模型名。
- 用户模型每日用量行：`(user_id, model, usage_date)` 唯一；`model` 是实际上游模型名。
- 分组候选每日用量行：`(group_id, route_alias, upstream_model, usage_date)` 唯一。这里 `route_alias` 是客户端请求的分组模型名，`upstream_model` 是候选实际模型；该组合可稳定区分同一模型在不同分组或路由别名下的候选额度。
- `usage_date` 由 `timezone.StartOfDay(now)` 所在项目全局时区的日历日期产生，不按用户时区拆分。

### 跨日与限额语义

- 三类表的用量按 `usage_date` 每日新建，`used_tokens` 从 `0` 开始，不继承前一日用量。
- 全局/用户模型限额在创建新日行时继承该维度最近一次已配置的 `daily_limit_tokens`；管理接口更新限额后，该值同时成为随后日期的配置来源。分组候选限额以当前 `groups.model_routing` 候选配置为来源，新日行快照当时的配置。
- 新增的三类 Token 配额统一规定：`daily_limit_tokens` 为 `NULL` 或 `0` 都表示不限额，正数表示上限；负数非法。该规则仅适用于本计划的新 Token 配额，不改变现有 `user_platform_quotas` 中 `0` 表示禁用的 USD 配额语义。

### 最终错误优先级

1. 逐候选执行配额检查与账号调度；任一候选成功即不返回终局错误。
2. 若所有候选都仅因分组候选、全局模型或用户模型 Token 配额耗尽而被淘汰，返回 HTTP `429`（可重试但需等待配额窗口或配置变化）。
3. 若至少一个候选通过 Token 配额检查、但账号可调度性、平台/模型映射、模型限流、账号 quota、window cost、RPM、并发或上游失败导致最终无候选可用，返回 HTTP `503`。混合出现配额耗尽与非配额不可调度时也以 `503` 为准，避免把服务可用性故障误报为纯配额问题。
