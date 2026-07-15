# MVP-002：实现三类 Redis 当天数据批量读取

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 聚焦 Redis Hash 的批量读取、codec 复用和错误分类，不包含写修复与报表接线，约 40 分钟可独立完成并测试。
- Dependencies: `MVP-001`

## 预期成果

Repository 层可以按日期批量读取并解码模型、路由候选和用户模型三类 Redis 当天累计数据，并明确区分成功空集合与 Redis 故障。

## 背景

现有 `backend/internal/repository/token_statistics_codec.go` 已提供 key 与 field codec，`token_statistics_sync.go` 已有批量遍历思路，但查询链路尚无统一批量读取能力。

## 范围内

- 定义并实现 `CurrentTokenUsageReader` 的 Redis 批量读取部分或等价端口。
- 复用 `TokenStatisticsKey` 与三个 `Decode*TokenStatisticsField`。
- 解析非负整数 `used_tokens`，将连接失败、空 Hash、非法 field/value 分开表达。
- 支持查询过滤可安全下推时使用 `HMGET`，否则使用受控 `HSCAN`。
- 为三类 Hash、空集合、连接失败和非法数据补测试。

## 范围外

- 不执行 MySQL 查询。
- 不执行 Redis 写修复。
- 不接入报表服务。

## 实现说明

- 不得新增第二套 field 编码。
- 正常读取应为批量操作，禁止循环逐项 `HGET`。
- 非法条目不得进入最终集合。

## 验收标准

- [x] 三类 Redis Hash 均能解码为正确业务记录。
- [x] 空 Hash 被识别为成功读取的空集合，而不是故障。
- [x] Redis 故障具有独立结果或错误语义，可供调用方降级。
- [x] 非法值与非法 field 有测试覆盖。

## 验证计划

- `cd backend && go test ./internal/repository -run 'Test.*CurrentTokenUsage.*Read|Test.*TokenStatistics.*Bulk'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 端口 | `backend/internal/service/current_token_usage_reader.go` | 定义三维度批量读取端口与包含非法条目计数的成功结果。 |
| 实现 | `backend/internal/repository/current_token_usage_reader.go` | 复用既有 key/field codec；明确过滤键时使用单次 `HMGET`，否则使用受控 `HSCAN`；Redis 命令失败返回错误，非法条目跳过并计数。 |
| 测试 | `backend/internal/repository/current_token_usage_reader_test.go` | 覆盖三类 Hash、空 Hash、过滤读取、连接失败、非法 field、负数 value。 |
| 聚焦验证 | `cd backend && go test ./internal/repository -run 'Test.*CurrentTokenUsage.*Read|Test.*TokenStatistics.*Bulk'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.669s`。 |
| 包回归 | `cd backend && go test ./internal/repository` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 13.591s`。 |

## 执行记录

2026-07-14 完成。空 Hash 返回无错误空集合；连接/命令失败返回独立错误供调用方降级；非法 field/value 不进入结果并通过 `InvalidEntries` 计数。首次测试构建因 `miniredis.HSet` 夹具签名不匹配失败，修正为逐 field/value 写入后聚焦测试及仓储包回归均通过。
