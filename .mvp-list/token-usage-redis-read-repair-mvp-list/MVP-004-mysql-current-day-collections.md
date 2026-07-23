# MVP-004：提供 MySQL 当天集合与历史快速路径契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 聚焦 Repository 查询边界和调用次数测试，不混入 Redis 或具体报表编排，约 40 分钟可验收。
- Dependencies: `MVP-001`

## 预期成果

报表服务可以一次性获得符合筛选条件的 MySQL 当天 usage/config 元数据集合，同时保留不含今天查询的现有数据库聚合、排序和分页路径。

## 背景

`backend/internal/repository/token_usage_report_repo.go` 当前把日期展开、聚合、排序和分页封装在单个方法内，混合查询需要可复用的当天集合而不能在 Redis miss 后逐项点查。

## 范围内

- 扩展 `ModelTokenUsageRepository` 或拆分等价端口，提供三维度当天集合查询。
- 一次查询组合 usage、limit config 及必要用户/分组元数据。
- 保留现有历史报表方法及 SQL 行为。
- 为筛选不存在项返回空集合、无额外点查增加仓储契约测试。

## 范围外

- 不读取 Redis。
- 不实现服务层合并、排序和分页。
- 不改变数据库 schema。

## 实现说明

- 不要删除现有 SQL 快速路径。
- Repository 方法命名需清晰区分“已分页报表”和“未分页/候选集合”。

## 验收标准

- [x] 三维度均有可批量取得当天候选集合的仓储能力。
- [x] 不存在筛选项返回空集合且不触发二次点查。
- [x] 现有历史报表仓储测试继续通过。
- [x] 无数据库迁移。

## 验证计划

- `cd backend && go test ./internal/repository -run 'Test.*TokenUsageReport|Test.*Today.*Usage'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 端口 | `backend/internal/service/token_usage_report_service.go` | 新增独立 `TodayTokenUsageRepository`，保留原分页报表接口不变。 |
| 实现 | `backend/internal/repository/token_usage_report_repo.go` | 三维度各使用一次集合 SQL 联合 usage/config，并补齐用户、分组、优先级元数据。 |
| 测试 | `backend/internal/repository/token_usage_today_repo_test.go` | 验证每个维度单次查询及不存在筛选返回空集合且无后续点查。 |
| 聚焦验证 | `cd backend && go test ./internal/repository -run 'Test.*TokenUsageReport|Test.*Today.*Usage'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.367s`。 |
| 包回归 | `cd backend && go test ./internal/repository` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 13.174s`。 |
| Schema | `backend/migrations`、`backend/sqlArchiving` | 未新增或修改数据库迁移/DDL。 |

## 执行记录

2026-07-15 完成。通过独立端口明确区分历史分页快速路径与当天未分页候选集合；当天查询不读取 Redis，也不在空结果后执行额外 MySQL 点查。首次测试构建因 sqlmock 行值类型需为 `driver.Value` 失败，修正夹具类型后通过。
