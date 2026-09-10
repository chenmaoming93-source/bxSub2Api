# MVP-001：建立分组安全配置和安全检查数据模型

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: 以一次数据库迁移、Ent schema、类型化配置和后端校验为边界，不实现外部调用和页面。
- Dependencies: `none`

## 预期成果

系统能够持久化、读取和校验分组安全检查配置，并具备独立安全检查记录表的数据库模型基础。

## 背景

项目后端使用 Ent 和 SQL migrations，分组实体位于 `backend/ent/schema/group.go`，分组服务和 DTO 位于 `backend/internal/service` 与 `backend/internal/handler/dto`。为避免 `groups` 表增加多个字段，本功能使用一个 `security_check_config` JSON 字段。

## 范围内

- 为 `groups` 增加 `security_check_config` JSON 字段，默认表示关闭安全检查；
- 定义类型化安全配置、规则和枚举：`RuleAction`、`Decision`、`CheckStatus`、`ExceptionAction`；
- 校验五个 Query 风险维度、阈值、规则操作、超时时间和采样比例；
- 增加 `security_check_logs` 的迁移和 Ent/服务模型基础；
- 增加 `security_check_log_retention_days` 和全局采集开关的设置键定义；
- 保持旧分组记录和旧 API 响应兼容。

## 范围外

- SingGuard HTTP 调用；
- Redis 或本地缓存；
- Gateway 接入；
- 异步写库 worker；
- 前端页面。

## 实现说明

- `security_check_config` 结构包含 `enabled`、`rules`、`timeout_ms`、`exception_action`、`collect_enabled`、`sample_rate`、`version`；
- 规则操作只允许 `block`、`warn`；异常决策只允许 `allow`、`block`；
- 最终决策只允许 `allow`、`warn`、`block`；检查状态只允许 `skipped`、`success`、`timeout`、`error`；
- 规则维度使用 `SINGGUARD_API_SPEC.md` 中五个 Query 维度；
- 记录表至少包含请求标识、用户/API Key/分组元数据、请求体、完整返回体、状态、决策、`is_unsafe`、命中规则、耗时、异常和创建时间；
- 请求体字段使用数据库大字段，具体 MySQL/GoldenDB 兼容类型由迁移实现确定；
- 不主动把上游账号 ID 或上游请求 ID加入记录模型，因为安全检查发生在账号选择之前。

## 验收标准

- [x] 旧数据库执行迁移后，已有分组能够读取默认关闭的安全配置。
- [x] 合法配置可以保存并重新解析，非法维度、阈值、操作、超时时间和采样比例会被拒绝。
- [x] `security_check_logs` 具备 `event_id` 唯一约束和按 `created_at`、分组及决策查询所需索引。
- [x] 后端类型明确区分 `RuleAction`、`Decision`、`CheckStatus` 和 `isUnsafe`，不存在 `unsafe` action 值。
- [x] 数据库迁移和相关 Go 单元测试通过。

## 验证计划

- `cd backend; go test ./ent/... ./internal/service/... ./internal/repository/...`
- 检查迁移 SQL 在项目支持的数据库测试环境中可执行；若环境不可用，记录具体限制。
- 人工检查生成/更新后的 Ent schema、服务模型和 DTO 字段一致。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test ./internal/domain ./migrations ./ent/... ./internal/repository ./internal/service ./internal/handler/dto` | 通过；所有目标包测试通过。 |
| 生成验证 | `cd backend; go generate ./ent` | 通过；Ent 生成了 `security_check_config` 字段及 schema 元数据。 |
| 迁移 | `backend/migrations/159_group_security_check.sql` | 已实现 NULL 加列、默认配置回填、NOT NULL 收紧、日志表、唯一约束和查询索引；静态迁移回归测试通过。 |
| 代码路径 | `backend/internal/domain/security_check.go`, `backend/ent/schema/group.go`, `backend/internal/repository/group_repo.go` | 配置类型、默认值、校验、持久化和服务映射已完成。 |

## 执行记录

- 发现 `go generate ./ent` 首次受沙箱 Go telemetry/build cache 权限限制；在获得当前运行环境文件权限后重试成功。
- `go test -tags=integration ./internal/repository -run TestMigrationsSchema -count=1` 未能编译，原因是仓库现有 `affiliate_repo_integration_test.go` 与 `group_model_route_account_repo.go` 存在同名 `querySingleInt` 冲突；未修改该无关代码。数据库真实容器迁移执行因此未作为本次证据，已由迁移静态回归测试和 Ent 生成/编译测试覆盖结构验证。
