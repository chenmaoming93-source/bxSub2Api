# MVP-008：usage 路由别名后端查询口径

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `将用量、错误和统计查询统一到一个兼容表达式，并通过 Repository 测试固定语义。`
- Dependencies: `none`

## 预期成果

现有 `model` 查询参数在 usage 相关后端统一解释为路由别名，使用 `requested_model` 优先、历史 `model` 回退的精确匹配。

## 背景

用量查询位于 `backend/internal/repository/usage_log_repo.go`，错误请求查询位于 `backend/internal/repository/ops_repo.go`。当前两者过滤列不一致。

## 范围内

- 建立或复用统一 requested-model 过滤表达式。
- 覆盖用量明细、错误请求、汇总、趋势和 usage 页面相关图表查询。
- 保留现有 HTTP 参数名 `model`，只重构业务语义。
- 保留历史日志回退：`COALESCE(NULLIF(TRIM(requested_model), ''), model)`。
- 更新 Repository 和 Handler 契约测试。

## 范围外

- 新增 `route_alias` HTTP 参数。
- 按 `upstream_model` 搜索。
- 回填所有历史日志。

## 实现说明

- 避免仅修改列表而遗漏统计/图表接口。
- 精确匹配语义保持与当前管理员筛选一致。
- 检查 SQL 执行计划；若表达式索引问题显著，只记录后续索引方案，不扩大本 MVP 数据迁移范围。

## 验收标准

- [x] `requested_model` 存在时仅按其匹配。
- [x] `requested_model` 为空时可按历史 `model` 匹配。
- [x] 相同 `upstream_model` 不会导致误匹配。
- [x] 用量、错误、汇总、趋势和图表接口使用同一业务口径。
- [x] 现有 `model` 参数调用方保持兼容。

## 验证计划

- `cd backend && go test ./internal/repository -run 'UsageLog|OpsError|ModelQuery|RequestedModel'`
- `cd backend && go test ./internal/handler/admin -run 'Usage|Ops'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 统一表达式 | `usageLogRequestedModelExpr` | `COALESCE(NULLIF(TRIM(requested_model), ''), model)`，requested_model 非空时优先，空白/历史记录回退 model。 |
| usage 接入 | `usage_log_repo.go` | 列表条件、统计条件、按模型时间范围、汇总/趋势/图表共用现有 model 参数及统一表达式。 |
| 错误请求 | `ops_repo.go` | 精确和用户端模糊两种过滤均改为相同的 TRIM/NULLIF requested-model 回退语义。 |
| Repository 测试 | `go test ./internal/repository -run 'UsageLog|OpsError|ModelQuery|RequestedModel' -count=1` | 通过。 |
| Handler 测试 | `go test ./internal/handler/admin -run 'Usage|Ops' -count=1` | 通过。 |

## 执行记录

- HTTP 参数仍名为 `model`，前端和其他调用方无需迁移；业务含义统一为路由别名/requested model。
- 表达式未引用 upstream_model，测试明确断言同名上游模型不会成为匹配条件。
- 当前表达式过滤通常不能直接利用单列 requested_model 索引；本 MVP 按计划不新增迁移。数据量增长后可考虑生成列/函数索引 `COALESCE(NULLIF(TRIM(requested_model),''),model)`。
