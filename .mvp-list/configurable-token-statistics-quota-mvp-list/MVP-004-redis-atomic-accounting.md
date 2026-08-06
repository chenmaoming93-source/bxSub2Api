# MVP-004：交付 Redis 原子多周期累计

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `独立完成新命名空间、分片、Lua累计、版本和脏集合，可由 Redis 单元测试完整验证。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

给定统一用量事件和活跃投影，系统能在新 Redis 命名空间中原子累计自然日、周、月计数、版本和脏标记。

## 背景

实时统计必须使用精确定位和一次 Lua 调用，不得写旧 `sub2api:token_stats:*`。

## 范围内

- 新 Redis Key/field 编码。
- 固定分片算法。
- Lua 多投影、多指标、多周期累计。
- 版本 Hash。
- 脏集合。
- 兜底 TTL。
- 操作数上限和输入验证。
- miniredis 或等效测试。

## 范围外

- 请求异步队列。
- MySQL同步。
- 限额查询。

## 实现说明

- Key 前缀必须是 `sub2api:dynamic_token_stats:v1:*`。
- Lua执行失败不得产生部分投影成功的可见状态。
- 维度原始JSON保留给后续同步，不仅保存不可逆哈希。

## 验收标准

- [x] 单事件正确生成三周期计数。
- [x] 多投影和多指标在一次 Lua 调用中原子增加。
- [x] 版本与计数同步递增。
- [x] 脏集合包含可恢复的精确统计身份。
- [x] 并发累计结果无丢失。
- [x] 新代码不读写旧 Redis 前缀。

## 验证计划

- `cd backend && go test ./internal/repository/... -run "DynamicToken|TokenStat"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/tokenstat/redis_accumulator.go` | 独立 Key、固定分片、单 Lua 多操作累计、版本、脏身份和 TTL 已实现 |
| 测试 | `cd backend && go test ./internal/repository/tokenstat -run "DynamicToken\|TokenStat"` | 通过；覆盖三周期、并发累计、版本、TTL 和旧前缀隔离 |

## 执行记录

2026-07-30：完成 Redis 原子累计和并发测试，全部验收项通过。
