# MVP-010：切换 OpenAIGatewayService 用量记录路径

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: OpenAI RecordUsage 路径独立且测试丰富，适合作为单独可验证切片。
- Dependencies: `MVP-003, MVP-008`

## 预期成果

OpenAI 成功请求继续写 `usage_logs`，停止价格解析和金额计费，停止逐请求三表事务，改用 Redis 三 Hash 累计。

## 背景

主要代码位于 `backend/internal/service/openai_gateway_service.go`，覆盖 chat、responses、embeddings、images 和 websocket 的共享用量服务。

## 范围内

- 旁路 OpenAI 价格计算、账号金额统计和 `applyUsageBilling`。
- 保持 `UsageLog` 字段和写入方式。
- 接入 Redis accumulator。
- 验证图片和零 Token 分支。
- 保留原金额实现。

## 范围外

- Handler 构造注入和完整端点覆盖审计。

## 实现说明

- 不再产生 fallback pricing 日志。
- 图片等特殊使用明细仍保留，金额字段归零。

## 验收标准

- [x] OpenAI 用量明细继续记录。
- [x] 价格解析与金额 Repository 不再调用。
- [x] 三表逐请求增量不再调用。
- [x] 正常 Token 请求写入 Redis，零 Token 请求跳过。

## 验证计划

- `cd backend; go test ./internal/service -run "OpenAIGatewayServiceRecordUsage|OpenAIRecordUsage"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/openai_gateway_service.go` | accumulator 注入后旁路价格解析、账号金额统计、金额 Repository 和旧三表增量；保留旧实现作为未注入回滚路径。 |
| 测试 | `backend/internal/service/openai_token_statistics_record_usage_test.go` | 验证 usage log 写入 1 次、金额 Repository 0 次、Redis accumulator 1 次、15 Token 及金额字段归零。 |
| 回归 | `cd backend; go test ./internal/service -run "OpenAIGatewayServiceRecordUsage|OpenAIRecordUsage" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.376s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：新增显式 runtime switch；生产注入 accumulator 后使用 Token 统计路径，未注入时保留原金额实现以支持回滚和历史测试。图片明细仍构建，公共累计适配对零 Token 自动跳过。
- 首次运行历史测试时，旧测试对金额调用的断言与新语义冲突；改为注入式切换后，旧回滚路径与新 Token 统计路径均有独立证据并通过。
