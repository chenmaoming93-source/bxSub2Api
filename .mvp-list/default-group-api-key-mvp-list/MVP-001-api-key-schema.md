# MVP-001：扩展 API Key 平台与用途数据模型

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `none`

## 预期成果

数据库与 Ent schema 能持久化 API Key 的 `platform` 和 `purpose`，现有记录保持兼容。

## 背景

涉及 `backend/ent/schema/api_key.go`、`backend/sqlArchiving/` 及 Ent 生成代码。DDL 按项目规约归档为可直接执行的 SQL，不加入运行时迁移目录；本 MVP 先完成字段与默认值，不包含唯一性约束。

## 范围内

- 新增 `platform` 可空字段，最大长度 50。
- 新增 `purpose` 非空字段，默认 `user_created`。
- 增加向前迁移及必要的 schema/生成代码更新。
- 验证旧记录读取时获得兼容默认值。

## 范围外

- 平台唯一索引。
- 业务服务与接口。

## 实现说明

- 遵循 `PROJECT_CONVENTIONS.md` 的 SQL 归档编号和 Ent 生成流程。
- 领域实体、Repository 映射和 DTO 暂只做编译所需的最小贯通。

## 验收标准

- [x] 迁移可在现有数据库结构上执行，字段类型和默认值符合 Plan。
- [x] `go test ./ent/... ./internal/repository/...` 通过。

## 验证计划

- `cd backend; go generate ./ent`
- `cd backend; go test ./ent/... ./internal/repository/...`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 生成 | `cd backend; go generate ./ent` | 通过，退出码 0。 |
| 聚焦测试 | `cd backend; go test ./ent/schema -run TestAPIKeyPlatformPurposeFields -count=1` | 通过。 |
| 聚焦测试 | `cd backend; go test ./migrations -run TestAPIKeyPlatformPurposeMigration -count=1` | 通过。 |
| Repository 测试 | `cd backend; go test ./internal/repository/... -count=1` | 通过。 |
| 规定测试 | `cd backend; go test ./ent/... ./internal/repository/...` | 未通过：`ent/schema/model_token_daily_usage_test.go` 期待当前 schema 不存在的 `daily_limit_tokens`，随后因索引越界 panic；`internal/repository` 通过。该失败位于执行前已有的 token-usage 工作区改动范围，未擅自修改。 |
| 规定测试重跑 | `cd backend; go test ./ent/... ./internal/repository/...`（2026-07-08 数据库字段更新后） | 仍被同一纯 Go schema 单元测试阻塞；该测试不连接数据库，因此数据库 DDL 更新不会改变结果。`internal/repository` 仍通过。 |
| 过期测试修复 | `backend/ent/schema/model_token_daily_usage_test.go` | 改为验证用量表不含 `daily_limit_tokens`、独立限额配置表包含该字段，并移除 `fields[3]` 硬编码访问。 |
| 聚焦重跑 | `cd backend; go test ./ent/schema -run 'ModelTokenDailyUsage|ModelTokenDailyLimitConfig' -count=1` | 通过。 |
| 规定测试最终重跑 | `cd backend; go test ./ent/... ./internal/repository/...` | 通过，退出码 0。 |
| 变更路径 | `backend/ent/schema/api_key.go`、`backend/sqlArchiving/160_api_key_platform_purpose.sql`、Ent 生成代码、API Key service/repository/DTO 映射 | 已实现 `platform` 可空最大 50、`purpose` 非空默认 `user_created`。 |

## 执行记录

实现与验证完成。过期的 token-usage schema 测试已按当前“限额配置与每日用量分表”结构修正，规定整包测试最终通过，原阻塞解除。
