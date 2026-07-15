# MVP-001：冻结统计查询契约并验证索引

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `none`

## 预期成果

形成可直接实现的三维统计查询契约、日期口径和索引决策。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。重点检查 `backend/ent/schema/*token_daily*`、`backend/internal/repository/daily_token_quota_repo.go` 和现有 migration；最大日期范围暂不在此强制，记录性能证据。

## 范围内

- 核对六张配额/用量表、软删除用户与当前路由配置的关联语义
- 定义分页、排序、汇总、使用率、状态及错误边界
- 为三类核心查询与默认目标查询编写 SQL/Repository 测试样例并执行 EXPLAIN
- 仅在证据表明必要时新增 migration 索引

## 范围外

- 不实现 HTTP API 或页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 三类统计与默认目标查询的输入、输出、排序和时区口径有自动化测试或书面代码契约
- [x] EXPLAIN 证据已记录，新增索引均有理由且无明显重复
- [x] `go test ./internal/repository/...` 通过

## 验证计划

- `cd backend; go test ./internal/repository/...`
- `对三类代表性 SQL 执行 EXPLAIN 并记录结果`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 查询契约 | `backend/internal/repository/token_usage_report_contract.go` | 固化日期范围、分页 1..100、排序白名单、稳定次序及三类有界 SQL |
| 自动化测试 | `cd backend; go test ./internal/repository -run 'TestTokenUsage' -count=1` | 通过；包含真实执行的 SQLite `EXPLAIN QUERY PLAN`，六个代表性查询均命中索引 |
| 索引审计 | `backend/migrations/159_token_usage_report_indexes.sql` | 新增日期默认目标、路由无候选模型、用户无模型查询所需索引；未复制现有完整目标唯一键 |
| Migration 测试 | `cd backend; go test ./migrations -run 'TestTokenUsageReportIndexes' -count=1` | 通过 |
| 完整 Repository 验收 | `cd backend; go test ./internal/repository/...` | 通过（`ok github.com/Wei-Shaw/sub2api/internal/repository`） |
| 完整 Migration 验收 | `cd backend; go test ./migrations/...` | 通过（`ok github.com/Wei-Shaw/sub2api/migrations`） |

## 执行记录

- 2026-07-06：开始实现并完成查询契约、索引 migration 与定向测试。
- 日期按项目全局时区解析为闭区间 `DATE`；响应总量口径保持 `used_tokens` 求和，限额状态留给后续 Service 计算。
- `used_tokens` 排序允许数据库排序，但始终受目标、日期和 `LIMIT <= 100` 约束；以 `id ASC` 作为稳定次序兜底。
- 2026-07-06：仅更新过时测试断言以适配当前 GoldenDB/MySQL 方言和 `Exec + LastInsertId` 写入路径；未修改生产代码。完整 Repository 与 migration 测试均已通过，阻塞解除。
