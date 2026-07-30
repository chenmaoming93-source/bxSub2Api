# MVP-007：迁移全部用户个人路由到声明式 RBAC

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `65 条个人接口位于集中路由文件，可作为一个兼容性切片迁移并验证。`
- Dependencies: `MVP-006`

## 预期成果

全部用户个人接口声明 self 权限，同时保留 JWT、BackendMode 和现有资源归属校验。

## 背景

主要路径为 `backend/internal/server/routes/user.go` 和支付用户路由。

## 范围内

- profile、TOTP、通知邮箱、API Key、分组、渠道、usage。
- announcement、redeem、subscription、channel monitor。
- 用户支付和订单接口。
- 用户个人权限的契约测试。

## 范围外

- 管理端路由。
- 修改 Handler 中现有数据归属规则。

## 实现说明

- 每条路由就近声明权限。
- 同一业务 read 接口可共享权限，但写操作按动作拆分。
- 管理员因 `*` 应继续能调用原个人功能。

## 验收标准

- [x] 当前源码中的 64 条个人接口全部进入 Registry；清单中多出的已移除接口已记录。
- [x] user 角色的初始化权限覆盖全部原个人行为。
- [x] 普通用户仍不能读取其他用户资源。
- [x] 未登录、无权限和有权限结果符合 401/403/原响应。

## 验证计划

- `cd backend && go test ./internal/server/... -run User`
- `cd backend && go test ./internal/handler/... -run Ownership`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 路由迁移 | `cd backend && go test ./internal/server/... -run 'User|RBAC' -count=1` | 通过；用户、支付、认证当前用户和自定义页面共 64 条现存接口均改为声明式权限注册。 |
| Handler 回归 | `cd backend && go test ./internal/handler/... -run Ownership -count=1` | 相关包编译通过；未修改任何 Handler 归属判断。 |
| 归属测试 | `cd backend && go test ./internal/service -run TestGetUserErrorRequestDetail_OwnershipEnforced -count=1` | 通过；其他用户的错误请求详情继续被拒绝。 |
| 应用构建 | `cd backend && go test ./cmd/server -run '^$'` | 通过；RBAC Repository、Redis 权限服务、Registry 和 Router 依赖注入可编译。 |
| 数量差异 | `RBAC_ENDPOINT_INVENTORY.md` 对比当前源码 | 清单中的 `GET /api/v1/usage/group-model-daily` 在当前代码已不存在且没有对应 Handler；实际个人接口为 64，不虚构恢复已删除 API。 |

## 执行记录

原清单为 65，当前 Gin 源码为 64；唯一差异是已删除的 `/api/v1/usage/group-model-daily`。其余 64 条均在路由旁声明 self 权限。JWT、BackendMode 中间件和 Handler 内按 `user_id` 的归属校验保持原样。Wire 当前主干已有 3 个与本次无关的 ProviderSet 缺失，生成器无法全量重跑；已对现有 `wire_gen.go` 做最小同步并以 `go test ./cmd/server` 验证。
