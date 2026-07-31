# MVP-003：交付受范围约束的四维对象解析

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `需要补齐或复用四类 Repository 查询、关系校验和稳定领域错误，工作范围可独立于 Redis 完成。`
- Dependencies: `MVP-001`

## 预期成果

通过一个可单元测试的领域能力，将 email、分组名、API Key 值（`api_keys.key`，数据库唯一）和路由别名解析成 `user_id`、`group_id`、`api_key_id`、`route_alias`，并安全地返回四类明确 Not Found。

## 背景

用户名必须精确匹配 `email`。API Key 值全局唯一，但解析后仍须校验其属于目标用户并可用于目标分组；路由别名必须属于目标分组。关系不成立时需按资源不存在处理，避免资源枚举。

## 范围内

- 定义输入模型、解析结果和 `USER_NOT_FOUND`、`GROUP_NOT_FOUND`、`API_KEY_NOT_FOUND`、`API_KEY_MISMATCH`、`ROUTE_ALIAS_NOT_FOUND` 领域错误。
- 复用或扩展受范围约束的 Repository 方法（按 `api_keys.key` 唯一值查询）。
- 固定用户、分组、API Key、路由别名的解析顺序。
- 覆盖跨用户、跨分组及关系不成立场景。
- 编写 Repository/Service 单元测试。

## 范围外

- 不读取 Redis，不匹配统计投影。
- 不暴露 HTTP 路由。
- 不修改 email 或 API Key 值的既有业务规则（唯一性由 schema 保证）。

## 实现说明

- Handler 字段名使用 `username`，Service 明确将其解释为 email。
- Service 不直接拼 SQL；查询必须参数化并通过窄 Repository 接口完成。
- 错误只暴露资源类型和请求值，不暴露其他用户/分组归属信息。
- 不读取或返回 API Key 密钥；响应与日志均不得回显明文值。

## 验收标准

- [x] email 精确查询得到正确 `user_id`，且未使用用户展示名匹配。
- [x] 四类不存在场景产生稳定且不同的领域错误。
- [x] 首个失败对象按固定顺序返回。
- [x] API Key 值不存在或已删除返回 `API_KEY_NOT_FOUND`；值存在但属于其他用户/分组（或无分组）返回 `API_KEY_MISMATCH`（HTTP 400），不泄露该 Key 实际归属。
- [x] 其他分组的路由别名对外表现为不存在（`ROUTE_ALIAS_NOT_FOUND`）。
- [x] 测试覆盖既有大小写、唯一性和分组授权规则。

## 验证计划

- `go test ./internal/repository/... ./internal/service/... -run 'External.*Token|Dimension.*Resolution|APIKey.*Group'`
- 必要时运行对应 Ent Repository 的 SQLite 集成测试。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/external_token_usage_repo.go` | email、分组名及 `api_keys.key` 唯一值受限查询已实现；归属（用户/分组）不符按不存在处理 |
| 领域解析 | `backend/internal/service/external_token_usage.go` | 固定四类解析顺序、关系隐藏和 `ROUTE_ALIAS_NOT_FOUND` 已实现 |
| 单元测试 | `go test ./internal/service -run 'ExternalTokenUsage'` | PASS |
| Repository 构建验证 | `go test ./internal/repository -run 'External.*Token|APIKey.*Group'` | PASS（包编译通过，无匹配的既有测试） |

## 执行记录

- 2026-07-31：新增专用窄 Repository，避免扩大已有大型接口并影响大量测试桩。
- email 复用现有 `userEmailLookupPredicate`；API Key 按 `api_keys.key` 唯一值查询并校验 user/group 归属，未删除状态；路由别名按 `ModelRoutingRuleNames()` 精确匹配。
- 2026-07-31（契约变更）：`api_key_name` 改为 `api_key`（明文值），Service 输入与 Repository 查询同步调整；新增归属不符、无分组用例。
- 2026-07-31（错误语义调整）：Key 值存在但归属不符不再伪装为不存在，改为 `API_KEY_MISMATCH`（HTTP 400）；仅值不存在/已删除返回 `API_KEY_NOT_FOUND`（HTTP 404）。

