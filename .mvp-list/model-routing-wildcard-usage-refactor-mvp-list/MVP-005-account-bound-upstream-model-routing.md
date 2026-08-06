# MVP-005：账号绑定上游模型的路由调度

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个专注开发日`
- Estimate rationale: `需重构候选内部数据传递并覆盖模型检查、配额、日志和原调度算法回归，测试面较大。`
- Dependencies: `MVP-001, MVP-002, MVP-003`

## 预期成果

模型路由为每个候选账号从已加载 `Account.Credentials.model_mapping` 的 key 字典序第一项取得上游模型，并携带“账号＋模型”参与现有调度。

## 背景

当前 `trySelectRouteCandidateAccounts` 接收候选级 `routingModelForSelection`。删除候选 `model` 后，同一候选不同账号可能拥有不同模型，必须按账号绑定。

## 范围内

- 增加仅供模型路由使用的账号级候选结构。
- 确定性解析 `model_mapping` key。
- 将账号自身模型用于支持检查、模型限流、动态限额、上游请求和日志。
- 账号无有效模型时过滤并记录原因。
- 保持现有候选 priority、粘性、账号 priority、负载、LRU、随机打散、并发槽和等待逻辑。

## 范围外

- 修改普通调度或账号测试的模型映射逻辑。
- 根据账号 ID 再查数据库获取模型。
- 使用旧路由候选的 `model`。

## 实现说明

- 可在 `Account` 上增加只供此路径调用的确定性 helper，或在 Gateway 路由模块实现局部解析。
- selection result 必须保存所选账号对应的模型，不能在选择后重新从其他来源推导。
- 历史多 key 账号按 key 字典序升序取第一项。

## 验收标准

- [x] 同一候选中两个账号可分别携带不同上游模型。
- [x] 最终上游请求、限额检查和日志使用被选账号对应模型。
- [x] 历史多 key 账号多次执行结果确定一致。
- [x] 无模型账号被跳过，其他候选仍可继续。
- [x] 现有多账号选择顺序和非模型路由行为未改变。

## 验证计划

- `cd backend && go test ./internal/service -run 'ModelRouting|RouteCandidate|GatewayMultiPlatform|GroupRoute'`
- `cd backend && go test ./internal/integration -run 'TokenQuotaModelRouting'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 确定性解析 | `Account.FirstModelMappingKey` | 仅供模型路由调用，去空白后按 key 升序取第一项；不改变其他 model_mapping 逻辑。 |
| 账号级传递 | `GatewayService.trySelectRouteCandidateAccounts` | 过滤阶段建立 accountID→upstreamModel；模型限流、动态限额、日志及最终 selection 均使用对应账号模型。 |
| 选择结果 | `AccountSelectionResult` | 选中后同时保存 RequestedModel/RouteAlias 与该账号 UpstreamModel，下游请求和 usage 继续使用既有 selection 身份字段。 |
| 服务测试 | `cd backend && go test ./internal/service -run 'ModelRouting|RouteCandidate|GatewayMultiPlatform|GroupRoute|AccountFirstModel' -count=1` | 通过。 |
| 集成契约 | `cd backend && go test ./internal/integration -run 'TokenQuotaModelRouting' -count=1` | 通过。 |

## 执行记录

- 账号级候选通过已有 `Account` 列表加局部 `accountModels map[int64]string` 表达，避免二次查库或在选择后重新推导。
- 同候选测试中账号 1=`model-a`、账号 2=`model-b`，负载算法选中账号 2 后 UpstreamModel 明确为 `model-b`。
- 无模型账号被过滤，后续有效账号继续参与；历史三 key 连续 20 次均得到 `alpha-model`。
- 保留候选 priority、粘性、账号 priority、负载、LRU、随机打散、槽位和等待分支；非路由逻辑没有调用新 helper。
- `go test -tags unit` 额外检查被仓库既有 `gateway_record_usage_test.go` 中过时的 `NewGatewayService` 参数数量阻断，与本 MVP 改动无关；清单规定的普通测试命令均通过。
