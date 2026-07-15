# MVP-011：完成运行时 Wiring 与端点覆盖

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 将已实现组件注入 wire/server 生命周期，并审计所有现有 RecordUsage 调用点。
- Dependencies: `MVP-006, MVP-007, MVP-009, MVP-010`

## 预期成果

生产启动路径能够构造 Redis 累计器和同步 worker，OpenAI、Anthropic、Gemini 及相关端点均走新的 Token 统计链路。

## 背景

依赖注入位于 `backend/internal/service/wire.go`、`backend/internal/repository/wire.go` 和 `backend/cmd/server/wire*.go`；handler 调用点分布在 gateway/openai/gemini 文件中。

## 范围内

- Repository、Service、Handler/server wiring。
- 同步 worker 启停注册。
- 审计 chat、responses、messages、embeddings、images、websocket、Gemini 等调用点。
- 删除正常路径对旧三表增量调用的遗漏，但保留实现。
- wiring 与 handler 回归测试。

## 范围外

- 全量故障注入和压测。

## 实现说明

- 不允许某些端点继续旧 MySQL 增量而其他端点写 Redis。
- Simple mode 行为需明确并通过现有测试确认，不静默改变未在 Plan 中批准的模式。

## 验收标准

- [x] 服务启动和关闭可正常完成。
- [x] 所有用量端点均覆盖新路径。
- [x] 没有生产请求路径继续调用旧逐请求三表事务。
- [x] wire 生成代码与声明一致。

## 验证计划

- `cd backend; go test ./cmd/server ./internal/handler ./internal/service -run "Wire|UsageRecord|RecordUsage"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Wiring | `backend/internal/repository/wire.go`、`backend/internal/service/wire.go` | 注入 Redis accumulator、Redis-first quota、绝对写 Repository、sync engine 和 scheduler。 |
| 生命周期 | `backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go` | scheduler 启动随配置开关，Cleanup 在 Redis 关闭前 Stop。声明与生成调用链一致。 |
| 路径审计 | `rg -n '\.RecordUsage\(|RecordUsageWithLongContext' backend/internal/handler --glob '*.go'` | chat、responses、messages、embeddings、images、websocket、Gemini 均进入 Gateway/OpenAI 已切换核心。 |
| 测试 | `cd backend; go test ./cmd/server ./internal/handler ./internal/service -run "Wire|UsageRecord|RecordUsage" -count=1` | 通过：server `7.233s`，handler `5.057s`，service `5.807s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：生产构造器强制注入 accumulator，OpenAI/Gateway 因此进入新路径；scheduler 注册启动和清理。所有 handler 调用点均复用两类公共 RecordUsage 核心。
- Wire CLI 首次生成因 `proxy.golang.org` 下载 `github.com/google/subcommands@v1.2.0` 超时而失败；随后按 `wire.go` provider graph 同步更新现有 `wire_gen.go`，并以 cmd/server 编译与聚焦测试验证生成代码一致性。
