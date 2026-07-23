# MVP-007：完成接口集成与安全回归验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `聚焦跨层场景测试和全后端单元回归，不再加入新的业务实现。`
- Dependencies: `MVP-003, MVP-006`

## 预期成果

通过跨层测试证明指定分组、专属授权、三元组幂等、多分组 Key、软删除重建及现有安全中间件均符合批准的 Plan，并形成可执行的最终证据。

## 背景

前序 MVP 分别交付服务、仓储、接口和索引。本 MVP 负责验证组合后的外部可观察行为，避免局部测试通过但完整链路语义不一致。

## 范围内

- 增加或完善集成级 getOrCreate 场景测试。
- 验证公开分组、已授权专属分组、未授权专属分组和订阅分组。
- 验证相同三元组幂等及不同分组创建不同 Key。
- 验证 platform 与 `groups.platform` 不匹配仍成功。
- 验证软删除后可重建。
- 验证 Bearer Token、Content-Type、禁止缓存和限流未回归。
- 运行全后端 unit 测试并记录证据。

## 范围外

- 不增加新功能。
- 不执行生产数据库升级。
- 不修改前端。
- 不扩展订阅分组支持。

## 实现说明

- 优先复用现有路由和仓储测试夹具。
- 测试必须验证失败场景未新增 Key，而不只检查 HTTP 状态。
- 若 integration tag 依赖本地数据库且环境不可用，必须记录为 BLOCKED 证据，不能将未运行测试视为通过；unit 回归仍需执行。

## 验收标准

- [x] 公开标准分组创建返回 201，重复请求返回 200 和同一 Key。
- [x] 已授权专属分组成功，未授权专属分组返回 403 且无 Key。
- [x] 订阅分组被拒绝且无 Key。
- [x] 同一用户/platform 的不同分组产生不同 Key。
- [x] platform 与分组平台不匹配仍成功。
- [x] 软删除相同三元组后可以重新创建。
- [x] 认证、Content-Type、限流和禁止缓存行为未回归。
- [x] `go test -tags=unit ./...` 通过。

## 验证计划

- `cd backend && go test -tags=unit ./...`
- `cd backend && go test -tags=integration ./...`
- 如果完整 integration 套件依赖不可用，执行可运行的定向集成测试并在证据中记录环境限制。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 功能聚合验证 | `cd backend && go test ./internal/service ./internal/handler ./internal/repository ./migrations -run 'TestGetOrCreatePlatformKey|TestEnsurePlatformKey|TestExternalProvisioningHandler|TestAPIKeyRepository_ActivePlatformAndDefaultUniqueness|TestAPIKeyPlatformGroupIndex'` | 通过：四个相关包全部成功。 |
| 仓储 integration | `cd backend && go test -tags=integration ./internal/repository -run 'TestAPIKeyRepoSuite/TestActivePlatformAndDefaultKeyUniquenessAllowsReplacementAfterSoftDelete'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.646s`，包含软删除后重建。 |
| 安全中间件 | `cd backend && go test ./internal/server/middleware -run 'Test.*Provisioning'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/server/middleware 4.656s`。 |
| 全量 unit 回归 | `cd backend && go test -tags=unit ./...` | 通过：所有包成功，`internal/service` 用时 `105.632s`。 |
| 恢复后定向修复 | 构造器测试、usage-log 测试、日志脱敏、SQL 归档测试、唯一约束识别、OAuth 首次登录、token 统计日期 | 对应定向测试均已通过；全量测试已从编译失败推进到运行期语义冲突。 |
| 恢复后功能复验 | `cd backend && go test ./internal/service ./internal/handler ./internal/repository ./migrations -run 'TestGetOrCreatePlatformKey|TestEnsurePlatformKey|TestExternalProvisioningHandler|TestAPIKeyRepository_ActivePlatformAndDefaultUniqueness|TestAPIKeyPlatformGroupIndex'` | 通过：service、handler、repository、migrations 全部成功。 |
| 完整 integration | `cd backend && go test -tags=integration ./...` | 通过：所有包成功，包含 `internal/integration`、repository、server、service 与 migrations。 |
| 货币计费测试清理 | `backend/internal/service/gateway_record_usage_test.go`、`payment_fulfillment_test.go`、`internal/server/middleware/api_key_auth_google_test.go` | 按用户确认，将余额扣减、billing fingerprint、affiliate rebate、余额不足拦截等旧货币计费测试移出默认 unit 测试集合；保留非货币使用日志与 token 统计测试。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

2026-07-16：相关功能、仓储 integration 和供应安全中间件测试均通过；全量 unit 回归实际运行但失败。用户要求继续尝试后，已修复旧构造器参数、失效 usage-log 测试、日志脱敏替换、159/160/161 SQL 归档测试、PostgreSQL 唯一约束识别、管理员测试 stub、API 契约快照、动态 token 统计日期，以及 OAuth 首次注册变量遮蔽空指针；对应定向测试通过。

用户确认货币计费相关测试可以移除后，已将余额扣减、billing fingerprint、affiliate rebate 和余额不足拦截等旧测试移出默认 unit 集合；保留非货币使用日志、模型字段与 token 统计测试。同时隔离配置测试与实际开发配置、对齐严格模型路由语义。最终 `go test -tags=unit ./...` 与 `go test -tags=integration ./...` 均通过，MVP 状态更新为 `VERIFIED`。
