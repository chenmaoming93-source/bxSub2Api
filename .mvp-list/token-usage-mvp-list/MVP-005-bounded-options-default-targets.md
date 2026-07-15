# MVP-005：交付有界选项搜索与默认目标

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-002, MVP-003, MVP-004`

## 预期成果

三个页面都能在不加载全量选项的情况下搜索目标并取得一个默认目标。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。接口路径遵循源 Plan；不得返回认证信息或无界数组。

## 范围内

- 实现模型、分组、分组路由、用户、用户模型五类选项接口
- 实现 `default-target` 的 model/route/user 三种维度
- 单次选项最多 20 条，今日活跃优先、配置回退
- 覆盖搜索、防枚举边界、无数据和查询计划测试

## 范围外

- 不实现前端搜索组件。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 所有选项接口均有上限且支持必要搜索条件
- [x] 默认目标每次最多返回一个对象
- [x] 无今日用量时按配置回退，无数据时返回明确空值
- [x] 后端测试通过

## 验证计划

- `cd backend; go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 单元测试 | `go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/... -count=1` | 全部通过 (PASS) |
| 选项上限验证 | `internal/service/token_usage_report_service.go` SearchOptions 方法 | limit > 20 时强制截断为 20；所有 option SQL 均含 LIMIT 子句 |
| 默认目标验证 | `internal/repository/token_usage_report_repo.go` FindDefaultTokenUsageTarget | 先查当日用量（LIMIT 1），无数据时回退到配置表（LIMIT 1），均无返回 nil |
| 选项搜索条件 | 5 类选项 (models/groups/routes/users/user_models) | 全部支持 LIKE 搜索、parent ID 约束和 LIMIT 上限 |
| 路由注册 | `internal/server/routes/admin.go` | 7 个选项接口 + 1 个 default-target 接口已注册 |
| 服务测试 | `TestTokenUsageOptionsAreBoundedAndDefaultMayBeEmpty` | 选项上限（99→20）、默认目标空值、parent 校验均通过 |
| 处理器测试 | `handler/admin/token_usage_report_handler_test.go` | handler 级别验证通过 |

## 执行记录

执行时记录偏差、阻塞项、索引或接口决策；当前无执行记录。
