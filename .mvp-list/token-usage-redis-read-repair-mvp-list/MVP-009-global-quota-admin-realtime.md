# MVP-009：全局模型限额配置显示实时消耗

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只改一个配置接口及其返回链路和前端回归，约 40 分钟。
- Dependencies: `MVP-001, MVP-002, MVP-003, MVP-004`

## 预期成果

全局模型限额配置弹窗的“今日已使用”与请求限额判断一致，并在 Redis 缺失时使用 MySQL 后修复 Redis。

## 背景

后端 `model_token_quota_admin_service.go` 及 `daily_token_quota_repo.go` 当前从 MySQL 返回 `used_tokens`；前端由 `GlobalModelTokenQuotaModal.vue` 展示。

## 范围内

- 让全局模型限额 List/Set 响应使用统一当天集合读取。
- 对配置项批量关联 Redis 消耗，不产生逐行 HGET。
- 保持限额配置仍由 MySQL 保存。
- 补后端 service/handler 与现有前端弹窗测试。

## 范围外

- 不改用户模型配置。
- 不改变限额编辑交互和 API DTO。

## 实现说明

- 新增但尚无任何消耗的配置仍显示 0，且不写无意义 Redis field。

## 验收标准

- [x] Redis 与 MySQL 都有时页面/API 返回 Redis 值。
- [x] Redis 缺失且 MySQL 有 usage 时返回 MySQL 并修复。
- [x] 列表读取无 N+1 Redis 调用。
- [x] 保存限额后的响应仍返回最终实时消耗。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*ModelTokenQuota'; pnpm --dir frontend exec vitest run src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/model_token_quota_admin_service.go` | List/Set 均批量调用统一 reader，按模型集合合并实时值；MySQL-only 正数 usage 原子修复，零值不写 Redis。 |
| 接线 | `backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go` | 管理服务注入共享 reader/repairer。 |
| 测试 | `backend/internal/service/model_token_quota_realtime_test.go` | 覆盖重叠取 Redis、MySQL miss 修复、零值不修复及 Set 返回实时值。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*ModelTokenQuota'` | 通过。 |
| 前端回归 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts` | 通过：1 file、1 test。 |
| 包回归 | `cd backend && go test ./internal/service ./internal/repository ./internal/handler/admin ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。列表通过一次 `HMGET` 读取全部配置模型，无逐行 HGET；保存仍由 MySQL 完成，缓存失效后对单个模型执行同一实时合并。同步修正业务键日期为 `YYYY-MM-DD`，避免 MySQL 午夜日期与请求时刻不同导致当天键不匹配。
