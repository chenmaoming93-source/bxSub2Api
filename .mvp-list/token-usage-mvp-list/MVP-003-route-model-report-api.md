# MVP-003：交付模型路由统计 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-001`

## 预期成果

管理员可分页查看一个分组路由别名下各候选上游模型的每日用量。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。历史用量以 `group_candidate_token_daily_usages` 为准；当前配置缺失时字段可空，不得丢弃历史行。

## 范围内

- 实现路由候选统计查询及汇总
- 关联 groups 和当前候选限额/优先级，兼容历史候选已删除
- 注册 `GET /api/v1/admin/token-usage/routes`
- 覆盖必选分组/别名、可选上游模型和分页排序测试

## 范围外

- 不实现路由选项接口和页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 指定 `group_id + route_alias` 可获得候选模型分页结果
- [x] 可用 `upstream_model` 进一步筛选
- [x] 历史候选配置缺失时用量仍可见
- [x] 相关 Go 测试通过

## 验证计划

- `cd backend; go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `token_usage_report_{repo,service,handler}.go` | 路由日报、筛选、分页、汇总、当前限额与 JSON 路由优先级 |
| 历史兼容 | `TestTokenUsageReportRepositoryRouteKeepsHistoricalCandidate` | 当前限额和候选配置缺失时历史行仍返回 |
| 完整验收 | `go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...` | 全部通过 |

## 执行记录

以用量表为主表并对 groups、限额配置和 JSON 候选配置使用 `LEFT JOIN`；历史候选的优先级和限额允许为空。
