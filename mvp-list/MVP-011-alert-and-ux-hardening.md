# MVP-011: 接入模型级告警并完善可访问性

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 复用现有 Ops 告警引擎和事件表，补模型/账号维度与 UI 链接，不重做通知基础设施。
- Dependencies: `MVP-005`, `MVP-009`

## Outcome

观测页可展示账号/路由级容量与质量告警，告警可深链到对应对象，并通过既有 Ops 通知配置发送。

## Context

`ops_alert_rules`、`ops_alert_events`、通知配置和 evaluator 已存在；当前设计不应另建告警系统。需要增加对 account/group/route_alias 指标的规则输入。

## In Scope

- 扩展现有 alert evaluator 的指标/维度映射，支持有效并发饱和、最终失败率、TTFT P95。
- 告警事件带 group_id、route_alias、account_id 等可定位字段，账号节点携带唯一绑定模型。
- 观测页显示待处置告警、级别、样本数、数据时间和深链。
- 3 个连续采样窗口去抖；业务限制与 SLA 错误分级。
- 键盘/ARIA、图例文本、色彩对比和窄屏验证。

## Out of Scope

- 自动修改路由。
- 新通知渠道。
- 删除现有告警配置的兼容行为。

## Implementation Notes

- 沿用现有 alert rule/event 表和通知配置；新增指标 code 与维度只做向前兼容。
- 同一对象重复事件合并/抑制，不让 5 秒采样洪泛通知。
- 告警详情可导航到 `/admin/observability?...` 并携带 canonical filters。
- 颜色状态都配文字与图标；键盘可操作深链和筛选。

## Acceptance Criteria

- [x] 无账号维度的旧告警规则继续工作。
- [x] 连续三窗越限触发，单窗毛刺不触发；恢复后关闭。
- [x] 告警事件能定位到至少 group/route/account 三元组。
- [x] 通知复用现有配置，数据库没有重复告警表。
- [x] 观测页告警深链能恢复 filters 和对象。
- [x] 主要颜色对比度和键盘操作通过人工检查。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service ./internal/repository ./internal/handler
cd ..
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run -- src/views/admin/ops
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Legacy compatibility | `backend/internal/service/ops_alert_evaluator_service.go` | Existing metric switch and old rule state path remain unchanged when metric is not prefixed `observability_` |
| New metrics | Optional `OpsObservabilityAlertMetricReader` | Client failure, business-limited, SLA error, TTFT P95 and effective concurrency saturation |
| Debounce | `observability.RequiredAlertEvaluations` + test | PASS: observability rules require at least 3 consecutive evaluation windows; existing recovery still resolves active event on first non-breach |
| Scope | Rule create/update validation | New observability rules require group_id, route_alias and account_id; invalid rules return `OBSERVABILITY_ALERT_SCOPE_REQUIRED` |
| Event dimensions | Evaluator event creation | Adds group_id, route_alias, account_id, sample_count and data_at to the existing `ops_alert_events.dimensions` JSON |
| Notification reuse | Existing `maybeSendAlertEmail` + Ops notification config | No new alert/event/notification tables or channels |
| Actual metric verification | Temporary local MariaDB verifier | PASS: failure rate 3.00 with 100 samples and concurrency saturation 90.00 for scoped triple |
| Alert UI | `ObservabilityAlertSummary.vue` | Firing model-scoped alerts show severity text, object tuple, metric, samples and data time; keyboard button opens filters |
| Deep link | `ModelObservabilityView.vue` | Alert click writes `/admin/observability` canonical query and route watcher restores user/group/alias/account/department filters |
| Rule editor | `OpsAlertRulesCard.vue` | New metric group plus route_alias/account_id fields; legacy metric options remain |
| Deep-link tests | `ObservabilityAlertSummary.spec.ts`, `ModelObservabilityView.spec.ts` | PASS: system-only alerts are excluded and canonical object filters are emitted/restored |
| Accessibility/contrast | Responsive screenshots and component markup | Buttons/labels/tab roles, text severity badges, keyboard Enter handlers and dark-mode select contrast verified |
| Unit tests | `go test -tags=unit ./internal/service/observability`; frontend focused Vitest | PASS |
| Build | `go build -p 1 ./cmd/server` | PASS |
| Known limitation | Full `./internal/service` unit package | Existing test mocks predate `AccountRepository.ExistsByName`; production build and new pure evaluator timing helper tests pass |

## Execution Notes

如果现有 alert evaluator 无法在不破坏规则兼容性的前提下扩展，先限制为只读事件展示，并把扩展另立 MVP。
