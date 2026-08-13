# MVP-001：后端数据层（domain 类型 + ent schema + SQL 归档）

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: 1 个开发者日
- Estimate rationale: 涉及新增 domain 类型与单测、修改 ent schema 并全量重新生成、diff 范围核对、SQL 归档与方言核验，工作量约一个工作日。
- Dependencies: `none`

## 预期成果

`accounts` 表具备 `model_attributes` JSON 列（DDL 已归档），后端具备强类型 `domain.ModelAttributes`（map）及最小规整函数，ent 生成代码已包含该字段且全仓库可编译。

## 背景

- 项目：bxSub2Api；后端 Go + ent v0.14.5，数据库 MySQL 8 / GoldenDB。
- 相关路径：`backend/internal/domain/`（新增类型）、`backend/ent/schema/account.go`（schema）、`backend/ent/`（生成代码）、`backend/sqlArchiving/`（DDL 归档）。
- 项目规约：表结构变更 SQL 必须归档到 `backend/sqlArchiving/`（不参与运行时迁移），编号取当前最大 167 的下一个（168）；方言必须为 MySQL 8 / GoldenDB。
- 数据结构：`{ 属性名(英文): { "description": 中文描述, "value": 动态类型 } }`；后端信任前端，value 原样存储。

## 范围内

- 新增 `backend/internal/domain/model_attributes.go`：`ModelAttributeItem`、`ModelAttributes`（`map[string]ModelAttributeItem`）、`Normalize()`（丢弃 key 去空白后为空的条目，其余原样保留）。
- `backend/ent/schema/account.go` 新增 `model_attributes` JSON 字段（`Optional()`，MySQL 方言 `json`）。
- 执行全量 `go generate ./ent` 并核对 diff 范围（预期仅 account 实体相关文件 + `mutation.go`/`migrate/schema.go`/`runtime/runtime.go` 等少量共享文件）。
- 新增 `backend/sqlArchiving/168_add_account_model_attributes.sql`：`ALTER TABLE accounts ADD COLUMN model_attributes JSON NULL COMMENT ...`。
- domain 规整函数单测（空 key 过滤、去空白、nil 安全、value/description 原样保留）。

## 范围外

- service / repository / handler / dto 的业务链路（见 MVP-002）。
- 任何前端改动。
- value 的类型解析、枚举校验或长度限制。

## 实现说明

- 生成前确认无残留 `go.exe`/gopls 进程，防止 Windows 文件锁中断生成导致损坏文件；生成后立即 `git status` + 编译核对，异常时 `git checkout -- backend/ent/` 还原重试。
- `Normalize()` 仅做最小防御：遍历 map，key 去首尾空白后为空的条目丢弃，其余（description/value）原样拷贝。
- SQL 文件需可独立执行、每条语句以分号结尾；声明可重复执行则需在 MySQL 8 测试库连续执行两次验证。

## 验收标准

- [x] `backend/internal/domain/model_attributes.go` 存在且 `go build ./internal/domain/...` 通过
- [x] `go generate ./ent` 执行成功，`git status` 显示 diff 仅涉及 account 实体相关文件（无大规模无关改动）
- [x] `go build ./...`（backend 全量）通过
- [x] domain 单测通过：空 key 条目被丢弃、key 首尾空白被剔除、nil 安全、description/value 原样保留
- [x] `backend/sqlArchiving/168_add_account_model_attributes.sql` 存在且无 PostgreSQL 专属语法，方言为 MySQL 8 / GoldenDB

## 验证计划

- `cd backend && go build ./...`
- `cd backend && go test ./internal/domain/...`
- `cd backend && go generate ./ent`（随后 `git status --short backend/ent/` 核对 diff 范围）
- 在 MySQL 8 / GoldenDB 测试库执行 `backend/sqlArchiving/168_add_account_model_attributes.sql`（声明可重复执行则连续执行两次）

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 新增文件 | `backend/internal/domain/model_attributes.go` | `ModelAttributeItem` / `ModelAttributes`(map) / `Normalize()` 已实现 |
| 新增文件 | `backend/internal/domain/model_attributes_test.go` | 8 个用例（nil 安全、空 key 丢弃、去空白、原样保留、不修改入参等） |
| 修改 | `backend/ent/schema/account.go` | 新增 `model_attributes` JSON 字段（Optional，MySQL json） |
| go generate | `cd backend && go generate ./ent` | 成功（exit 0）；diff 仅 9 个 account 相关文件：account.go、account/{account,where}.go、account_create.go、account_update.go、migrate/schema.go、mutation.go、runtime/runtime.go、schema/account.go（+238/-20，无无关改动） |
| go build | `cd backend && go build ./...` | 通过（exit 0） |
| go test | `cd backend && go test ./internal/domain/...` | ok（1.590s，8 用例全绿） |
| SQL 归档 | `backend/sqlArchiving/168_add_account_model_attributes.sql` | 已创建；方言 MySQL 8/GoldenDB，风格与 160/161 一致（ADD COLUMN IF NOT EXISTS）；无 PostgreSQL 专属语法。注：当前环境无 MySQL 测试库，实际执行验证留待 MVP-005/部署时进行 |

## 执行记录

- 2026-08-05：MVP-001 完成。执行 `go generate ./ent` 时工作区存在用户启动的 `go run ./cmd/server/` 与 `go test ./...` 进程，未终止用户进程，生成未受影响（无文件锁错误）。
- 生成 diff 范围符合预期（仅 account 实体 + 少量共享文件），无需暂停确认。
- SQL 文件采用与 `160_api_key_platform_purpose.sql` 一致的 `ADD COLUMN IF NOT EXISTS` 幂等写法（GoldenDB MySQL 兼容模式）。
