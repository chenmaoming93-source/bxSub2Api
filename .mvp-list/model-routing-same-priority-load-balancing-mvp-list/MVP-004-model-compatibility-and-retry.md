# MVP-004：账号上游模型兼容与失败重试

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Dependencies: `MVP-003`

## 目标

明确候选对象中的 `model` 只是历史兼容字段，不参与账号选择或上游模型决定。账号选中后，真实上游模型始终来自该账号自己的 model mapping；选择失败时，继续遵循当前 priority 层和候选耗尽语义。

## 实现范围

- 新候选对象路径使用请求模型/路由别名进行路由匹配，不使用候选对象 `model` 覆盖选择结果。
- 过滤和上游转发使用选中账号的 `FirstModelMappingValue()`。
- `AccountSelectionResult` 保留请求模型、路由别名和账号上游模型三者的独立身份。
- 当前层账号因负载、route slot 或 account slot 失败时继续尝试当前层剩余账号；当前层耗尽后由上层进入下一 priority 层。
- 保留上游 `UpstreamFailoverError` 及 handler 的排除账号/重试机制，不改变已有错误分类。

## 验收标准

- [x] 候选 `model` 缺失时可以正常选择账号。
- [x] 候选 `model` 修改不会改变真实上游模型。
- [x] 真实上游模型来自选中账号的 model mapping。
- [x] 上游失败后，同 priority 仍有可用账号时优先继续使用同层账号。
- [x] 同 priority 全部耗尽后才进入下一 priority。

## 验证命令与结果

| 类型 | 命令 | 结果 |
|---|---|---|
| 路由字段兼容与上游模型 | `cd backend; go test -tags unit ./internal/domain -run 'TestParseModelRoutingConfigCandidatesStableSort|TestGroupCandidatesByPriority' -count=1` | PASS |
| 账号绑定上游模型 | `cd backend; go test -tags unit ./internal/service -run 'TestRouteCandidate|TestAccountFirstModelMappingValueResolvesUpstreamModel' -count=1` | PASS |
| 上游失败信号 | `cd backend; go test -tags unit ./internal/service -run 'TestHandleNonStreamingResponse_NonJSON2xxTriggersFailover|TestHandleStreamingResponse_StreamReadErrorBeforeOutput_TriggersFailover' -count=1` | PASS |

## 已知验证环境问题

尝试运行 `cd backend; go test -tags unit ./internal/handler` 时，仓库内两个既有测试因 `NewGatewayService`、`NewOpenAIGatewayService` 调用参数数量与当前构造函数签名不一致而无法编译；该问题与本 MVP 的路由改动无关。服务层的上游失败信号测试已独立通过。

## 变更证据

- `backend/internal/service/gateway_service.go`
- `backend/internal/service/gateway_account_bound_model_routing_test.go`
- `backend/internal/domain/model_routing.go`
- `backend/internal/domain/model_routing_test.go`

## 执行记录

- 2026-08-17：确认候选 `model` 不再参与账号选择；补充候选字段被忽略且账号 mapping 决定上游模型的测试；通过路由与上游失败信号验证。
