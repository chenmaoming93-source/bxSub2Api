# MVP-005：接入非阻塞用量事件采集

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `以统一事件构造、有界队列、Worker和代表性协议接入构成一个可观察的非阻塞纵向切片。`
- Dependencies: `MVP-003, MVP-004`

## 预期成果

模型调用完成后能够非阻塞提交统一事件，异步Worker写入新Redis统计；队列满或Redis失败不改变模型响应。

## 背景

统计入库必须与模型主流程隔离。所有协议最终都需提供六个首期维度及 `total_tokens`，缺失时明确记录而非伪造。

## 范围内

- 有界内存队列和Worker生命周期。
- 批量、超时、短重试和优雅关闭。
- 统一事件构造器。
- 主要模型调用完成路径接入。
- 队列满、字段缺失、Redis失败监控。
- fail-open测试。

## 范围外

- MySQL同步。
- 限额读取。
- 性能压测报告。

## 实现说明

- 请求线程使用 `tryEnqueue`，不得等待队列。
- 请求完成时间决定周期归属。
- 不调用旧 `TokenStatisticsIncrement` 或旧累计函数。

## 验收标准

- [x] 请求线程不等待Redis或MySQL。
- [x] 正常调用产生完整统一事件。
- [x] 队列满时响应内容和状态不变。
- [x] Redis失败并重试耗尽后响应不变。
- [x] 缺失维度或指标有可观察错误且不写错误桶。
- [x] 代表性流式、非流式和至少三类协议测试通过。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/handler/... -run "DynamicToken|AsyncUsage|RecordUsage"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 队列与 Worker | `backend/internal/service/tokenstat/async_pipeline.go` | 有界 `tryEnqueue`、超时、短重试、排空关闭和可观察计数已实现 |
| 网关接入 | `backend/internal/service/dynamic_token_usage.go`、`backend/internal/service/gateway_service.go` | 使用日志成功后提交完整六维事件；调用仅做内存非阻塞入队 |
| 装配 | `backend/cmd/server/wire_gen.go` | 独立配置启用时装配活动投影、Redis Writer 和 Pipeline |
| 测试 | `cd backend && go test ./internal/service/... ./internal/handler/... ./internal/repository/tokenstat ./cmd/server -run "DynamicToken\|AsyncUsage\|RecordUsage\|AsyncPipeline"` | 通过；覆盖 sync/stream/ws、六维事件、队列满、缺维度和 Redis 失败 |

## 执行记录

2026-07-30：完成非阻塞采集、网关共享记录路径接入、运行装配和 fail-open 测试。
