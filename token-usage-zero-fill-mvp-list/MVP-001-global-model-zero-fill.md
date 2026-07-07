# MVP-001：全局模型每日零用量补全

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `包含公共日期序列查询骨架、全局模型查询改造及独立 Repository 测试，范围可在一个开发日内闭环。`
- Dependencies: `none`

## 预期成果

全局模型存在限额配置时，查询日期范围内每天都有报表行；缺少实际调用的日期显示 `used_tokens=0`，同时保留无配置但有用量的数据。

## 背景

当前 `backend/internal/repository/token_usage_report_repo.go` 以 `model_token_daily_usages` 为主表，没有用量记录的日期不会出现。本 MVP 同时建立后续两个维度复用的日期序列、排序和分页实现方式。

## 范围内

- 在 `token_usage_report_contract.go` 或 Repository 内建立安全的日期序列与补零查询结构。
- 改造 `ListModelTokenUsage` 的列表、总数及 Token 合计查询。
- 保证配置值 `0` 保持为显式不限额，缺少配置时返回 `null`。
- 添加配置无用量、部分日期有用量、无配置有用量、排序与分页测试。

## 范围外

- 用户模型及分组路由查询。
- 数据库 Schema、迁移、用量入库、计费或限额判断。
- API 和前端结构调整。

## 实现说明

- 使用递归 CTE 生成闭区间日期集合。
- 将当前模型配置和查询范围内实际用量组合取并集，再与日期集合组合。
- 对每日用量使用 `COALESCE(..., 0)`，汇总与分页必须基于补全结果。
- SQL 排序字段继续通过白名单映射，禁止用户输入直接进入 SQL 标识符。

## 验收标准

- [x] 已配置模型查询 7 天时返回 7 条数据，无调用日期为 `0`。
- [x] `daily_limit_tokens=0` 在响应中保持 `0`。
- [x] 无配置但存在用量的模型仍返回实际用量，限额为 `null`。
- [x] `total`、`total_used_tokens`、日期排序、Token 排序及分页结果正确。
- [x] 不包含任何 Schema 或用量入库改动。

## 验证计划

- `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryModel|TestTokenUsageReportContract"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_usage_report_repo.go` | 使用递归日期 CTE、配置/用量目标并集及补全结果集完成全局模型查询。 |
| 契约 | `backend/internal/repository/token_usage_report_contract.go` | 增加绑定日期序列与补全结果安全排序。 |
| 测试 | `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryModel\|TestTokenUsageReportContract\|TestTokenUsageDateSeries" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository`。 |
| 差异检查 | `git diff --check -- backend/internal/repository/token_usage_report_*` | 通过，无空白错误；未修改 Schema、迁移或用量入库。 |

## 执行记录

- `2026-07-07T15:00:31+08:00`：实现并验证全局模型 7 日补零、显式零限额、无配置历史用量、补全后汇总排序及分页。
- 首次聚焦测试因冷编译超过 120 秒超时；缓存建立后以 300 秒上限重跑并成功完成。
