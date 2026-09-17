# MVP-007：完成跨入口回归与发布运行保障

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: `聚焦最终跨层验收、故障场景、文档和发布/回滚证据，范围固定且可在一个开发日内完成，不重复实现前序功能。`
- Dependencies: `MVP-004, MVP-005, MVP-006`

## 预期成果

管理员页面、管理员 API、integrations API 和网关限额检查形成经过验证的完整闭环，并具备 Redis-only 持久性边界、故障降级、发布及回滚所需的测试和运维说明。

## 背景

- 源 Plan 要求真实统计和 MySQL 同步完全不受影响，同时明确 Redis AOF/RDB 只是运维增强，不构成应用层恢复保证。
- 应用或 Redis 在多条重置中途失败时允许部分成功，调用方可重试。
- 开发指引位于 `docs/token-statistics-development-guide.md`，应补充快照 namespace、读取公式和禁止事项。

## 范围内

- 执行并修复 Token statistics repository/service/handler/routes 的完整相关测试。
- 执行前端 API、页面、类型检查和构建。
- 增加跨层测试，证明管理员与 integrations 入口调用同一重置语义。
- 验证重置前后原统计值、version、dirty set 和 MySQL 聚合不被篡改。
- 验证应用重新构造服务但 Redis 数据保留时，快照继续影响限额读取。
- 验证 Redis 不可用时重置结果和限额 `fail-open`。
- 验证单次 dirty、无同步锁、不扫描 processing 的既定边界。
- 更新 Token statistics 开发/运维文档，说明 namespace、TTL、AOF 建议、数据丢失和回滚行为。
- 记录发布检查及回滚验证证据。

## 范围外

- 不新增 Redis 快照恢复数据库表。
- 不实施或修改生产 Redis AOF 配置，只提供明确运维建议。
- 不扩展为严格一致、幂等或异步后台重置任务。
- 不执行生产部署或生产数据操作。

## 实现说明

- 回归测试必须证明旧版本语义：快照缺失时 effective usage 等于 raw usage。
- 隔离检查应确认没有新代码归零统计 Key、修改 `metric_value` 或将快照加入同步 dirty 流程。
- 文档必须明确 processing 窗口遗漏及“必要时再次重置”的产品决策，避免后续开发误加锁或分布式事务。
- 若完整仓库测试存在与本功能无关的既有失败，必须记录准确命令、失败用例和归因，不能标记为通过。

## 验收标准

- [x] 后端相关包测试全部通过，或对非本功能既有失败提供可复现证据且本功能测试通过。
- [x] 前端定向测试、typecheck 和 build 通过。
- [x] 管理员页面、管理员 API、integrations API 均能触发相同的重置服务结果。
- [x] 重置后原统计 Hash、version Hash、dirty identity 和 MySQL 聚合值保持真实累计语义。
- [x] 应用重启且 Redis 保留时快照继续生效；Redis 快照丢失时不会污染 MySQL。
- [x] Redis 故障时重置接口和限额 fail-open 行为符合 Plan。
- [x] 大批量分批处理、部分成功和重复重置场景有自动化或明确人工验证证据。
- [x] 开发文档包含快照 Key、差值公式、TTL、最终一致性、AOF 建议和回滚说明。
- [x] 没有引入旧固定 Token 统计表、旧 Redis namespace 或旧 API 依赖。

## 验证计划

- `cd backend && go test ./internal/service/tokenstat ./internal/repository/tokenstat ./internal/handler/admin ./internal/handler ./internal/server/routes`
- `cd backend && go test ./...`
- `cd frontend && pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- 检查 `docs/token-statistics-development-guide.md` 与最终行为一致，并静态确认新代码未写入旧固定统计 namespace。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端功能套件 | `go test ./cmd/server ./internal/handler ./internal/server/routes ./internal/service/tokenstat ./internal/repository/tokenstat` | 全部通过。 |
| 跨入口/持久性 | `quota_reset_contract_test.go`、`quota_reset_test.go` 定向命令 | 管理员与 integrations 返回同一服务语义；重建 reader 后 baseline 保留，删除 baseline 后回到 raw。 |
| 完整后端 | `cd backend && go test ./...` | 未全绿：既有 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting`、RBAC compatibility seed 2 项、`TestGroupSecurityCheckMigrationAddsSafeDefaultsAndIndexes` 失败；token reset 相关包均通过。 |
| 前端 | 定向 Vitest（2 files/15 tests）、`pnpm typecheck`、`pnpm build` | 全部通过；build 只有既有 warning。 |
| 数据隔离 | `quota_reset_test.go`、`reset_identity_sources_test.go`、静态 grep | raw/version/dirty 保持不变；MySQL 数据源只读；无 DDL 或旧 namespace/API 依赖。 |
| 文档 | `docs/token-statistics-development-guide.md` 第 8 节 | 记录 namespace、公式、TTL、一致性、AOF/RDB、丢失、重试及回滚。 |
| 格式 | `git diff --check`、Go `gofmt` | 通过；文档仅有 Git LF→CRLF 工作树提示。 |

## 执行记录

2026-09-10 完成。新增自动化覆盖大批量多 pipeline、单条 WRONGTYPE 部分失败、重复重置、Redis fail-open、应用服务重建、baseline 丢失以及两个 HTTP 入口共享结果。完整仓库测试的四个失败均位于本功能未修改的既有 group/RBAC/migration 逻辑；已准确保留命令和用例名，未误报全绿。未执行生产部署、Redis 配置或生产数据操作。
