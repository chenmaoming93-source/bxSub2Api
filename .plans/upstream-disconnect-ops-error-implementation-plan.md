# 上游服务断开后取消下游请求并记录 Ops 错误实施计划

> **状态：Final — user approved**  
> **版本：v1.0**  
> **日期：2026-09-16**  
> **变更摘要：** 当调用 sub2api 的上游客户端/服务断开时，立即取消下游模型请求，并仅向 `ops_error_logs` 写入专用错误记录，使其出现在 `/admin/usage` 的“错误请求”菜单中；不写入 `usage_logs`，不产生 Token 或费用统计。

---

## 1. 背景与目标

### 1.1 背景

当前部分流式请求在调用方断开后，仍会继续读取和处理下游模型响应：

- 流式请求使用了 `context.WithoutCancel`；
- 检测到客户端写出失败后，代码仍会继续 drain 下游响应；
- 并发槽位可能已经释放，但下游模型连接仍然保持；
- 导致系统显示的账号并发低于下游模型实际承受的并发。

相关现有逻辑主要位于：

- `backend/internal/service/gateway_service.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_ws_forwarder.go`
- `backend/internal/handler/gateway_handler*.go`
- `backend/internal/handler/openai_gateway_handler.go`

### 1.2 目标

| ID | 目标 |
|---|---|
| G-01 | 调用 sub2api 的上游客户端断开后，立即取消下游模型请求 |
| G-02 | 关闭下游 HTTP、SSE、WebSocket 等连接及响应体 |
| G-03 | 该请求不执行账号切换、模型降级或重试 |
| G-04 | 只写入 `ops_error_logs`，不写入 `usage_logs` |
| G-05 | 不产生 Token、费用、配额或动态统计数据 |
| G-06 | 在 `/admin/usage` 的“错误请求”菜单中可查询 |
| G-07 | 使用专用错误类型明确标识“上游服务断开” |

### 1.3 非目标

- 不改变正常模型上游错误的 failover、账号切换和路由降级逻辑；
- 不改变正常请求响应格式；
- 不为断开请求补记 Token；
- 不为该场景新增 `usage_logs` 字段；
- 不把该错误标记为 `cyber`、429 或普通 `upstream_error`；
- 不改变调用方已经断开后的响应内容。

---

## 2. 已确认的决策与假设

### 2.1 已确认决策

1. “上游服务断开”指**调用 sub2api 的上游客户端/服务断开**，包括：
   - 入站请求 Context 被取消；
   - SSE 向调用方写数据失败；
   - WebSocket 调用方连接断开。

2. 覆盖全部网关协议和路径：
   - HTTP 非流式；
   - SSE 流式；
   - OpenAI WebSocket；
   - Gemini、Anthropic、OpenAI 兼容路径；
   - 图片流式路径如适用。

3. 错误类型使用：

   ```text
   error_type = client_disconnected
   ```

   前端显示名称为：

   ```text
   上游服务断开
   ```

4. 不写入 `usage_logs`，只写入 `ops_error_logs`。

5. 遵循现有 Ops monitoring 开关：
   - Ops monitoring 开启：写入并可查询；
   - Ops monitoring 关闭：沿用当前 Ops 行为，不强制绕过开关。

6. 保留 `context.WithoutCancel` 在原有需要脱离请求取消的场景中的用途。下游请求不直接依赖其取消传播，而是基于它创建可主动取消的 Context，并监听原始入站 Context，将取消信号桥接到下游请求。

7. 不修改模型路由降级策略。仅增加调用方已断开时的提前终止保护；模型供应商自身的 429、5xx、连接重置和超时仍按原有 failover 逻辑处理。

### 2.2 保守假设

- 数据库正常可用时，系统应确保该错误记录进入 Ops 写入链路；
- 数据库不可用时无法承诺绝对持久化，但必须记录失败日志和指标；
- 同一个请求在单次处理生命周期内只记录一次断开错误；
- `request_type` 继续表示 `sync`、`stream`、`ws_v2`，不用于表示错误类型。

---

## 3. 功能设计

### 3.1 请求断开检测与下游取消

#### 目的

调用方断开后，终止当前请求关联的下游模型调用，避免残留连接和模型侧幽灵并发。

#### 行为

1. 为每个下游转发请求创建可取消 Context；
2. 监听原始入站请求 Context；
3. 检测到 SSE/HTTP 写出失败或 WebSocket 断开时，调用下游 Context 的 `cancel()`；
4. 下游 HTTP 请求收到取消信号；
5. 关闭下游响应体和相关连接；
6. 释放账号、用户、队列等并发资源；
7. 进入错误记录流程；
8. 不进入账号切换或模型降级流程。

### 3.2 路由降级保护

取消下游请求本身不会改变正常路由降级，但必须区分错误原因。

| 错误原因 | 行为 |
|---|---|
| 调用方断开 | 取消当前下游请求，停止重试和 failover |
| 模型供应商返回 429/5xx | 保留现有 failover |
| 模型供应商连接异常，但调用方仍在线 | 保留现有 failover |
| 模型供应商响应超时 | 按现有策略处理 |
| 调用方断开后出现 `context.Canceled` | 不当作模型账号故障，不切换账号 |

特别要求：

- failover 前先判断是否为入站请求取消；
- 不能把调用方断开误判为模型账号异常；
- 不能因调用方断开而触发账号临时禁用或错误计数；
- 不调整模型候选排序、账号选择、重试次数或降级优先级。

### 3.3 Ops 错误记录

#### 存储位置

```text
ops_error_logs
```

不写入：

```text
usage_logs
```

#### 记录字段

| 字段 | 值 |
|---|---|
| `error_type` | `client_disconnected` |
| `error_phase` | `network` |
| `error_owner` | `client` |
| `error_source` | `client_request` |
| `status_code` | `499` |
| `is_business_limited` | `false` |
| `request_type` | 原请求类型：sync/stream/ws_v2 |
| `stream` | 原请求流式标记 |
| `error_message` | 稳定、可检索的断开错误描述 |
| `request_id` | 当前请求 ID |
| `client_request_id` | 客户端请求 ID，如存在 |
| `user_id` | 当前用户 |
| `api_key_id` | 当前 API Key |
| `account_id` | 已选模型账号 |
| `group_id` | 当前分组 |
| `model` | 请求模型 |
| `requested_model` | 客户端请求模型 |
| `upstream_model` | 实际下游模型，如存在 |
| `request_path` | 入站路径 |
| `inbound_endpoint` | 入站标准端点 |
| `upstream_endpoint` | 下游标准端点 |
| `duration_ms` | 请求持续时间 |
| `time_to_first_token_ms` | 已获得时记录，否则为空 |

> 注意：字段表只描述**逻辑记录内容**，不等于线上 schema。`is_retryable` 属于已下线的 Ops 重试功能——迁移 `136_remove_ops_retry_replay.sql` 已随 `ops_retry_attempts` 一起删除该列，生产库中并不存在，**不得**再作为持久化字段写入 INSERT；断开的「不可重试」语义由 `error_type` / `status_code` / `error_owner` 表达即可。任何新增落库字段都必须先核对线上 schema，并按 `PROJECT_CONVENTIONS.md` §2.2 把 DDL 放进 `backend/sqlArchiving/`。

不记录：

- Token 数；
- 费用；
- 请求 Body；
- 敏感凭证；
- 下游供应商错误码，除非已有安全的诊断信息。

### 3.4 `/admin/usage` 错误请求展示

当前页面已经通过以下链路读取错误记录：

```text
/admin/usage
  └── 错误请求 Tab
      └── GET /api/v1/admin/ops/errors
          └── ops_error_logs
```

实施内容：

- 增加 `client_disconnected` 的前端显示名称；
- 显示为“上游服务断开”；
- 支持按错误类型筛选；
- 保持现有错误详情弹窗和分页格式；
- 不改变 `/admin/usage` 的接口返回结构。

### 3.5 统计与计费隔离

断开请求不得进入：

- `usage_logs`；
- Token 统计投影；
- 动态 Token 用量；
- 用户余额扣费；
- 订阅用量；
- API Key 配额；
- 账号用量统计；
- 成功请求统计。

该请求只作为 Ops 错误事件存在。

---

## 4. 技术设计

### 4.1 Context 设计

#### 下游请求 Context

保留 `context.WithoutCancel` 的原有价值，但增加显式取消桥接：

```text
inboundCtx = c.Request.Context()
detachedCtx = context.WithoutCancel(inboundCtx)
downstreamCtx, cancel = context.WithCancel(detachedCtx)
```

同时监听原始 `inboundCtx.Done()`：

```text
when inboundCtx.Done():
    cancel()
```

下游 HTTP/SSE/WebSocket 请求使用 `downstreamCtx`。

当 SSE/HTTP 写出失败或 WebSocket 断开时，也调用同一个 `cancel()`。

需要调整现有 `detachStreamUpstreamContext` 的生命周期：不能在仅构造下游请求后立即释放取消控制权，而应持有到下游请求完成。这样既保留 `WithoutCancel` 的上下文值和隔离用途，又能在调用方断开时主动取消下游请求。

#### 后置记录 Context

错误入库不能依赖已取消的入站 Context。记录 Ops 错误时使用：

```text
background context
+ request metadata
+ bounded timeout
```

因此：

- 下游请求：允许被取消；
- Ops 入库：脱离客户端连接生命周期；
- 计费、错误入库等后置操作仍可使用脱离客户端取消的 Context；
- 不能全局删除所有 `context.WithoutCancel` 使用。

### 4.2 断开事件协调器

建议抽取统一的请求断开协调逻辑，避免各协议重复实现。

职责：

1. 保证断开事件只触发一次；
2. 调用下游 `cancel()`；
3. 标记当前请求为 `client_disconnected`；
4. 阻止后续 failover；
5. 触发一次 Ops 错误记录；
6. 协调连接关闭与并发槽位释放。

### 4.3 Ops 写入可靠性

现有 Ops 写入链路为异步队列。需要补充以下保障：

- 使用脱离入站请求的短超时 Context；
- 队列成功提交后由后台 worker 写入；
- 队列满时提供同步 fallback；
- 记录队列丢弃、写入失败和 fallback 次数；
- 同一请求生命周期内通过请求级标记避免重复写入；
- 保留现有敏感字段清理和错误体截断逻辑。

在 Ops monitoring 关闭时，遵循现有 `OpsService` 行为，不写入并且不展示。

### 4.4 路由降级判定

建议使用以下判定顺序：

```text
forward request

if inbound request is disconnected:
    cancel downstream request
    close downstream body/connection
    release concurrency
    record client_disconnected
    return
else if error is failover-worthy model upstream error:
    execute existing retry/failover logic
else:
    execute existing ordinary error handling
```

关键约束：

- `context.Canceled` 只有在确认来自入站调用方断开时，才归类为 `client_disconnected`；
- 不能把模型供应商主动关闭连接误分类为客户端断开；
- 模型供应商断开仍然走原有 failover；
- 不修改模型路由降级算法，仅在进入该算法前增加断开请求的终止保护。

---

## 5. 关键流程伪代码

### 5.1 普通 HTTP/SSE 请求

```text
function forwardRequest(request):
    inboundCtx = request.Context
    detachedCtx = context.WithoutCancel(inboundCtx)
    downstreamCtx, cancel = context.WithCancel(detachedCtx)
    stopParentBridge = watch(inboundCtx.Done, cancel)

    disconnectOnce = once()
    markDisconnected = function(reason):
        disconnectOnce.do:
            state = CLIENT_DISCONNECTED
            cancel()
            close downstream response/body
            enqueueOpsDisconnectError(request metadata, reason)

    result, err = forwardToModel(downstreamCtx)
    stopParentBridge()

    if state == CLIENT_DISCONNECTED:
        release concurrency
        stop retry/failover
        return

    if err is failover-worthy model upstream error:
        release current account slot
        continue existing failover

    if err:
        execute existing ordinary error path
        return

    record normal usage only when request completed normally
```

### 5.2 SSE 写出失败

```text
when write response event fails:
    markDisconnected("downstream client write failed")
    stop writing further response events
    cancel model request
    close upstream response body
    do not drain remaining upstream response
    do not retry or switch account
    write Ops error asynchronously
```

### 5.3 WebSocket 请求

```text
when client WebSocket read/write/close indicates disconnect:
    markDisconnected("client websocket disconnected")
    cancel model-side WebSocket/HTTP relay
    close relay and model connection
    do not start another account retry
    write one Ops error record
```

### 5.4 Ops 错误入库

```text
function recordClientDisconnected(meta):
    if request already recorded:
        return

    if ops monitoring disabled:
        return

    entry = OpsInsertErrorLogInput(
        RequestID = meta.RequestID,
        ClientRequestID = meta.ClientRequestID,
        UserID = meta.UserID,
        APIKeyID = meta.APIKeyID,
        AccountID = meta.AccountID,
        GroupID = meta.GroupID,
        Model = meta.Model,
        RequestType = meta.RequestType,
        Stream = meta.Stream,
        ErrorPhase = "network",
        ErrorType = "client_disconnected",
        ErrorOwner = "client",
        ErrorSource = "client_request",
        StatusCode = 499,
        IsBusinessLimited = false,
        ErrorMessage = stable disconnect message
    )

    submit with detached bounded context
    fallback to synchronous insert when queue is full
```

---

## 6. 安全、隐私与兼容性

### 6.1 安全

- 不保存 API Key 明文；
- 不保存完整请求 Body；
- 错误消息经过现有 Ops 清理与截断逻辑；
- 沿用现有管理员权限 `ops.read`；
- 不新增公开接口；
- 不改变用户权限边界。

### 6.2 兼容性

- 不修改 `usage_logs` 表；
- 不修改现有 usage API 数据结构；
- 不修改正常请求计费逻辑；
- 不修改正常模型上游错误的路由降级逻辑；
- `request_type` 保留原协议语义；
- `error_type` 扩展为新的业务错误值；
- 保留 `context.WithoutCancel` 在计费、错误入库及其他后置逻辑中的既有用途。

### 6.3 数据库迁移

本方案原则上不需要数据库结构迁移：

- `ops_error_logs.error_type` 已为字符串字段；
- `ops_error_logs` 已具备请求、账号、模型、错误分类等字段；
- 只需增加应用层错误类型识别、展示和筛选支持。

如当前部署版本缺少相关 Ops 字段，应先做版本兼容检查，不得直接假设线上 schema 与代码完全一致。

---

## 7. 验证策略

### 7.1 单元测试

| ID | 验证内容 |
|---|---|
| UT-01 | 入站 Context 取消会通过桥接触发下游 Context 取消 |
| UT-02 | `context.WithoutCancel` 仍保留上下文值，但不会阻止显式下游取消 |
| UT-03 | SSE 写出失败会触发下游取消 |
| UT-04 | WebSocket 断开会触发 relay 关闭 |
| UT-05 | 断开事件重复触发时只生成一条 Ops 记录 |
| UT-06 | `client_disconnected` 不进入 failover |
| UT-07 | 模型供应商 429/5xx 仍然执行原有 failover |
| UT-08 | 断开错误不调用 usage 记录服务 |
| UT-09 | Ops 队列满时会执行同步 fallback |
| UT-10 | Ops monitoring 关闭时遵循现有跳过逻辑 |

### 7.2 集成测试

- 使用可控 HTTP 上游，确认客户端断开后下游请求 Context 收到取消；
- 确认下游响应 Body 被关闭；
- 确认账号并发槽位释放；
- 确认未发生账号切换；
- 确认 `ops_error_logs` 记录字段完整；
- 确认 `usage_logs` 没有新增记录；
- 确认 Token 统计、费用和配额没有变化；
- 确认模型供应商错误仍按原逻辑执行降级。

### 7.3 端到端测试

覆盖：

- Anthropic Messages；
- OpenAI Chat Completions；
- OpenAI Responses；
- Gemini 兼容接口；
- OpenAI WebSocket；
- SSE 首 Token 前断开；
- SSE 首 Token 后断开；
- 非流式请求取消；
- 图片流式请求取消。

### 7.4 管理后台验证

确认：

```text
/admin/usage
  -> 错误请求
  -> 请求类型/错误类型列表
  -> 显示“上游服务断开”
  -> 可分页、筛选、查看详情
```

并验证该记录不会出现在“用量明细”中。

---

## 8. 验收标准

| ID | 验收标准 |
|---|---|
| AC-01 | 调用方断开后，下游模型连接在可接受的短时间内被取消 |
| AC-02 | 下游响应体、HTTP/SSE/WebSocket relay 均被关闭 |
| AC-03 | 断开请求不会触发账号切换、模型降级或重试 |
| AC-04 | 正常模型上游 429/5xx/连接错误仍保留原有 failover |
| AC-05 | 每次断开请求最多产生一条 `client_disconnected` Ops 记录 |
| AC-06 | Ops 记录的 `status_code` 为 499，错误类型为 `client_disconnected` |
| AC-07 | `/admin/usage` 的“错误请求”菜单能查到该记录 |
| AC-08 | 该请求不会新增 `usage_logs` |
| AC-09 | 该请求不会产生 Token、费用、余额、配额或动态统计 |
| AC-10 | Ops monitoring 关闭时遵循现有 Ops 行为 |
| AC-11 | 不泄露请求 Body、API Key 或内部网络地址 |
| AC-12 | 所有目标协议的自动化测试通过 |
| AC-13 | 原有模型路由降级测试不回归 |

---

## 9. 实施顺序与后续任务拆分边界

### 阶段一：现状基线与断开分类

产出：

- 梳理所有协议的下游转发入口；
- 标记当前使用 `context.WithoutCancel` 或 drain 行为的路径；
- 统一定义 `client_disconnected` 错误分类；
- 明确正常 failover 与调用方断开的判定边界。

### 阶段二：下游取消协调能力

产出：

- 统一可取消的下游请求 Context；
- 保留 `context.WithoutCancel` 基础 Context 并增加入站取消桥接；
- 统一断开事件协调器；
- HTTP/SSE/WebSocket 关闭和资源释放逻辑；
- 断开后停止 failover/retry。

### 阶段三：Ops 错误记录

产出：

- 构造 `OpsInsertErrorLogInput`；
- 接入 `OpsService` 异步写入；
- 增加队列满时的同步 fallback；
- 增加重复记录保护和失败指标。

### 阶段四：全部协议接入

产出：

- HTTP 非流式；
- SSE 流式；
- OpenAI WebSocket；
- Gemini/Anthropic/OpenAI 兼容路径；
- 图片流式路径。

### 阶段五：管理后台展示

产出：

- `client_disconnected` 类型中文/英文显示；
- 错误类型筛选支持；
- 详情显示字段；
- `/admin/usage` 错误请求 Tab 验证。

### 阶段六：回归与灰度

产出：

- 单元、集成、端到端测试；
- 并发与连接释放验证；
- 监控指标和日志验证；
- 灰度启用及回滚方案。

---

## 10. 风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| 将模型供应商断开误判为调用方断开 | 错误跳过 failover | 使用入站 Context、写出错误和 relay 状态联合判定 |
| 取消传播过早导致正常请求中断 | 请求失败率升高 | 先增加分类和测试，再启用取消行为 |
| Ops 队列满导致记录丢失 | 错误无法查询 | 增加同步 fallback、队列丢弃指标和告警 |
| 断开后仍有异步统计 | 污染 Token/费用统计 | 断开路径不得调用任何 usage 记录函数 |
| WebSocket 仍保留独立 drain 逻辑 | 下游连接继续存在 | 单独覆盖 WS relay 测试 |
| Ops monitoring 关闭 | 页面不可见 | 明确沿用现有开关行为 |
| 多层 handler 重复写入 | 重复错误记录 | 使用请求级 once/标记统一记录 |
| 修改 Context 生命周期影响已有请求 | 正常请求异常取消 | 保留 `WithoutCancel` 基础语义，使用独立显式 cancel，并补充回归测试 |

---

## 11. 追踪矩阵

| 目标 | 功能模块 | 主要技术组件 | 验收标准 |
|---|---|---|---|
| G-01 | 下游取消 | Gateway Service、HTTP/SSE/WS relay | AC-01、AC-02 |
| G-02 | 连接关闭 | Forwarder、Response Body、WS relay | AC-02 |
| G-03 | 路由保护 | Handler failover guard | AC-03、AC-04、AC-13 |
| G-04 | Ops 持久化 | OpsService、OpsRepository | AC-05、AC-06 |
| G-05 | 统计隔离 | Gateway usage recording boundary | AC-08、AC-09 |
| G-06 | 后台查询 | `/admin/ops/errors`、UsageView | AC-07 |
| G-07 | 专用错误特征 | `error_type=client_disconnected` | AC-06、AC-07 |
| G-08 | 安全合规 | Ops sanitize、权限控制 | AC-11 |

---

## 12. Review Record

### v0.1

- 确认断开对象为调用 sub2api 的上游客户端/服务；
- 确认覆盖全部网关协议和路径；
- 确认错误类型使用 `client_disconnected`；
- 确认只写 `ops_error_logs`，不写 `usage_logs`；
- 确认遵循现有 Ops monitoring 开关；
- 确认保留 `context.WithoutCancel`，通过原始入站 Context 的取消桥接来取消下游请求；
- 确认不修改模型路由降级策略，仅增加调用方断开时的提前终止保护。

### v1.0

- 用户已明确回复“好的，这个plan我批准了”；
- 状态：**用户已批准，可进入后续任务拆分与实施阶段**。
