# MVP-002：仓储按用户、调用系统和分组查询 Key

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `聚焦一个仓储查询和创建前重复检查，并以仓储测试验证软删除与多分组行为。`
- Dependencies: `MVP-001`

## 预期成果

API Key 仓储使用 `(user_id, platform, group_id, deleted_at IS NULL)` 定位平台 Key，创建前的并发重复检查也采用相同三元组语义。

## 背景

当前 `backend/internal/repository/api_key_repo.go` 的 `GetByUserIDAndPlatform` 和创建前检查只包含用户与 platform，会阻止同一调用系统在不同分组下创建 Key。

## 范围内

- 实现或重命名三元组仓储查询方法。
- 查询只命中未软删除记录。
- 创建前 uniqueness check 加入 `group_id`。
- 保留用户行锁策略和冲突错误转换。
- 更新仓储 stub、接口实现和相关测试。
- 验证已软删除三元组可重新创建。

## 范围外

- 不增加数据库索引。
- 不改变普通 `purpose = user_created` Key 的创建规则。
- 不实现分组访问授权。

## 实现说明

- platform Key 的重复条件必须同时满足 `user_id`、`platform`、`group_id` 且 `deleted_at IS NULL`。
- 默认 Key 的单用户唯一规则保持不变。
- 如果旧的二元组方法无其他调用方，应移除或明确重命名，防止继续误用。

## 验收标准

- [x] 三元组仓储方法能读取目标 Key。
- [x] 不同 groupID 不互相冲突。
- [x] 相同三元组重复创建返回 `ErrAPIKeyExists`。
- [x] 已软删除相同三元组不阻止重新创建。
- [x] 默认 Key 唯一行为未发生回归。
- [x] 定向仓储测试通过。

## 验证计划

- `cd backend && go test ./internal/repository -run 'Test.*APIKey.*Platform|Test.*Platform.*Purpose'`
- 若定向名称与现有测试不匹配，则执行 `cd backend && go test ./internal/repository` 并在执行记录中注明。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/api_key_repo.go` | 查询及创建前重复检查均使用 `user_id + platform + group_id + deleted_at IS NULL`；保留用户行锁与默认 Key 唯一逻辑。 |
| 测试 | `backend/internal/repository/api_key_repo_last_used_unit_test.go` | 覆盖同三元组冲突、不同分组共存、三元组读取、软删除后重建和默认 Key 回归。 |
| 验证 | `cd backend && go test ./internal/repository -run 'Test.*APIKey.*Platform|Test.*Platform.*Purpose'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 6.016s`。 |

## 执行记录

2026-07-16：完成真实仓储三元组查询和创建前重复检查，并通过定向仓储测试。
