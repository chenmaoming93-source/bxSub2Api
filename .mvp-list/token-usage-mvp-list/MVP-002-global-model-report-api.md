# MVP-002：交付全局模型统计 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-001`

## 预期成果

管理员可分页查询指定实际上游模型在日期范围内的每日用量与汇总。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。建议落在 `backend/internal/{repository,service,handler/admin}`；必须要求 `model`，`page_size` 默认 20、最大 100。

## 范围内

- 实现全局模型 Repository、Service、Handler
- 关联 `model_token_daily_limit_configs` 并计算使用率和状态
- 注册 `GET /api/v1/admin/token-usage/models`
- 覆盖参数、权限、空数据、排序、分页和汇总测试

## 范围外

- 不实现选项搜索、默认目标或前端页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 管理员可按模型和日期获得有界分页响应
- [x] 缺少模型或非法日期/排序返回 400，非管理员被拒绝
- [x] 当前限额为空时返回不限额语义
- [x] 相关 Go 测试通过

## 验证计划

- `cd backend; go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/{repository,service,handler/admin}/token_usage_report_*` | 完成模型日报、汇总、当前限额、使用率和状态计算 |
| 路由与权限 | `backend/internal/server/routes/admin.go`；`TestTokenUsageReportAdminRouteRequiresAdmin` | 注册 `GET /api/v1/admin/token-usage/models`，非管理员返回 401 |
| 定向测试 | `go test` 分别运行 `TestTokenUsageReport*` | Repository、Service、Handler、Server 均通过 |
| 完整验收 | `cd backend; go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...` | 全部通过 |

## 执行记录

- 日期缺省为项目全局时区的今天；日期范围为闭区间。
- 未配置或非正限额返回 `status=unlimited` 且 `usage_rate=null`；80% 起为 `warning`，达到或超过 100% 为 `exceeded`。
- Wire 生成因外网依赖下载失败，已按现有生成结构同步更新 `cmd/server/wire_gen.go`，编译与测试验证通过。
