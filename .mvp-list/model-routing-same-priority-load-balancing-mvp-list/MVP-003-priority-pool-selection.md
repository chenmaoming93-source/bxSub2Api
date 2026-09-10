# MVP-003：Gateway 同 priority 账号池选择

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Dependencies: `MVP-001`、`MVP-002`

## 目标

把同一模型路由别名下、priority 相同的候选账号合并为一个选择层。在该层内按候选 route LoadRate、最近使用时间选择账号；只有当前层没有可用结果时，才降级到下一 priority 层。

## 实现范围

- 使用 `GroupCandidatesByPriority` 将候选按 priority 升序分层，并合并同层账号。
- 同层账号去重；同一账号出现在不同 priority 层时拒绝非法路由配置。
- 新候选对象格式进入分层选择路径，候选对象中的 `model` 不参与账号选择。
- 同层排序使用 route LoadRate → `LastUsedAt`，不再以账号自身 priority 作为同层主排序键。
- 当前层账号静态过滤、LoadRate 达到 100%、route slot 竞争失败或 account slot 竞争失败，只影响当前层；当前层成功获取账号后立即返回。
- 显式旧格式路由在路由账号全部被过滤后返回 `ErrNoAvailableAccounts`，不再隐式回退到普通账号池。
- 保留 route slot 与 account slot 的双槽位获取及失败释放逻辑。

## 验收标准

- [x] 两个 priority 相同的候选账号进入同一排序池。
- [x] 当前层存在可成功获取槽位的账号时，不选择下一 priority。
- [x] 当前层所有账号 `LoadRate >= 100` 时，才进入下一 priority。
- [x] 当前层排序靠前账号抢槽失败时，会尝试当前层下一个账号。
- [x] route slot 已获取但 account slot 失败时，route slot 被释放。
- [x] 所有层耗尽时返回候选耗尽或无可用账号错误。

## 验证命令与结果

| 类型 | 命令 | 结果 |
|---|---|---|
| priority 分层 | `cd backend; go test -tags unit ./internal/domain -run 'TestGroupCandidatesByPriority' -count=1` | PASS |
| 路由账号池选择 | `cd backend; go test -tags unit ./internal/service -run 'TestRouteCandidate' -count=1` | PASS |
| Gateway 回归 | `cd backend; go test -tags unit ./internal/service -run '^TestGatewayService_SelectAccountWithLoadAwareness$' -count=1` | PASS |
| 服务包编译 | `cd backend; go test -tags unit ./internal/service -run '^$'` | PASS |

## 变更证据

- `backend/internal/domain/model_routing.go`
- `backend/internal/domain/model_routing_test.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/gateway_account_bound_model_routing_test.go`

## 执行记录

- 2026-08-17：完成 priority 分层、同层合并与 route LoadRate 选择；补充旧格式显式路由无可用账号时的严格返回；通过验收测试。
