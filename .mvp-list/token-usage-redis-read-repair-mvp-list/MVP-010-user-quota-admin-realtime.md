# MVP-010：用户模型限额配置显示实时消耗

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只改用户模型限额配置的批量读取、保存响应和页面回归，约 40 分钟。
- Dependencies: `MVP-001, MVP-002, MVP-003, MVP-004`

## 预期成果

用户模型限额配置弹窗的“今日已使用”与实时限额判断一致，且保持批量读取和原有保存语义。

## 背景

后端位于 `user_model_token_quota_admin_service.go` 与 `daily_token_quota_repo.go`，前端由 `UserModelTokenQuotaModal.vue` 展示。

## 范围内

- 让用户模型限额 List/Upsert 响应使用统一当天集合读取。
- 按 user_id+model 批量关联 Redis 消耗。
- 保持配置新增、删除和默认值行为。
- 补 Redis-only/MySQL-only/无消耗配置及前端弹窗回归测试。

## 范围外

- 不改全局模型配置。
- 不修改默认用户配额设置格式。

## 实现说明

- 不得对每个 model 单独访问 Redis 或 MySQL。
- 用户不存在等现有校验规则保持不变。

## 验收标准

- [x] 弹窗和 API 返回 Redis 实时 `used_tokens`。
- [x] Redis 缺失且 MySQL 有 usage 时回退并修复。
- [x] 无 usage 的配置显示 0 且不创建 Redis field。
- [x] 保存、删除和默认配额相关回归测试通过。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*UserModelTokenQuota'; pnpm --dir frontend exec vitest run src/views/admin/__tests__/UsersView.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/user_model_token_quota_admin_service.go` | List/Upsert 按 user_id+model 批量 `HMGET` 并合并实时值，正数 MySQL-only usage 修复，零值不写入。 |
| 接线 | `backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go` | 用户模型管理服务注入共享 reader/repairer。 |
| 测试 | `backend/internal/service/user_model_token_quota_realtime_test.go` | 覆盖 List/Upsert 实时值、MySQL 回退修复、Redis-only 非配置项排除和零 usage。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*UserModelTokenQuota'` | 通过。 |
| 前端回归 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/UsersView.spec.ts` | 通过：1 file、2 tests。 |
| 包回归 | `cd backend && go test ./internal/service ./internal/repository ./internal/handler/admin ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。管理响应仅包含 MySQL 中存在的配置项，Redis-only 非配置项不扩展配置列表；保存、删除、默认配置格式及用户存在性校验均保持原行为。
