# MVP-002：用户模型每日零用量补全

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `一个完整用户维度查询切片，包含可选模型筛选、软删除用户兼容、汇总分页和测试。`
- Dependencies: `MVP-001`

## 预期成果

用户当前定义的每个模型限额条目在查询范围内每天可见，没有调用时显示 `0`；历史上有用量但当前无配置的模型也不会丢失。

## 背景

当前 `ListUserTokenUsage` 只查询 `user_model_token_daily_usages`，导致配置存在但零调用的日期或模型不显示。现有接口允许查询用户全部模型或指定单一模型。

## 范围内

- 改造用户模型报表的目标组合、日期补零、限额关联、汇总与分页。
- 同时支持未指定模型和指定模型两种查询。
- 保持软删除用户的邮箱、用户名和删除状态展示。
- 添加零用量、混合用量、无限额、历史用量、筛选及分页测试。

## 范围外

- 修改用户限额配置写入语义。
- 修改用户、限额或用量表结构。
- 修改 Handler、API 响应和前端页面。

## 实现说明

- 复用 MVP-001 确立的日期序列和排序方案。
- 目标模型集合为当前用户限额配置模型与日期范围内实际用量模型的并集。
- 指定 `model` 时在配置集合和用量集合两侧一致过滤。
- 最终结果左连接 `users`，不得过滤软删除用户。

## 验收标准

- [x] 用户存在配置但完全无调用时，查询范围内每天返回 `used_tokens=0`。
- [x] 未指定模型时，每个当前配置模型都按日补零。
- [x] 指定模型时仅返回该模型。
- [x] 软删除用户及无配置历史用量仍能正确返回。
- [x] 总数、Token 合计、排序和分页正确。

## 验证计划

- `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryUser"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_usage_report_repo.go` | 用户配置与范围内用量组成模型并集，和日期序列组合后关联用户、用量与当前限额。 |
| 测试 | `cd backend; go test ./internal/repository -run "TestTokenUsageReportRepositoryUser" -count=1` | 通过：覆盖完全零用量、多配置模型、指定模型、软删除用户、无配置历史用量及汇总分页。 |
| 差异检查 | `git diff --check -- internal/repository/token_usage_report_repo.go internal/repository/token_usage_report_repo_test.go` | 通过，无空白错误。 |

## 执行记录

- `2026-07-07T15:04:37+08:00`：实现并验证用户模型按日补零及可选模型筛选。
