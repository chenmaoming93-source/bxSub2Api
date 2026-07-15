# MVP-005：打通模型消耗报表的实时混合查询

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只处理模型维度的一条完整垂直链路，包括日期分流、合并、汇总、排序和分页，约 40 分钟。
- Dependencies: `MVP-001, MVP-002, MVP-003, MVP-004`

## 预期成果

模型 Token 消耗报表在历史范围保持 MySQL 快速路径，在包含今天时正确使用 Redis 实时值并完成缺失修复。

## 背景

入口为 `/admin/token-usage/models`，现有实现位于 `token_usage_report_service.go` 与 `token_usage_report_repo.go`。

## 范围内

- 实现模型报表的业务日期分流。
- 历史日期继续调用现有 MySQL 报表路径。
- 包含今天时合并 MySQL 历史、MySQL 当天集合与 Redis 当天集合。
- 基于合并结果计算 summary、usage_rate、status、排序、total 和分页。
- 覆盖 Redis-only、MySQL-only、两边都有和不存在模型筛选测试。

## 范围外

- 不改路由候选和用户模型报表。
- 不改配置页面。

## 实现说明

- 不得先分页再覆盖 Redis 数字。
- API DTO 与路径保持不变。

## 验收标准

- [x] 不包含今天时测试证明 Redis reader 未被调用。
- [x] 包含今天时两边都有的数据使用 Redis 且不相加。
- [x] Redis-only 模型可进入正确排序与页码。
- [x] MySQL-only 模型返回 MySQL 值并触发修复候选。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*Model.*TokenUsage'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_usage_report_service.go` | 增加业务日分流；历史保持分页 MySQL 快速路径，包含今天时先收集历史再合并 MySQL/Redis 当天集合，最后汇总、排序、分页并触发 MySQL-only 修复。 |
| 接线 | `backend/internal/repository/wire.go`、`backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go` | 生产链路注入当天 MySQL 集合、Redis reader 与原子 repairer。 |
| 测试 | `backend/internal/service/token_usage_model_hybrid_test.go` | 覆盖历史不读 Redis、重叠取 Redis、不相加、Redis-only 排序分页、MySQL-only 修复和不存在筛选。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*Model.*TokenUsage'` | 通过。 |
| 相关回归 | `cd backend && go test ./cmd/server ./internal/service ./internal/repository ./internal/handler/admin` | 通过：server、service、repository、admin handler 包均通过。 |

## 执行记录

2026-07-15 完成。系统无全局 `wire`，尝试 `go run` 生成时因外网依赖下载失败；依据现有生成格式同步更新 `wire_gen.go`，随后 `cmd/server` 构建测试通过。Redis 整体读取失败时保留 MySQL 当天集合且不触发修复。
