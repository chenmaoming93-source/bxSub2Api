# MVP-008：旁路金额资格检查并保留非金额控制

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 仅调整资格检查边界并补充针对性回归测试，范围可控。
- Dependencies: `none`

## 预期成果

余额、订阅金额、API Key 金额额度和账号金额额度不再阻断请求，身份、状态、RPM、并发和 Token 配额仍生效。

## 背景

资格入口主要为 `BillingCacheService.CheckBillingEligibility`，由多个 handler 调用。

## 范围内

- 在最小范围注释或旁路金额检查分支。
- 保留金额检查具体实现。
- 增加余额为零仍可通过的测试。
- 增加 Token 配额及状态控制不被误伤的测试。

## 范围外

- 请求后金额计费和 Redis Token 累计。

## 实现说明

- 注释必须说明 token-statistics-only 停用原因和恢复入口。
- 不大范围重构 `BillingCacheService`。

## 验收标准

- [x] 金额不足不再返回 `billing_error`。
- [x] 非金额限制仍返回原有分类错误。
- [x] 被停用的具体实现仍存在。
- [x] 资格检查测试通过。

## 验证计划

- `cd backend; go test ./internal/service -run "CheckBillingEligibility|BillingCacheService"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/billing_cache_service.go` | 在 `CheckBillingEligibility` 最外层旁路余额、订阅金额、API Key 金额及用户平台金额分支，保留 RPM 调用。 |
| 保留实现 | `checkBalanceEligibility`、`checkSubscriptionEligibility`、`checkUserPlatformQuotaEligibility`、`checkAPIKeyRateLimits` | 均保留且可调用，注释标明恢复入口。 |
| 测试 | `cd backend; go test ./internal/service -run "CheckBillingEligibility|BillingCacheService" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.135s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：验证 standard 模式在无余额缓存、失效订阅及已配置 API Key 金额窗口的组合下仍可通过；原 RPM 检查继续位于资格入口末尾，既有 BillingCacheService 回归测试通过。
