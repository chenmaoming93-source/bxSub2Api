# MVP-006：打通路由候选消耗报表的实时混合查询

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 围绕 group_id、route_alias、upstream_model 复合键完成一条独立报表链路，工作量约 40 分钟。
- Dependencies: `MVP-001, MVP-002, MVP-003, MVP-004`

## 预期成果

路由候选 Token 消耗报表对今天使用 Redis 实时累计，并保持分组、路由、上游模型、优先级和限额元数据正确。

## 背景

入口为 `/admin/token-usage/routes`，该维度使用复合业务键，Redis field codec 位于 `token_statistics_codec.go`。

## 范围内

- 实现路由候选报表的日期分流与混合查询。
- 按 group_id、route_alias、upstream_model 业务键合并。
- 用 MySQL 元数据补齐 group_name、priority 和 daily_limit_tokens。
- 基于合并结果完成汇总、状态、多字段排序和分页。
- 覆盖 Redis 孤立项、无效实体关联与不存在筛选项。

## 范围外

- 不改模型和用户模型报表。
- 不改路由配置写入逻辑。

## 实现说明

- 无法关联有效业务实体的 Redis 孤立项不进入候选展示，并应留下诊断证据。
- 保持 `include`/筛选语义与现有接口一致。

## 验收标准

- [x] 历史查询不访问 Redis。
- [x] Redis 实时值正确覆盖当天 MySQL 快照。
- [x] 复合键不会因特殊字符或同名路由发生碰撞。
- [x] 分组名、优先级、限额、汇总、排序和分页正确。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*Route.*TokenUsage'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_usage_report_service.go` | 路由报表日期分流、结构化复合键合并、MySQL 元数据补齐、无有效分组元数据的 Redis 孤立项过滤、合并后汇总/多字段排序/分页。 |
| 修复 | `backend/internal/service/current_token_usage_reader.go`、`backend/internal/repository/token_usage_read_repair.go` | 共享 repairer 端口增加路由候选原子修复。 |
| 测试 | `backend/internal/service/token_usage_route_hybrid_test.go` | 覆盖历史不读 Redis、覆盖不相加、特殊字符复合键、孤立项过滤、元数据、汇总和多字段排序。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*Route.*TokenUsage'` | 通过。 |
| 包回归 | `cd backend && go test ./internal/service ./internal/repository ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。Redis-only 路由项只有在能从 MySQL 当天候选集合获得有效分组元数据时才展示；无法关联的孤立复合键被排除。未改变路由配置写入逻辑或 API DTO。
