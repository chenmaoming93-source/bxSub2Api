# MVP-001：建立可取消的下游 Context 桥接

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个聚焦开发日`
- Estimate rationale: 聚焦现有 `detachStreamUpstreamContext` 生命周期、`context.WithoutCancel` 保留语义和单元测试，形成所有协议可复用的取消基础。
- Dependencies: `none`

## 预期成果

下游模型请求继续保留 `context.WithoutCancel` 的上下文值和隔离用途，但能够在原始入站请求取消或显式断开事件发生时被主动取消。

## 背景

当前流式转发路径在构造下游请求时使用 `context.WithoutCancel`，调用方断开后下游请求可能继续 drain。现有 `backend/internal/service/gateway_service.go` 中的 `detachStreamUpstreamContext` 还需要调整生命周期，避免过早释放取消控制权。

本 MVP 不接入具体协议的最终业务分支，也不改变模型路由降级策略。

## 范围内

- 设计并实现可复用的下游 Context 取消桥接；
- 保留 `context.WithoutCancel` 作为基础 Context；
- 监听原始入站 Context 的 `Done()` 并桥接到下游 `cancel()`；
- 明确下游请求完成后的取消控制器释放时机；
- 覆盖父 Context 取消、显式取消、正常完成三种生命周期；
- 增加 Context 值保留和取消传播测试；
- 增加与正常 failover 判定边界相关的基础测试或测试辅助方法。

## 范围外

- 不接入 `ops_error_logs`；
- 不修改 `/admin/usage`；
- 不修改模型账号选择、候选排序、重试次数或降级优先级；
- 不覆盖具体 HTTP/SSE/WebSocket 协议的完整业务处理。

## 实现说明

- 重点检查 `backend/internal/service/gateway_service.go` 中 `detachStreamUpstreamContext` 的返回值和生命周期；
- 下游 Context 的取消函数必须持有到模型请求完成，而不是只持有到请求对象构造完成；
- 原始入站 Context 的取消只负责通知下游取消，不应直接改变正常模型上游错误的 failover 分类；
- 设计应允许后续 SSE 写失败和 WebSocket 断开复用同一个显式 `cancel()`；
- 取消桥接停止后不得遗留 watcher 或 goroutine。

## 验收标准

- [x] `context.WithoutCancel` 仍能保留原始 Context 的值；
- [x] 原始入站 Context 取消后，下游 Context 的 `Done()` 能够关闭；
- [x] 显式调用下游 `cancel()` 不会修改原始入站 Context；
- [x] 正常完成后取消桥接被释放，不产生 goroutine 泄漏；
- [x] 下游 Context 生命周期覆盖真实转发过程；
- [x] 现有 failover 相关测试不因该基础能力失败。

## 验证计划

- `go test ./internal/service/...`
- 重点检查并补充 `gateway_context_management_test.go`、`gateway_streaming_test.go` 相关测试；
- 使用测试 Context 验证 `Done()`、`Err()`、Context value 和取消控制器生命周期。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 代码 | `backend/internal/service/gateway_service.go` | 新增 `streamUpstreamContextControl`，保留 `context.WithoutCancel` 基础语义并桥接原始 Context 取消。 |
| 测试 | `backend/internal/service/stream_upstream_context_test.go` | 覆盖 Context value 保留、父取消、显式取消、Release 幂等和非流式路径。 |
| 命令 | `go test ./internal/service/...`（工作目录：`backend`） | 通过：service、openai_ws_v2、tokenstat 测试全部通过。 |

## 执行记录

- 2026-09-16：实现可复用 Context 控制器和父 Context 取消桥接。
- 2026-09-16：保留现有 `detachStreamUpstreamContext` 调用兼容性；具体协议的显式 cancel 生命周期由后续协议 MVP 接入。
- 2026-09-16：服务层测试通过，MVP 验证完成。

