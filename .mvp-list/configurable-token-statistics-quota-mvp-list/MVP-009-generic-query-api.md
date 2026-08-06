# MVP-009：交付管理员通用多维查询 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `投影驱动的白名单查询、汇总、趋势、排行榜、分页和CSV可在统一服务中独立交付。`
- Dependencies: `MVP-002, MVP-003, MVP-006`

## 预期成果

具备 `token_usage.read` 的管理员可以通过一个通用API查询任意已采集投影的MySQL历史数据，并获得完整性和同步时间。

## 背景

查询不能跨投影拼接，也不能查询未采集的组合。自定义日期范围基于日数据求和。

## 范围内

- `POST /admin/token-statistics/query`。
- 投影、指标、筛选和group_by白名单验证。
- 日/周/月与自定义范围规则。
- 汇总、时间序列、排行榜和分页。
- CSV导出能力及行数限制。
- 投影启用时间、完整性和最后同步时间。
- 参数化SQL和查询范围保护。

## 范围外

- Redis实时数据合并。
- 前端页面。

## 实现说明

- 默认只读MySQL，明确展示最终一致延迟。
- 动态字段不能直接拼接用户输入；使用注册表映射到允许列或JSON表达式。

## 验收标准

- [x] 可查询低基数和多维投影。
- [x] 非投影维度、非投影指标和非法排序被拒绝。
- [x] 自定义日期范围只汇总日粒度。
- [x] 不混合周期重复计数。
- [x] 返回完整性和最后同步时间。
- [x] 仅管理员且要求 `token_usage.read`。
- [x] SQL注入、超大范围和超大分页测试通过。

## 验证计划

- `cd backend && go test ./internal/repository/... ./internal/service/... ./internal/handler/admin/... -run "DynamicToken.*Query|TokenStat.*Query"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 查询服务 | `backend/internal/service/tokenstat/query.go` | 基于投影和注册表白名单完成筛选、分组、汇总、趋势/排行排序、分页、范围保护及同步完整性元数据；仅查询 `token_stat_aggregates` |
| 管理 API | `backend/internal/handler/admin/dynamic_token_statistics_handler.go`、`backend/internal/server/routes/admin.go` | 完成 `POST /admin/token-statistics/query` 和 CSV 输出，路由要求 `token_usage.read` |
| 自动化测试 | `cd backend && go test ./internal/repository/... ./internal/service/... ./internal/handler/admin/... ./internal/server/... -run "DynamicToken.*Query\|TokenStat.*Query\|RBACAdminOps"` | 通过；覆盖多维筛选/分组、汇总/排行/分页、非法维度、非法排序注入、超大日期范围与权限映射 |

## 执行记录

2026-07-30：完成管理员通用查询 API。查询只使用单一投影、单一周期粒度的 MySQL 聚合行，不合并 Redis 实时值，不访问旧统计表。
