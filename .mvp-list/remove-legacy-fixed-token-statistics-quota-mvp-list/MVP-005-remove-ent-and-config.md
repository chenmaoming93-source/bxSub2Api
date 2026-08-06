# MVP-005：移除旧 Ent 模型、配置与持久化依赖

- Protocol: `mvp-list/v1`
- State: `DONE`
- Estimate: `1.5 开发日`
- Estimate rationale: `六张旧表 schema、关联 edge、生成代码和旧配置删除需要 Ent 再生成及全后端编译。`
- Dependencies: `MVP-002, MVP-003`

## 预期成果

应用代码和 Ent 不再认识六张旧表，旧统计配置被删除，开发、资源和示例配置同步；新动态配置及表完整保留。

## 背景

生成 Ent 代码会产生全局机械差异，必须以 schema 删除为源并防止误删新 `token_stat_*` schema。

## 范围内

- 删除六个旧 Ent schema、edge、生成实体与 predicate。
- 删除旧固定统计和限额配置、默认值、校验与样例。
- 重新生成 Ent 并更新测试。

## 范围外

- 在数据库执行 DROP。
- 删除 `usage_logs` 或新 `token_stat_*`。

## 实现说明

- 配置变更同步 `backend/config/config.yaml`、`backend/resources/config.yaml`、`deploy/config.example.yaml`。
- 生成后运行全 Ent/schema 和配置测试。

## 验收标准

- [x] 六个旧 Ent schema 与生成 API 不再存在。
- [x] 旧配置已从结构、默认值、校验和三份 YAML 删除。
- [x] 新动态 schema、配置和 API 完整。
- [x] Ent、配置、服务和 server 测试通过。

## 验证计划

- `cd backend && go generate ./ent && go test ./ent/... ./internal/config/... ./internal/service/... ./internal/server/...`
- `rg -n "ModelTokenDaily|UserModelTokenDaily|GroupCandidateTokenDaily|token_statistics:" backend/ent backend/internal/config backend/config backend/resources deploy`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Ent 生成 | `go run ... ent generate --target ./entgen ...` 后原子替换生成层 | 通过；规避 Windows 原地覆盖文件映射锁 |
| 后端测试 | `go test ./ent/... ./internal/config/... ./internal/service/... ./internal/server/...` | 通过 |
| 保护检查 | `backend/ent/schema/token_stat_*.go`、`gateway.dynamic_token_statistics` | 五个新实体与新配置完整保留 |

## 执行记录

2026-07-30：开始移除旧 Ent schema/生成代码、旧配置项与三份 YAML 示例，保护五张新动态统计表。

2026-07-31：完成六个旧 schema、edge、生成 API、旧配置结构/默认值/校验及三份 YAML 的删除；Ent 全量重新生成并通过相关后端测试。
