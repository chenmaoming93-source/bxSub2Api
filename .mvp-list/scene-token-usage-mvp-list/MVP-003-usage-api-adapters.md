# MVP-003：管理端与外部场景用量接口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 两个接口仅承担不同鉴权和 HTTP 适配，共用 MVP-002 的查询用例；包含路由、契约和错误测试，能独立验证接口一致性。
- Dependencies: `MVP-002`

## 预期成果

管理端和外部系统都可以通过正式 HTTP 接口查询相同的场景 Token 用量结果，且配置错误原因一致。

## 背景

管理端路由集中在 `backend/internal/server/routes/admin.go`，外部 integrations 路由位于 `backend/internal/server/routes/integrations.go`。管理端使用 RBAC，外部端使用现有外部 Bearer Token、限流和请求加固中间件。

## 范围内

- 新增管理端接口：`GET /api/v1/admin/usage/scene-account/daily`。
- 新增外部接口：`POST /api/v1/integrations/token-usage/query/scene-account/daily`。
- 管理端参数使用查询参数，外部端参数使用 JSON 请求体。
- 两个接口均支持 `start_date`、`end_date` 和可选 `group_name`。
- 管理端使用 `usage.admin.read` 权限，外部端复用既有 integrations 鉴权。
- 统一处理 400、409、503 和动态查询行数超限错误。
- 增加管理端和外部端路由/Handler 契约测试。
- 记录 source、日期范围、Projection、结果完整性和错误码等结构化日志。

## 范围外

- 不在 Handler 中复制统计查询、聚合或名称补全逻辑。
- 不修改现有 `/admin/token-statistics/query` 和既有外部 Token 用量接口的契约。
- 不返回 API Key 或其他凭证。

## 实现说明

- 管理端和外部端调用同一个共享查询 Service，确保响应数据一致。
- `end_date` 按包含当天处理，内部传给 tokenstat 时使用结束日期次日零点。
- 错误至少包含：`SCENE_USAGE_STATISTICS_NOT_CONFIGURED`、`SCENE_USAGE_STATISTICS_NOT_ACTIVE`、`TOKEN_STATISTICS_DISABLED`。
- 成功响应包含 `timezone`、`complete`、`consistency`、Projection 元数据和嵌套 `days`。

## 验收标准

- [x] 两个接口都能接受日期范围和可选技术 `group_name`。
- [x] 管理端需要 `usage.admin.read`，外部端需要现有外部 Bearer Token。
- [x] 两个接口对相同查询返回等价的统计结果。
- [x] 缺少必要配置时返回明确错误，不返回空成功结果。
- [x] `group_name` 只匹配技术分组名，省略时可以查询所有分组。
- [x] 非法日期、超长范围和超出查询行数限制时返回明确错误。
- [x] 接口契约测试和相关后端测试通过。

## 验证计划

- `cd backend && go test ./internal/handler/... ./internal/server/routes/...`
- 使用 Handler stub 验证两个接口的请求校验、响应结构、错误映射和脱敏行为。
- 使用路由测试验证管理端权限与外部 Bearer Token 保护。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 管理端适配器 | `backend/internal/handler/admin/scene_account_usage_handler.go`, `backend/internal/server/routes/admin.go` | 新增 GET 接口并挂载 `usage.admin.read`，参数为查询参数。 |
| 外部适配器 | `backend/internal/handler/external_scene_account_usage_handler.go`, `backend/internal/server/routes/integrations.go` | 新增 POST 接口，复用 integrations Bearer 鉴权、限流与请求加固，参数为 JSON。 |
| 共享依赖 | `backend/internal/handler/wire.go`, `backend/internal/handler/handler.go`, `backend/internal/server/router.go` | 两个 Handler 注入同一个 `SceneAccountDailyUsageService`，未复制查询逻辑。 |
| Handler 契约 | `go test ./internal/handler -run 'Test(External|Admin|SceneAccountDailyUsage)' -count=1` | 通过。覆盖 JSON/query 绑定、等价响应和配置错误映射。 |
| 外部路由鉴权 | `go test ./internal/server/routes -run 'TestIntegrationRoutes_(SceneAccountDailyAuthContract|TokenUsageExternalTokenOnlyAndUnique)' -count=1` | 通过；Bearer Token 保护和新路径验证通过。 |
| 相关回归 | `go test ./internal/handler/... ./internal/server/routes/...` | `internal/handler` 与 `internal/server/routes` 通过；命令因既有 `group_model_routing_test.go` 候选账号断言失败退出 1，与本 MVP 无关。 |
| 变更路径 | `backend/internal/handler/{handler.go,wire.go,external_scene_account_usage_handler.go,scene_account_usage_handler_test.go}`, `backend/internal/handler/admin/scene_account_usage_handler.go`, `backend/internal/server/{router.go,routes/{admin.go,integrations.go,integrations_test.go}}` | 完成两个 HTTP 适配器、路由和结构化日志。 |

## 执行记录

- 2026-08-26：完成管理端与外部端场景 Token 用量接口；二者仅鉴权/HTTP 适配不同，统一调用 MVP-002 共享 Service。

