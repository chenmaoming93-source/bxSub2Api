# MVP-002：移除旧请求路径统计、限额与后台任务

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 开发日`
- Estimate rationale: `涉及模型请求成功记账、调度前后限额、Redis 修复/同步和 Wire 启动链，需跨协议回归。`
- Dependencies: `MVP-001`

## 预期成果

模型请求不再写入或读取三套旧固定 Redis/限额逻辑，旧同步与读修复任务不再启动，同时新动态统计与限额保持工作。

## 背景

旧逻辑与模型调用、路由候选选择和成功 usage 记录共享调用点，只能删除旧分支，不能破坏公共主流程。

## 范围内

- 删除旧固定统计累计和 Redis codec。
- 删除旧 DailyTokenQuota 请求/候选检查与缓存。
- 删除旧 Redis→MySQL 同步、读修复和调度器启动。
- 更新 Wire 和相关单元测试。

## 范围外

- 旧管理 API、前端和 Ent schema。

## 实现说明

- 保留统一 `recordUsage` 对新 `TryEnqueueDefault` 的调用。
- 保留计费、余额、API Key、订阅、路由、账号调度和错误重试。

## 验收标准

- [x] 三类旧统计 Redis 写入停止。
- [x] 三类旧限额不再参与请求或候选调度。
- [x] 旧同步和读修复后台任务不再启动。
- [x] 新动态统计、动态限额、模型调用和计费测试通过。
- [x] 生产代码不再引用旧 Redis 前缀。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/repository/... ./cmd/server/...`
- `rg -n "sub2api:token_stats:|quota:daily_token:|dailyTokenQuotaRepo|TokenStatisticsAccumulator" backend --glob "*.go"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 请求链路 | `backend/internal/service/gateway_service.go`、`openai_gateway_service.go`、`openai_group_route_quota.go` | 已移除旧累计器、旧 DailyTokenQuota 仓储注入和候选限额判断；保留动态配额检查与 `submitDynamicTokenUsage` |
| 后台任务 | `backend/internal/{repository,service}/wire.go`、`backend/cmd/server/wire*.go` | 旧同步引擎、调度器及 cleanup 启停依赖已移除 |
| 测试 | `cd backend && go test ./internal/service/... ./internal/repository/... ./cmd/server/...` | 通过 |
| 静态扫描 | `rg -n "sub2api:token_stats:|quota:daily_token:|dailyTokenQuotaRepo|TokenStatisticsAccumulator|TokenStatisticsScheduler" backend --glob "*.go"` | 生产代码无旧前缀或旧运行时依赖；剩余仅为测试中的保护断言/待清理注释 |

## 执行记录

2026-07-30：开始移除旧请求路径统计、限额检查和后台调度链。
2026-07-30：恢复并验证原计费语义，确认删除旧统计不会改变计费；完成运行时、Wire 与后台任务清理并通过回归。
