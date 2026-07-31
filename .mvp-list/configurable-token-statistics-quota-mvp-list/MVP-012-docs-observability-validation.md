# MVP-012：完成文档、观测性与系统级验收

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 开发日`
- Estimate rationale: `完整文档、指标告警、性能与隔离审计需要跨模块验证，作为最终系统交付门禁。`
- Dependencies: `MVP-007, MVP-008, MVP-011`

## 预期成果

新体系具备可操作的监控和告警、两份正式文档、性能证据、全协议回归和新旧隔离报告，可进入后续旧体系移除阶段。

## 背景

Plan要求后续开发者或AI能安全增加指标，管理员能完成日常操作；同时新体系不得与旧三套固定逻辑产生交集。

## 范围内

- `docs/token-statistics-development-guide.md`。
- `docs/token-statistics-operation-guide.md`。
- 队列、丢弃、Redis、同步、封账、限额和配置版本指标。
- 告警建议与故障演练记录。
- 容量和性能测试。
- 全协议回归。
- 新旧依赖、Redis、MySQL、API和前端隔离审计。
- 最终验收报告。

## 范围外

- 删除旧三套统计和限额。
- 迁移旧数据或配置。

## 实现说明

- 开发指引必须包含新增指标实例、语义版本、测试清单和AI禁止事项。
- 操作手册必须覆盖投影、限额、查询、同步、封账和故障。
- 隔离审计应可重复运行。

## 验收标准

- [x] 两份文档完整并通过一次模拟演练。
- [x] 关键指标和告警可观察。
- [x] 性能目标有真实命令和结果证据。
- [x] 全协议统计与fail-open回归通过。
- [x] 新体系不读写旧Redis、旧表或旧API。
- [x] 旧固定统计限额删除工作仍明确留给专门Plan。
- [x] 后端、前端和相关集成测试通过。

## 验证计划

- `cd backend && go test ./...`
- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- `rg -n "TokenStatisticsType|model_token_daily_|user_model_token_daily_|group_candidate_token_daily_|/admin/token-usage/" <新体系路径>`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 开发与操作文档 | `docs/token-statistics-development-guide.md`、`docs/token-statistics-operation-guide.md` | 覆盖新增指标/维度、语义版本、AI 禁止事项、投影/限额/查询/同步/封账/故障和演练 |
| 验收报告 | `docs/token-statistics-acceptance-report.md` | 记录自动化故障演练、性能结果、边界和可重复隔离审计 |
| 可观测性 | `backend/internal/service/tokenstat/observability.go`、`GET /admin/token-statistics/status` | 固定基数暴露队列/丢弃/Redis/同步/封账/限额/config version 指标；操作手册给出告警阈值 |
| 性能 | `cd backend && go test ./internal/service/tokenstat -run '^$' -bench DynamicTokenStatisticsQueueFullFailOpen -benchtime=200ms` | 13,915,462 次，19.25 ns/op；队列满失败路径不执行阻塞 I/O |
| 后端全量 | `cd backend && go test ./...` | 全部通过，包含协议、服务、仓储、handler、server 和集成包 |
| 前端验证 | `pnpm vitest run ...dynamicTokenStatistics... ...TokenStatisticsView... ...permissionMatrix...`、`pnpm typecheck`、`pnpm build` | 新功能 9 项测试、类型检查和生产构建通过；全量测试的 6 项既有 OAuth/EmailVerify 跳转断言失败已如实记录 |
| 隔离扫描 | 验收报告中的 `rg` 命令 | 无匹配；新路径未引用旧表、旧 API、旧 Redis 或 `dailyTokenQuotaRepo` |

## 执行记录

2026-07-30：完成文档、可观测性、容量性能、故障测试映射、全量后端回归、前端验证和新旧隔离审计。旧固定逻辑的删除仍严格留给独立 Plan。
