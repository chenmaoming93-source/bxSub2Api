# MVP-013：完成接口契约、调用次数与性能回归

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 以可执行回归证据收口 API 兼容、无 N+1、历史快速路径和混合查询性能，约 40 分钟。
- Dependencies: `MVP-009, MVP-010, MVP-011, MVP-012`

## 预期成果

整套改造具备端到端回归证据：现有 API 契约不变、历史查询无 Redis 调用、当天查询无逐项点查，前端页面无需协议调整。

## 背景

批准 Plan 的最终验收不仅要求数值正确，还要求 API 兼容、集合式访问和历史性能不退化。

## 范围内

- 补或整理 handler/API contract 测试，锁定请求参数和响应 JSON。
- 加入 Repository/Redis mock 调用次数断言，证明不存在项无额外 MySQL 点查且列表无 N+1。
- 验证历史范围走原有 MySQL 快速路径。
- 为典型规模的混合集合合并增加 benchmark 或有界性能测试。
- 运行相关后端、前端测试及构建/typecheck，并记录证据。

## 范围外

- 不新增业务功能。
- 不做生产发布或真实生产压测。

## 实现说明

- 若全量测试存在与本改造无关的既有失败，需记录命令、失败用例和隔离后的相关测试结果，不得伪报通过。

## 验收标准

- [x] 三类报表和两类配置 API 契约保持不变。
- [x] 不存在筛选项没有 Redis miss 后的额外 MySQL 点查。
- [x] 历史查询测试证明 Redis reader 未被调用。
- [x] 相关后端测试、前端测试、typecheck 与构建通过，或对无关既有失败提供可复现证据。

## 验证计划

- `cd backend && go test ./internal/repository ./internal/service ./internal/handler/admin`
- `pnpm --dir frontend exec vitest run src/api/__tests__/admin.tokenUsage.spec.ts src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts src/views/admin/__tests__/UsersView.spec.ts`
- `pnpm --dir frontend run typecheck`
- `pnpm --dir frontend run build`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API 契约 | `backend/internal/handler/admin/token_usage_report_handler_test.go` | 三类报表继续返回 `items/summary.used_tokens/pagination.page,page_size,total`；两类配置 handler 既有契约测试继续通过。 |
| 调用次数 | `backend/internal/repository/token_usage_today_repo_test.go`、实时配置 service 测试 | 不存在筛选仅一次集合 SQL；配置 List 每次仅一次 reader 批量调用，无 miss 后点查和逐项 Redis 访问。 |
| 历史路径 | `backend/internal/service/token_usage_*_hybrid_test.go` | 三维度历史查询均断言 Redis reader 调用次数为 0。 |
| 性能 | `cd backend && go test ./internal/service -run '^$' -bench 'BenchmarkModelTokenUsageMerge10000' -benchtime=3x` | Windows/amd64：10,000+10,000 行合并约 `6230400 ns/op`。 |
| 后端回归 | `cd backend && go test ./internal/repository ./internal/service ./internal/handler/admin ./cmd/server` | 全部通过。 |
| 前端测试 | `pnpm --dir frontend exec vitest run src/api/__tests__/admin.tokenUsage.spec.ts src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts src/views/admin/__tests__/UsersView.spec.ts` | 通过：3 files、18 tests。 |
| Typecheck | `pnpm --dir frontend run typecheck` | 通过，退出码 0。 |
| Build | `pnpm --dir frontend run build` | 通过，退出码 0；仅有既有 chunk/dynamic-import 警告。 |

## 执行记录

2026-07-15 完成。补齐模型维度多字段排序、业务日键归一化、零 usage 不修复及三类报表响应结构断言。一次复核因在仓库根目录运行 Go 命令而提示找不到 module，切换到 `backend` 后同一目标包全部通过；该路径错误不属于代码失败。
