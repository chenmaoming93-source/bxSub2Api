# Token 用量报表零用量补全实施 Plan

- 状态：`最终版——用户已批准`
- 版本：`v1.0`
- 日期：`2026-07-07`
- 变更摘要：让已定义限额条目在查询日期范围内每天都有统计行，无调用时显示 `0`。

## 1. 背景与目标

当前报表以每日用量表为主要数据源。某个限额组合当天没有调用时，不会产生用量记录，因此报表缺少对应日期。

目标：

- 已定义限额条目在查询日期范围内每天可见。
- 当天没有调用时，`used_tokens` 显示 `0`。
- `daily_limit_tokens=0` 仍表示已定义且不限额。
- 无限额配置但产生过调用的数据继续显示。
- 覆盖全局模型、分组路由、用户模型三个维度。

成功标准：

- 配置存在且查询 7 天时，返回完整 7 条每日记录。
- 无调用日期显示 `0`。
- 总记录数、分页、排序和 Token 合计与补全后的结果一致。

## 2. 范围

### 范围内

- 改造 Token 用量报表的 SQL 查询与汇总逻辑。
- 为三个报表维度生成日期序列。
- 合并限额配置与实际用量。
- 补充 Repository 测试及必要的接口回归测试。

### 范围外

- 不修改 Token 用量入库流程。
- 不修改计费和限额拦截逻辑。
- 不修改现有 API 路径和响应结构。
- 不新建数据库表。
- 不保存限额配置的历史生效区间。

## 3. 假设与决策

- 当前限额配置代表查询时刻有效的配置。
- 当前存在的配置会覆盖用户选择的整个日期范围。
- 配置删除后，历史报表只保留已有实际用量，不再为该配置补零。
- 若未来需要准确还原“某条限额在哪一天创建或删除”，必须增加配置历史或生效区间；本次不处理。
- 日期边界继续使用项目现有时区规则。
- 现有用量表已经独立于限额配置进行写入，无需改造入库。

## 4. 功能设计

### FR-01 全局模型报表

查询指定模型时：

- 实际用量日期正常显示。
- 如果该模型存在限额配置，则查询范围内缺失日期补 `0`。
- 没有限额配置但存在用量时，继续显示实际用量。

### FR-02 分组路由报表

查询指定分组和路由时：

- 当前已定义限额的上游模型每天都显示。
- 未指定上游模型时，每个已配置候选模型分别补零。
- 指定上游模型时，仅补该模型。
- 无配置但存在历史用量的候选组合继续显示。

### FR-03 用户模型报表

查询指定用户时：

- 当前已配置的用户模型每天都显示。
- 未指定模型时，每个已配置模型分别补零。
- 指定模型时，仅补该模型。
- 无配置但存在历史用量的组合继续显示。

### FR-04 汇总、排序与分页

- `total` 统计补零后的总行数。
- `total_used_tokens` 仍为实际 Token 总量，补零行不增加总量。
- 日期和 Token 排序作用于补全后的结果集。
- 分页在合并与补零完成后执行。

## 5. 技术设计

### 5.1 改造位置

主要修改：

- `backend/internal/repository/token_usage_report_repo.go`
- `backend/internal/repository/token_usage_report_contract.go`
- `backend/internal/repository/token_usage_report_repo_test.go`
- `backend/internal/repository/token_usage_report_contract_test.go`

Service、Handler 和前端原则上无需修改，因为接口结构保持不变。

### 5.2 查询模型

每个报表构造两个集合：

1. 当前配置条目与查询日期序列的笛卡尔积。
2. 查询范围内存在的实际用量记录。

通过 `UNION` 或等价 CTE 合并组合键，再左连接用量：

```text
目标组合 = 当前限额配置组合 UNION 实际用量组合
日期集合 = start_date 至 end_date
报表行 = 目标组合 CROSS JOIN 日期集合
         LEFT JOIN 每日用量
```

输出：

```text
used_tokens = COALESCE(usage.used_tokens, 0)
daily_limit_tokens = 当前配置中的限额；无配置时为 null
```

### 5.3 日期序列

优先采用 MySQL 8 兼容的递归 CTE：

```sql
WITH RECURSIVE dates AS (
  SELECT start_date
  UNION ALL
  SELECT DATE_ADD(usage_date, INTERVAL 1 DAY)
  FROM dates
  WHERE usage_date < end_date
)
```

日期参数必须继续经过现有查询范围校验，避免生成异常规模的数据集。

### 5.4 数据表

复用现有表：

| 维度 | 限额配置表 | 每日用量表 |
|---|---|---|
| 全局模型 | `model_token_daily_limit_configs` | `model_token_daily_usages` |
| 用户模型 | `user_model_token_daily_limit_configs` | `user_model_token_daily_usages` |
| 分组路由 | `group_candidate_token_daily_limit_configs` | `group_candidate_token_daily_usages` |

不新增表、不修改字段、不执行数据迁移。

### 5.5 API

保留现有接口：

- `GET /admin/token-usage/models`
- `GET /admin/token-usage/routes`
- `GET /admin/token-usage/users`

请求参数、鉴权、分页字段及响应字段保持不变。

## 6. 核心逻辑

```text
function queryReport(dimension, target, dateRange, pagination, sorting):
  validate dateRange, pagination and sorting
  build dates from startDate to endDate
  load current configured combinations matching target
  load usage combinations matching target and dateRange
  combinations = distinct(configured combinations + usage combinations)
  rows = combinations × dates
  left join daily usage
  left join current quota configuration
  used_tokens = usage value or 0
  calculate total rows and sum of actual usage
  sort completed rows
  apply pagination
  return report
```

错误处理维持现有行为，不增加自动重试或降级查询。

## 7. 验证策略

### AC-01 全局模型补零

存在模型配置、部分日期无调用时，查询范围内每天均返回记录，无调用日期为 `0`。

### AC-02 用户模型补零

用户模型配置存在但完全没有调用时，仍返回完整日期范围，Token 均为 `0`。

### AC-03 路由候选补零

候选模型存在限额配置但没有用量时，分组路由报表能够显示每日零用量。

### AC-04 无限额条目

限额值为 `0` 时仍参与补零，响应中的限额保持为 `0`，不转换成 `null`。

### AC-05 无配置用量

不存在限额配置但存在实际用量时，报表继续返回这些数据，限额为 `null`。

### AC-06 汇总与分页

`total`、Token 合计、排序和分页均基于补全后的结果正确计算。

建议验证命令：

```powershell
cd backend
go test ./internal/repository/... ./internal/service/... ./internal/handler/admin/...
```

## 8. 实施顺序

1. 提取三个维度共用的日期序列与排序约束。
2. 改造全局模型查询并验证。
3. 改造用户模型查询并验证。
4. 改造分组路由查询并验证。
5. 执行 Repository、Service、Handler 回归测试。
6. 检查前端现有页面对零用量行的展示，无兼容问题则不修改前端。

后续 MVP 拆分建议按三个报表维度形成独立、可验证的垂直任务，最后增加统一回归任务。

## 9. 风险

| 风险 | 影响 | 应对 |
|---|---|---|
| 日期范围和配置数量乘积较大 | 查询变慢 | 保留日期范围及分页校验，测试典型数据规模 |
| 当前配置被用于历史日期 | 历史语义不精确 | 明确这是本次约定；未来通过配置历史解决 |
| 三个查询的汇总逻辑不一致 | 总数或分页错误 | 为每个维度分别覆盖补零、实用量和混合场景 |
| MySQL 版本不支持递归 CTE | 查询无法执行 | 项目当前 SQL 已使用 MySQL 8 能力；实施时通过集成测试确认 |

## 10. 追踪矩阵

| 需求 | 模块 | 验收标准 |
|---|---|---|
| FR-01 | 全局模型报表查询 | AC-01、AC-04、AC-05 |
| FR-02 | 分组路由报表查询 | AC-03、AC-04、AC-05 |
| FR-03 | 用户模型报表查询 | AC-02、AC-04、AC-05 |
| FR-04 | 汇总、排序、分页 | AC-06 |

## 11. 审核记录

- `v0.1`：根据“已定义限额条目每天均需展示，即使用量为零”的原则形成初稿。
- `v1.0`：用户于 `2026-07-07` 明确批准，内容定稿。
- 当前状态：用户已批准。
