# MVP-001：路由 priority 分层与配置校验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦领域层 priority 分组、重复账号校验和旧字段兼容，不修改 Gateway 调度主流程。
- Dependencies: `none`

## 预期成果

形成可被 Gateway 使用的 priority 分层数据，并在保存/解析阶段明确处理同别名重复账号和候选 `model` 兼容字段。

## 背景

当前 parser 已按 `priority` 稳定排序，但 Gateway 仍逐候选处理。数据库对 `(group_id, route_alias, account_id)` 有唯一约束，跨 priority 重复账号需要在业务层给出明确错误。

## 范围内

- 增加或整理按 priority 分组的领域辅助逻辑；
- 同一 priority 内账号去重；
- 同一别名跨 priority 重复账号校验；
- 保持旧数组格式解析；
- 确认候选 `model` 可缺省并仅兼容读取；
- 添加领域层单元测试。

## 范围外

- 不修改 Gateway 账号选择；
- 不新增 Redis 负载读取；
- 不修改前端布局。

## 实现说明

- 重点查看 `backend/internal/domain/model_routing.go`、`backend/internal/service/group.go` 和相关 repository 同步逻辑；
- 分层结果应保持 priority 升序；
- 跨 priority 重复账号错误需要包含 route alias、account ID 和 priority；
- 不能破坏旧格式 `map[string][]int64`。

## 验收标准

- [x] 相同 priority 的候选可以被分组成同一层。
- [x] 同一层重复账号不会在返回的账号池中出现多次。
- [x] 同一别名账号出现在不同 priority 时返回明确配置错误。
- [x] 旧格式和缺少 `model` 的候选格式仍可解析。

## 验证计划

- `cd backend; go test ./internal/domain ./internal/service ./internal/repository`
- 重点运行 `model_routing_test.go`、`group_model_routing_test.go` 和新增重复校验测试。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test ./internal/domain ./internal/service ./internal/repository` | 通过；domain、service、repository 均通过 |
| 代码 | `backend/internal/domain/model_routing.go` | 新增 priority tier 分组、同层去重和跨层重复校验 |
| 测试 | `backend/internal/domain/model_routing_test.go` | 新增同层去重和跨层冲突测试 |

## 执行记录

2026-08-17：完成 priority tier 领域辅助逻辑和重复账号校验；运行格式化及 domain/service/repository 回归测试，全部通过。
