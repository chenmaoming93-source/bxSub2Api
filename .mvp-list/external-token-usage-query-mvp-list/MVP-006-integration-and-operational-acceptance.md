# MVP-006：完成集成、回归与运维验收

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `集中完成 Wire、全链路契约、RBAC Coverage、性能边界、文档和回归证据，确保功能可安全发布。`
- Dependencies: `MVP-005`

## 预期成果

新接口完成生产依赖注入、全链路验证和运维交付，已有 integrations 路由及动态统计写入不受影响，具备明确发布与回滚证据。

## 背景

最后阶段需证明 Handler、Service、Repository 在真实依赖图中可构建，路由安全边界和 RBAC 排除正确，Redis 读写兼容，并形成可观测、可回滚的发布单元。

## 范围内

- 完成 Wire Provider 和生成代码更新。
- 运行 RBAC Route Coverage 与路由唯一性检查。
- 完成端到端或等价集成测试：日周月有值、单周期缺失、投影未配置、Redis 故障。
- 回归现有 `api-keys/getOrCreate`、`model-routes/list` 和动态统计写入测试。
- 验证每请求最多三个精确 HGET 且无扫描。
- 补充外部接口契约、指标、日志、告警、发布和回滚说明。
- 汇总可复现的测试证据。

## 范围外

- 不部署到生产或修改生产配置。
- 不创建统计投影或填充真实业务 Redis 数据。
- 不扩展历史查询或新增前端页面。

## 实现说明

- 使用项目既有 Wire 生成流程，不手工维护不一致的依赖图。
- 完整测试失败若源于无关既有问题，必须记录精确命令和失败证据，同时保证所有受影响包测试通过。
- 运维文档明确 Redis 为当前周期唯一数据源，故障返回 503。

## 验收标准

- [x] Wire 生成与构建成功，无手工依赖缺口。
- [x] RBAC Coverage 通过，目标路由只作为外部 Token 排除项。
- [x] 日、周、月全链路状态与 Redis 数据一致。
- [x] 原 integrations 路由及动态统计写入回归通过。
- [x] 测试证明无 Redis 扫描且操作数有界。
- [x] 运维、发布、回滚和接口契约文档完成。
- [x] 所有验收证据已写入本文件，随后立即更新 `mvp-progress.md`。

## 验证计划

- `go test ./internal/handler ./internal/server/... ./internal/service/... ./internal/repository/tokenstat/...`
- `go test ./...`
- 运行项目现有 Wire 生成/校验命令并确认工作树只包含预期生成差异。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Wire 生成 | `go run github.com/google/wire/cmd/wire ./cmd/server` | PASS，`wire_gen.go` 成功生成并包含外部查询、外部 provisioning 与动态统计 bootstrap |
| 相关包验收 | `go test ./cmd/server ./internal/handler ./internal/server/... ./internal/service/... ./internal/repository/tokenstat/...` | PASS |
| 全量后端测试 | `go test ./...` | PASS，所有包通过 |
| RBAC 与路由 | `internal/rbac/coverage.go`、`TestIntegrationRoutes_TokenUsageExternalTokenOnlyAndUnique` | integrations 前缀自动排除；目标 Method+Path 唯一且外部 Token 必需 |
| Redis 有界性 | `current_usage_reader_test.go`、`redis_accumulator_test.go` | 三周期精确 HGET 兼容；无 KEYS/SCAN；请求遇错提前结束 |
| 文档 | `docs/token-statistics-development-guide.md`、`docs/token-statistics-operation-guide.md` | 已补充接口契约、故障含义、发布与回滚说明 |

## 执行记录

- 2026-07-31：将此前生成文件中的动态统计手工启动逻辑提取为 composition-root Provider，确保今后 Wire 重生成不会丢失现有动态统计功能。
- 同时将既有 ExternalProvisioning Handler 纳入正式 Wire Provider，避免重生成后原 integrations 接口消失。
- 完整后端测试两次通过；最终一次在最新 Wire 生成结果和 DTO 契约修正后通过。
