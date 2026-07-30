# MVP-012：迁移支付、订阅、兑换与返利管理路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `围绕财务和商业操作形成一个独立高风险授权域。`
- Dependencies: `MVP-006`

## 预期成果

支付订单、套餐、服务商、订阅、兑换码、优惠码和返利接口具备清晰的财务权限边界。

## 背景

支付 webhook 不纳入 RBAC；只有登录后的用户支付与管理支付路由纳入。

## 范围内

- `/admin/payment` Dashboard、配置、订单、套餐和服务商。
- 管理订阅分配、延期、重置和删除。
- 兑换码、优惠码及使用记录。
- Affiliate 邀请、返利、转账和专属用户管理。

## 范围外

- 支付 webhook 和公开订单恢复。
- 用户个人订单路由（已在 MVP-007）。

## 实现说明

- 财务读取、退款、余额相关和服务商配置拆分。
- 支付回调保持 EXCLUDED 并保留签名校验。

## 验收标准

- [x] 范围内管理路由全部进入 Registry。
- [x] webhook 和公开支付路由保持明确排除。
- [x] 财务只读角色不能退款、取消、重试或修改服务商。
- [x] admin 行为保持一致。

## 验证计划

- `cd backend && go test ./internal/server/routes/... -run Payment`
- `cd backend && go test ./internal/handler/admin/... -run '(Payment|Subscription|Redeem|Promo|Affiliate)'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/server/routes/payment.go` | `/admin/payment` 改用身份认证、Principal、兼容回退和统一 registrar；读、订单操作、套餐管理、服务商管理分权。公开恢复与 webhook 保持原公开路由及签名校验。 |
| 实现 | `backend/internal/server/routes/admin.go` | 订阅、兑换码、优惠码和返利管理路由全部声明读或管理权限。 |
| 测试 | `go test ./internal/server/routes/... -run 'Payment|RBAC' -count=1` | 通过。 |
| 测试 | `go test ./internal/handler/admin/... -run '(Payment|Subscription|Redeem|Promo|Affiliate)' -count=1` | 通过。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过。 |

## 执行记录

受控边界为登录用户支付路由和 `/admin/payment`；`/payment/public/*` 与 `/payment/webhook/*` 不进入 RBAC，继续依赖恢复令牌、持久状态兼容校验或支付服务商签名校验。`billing.read` 不授予取消、重试、退款或服务商修改。
