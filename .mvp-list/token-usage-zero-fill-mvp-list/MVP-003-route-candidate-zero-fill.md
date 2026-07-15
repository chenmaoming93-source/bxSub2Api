# MVP-003：分组路由候选每日零用量补全

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `路由维度包含候选模型并集、可选上游模型筛选、分组信息和优先级关联，适合作为一个独立垂直切片。`
- Dependencies: `MVP-001`

## 预期成果

分组路由当前定义限额的候选模型在查询范围内每天可见，零调用日期显示 `0`，同时保留无配置但有历史用量的候选组合。

## 背景

当前 `ListRouteTokenUsage` 以 `group_candidate_token_daily_usages` 为主表。限额配置存在但候选当天无调用时，报表缺少该候选的日期行。

## 范围内

- 改造分组、路由、上游模型组合的日期补零查询。
- 支持指定或不指定 `upstream_model`。
- 保持分组名称、当前路由优先级及限额关联。
- 添加多候选、零用量、无限额、历史候选、筛选、排序和分页测试。

## 范围外

- 修改模型路由配置结构或候选选择算法。
- 修改限额配置同步、Token 入库或数据库 Schema。
- 修改 API 和前端页面结构。

## 实现说明

- 目标候选集合为当前限额配置候选和日期范围内实际用量候选的并集。
- 未指定上游模型时，对每个候选与每个日期生成报表行；指定时两侧统一过滤。
- `groups.model_routing` 的当前优先级仅作为展示信息关联，无法解析或历史候选不存在时允许为 `null`。
- 汇总、排序和分页在补全后的结果上执行。

## 验收标准

- [x] 已配置候选没有调用时仍按查询日期逐日显示 `0`。
- [x] 多候选查询生成正确的候选数乘日期数记录。
- [x] 指定上游模型时只返回目标候选。
- [x] 无配置历史候选仍显示实际用量，限额与优先级允许为 `null`。
- [x] 总数、Token 合计、排序和分页正确。

## 验证计划

- `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryRoute"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_usage_report_repo.go` | 路由候选配置与范围内用量组成并集，逐日补零并保留分组名称和当前路由优先级关联。 |
| 测试 | `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryRoute" -count=1` | 通过：覆盖多候选乘日期、零用量、显式零限额、指定候选、历史候选、汇总排序及分页。 |
| 差异检查 | `git diff --check -- internal/repository/token_usage_report_repo.go internal/repository/token_usage_report_repo_test.go` | 通过，无空白错误。 |

## 执行记录

- `2026-07-07T15:06:35+08:00`：实现并验证分组路由候选按日补零与可选上游模型筛选。
