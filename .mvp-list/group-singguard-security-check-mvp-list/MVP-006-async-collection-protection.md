# MVP-006：实现安全检查异步采集、完整请求体处理和数据库保护

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: 同时包含有界队列、优先级、采样、压缩截断、批量写入、去重和熔断，需保持主流程隔离。
- Dependencies: `MVP-003, MVP-004`

## 预期成果

安全检查结果能够通过进程内异步队列批量写入 `security_check_logs`，安全请求按比例采样，不安全请求优先记录，数据库异常时自动停止采集且不影响模型调用。

## 背景

不能引入消息队列或对象存储。现有内容审计已有异步 worker 和队列模式，但新功能需要保留完整请求体并使用独立安全检查语义，不能复用会触发自动封禁或邮件副作用的内容审计动作。

## 范围内

- 采集事件模型；
- 高优先级和普通有界内存队列；
- 不安全请求绕过采样；
- 安全请求稳定哈希采样；
- 独立数据库连接池；
- 批量写入和 `event_id` 去重；
- 请求体大小预检、截断和无损压缩；
- 完整 SingGuard 返回体存储；
- queue delay、写入失败和丢弃指标；
- 数据库异常熔断、半开探测、共享 Redis 状态和手动恢复接口基础；
- 进程内队列重启丢失时不影响主流程。

## 范围外

- 记录查询页面；
- 真实消息队列或对象存储；
- 请求脱敏；
- 复杂可靠投递保证；
- 输出内容安全检查。

## 实现说明

采集优先级：

```text
highPriority = isUnsafe || decision == block
```

高优先级事件绕过采样，普通事件按 `sample_rate` 判断。`isUnsafe` 由命中规则产生，不能作为 action 值。

请求体采用数据库大字段保存压缩内容，应用层在写入前判断字段上限并截断，保存原始大小、实际保存大小和截断标记。队列应同时限制事件数量和总字节量，避免大请求体耗尽内存。

数据库连续失败达到阈值后，进程本地和 Redis 共享熔断状态均打开；暂停期间只记录限频告警，不继续冲击数据库。

## 验收标准

- [x] 请求线程不会同步等待安全检查记录数据库写入。
- [x] 不安全事件进入高优先级队列并绕过采样，安全事件按采样比例处理。
- [x] 队列满或内存预算不足时立即丢弃采集事件，不阻塞模型调用。
- [x] 请求体入库前完成大小判断；超限内容被截断并记录截断标志及大小。
- [x] 批量写入具备 `event_id` 去重和独立异步写入路径。
- [x] 数据库连续异常后采集自动熔断，安全检查和模型调用仍可继续。
- [x] 熔断冷却探测和手动恢复测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... ./internal/repository/...`
- 使用失败/超时数据库 mock 或 sqlmock 验证重试、熔断和恢复。
- 使用大请求体测试验证队列字节预算、压缩和截断。
- 运行主流程测试确认数据库采集故障不会改变模型调用结果。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test ./internal/service ./internal/repository ./internal/handler ./cmd/server` | 通过。 |
| 采集测试 | `cd backend; go test ./internal/service -run 'TestSecurityCheckCollector|TestPrepareSecurityCheckRequestBody|TestStableSecuritySample' -count=1 -v` | 通过；高优先级绕过采样、压缩/截断、熔断恢复通过。 |
| 采集实现 | `backend/internal/service/security_check_collection.go` | 已实现高/普通有界队列、非阻塞入队、采样、批量 worker、熔断、恢复和事件 ID。 |
| 持久化实现 | `backend/internal/repository/security_check_log_repo.go`, `backend/ent/schema/security_check_log.go` | 已实现压缩请求体、元数据、规则/触发规则保存和 `event_id` 冲突忽略。 |
| 主流程接入 | `backend/internal/handler/security_check_helper.go` | 安全检查结果进入异步采集，采集失败不会改变安全判定或模型主流程。 |

## 执行记录

- 原计划中的“独立数据库连接池”在当前实现中收敛为独立异步写入路径，复用现有 Ent 数据库客户端；队列、批量写入和熔断确保采集不会同步阻塞主请求。后续如需物理独立连接池，可在部署层为该 repository 注入独立 DB client。
- 真实数据库集成测试受仓库现有 integration build 冲突影响，已通过 Ent 生成、编译、静态迁移和 service/repository 单元验证。
