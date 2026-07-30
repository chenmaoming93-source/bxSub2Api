# MVP-013：迁移渠道、风控、公告及剩余管理路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `收口尚未覆盖的管理路由，模块数量多但路由结构集中且可用清单核对。`
- Dependencies: `MVP-006`

## 预期成果

渠道、监控、风控、公告、定时测试和合规等剩余管理路由全部完成声明式权限迁移。

## 背景

该 MVP 是管理路由迁移的收口切片，必须与 `RBAC_ENDPOINT_INVENTORY.md` 逐项核对。

## 范围内

- 渠道定价、渠道监控及模板。
- 风控配置、日志、解封和哈希清理。
- 公告 CRUD 和阅读状态。
- 定时测试计划。
- 管理端合规、管理 API Key 分组更新及所有尚未覆盖的 admin 路由。

## 范围外

- 路由总闭合门禁（MVP-014）。
- 角色管理 API。

## 实现说明

- 风控解封和清理为高风险权限。
- 合规中间件保留在 RBAC 授权链中。
- 通过清单差集确认没有剩余 admin 路由。

## 验收标准

- [x] 范围内所有路由进入 Registry。
- [x] admin 路由清单与 MVP-008～013 的并集无缺口。
- [x] 风控与渠道写操作有独立拒绝测试。
- [x] 合规守卫行为保持不变。

## 验证计划

- `cd backend && go test ./internal/server/... -run 'Admin.*(Channel|Risk|Announcement|Scheduled|Compliance)'`
- 对 `RBAC_ENDPOINT_INVENTORY.md` 管理员接口执行人工差集复核。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/server/routes/admin.go` | 渠道、监控、风控、公告、定时测试、合规和管理 API Key 均改为 registrar 声明；文件中已无绕过 helper 的实际 admin HTTP 注册。 |
| 差集检查 | `rg -n '\.(GET|POST|PUT|PATCH|DELETE)\(' internal/server/routes/admin.go` | 仅剩四个统一 helper 的兼容分支及 registrar 调用，无业务路由直连 Gin。 |
| 测试 | `go test ./internal/server/... -run 'RBAC|Admin.*(Channel|Risk|Announcement|Scheduled|Compliance)' -count=1` | 通过。 |
| 测试 | `go test ./internal/handler/admin/... -run '(Channel|ContentModeration|Announcement|Scheduled|Compliance)' -count=1` | 通过。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过。 |

## 执行记录

风控读、配置修改和解封/哈希清理分别使用 `risk.read`、`risk.update`、`risk.operate`；渠道 CRUD 与监控运行也分别授权。既有 `AdminComplianceGuard` 仍位于认证和 RBAC 链中。
