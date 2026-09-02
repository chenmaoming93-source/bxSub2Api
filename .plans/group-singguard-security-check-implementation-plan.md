# 分组级 SingGuard 请求安全检查实施计划

> **状态：最终版——用户已批准（Final — user approved）**
> **版本：1.2.2**
> **日期：2026-09-02**
> **变更摘要：** 新增分组级请求安全检查、规则判定、超时/异常决策、采集、采样、管理页面及多实例配置缓存同步；补充独立安全日志页面、检查状态可视化、风险维度可读化展示，以及严格按配置时间执行的日志生命周期清理。

## 1. Introduction

### 1.1 背景

系统需要在模型调用进入上游模型前，根据请求所属分组的安全配置调用内网 SingGuard 服务，对请求内容进行风险分类。

SingGuard 的服务地址与部署环境相关，不能硬编码。当前只检查请求内容，不检查模型输出内容。

项目现有技术基础：

- 后端：Go、Gin、Ent；
- 数据库：MySQL/GoldenDB；
- 缓存：Redis；
- 前端：Vue 3 + TypeScript；
- 已存在分组管理、Redis 缓存、异步 worker 和内容审计相关实现。

### 1.2 目标

1. 支持按分组开启或关闭请求安全检查。
2. 支持配置多个风险维度、阈值和操作。
3. 支持阻断和告警两种规则操作。
4. 支持超时、外部接口异常时按配置放行或阻断。
5. 支持安全检查记录采集、采样和页面查询。
6. 采集异常不能影响模型调用主流程。
7. 支持多实例本地配置缓存自动失效。
8. 支持记录保留期限配置，默认 3 天。

### 1.3 范围

包含：

- 分组安全检查配置；
- SingGuard `/classify` 调用；
- Query 任务五个风险维度解析；
- 规则判定和模型调用阻断；
- 异步采集和采样；
- 独立安全检查记录表；
- 记录管理页面；
- 采集熔断、手动恢复和定时清理；
- Redis 配置缓存、本地缓存和 Pub/Sub 失效通知。

第一期覆盖以下请求：

- Anthropic Messages；
- OpenAI Chat Completions；
- OpenAI Responses；
- Gemini `generateContent`；
- OpenAI Responses WebSocket 的首帧和后续模型调用帧。

第一期不包含图片生成、图片编辑和 embeddings，因为本需求的输入定义是聊天内容列表。

不包含：

- SingGuard 服务部署、模型加载和健康维护；
- Response 输出内容安全检查；
- 消息队列；
- 对象存储；
- 请求数据脱敏；
- 外网环境下的真实 SingGuard 联调。

## 2. Assumptions and Decisions

### 2.1 外部接口

依据项目中的 `SINGGUARD_API_SPEC.md`：

```http
POST {base_url}/classify
```

请求只传：

```json
{
  "text": "聊天内容格式化后的字符串",
  "task": "query"
}
```

不传 `threshold`。本系统自行读取 `risks[*].risk_prob` 并与分组规则比较。

SingGuard Query 的五个风险维度为：

- `Dangerous_Operations_Tool_Abuse`
- `Malicious_Code_and_Cyberattack`
- `Prompt_Injection_and_Jailbreak`
- `Resource_Abuse`
- `Sensitive_Information_Stealing`

`base_url` 作为部署配置项注入，禁止硬编码 IP。

### 2.2 SingGuard 输入格式

`text` 的 JSON 类型必须是字符串，但字符串内容采用可读的聊天格式，不直接传整个请求体 JSON，也不直接传数组对象。

示例：

```text
[system]
你是一个助手

[user]
请查询订单

[assistant]
我会帮你查询

[tool_call]
name: query_order
arguments:
order_id=12345
```

格式化规则：

- 保留消息顺序和角色；
- 文本内容直接输出；
- 工具调用、工具结果转换为可读文本；
- 图片只保留类型或占位描述，不传 Base64 原文；
- 不传完整模型请求体；
- 不传 `threshold`；
- 超过 SingGuard 的 100000 字符限制时，按异常决策处理，不静默截断判定文本。

不同协议的聊天内容来源：

- Anthropic / OpenAI Chat：`messages`；
- OpenAI Responses：`input`；
- Gemini：`contents`。

### 2.3 分组配置存储方式

不在 `groups` 表中增加多个独立字段，增加一个 JSON 字段：

```sql
security_check_config JSON
```

示例：

```json
{
  "enabled": true,
  "rules": [
    {
      "dimension": "Prompt_Injection_and_Jailbreak",
      "threshold": 0.8,
      "action": "block"
    }
  ],
  "timeout_ms": 500,
  "exception_action": "allow",
  "collect_enabled": true,
  "sample_rate": 10,
  "version": 1
}
```

安全配置按分组 ID 整体加载和缓存，不按 JSON 子字段查询。

### 2.4 类型定义

规则操作：

```text
RuleAction = block | warn
```

最终决策：

```text
Decision = allow | warn | block
```

检查状态：

```text
CheckStatus = skipped | success | timeout | error
```

不安全标识：

```text
isUnsafe = 是否命中任意安全检查规则
```

`unsafe` 不是 action 值。

异常配置只允许：

```text
ExceptionAction = allow | block
```

### 2.5 建议默认值

以下是当前实现建议值，属于可配置默认值：

- `enabled=false`；
- `rules=[]`；
- `timeout_ms=500`；
- `exception_action=allow`；
- `collect_enabled=false`；
- `sample_rate=10`；
- 本地配置缓存 TTL：5 秒；
- 采集熔断冷却时间：60 秒；
- 记录保留期限：3 天。

### 2.6 采样策略

- 采集关闭时不采集；
- 安全请求按照 `sample_rate` 采集；
- 不安全请求尽量全量采集，绕过采样；
- `sample_rate=0` 时仍然允许记录不安全请求；
- `sample_rate=100` 时安全请求也全部进入采集流程；
- 采样可使用请求 ID 稳定哈希，保证重试时结果一致。

“不安全”包括命中 `warn` 规则的请求。

## 3. Conceptual Functional Design

### 3.1 分组安全配置

管理员在现有分组编辑页面配置：

- 安全检查开关；
- 规则列表；
- 风险维度；
- 分数阈值；
- 操作；
- 超时时间；
- 异常决策；
- 采集开关；
- 采样比例。

规则支持新增、删除和排序，同一个维度允许配置多条规则。

服务端校验：

- 维度必须属于五个 Query 风险域；
- 阈值范围为 `0～1`；
- 操作只能是 `block` 或 `warn`；
- 超时时间必须为正数；
- 采样比例必须为 `0～100`；
- 规则数量不能超过系统上限。

### 3.2 请求安全检查

模型调用流程：

1. 根据分组 ID 获取安全配置快照；
2. 配置关闭或规则为空时直接放行；
3. 从已有请求体中提取聊天内容；
4. 按角色和内容块转换为可读字符串；
5. 调用 SingGuard；
6. 解析五个风险维度的分数；
7. 按规则顺序判断；
8. 命中 `warn` 时打印结构化 `warn` 日志并继续；
9. 命中 `block` 时立即阻断，不调用上游模型；
10. 未阻断时继续原模型调用；
11. 按采集策略生成异步记录。

检查应位于现有 Handler 已经读取请求体、完成基本模型解析之后，且位于上游账号选择和模型转发之前。

不能把安全检查实现为再次读取 `c.Request.Body` 的通用 Middleware，因为现有 `ReadRequestBodyWithPrealloc` 会消费请求体。

### 3.3 规则判定

使用严格大于比较：

```text
risk_prob > rule.threshold
```

- 只命中告警规则：`check_status=success`、`decision=warn`、`is_unsafe=true`，模型调用继续；
- 命中阻断规则：`check_status=success`、`decision=block`、`is_unsafe=true`；
- 没有命中：`check_status=success`、`decision=allow`、`is_unsafe=false`；
- 超时或异常：`check_status=timeout/error`，`decision` 由 `exception_action` 决定，`is_unsafe=false`。

如果配置的维度不存在于返回体，视为外部结果异常。未配置的返回维度只保存，不参与判定。

### 3.4 采集策略

采集内容包括：

- 模型调用请求体；
- SingGuard 完整返回体；
- 请求和分组元数据；
- 配置版本和规则快照；
- 检查状态、最终决策、不安全标识；
- 触发规则；
- 检查耗时和异常信息。

采集不在请求线程同步写数据库，而是进入进程内有界队列。

队列划分为高优先级和普通队列：

```text
if isUnsafe == true or decision == block:
    高优先级队列，绕过采样
else:
    按 sample_rate 判断后进入普通队列
```

队列满时立即放弃采集并记录指标，不能等待或阻塞模型调用。

### 3.5 采集熔断

采集数据库连续超时、连接失败或批量写入失败时：

1. 采集 worker 进入熔断状态；
2. 暂停数据库写入；
3. 安全检查继续执行；
4. 模型调用继续执行；
5. 记录限频告警和监控指标；
6. 冷却后进行有限探测；
7. 页面支持手动重新开启采集。

### 3.6 记录查询页面

安全检查记录页面支持：

- 分页查询；
- 按时间范围筛选；
- 按分组筛选；
- 按最终决策筛选；
- 按检查状态筛选；
- 查看完整请求体；
- 查看完整 SingGuard 返回体；
- 查看触发规则、配置版本和耗时；
- 查看请求体是否被截断；
- 查看采集熔断状态；
- 手动恢复采集。

补充的管理端交互要求：

- 安全检查日志使用独立页面 `/admin/security-check-logs`，不再只依赖分组页面弹窗；
- 点击“查看详情”后使用独立 `BaseDialog` 弹窗展示大字段详情，不在列表页面底部内嵌详情；
- 详情弹窗在存在 `exception_type` 或 `exception_message` 时展示异常信息区域；
- 分组管理页面不再提供重复的“安全日志”入口，统一从左侧“安全检查日志”进入；
- 左侧管理菜单中将“安全检查日志”放在“可配置 Token 统计”之前；
- 列表明确展示 `check_status`，将 `success`、`timeout`、`error`、`skipped` 映射为可读状态；
- 五个风险维度在配置和详情页面展示中文名称及含义，同时保留英文 code 作为接口和排查标识。

列表页不加载大字段，详情页再加载请求体和返回体。

## 4. Detailed Technical Design

### 4.1 组件划分

```text
模型请求
  │
  ▼
各协议 Gateway Handler
  │  已读取 body []byte
  ▼
GroupSecurityCheck
  │
  ├── SecurityConfigProvider
  │      ├── 本地缓存
  │      ├── Redis
  │      └── 数据库回源
  │
  ├── SecurityTextFormatter
  │
  ├── SingGuardClient
  │
  ├── SecurityRuleEvaluator
  │
  └── SecurityRecordProducer
         │
         ▼
    进程内有界队列
         │
         ▼
    SecurityRecordWorker
         │
         ▼
    security_check_logs
```

### 4.2 现有代码接入点

当前各 Handler 已在模型调用前读取完整请求体并调用内容审计，安全检查可复用这些位置：

- `backend/internal/handler/gateway_handler.go`；
- `backend/internal/handler/gateway_handler_chat_completions.go`；
- `backend/internal/handler/gateway_handler_responses.go`；
- `backend/internal/handler/openai_chat_completions.go`；
- `backend/internal/handler/openai_gateway_handler.go`；
- `backend/internal/handler/gemini_v1beta_handler.go`。

可复用 `content_moderation_helper.go` 中的请求元数据组织方式，包括：

- request ID；
- user ID；
- API Key ID 和名称；
- group ID 和名称；
- 入站 endpoint；
- provider；
- model；
- protocol；
- body。

现有 `ExtractContentModerationInput` 只提取最后一条用户消息、做内容归一化和长度限制，不能直接复用。新功能需要增加独立的完整聊天内容格式化器。

### 4.3 请求体和元数据可用性

普通 HTTP 请求在安全检查位置可以拿到解压后的完整 `body []byte`。数据库记录保存该逻辑请求体，而不是传输层压缩字节。

OpenAI Responses WebSocket 没有普通 HTTP body：

- 首次模型请求使用 `firstMessage`；
- 后续模型请求使用 `BeforeRequest` 回调中的 `payload`；
- 每一轮 payload 都应独立执行安全检查和采集。

安全检查发生在账号选择前，因此本期不记录：

- 上游账号 ID；
- 上游请求 ID；
- 最终映射后的上游模型。

OpenAI 紧凑请求路径可能在检查前规范化 body。若“完整入参”指客户端原始请求体，应在规范化前保留 `originalBody`，安全检查使用规范化内容，采集使用原始 body。

### 4.4 SingGuard 客户端

请求：

```http
POST {base_url}/classify
Content-Type: application/json
Accept: application/json
```

请求体：

```json
{
  "text": "[user]\n请查询订单",
  "task": "query"
}
```

客户端要求：

- 独立 HTTP 连接池；
- 连接超时；
- 读取超时；
- 总超时；
- 限制响应体大小；
- 不进行无限重试；
- 分别记录网络错误、HTTP 错误、JSON 解析错误和字段缺失错误。

SingGuard 返回体原文完整保存，同时解析：

```text
risks[dimension].risk_prob
```

### 4.5 分组配置 JSON

Ent Group schema 增加 `security_check_config` JSON 字段，服务层使用类型化结构进行解析和校验。

安全配置建议通过独立接口更新：

```http
PUT /api/v1/admin/groups/{id}/security-check
```

更新流程：

1. 加载现有 JSON 配置；
2. 合并请求内容；
3. 校验所有字段；
4. 递增 `version`；
5. 更新数据库；
6. 更新 Redis；
7. 发布缓存失效通知。

API Key 的认证快速查询路径不需要加载完整安全 JSON，只需提供 `group_id`；安全配置由 `SecurityConfigProvider` 独立加载，避免将配置重复塞入每个 API Key 对象。

### 4.6 Redis 和本地缓存

Redis Key：

```text
sub2api:security-check:group:{group_id}
```

Redis Value 为完整配置 JSON。

通知频道：

```text
sub2api:security-check:config-change
```

通知内容：

```json
{
  "group_id": 123,
  "version": 2
}
```

各服务实例：

- 订阅配置变化通知；
- 收到通知后删除本地缓存；
- 下一次请求从 Redis 加载；
- Redis 未命中时回源数据库；
- Pub/Sub 丢消息时由本地 TTL 兜底；
- 使用版本号避免旧配置覆盖新配置。

缓存异常时：

- 有最近一次有效配置则继续使用；
- 没有任何有效配置时按关闭安全检查处理；
- 不因缓存故障返回系统错误；
- 记录缓存故障指标和告警。

### 4.7 安全检查记录表

新增 `security_check_logs` 表。

| 字段 | 类型 | 空值 | 说明 |
|---|---|---:|---|
| `id` | BIGINT | 否 | 主键 |
| `event_id` | VARCHAR(64) | 否 | 事件唯一 ID |
| `request_id` | VARCHAR(64) | 是 | 服务端请求 ID |
| `client_request_id` | VARCHAR(64) | 是 | 客户端请求 ID |
| `user_id` | BIGINT | 是 | 用户 ID |
| `api_key_id` | BIGINT | 是 | API Key ID |
| `api_key_name` | VARCHAR(100) | 是 | API Key 名称 |
| `group_id` | BIGINT | 是 | 分组 ID |
| `group_name` | VARCHAR(100) | 是 | 分组名称快照 |
| `model` | VARCHAR(100) | 是 | 客户端请求模型 |
| `provider` | VARCHAR(50) | 是 | 平台 |
| `protocol` | VARCHAR(32) | 是 | 请求协议 |
| `endpoint` | VARCHAR(255) | 是 | 入站规范化 endpoint |
| `config_version` | BIGINT | 否 | 使用的配置版本 |
| `rules_snapshot` | JSON/TEXT | 否 | 规则快照 |
| `request_body` | MEDIUMBLOB | 是 | 压缩后的请求体 |
| `request_body_original_bytes` | BIGINT | 否 | 原始字节数 |
| `request_body_stored_bytes` | BIGINT | 否 | 保存字节数 |
| `request_body_truncated` | BOOLEAN | 否 | 是否截断 |
| `singguard_response` | MEDIUMTEXT | 是 | 完整返回体 |
| `check_status` | VARCHAR(16) | 否 | skipped/success/timeout/error |
| `decision` | VARCHAR(16) | 否 | allow/warn/block |
| `is_unsafe` | BOOLEAN | 否 | 是否命中安全规则 |
| `triggered_rules` | JSON/TEXT | 是 | 已命中的规则 |
| `latency_ms` | INT | 是 | 安全检查总耗时 |
| `singguard_latency_ms` | INT | 是 | 外部接口耗时 |
| `queue_delay_ms` | INT | 是 | 队列等待耗时 |
| `exception_type` | VARCHAR(32) | 是 | 异常类型 |
| `exception_message` | TEXT | 是 | 异常摘要 |
| `created_at` | DATETIME(6) | 否 | 创建时间 |

约束和索引：

- `UNIQUE(event_id)`，避免重试产生重复记录；
- `INDEX(created_at)`，用于过期清理；
- `INDEX(group_id, created_at)`，用于分组时间查询；
- `INDEX(decision, created_at)`，用于结果筛选。

不强制外键绑定分组，保留分组名称快照，避免分组删除影响历史记录。

请求体存储流程：

1. 以 Handler 已有的原始逻辑请求体为输入；
2. 判断大小；
3. 超过应用逻辑上限时提前截断；
4. 无损压缩；
5. 检查压缩结果是否超过 `MEDIUMBLOB` 上限；
6. 必要时继续缩短；
7. 保存截断标记和大小信息。

数据库字段上限按 `MEDIUMBLOB` 的约 16MB 设计，应用层必须在插入前检查。若存储处理异常，只能放弃或降级采集，不能影响模型调用。

### 4.8 全局设置和熔断状态

复用 `settings` 表保存：

| Key | 默认值 | 说明 |
|---|---:|---|
| `security_check_log_retention_days` | 3 | 记录保留天数，范围 1～3650 |
| `security_check_log_cleanup_time` | `03:00` | 每日清理时间，服务器本地时区，格式 `HH:mm` |
| `security_check_collection_master_enabled` | true | 全局采集开关 |

Redis 保存共享熔断状态：

```text
sub2api:security-check:collection:circuit
```

Redis Pub/Sub 用于通知各实例暂停或恢复本地采集。Redis 只承担缓存和状态通知，不承担业务消息可靠投递。

### 4.9 管理接口

分组安全配置：

```http
PUT /api/v1/admin/groups/{id}/security-check
```

记录列表：

```http
GET /api/v1/admin/security-check-logs
```

支持参数：

- `page`；
- `page_size`；
- `group_id`；
- `decision`；
- `check_status`；
- `start_time`；
- `end_time`。

记录详情：

```http
GET /api/v1/admin/security-check-logs/{id}
```

采集状态：

```http
GET /api/v1/admin/security-check-collection/status
POST /api/v1/admin/security-check-collection/reopen
```

日志生命周期配置（当前实现路由）：

```http
GET /api/v1/admin/groups/security-check/retention
PUT /api/v1/admin/groups/security-check/retention
```

请求字段：`retention_days`、`cleanup_time`；响应额外返回服务器本地时区和下一次清理时间。

建议权限：

- `security_check.logs.read`；
- `security_check.config.update`；
- `security_check.collection.manage`。

## 5. Pseudocode and Operational Logic

### 5.1 请求安全检查

```text
function checkRequest(ctx, metadata, body):
    config = configProvider.Get(metadata.groupID)

    if config unavailable:
        return checkStatus=error, decision=allow, isUnsafe=false

    if not config.enabled or config.rules is empty:
        return checkStatus=skipped, decision=allow, isUnsafe=false

    create timeout context using config.timeout_ms

    chatContent = extractFullChatContent(metadata.protocol, body)
    if chatContent is empty:
        return checkStatus=skipped, decision=allow, isUnsafe=false

    text = formatReadableChatText(chatContent)

    if length(text) > 100000:
        decision = config.exception_action
        return checkStatus=error, decision=decision, isUnsafe=false

    response, err = singguard.classify({
        text: text,
        task: "query"
    })

    if err exists:
        decision = config.exception_action
        return checkStatus=timeout/error, decision=decision, isUnsafe=false

    triggeredRules = []
    isUnsafe = false

    for rule in config.rules:
        score = response.risks[rule.dimension].risk_prob

        if score > rule.threshold:
            append triggeredRules
            isUnsafe = true

            if rule.action == "warn":
                log WARN
                continue

            if rule.action == "block":
                return success, block, true, triggeredRules

    if triggeredRules is not empty:
        return success, warn, true, triggeredRules

    return success, allow, false, []
```

### 5.2 采集入队

```text
function enqueueRecordIfNeeded(result, requestBody, metadata, config):
    if not config.collect_enabled:
        return

    if global collection disabled or circuit open:
        record metric
        return

    highPriority = result.isUnsafe or result.decision == block

    if not highPriority:
        if not stableSample(result.requestID, config.sample_rate):
            return

    event = buildEvent(result, requestBody, metadata)

    if highPriority:
        try enqueue highPriorityQueue without blocking
    else:
        try enqueue normalQueue without blocking

    if queue full:
        increment dropped metric
        log rate-limited warning
```

### 5.3 采集 worker

```text
worker:
    while running:
        if circuit open:
            wait for half-open probe
            continue

        batch = dequeue bounded batch

        truncate request body before database field limit
        compress request body
        build storage metadata

        insert batch using event_id uniqueness

        if database failure:
            increment failure counter

            if failure threshold reached:
                open local and shared circuit

            retry only within bounded limit
            never block request handler
```

### 5.4 配置同步

```text
updateSecurityConfig(groupID, patch):
    lock group row
    load existing JSON
    merge patch
    validate
    increment version
    commit database

    set Redis full config
    publish groupID and version

onConfigChange(message):
    if local version <= message.version:
        evict local cache

onLocalCacheMiss(groupID):
    load Redis
    if not found:
        load database
        backfill Redis
    save local cache
```

### 5.5 过期清理

```text
cleanup_tick:
    config = load retention_days and cleanup_time
    localNow = now in server local timezone

    if localNow is before cleanup_time:
        return
    if cleanup already succeeded today:
        return

    retentionDays = config.retention_days
    cutoff = now - retentionDays

    repeat:
        delete at most 1000 rows
        where created_at < cutoff

    stop when affected rows == 0
    mark today as cleaned
```

默认保留 3 天、每日 03:00 执行。严格按配置时间触发：服务在当天清理时间之后启动，或配置在当天清理时间之后保存时，均等待下一次计划时间，不执行补偿清理。清理失败记录错误并等待下一次计划时间，不影响模型请求。

## 6. Verification Strategy

### 6.1 单元测试

覆盖：

- JSON 配置解析和默认值；
- 维度、阈值、规则操作校验；
- `RuleAction`、`Decision`、`CheckStatus` 枚举校验；
- 严格大于阈值；
- 同维度多规则；
- 告警继续判断；
- 阻断停止判断；
- 可读聊天文本格式化；
- 四类协议内容提取；
- 采样比例；
- 不安全请求绕过采样；
- 超时和异常决策；
- 请求体截断和压缩；
- 缓存版本比较；
- 熔断状态转换；
- 日志保留天数和 `HH:mm` 清理时间校验；
- 每日严格定时调度、错过时间不补偿。

### 6.2 外部接口契约测试

使用本地 HTTP 测试服务器模拟：

- 正常五维返回；
- 维度缺失；
- 400；
- 422；
- 503；
- 慢响应；
- 非法 JSON；
- 超大返回体。

不要求连接真实内网 SingGuard 服务。

### 6.3 集成测试

覆盖：

- 数据库迁移；
- 分组 JSON 配置读写；
- Redis 缓存和 DB 回源；
- 多实例 Pub/Sub 本地缓存失效；
- 采集批量写入；
- 唯一事件 ID 去重；
- 过期数据清理；
- 记录分页和详情查询；
- 采集熔断及手动恢复。

### 6.4 主流程验收标准

- **AC-01：** 关闭安全检查或规则为空时不调用 SingGuard。
- **AC-02：** SingGuard 的 `text` 是字符串，字符串内容是可读聊天格式，不是对象或数组。
- **AC-03：** `risk_prob > threshold` 才命中规则。
- **AC-04：** `warn` 不阻断且继续判断，`block` 立即阻断。
- **AC-05：** 超时、HTTP 错误、网络错误和返回格式错误按异常决策处理。
- **AC-06：** 阻断时不调用上游模型。
- **AC-07：** 采集写库不在请求线程等待数据库。
- **AC-08：** 不安全记录绕过采样进入高优先级队列。
- **AC-09：** 数据库异常达到阈值后自动熔断采集。
- **AC-10：** 请求体超限时入库前截断并保存截断标记。
- **AC-11：** 配置更新后本地缓存通过 Pub/Sub 失效，丢通知时 TTL 兜底。
- **AC-12：** 默认记录保留 3 天，并支持页面配置。
- **AC-13：** 管理员可以分页查询记录并查看请求体、完整返回体和判定结果。
- **AC-14：** 默认每日 03:00 按服务器本地时间清理，并支持页面配置清理时间；服务重启或保存配置晚于当天清理时间时，不补偿执行，等待下一次计划时间。

### 6.5 监控指标

- 安全检查次数；
- allow/warn/block 数量；
- timeout/error 数量；
- P50/P95/P99 延迟；
- 配置缓存命中率；
- 队列长度；
- 丢弃事件数量；
- 数据库写入失败数量；
- 熔断次数和持续时间；
- 请求体截断数量；
- 配置传播延迟。

## 7. Implementation Sequencing and Decomposition Guide

### 阶段一：数据和配置基础

产出：

- `groups.security_check_config` 字段；
- `security_check_logs` 表；
- 全局保留期限和采集总开关；
- Ent schema、服务模型、DTO；
- 配置校验和独立配置接口。

### 阶段二：缓存和传播

产出：

- Redis 配置缓存；
- 本地缓存；
- Pub/Sub 失效通知；
- 配置版本控制；
- Redis/DB 故障回退。

### 阶段三：安全检查核心链路

产出：

- SingGuard 客户端；
- 四类协议聊天内容提取；
- 可读文本格式化器；
- 响应解析；
- 规则判定；
- 超时和异常决策；
- 各 Gateway Handler 接入；
- WebSocket 首帧和后续帧接入。

### 阶段四：异步采集和保护

产出：

- 高低优先级有界队列；
- 批量写入 worker；
- 采样；
- 请求体压缩和截断；
- 熔断、半开探测和手动恢复；
- 过期清理。

### 阶段五：管理页面

产出：

- 分组安全检查配置面板；
- 独立安全检查记录页面（含日志列表和详情）；
- 安全检查记录列表中的检查状态可视化；
- 记录详情；
- 保留期限和每日清理时间配置；
- 清理时区和下一次执行时间展示；
- 风险维度中文名称和含义字典；
- 采集状态和恢复操作；
- RBAC 权限；
- 中英文文案。

### 阶段六：验证和发布

产出：

- 模拟接口测试报告；
- 数据库迁移验证；
- Redis 多实例测试；
- 主流程回归测试；
- 压力测试；
- 内网 SingGuard 联调配置说明；
- 灰度、监控和回滚说明。

## 8. Risks and Open Items

| 风险 | 影响 | 缓解措施 |
|---|---|---|
| SingGuard 延迟升高 | 增加请求耗时 | 总超时、独立连接池、延迟监控 |
| SingGuard 不可用 | 无法完成安全判定 | 按异常决策处理 |
| Pub/Sub 丢消息 | 配置短暂过期 | 本地 TTL、版本号、数据库回源 |
| Redis 不可用 | 配置读取失败 | 使用最近有效配置或安全检查关闭 |
| 数据库写入拥塞 | 影响主库 | 异步队列、独立连接池、批量写入、熔断 |
| 请求体过大 | 内存和数据库压力 | 有界队列、压缩、字段上限、截断 |
| 进程重启 | 内存队列记录丢失 | 不阻塞主流程，记录丢弃指标 |
| 原始请求包含敏感信息 | 数据泄露风险 | 限制管理权限、数据库权限和保留期限 |
| JSON 配置并发修改 | 配置覆盖 | 行锁、版本递增、独立更新接口 |
| MySQL/GoldenDB JSON 差异 | 迁移失败 | 两种数据库分别验证迁移和 JSON 行为 |
| 请求体规范化后丢失原文 | 审计内容不准确 | 规范化前保留 `originalBody` |

非阻塞可调项：

- 生产环境具体 SingGuard `base_url`；
- 是否需要 SingGuard 认证信息；
- 超时默认值、采样默认值和请求体逻辑上限的运营调整。

## 9. Traceability Matrix

| 需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| 分组开启检查 | 分组配置 | JSON 配置、配置接口 | AC-01 |
| 检查规则列表 | 规则判定 | SecurityRuleEvaluator | AC-03、AC-04 |
| 阻断请求 | 主流程控制 | Gateway 集成 | AC-06 |
| 告警不中断 | 规则判定 | Logger、Evaluator | AC-04 |
| 超时和异常决策 | 外部调用 | SingGuardClient | AC-05 |
| 可读文本入参 | 输入格式化 | SecurityTextFormatter | AC-02 |
| 是否采集 | 采集策略 | RecordProducer | AC-07 |
| 采样比例 | 采样策略 | Stable sampler | AC-08 |
| 不安全尽量采集 | 优先级队列 | High priority queue | AC-08 |
| 数据库保护 | 采集熔断 | Worker、Circuit Breaker | AC-09 |
| 大请求体处理 | 请求体存储 | MEDIUMBLOB、压缩、截断 | AC-10 |
| 多实例缓存同步 | 缓存管理 | Redis、本地缓存、Pub/Sub | AC-11 |
| 保留期限 | 数据清理 | Settings、Cleanup Job | AC-12 |
| 页面查询 | 管理功能 | Admin API、Vue 页面 | AC-13 |

## 10. Review Record

### v0.1

初版方案包含多个分组独立字段，并初步设计了异步采集、采样、异常决策和页面查询。

### v0.2

根据评审意见：

- 将分组安全配置收缩为单个 `security_check_config JSON` 字段；
- 增加配置版本和 Redis Pub/Sub 本地缓存失效；
- 明确 `RuleAction`、`Decision`、`CheckStatus` 和 `isUnsafe` 的区别；
- 修正采集优先级和伪代码；
- 增加请求体截断、压缩和字段大小处理；
- 根据代码审查结果，改为复用现有 Handler 已读取的 `body []byte`；
- 排除不能在安全检查时获得的上游账号和上游请求字段；
- 明确 WebSocket 和请求体规范化场景；
- 将 SingGuard 输入定义为“可读聊天内容格式的字符串”。

### v1.0

用户确认采用可读聊天内容格式，并批准形成最终 Plan。

本 Plan 已准备好用于后续任务拆解和 MVP 分解。

### v1.1 补充

用户批准以下增量修改：

- 安全检查日志从分组页面弹窗补充为独立管理页面；
- 独立页面的记录详情使用 `BaseDialog` 弹窗，不在列表底部展开；
- 左侧菜单中安全日志放置在可配置 Token 统计上方；
- 日志列表展示检查状态，避免将超时/异常与最终决策混淆；
- 风险维度展示中文名称和含义，英文 code 继续用于 API 传输和技术排查。

本补充不改变安全检查判定逻辑、日志表结构、后端日志字段和既有接口契约。

### v1.1.1 交互修订

根据用户反馈，独立日志页的“查看详情”改为 `BaseDialog` 弹窗展示，移除列表页底部内嵌详情区域；详情加载中和加载失败状态仍在弹窗内反馈。

### v1.1.2 错误信息与入口修订

根据用户反馈：

- 在详情弹窗中展示后端已记录的 `exception_type` 和 `exception_message`；
- 移除分组管理页面的“安全日志”按钮、弹窗挂载和相关状态，统一使用左侧“安全检查日志”页面；
- 不改变日志表结构、后端接口和安全检查判定逻辑。

### v1.2 日志生命周期配置补充

用户批准增加安全日志生命周期配置：

- 在安全检查日志页面配置保留天数和每日清理时间；
- 默认保留 3 天、服务器本地时间每日 03:00 清理；
- 复用 `settings` 表保存配置，不新增表字段或迁移；
- 清理 worker 动态读取配置，失败按分钟重试且不影响模型请求；
- 增加 `MVP-010` 覆盖设置 API、调度清理、页面配置和验证。

### v1.2.1 同日重配修订

验证发现：若服务启动后的补偿清理已经标记当天完成，管理员随后在同一天修改清理时间或保留天数，旧调度状态会阻止当天再次清理。该版本曾调整为检测到配置变化时重置当天清理状态。

### v1.2.2 严格定时修订

根据用户反馈，取消启动和同日保存配置后的补偿清理：

- 仅在服务器本地时间进入配置的 `HH:mm` 分钟时执行；
- 服务启动时如果已错过当天时间，等待下一天；
- 配置在当天时间之后保存，等待下一天；
- 清理失败不在非计划时间重试，等待下一次计划时间；
- 清理时间始终读取配置，不写死具体时间值。
