# MVP-001：确定路由与既有领域规则

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `需要完成路由表、RBAC 排除机制、对象仓储接口和动态统计配置入口的集中核验，并以测试或设计常量固化结论。`
- Dependencies: `none`

## 预期成果

得到可直接实施的最终 URL、安全中间件边界及用户、分组、API Key、路由别名的既有匹配规则，消除后续实现对仓库事实的猜测。

## 背景

候选 URL 必须避开已有路由；接口不能被登录或 RBAC 中间件保护。API Key 名唯一性、email 比较、Key 与分组关系、路由别名来源也必须以现有模型和 Repository 为准。

## 范围内

- 枚举 Gin 实际注册的 Method + Path，确认候选 URL 是否可用并记录最终 URL。
- 确认 integrations 路由中间件链与 RBAC known exclusion/coverage 机制。
- 核验用户 email、分组名、API Key 名和路由别名现有字段、索引、Repository 方法及大小写规则。
- 核验 API Key 与用户/分组的关联规则。
- 核验动态统计时区、shard count、活动投影初始化入口。
- 为最终路由唯一性和既有规则增加最小可执行测试或测试夹具。

## 范围外

- 不实现 Redis Reader、业务查询 Service 或 HTTP Handler。
- 不修改业务数据约束和数据库 schema。

## 实现说明

- 优先检查 `backend/internal/server/router.go`、`backend/internal/server/routes`、`backend/internal/rbac`、相关 Ent schema 和 Repository。
- 若候选 URL 无冲突，固定 `POST /api/v1/integrations/token-usage/query`；若冲突，记录替代路径与理由。
- 不通过删除、覆盖已有路由解决冲突。
- 发现既有规则与最终 Plan 假设冲突时，先记录并停止受影响后续 MVP，不能静默发明规则。

## 验收标准

- [x] 最终 Method + URL 已确认且在路由表中不存在冲突。
- [x] 已明确目标路由只能使用 integrations Token Auth、限流和 hardening。
- [x] 已明确 RBAC 排除登记方式，且不会引入 RBAC 判断。
- [x] email、API Key 名、分组关系、路由别名、时区和 shard count 的既有规则均有代码位置或测试证据。
- [x] 后续 MVP 不需要重新发现上述关键规则。

## 验证计划

- `go test ./internal/server/routes ./internal/rbac ./internal/repository/... ./internal/service/tokenstat/...`
- 人工检查测试 Router 的 `Routes()` 输出，确认最终 Method + Path 唯一。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 路由盘点 | `rg` 检查 `internal/server/routes` 与 `internal/server/router.go` | 候选 `POST /api/v1/integrations/token-usage/query` 未注册；最终采用该路径 |
| 安全边界 | `internal/server/routes/integrations.go`、`internal/server/middleware/provisioning_auth.go` | integrations 组只使用外部 Bearer Token 与 hardening，不依赖登录态 |
| RBAC 排除 | `internal/rbac/coverage.go` | `/api/v1/integrations/*` 自动登记为 `external integration key trust boundary` |
| 领域规则 | `internal/repository/user_repo.go`、`api_key_repo.go`、`group_repo.go`、`internal/service/group.go` | email 使用现有规范化精确查询；API Key 名非全局唯一，必须按 userID+name 查询；Key 的 `group_id` 为绑定分组；路由别名来自分组 `ModelRoutingRuleNames()` |
| 统计配置 | `internal/config/config.go`、`cmd/server/wire_gen.go` | 时区与 shard count 来自 `Gateway.DynamicTokenStatistics`；启动时执行 `RefreshActive` |
| 自动验证 | `go test ./internal/server/routes ./internal/rbac ./internal/repository/... ./internal/service/tokenstat/...` | PASS，相关包全部通过 |

## 执行记录

- 2026-07-31：确认候选 URL 无冲突。现有 RBAC coverage 会按 integrations 前缀自动排除，无需新增权限点。
- API Key schema 没有 `(user_id,name)` 唯一索引，因此后续查询必须检测多条匹配并作为内部数据冲突处理，不能任意选择。
- API Key 绑定 `group_id`；本接口要求与请求分组一致。路由别名按分组解析后的规则名精确匹配。

