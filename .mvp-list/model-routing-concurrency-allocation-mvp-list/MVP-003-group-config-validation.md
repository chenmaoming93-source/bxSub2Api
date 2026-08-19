# MVP-003：分组候选并发配置校验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Dependencies: `MVP-001`

## 目标

在分组编辑候选并发时校验账号额度，并返回可理解的超限错误。

## 范围

- 查询同账号全部具体候选分配；
- 分组保存和单候选更新时校验总和；
- 账号为 0/负数时跳过总和上限校验；
- `null`/缺失配置不计入总和；
- 返回超限账号、当前总和和账号上限；
- 校验过程提供事务或等价并发保护。

## 非范围

- 不改变候选达到上限后的请求降级行为。

## 验收标准

- [x] 合法配置可以保存；
- [x] 具体候选总和超限时保存被拒绝；
- [x] `null` 候选不导致误报；
- [x] 账号并发为 0/负数时不因总和超限拒绝；
- [x] 错误信息包含账号和额度信息；
- [x] 现有路由配置保存行为未被破坏。

## 验证计划

- 运行管理接口和 repository/service 测试；
- 覆盖多候选、跨分组、空配置和并发更新场景。

## 证据

完成证据：

- `backend/internal/repository/group_model_route_account_repo.go` 在候选并发更新前校验账号并发分配总和；
- 分组路由编辑器读取账号总并发和已分配额度，显示具体分配比例及无限制状态；
- `go test ./internal/repository ./internal/handler/admin` 通过；
- `npm run typecheck` 通过。
