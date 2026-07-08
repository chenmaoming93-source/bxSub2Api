# MVP-002：在代码中落实平台 Key 与默认 Key 唯一校验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-001`

## 预期成果

Repository 能够阻止同一用户重复创建未软删除的平台 Key 或默认 Key，不新增数据库字段或索引。

## 背景

平台 Key 和默认 Key 的有效记录唯一性由应用代码负责校验；软删除记录不参与查重。

## 范围内

- 在 Repository 创建逻辑中检查未软删除的 `(user_id, platform)` 重复记录。
- 在 Repository 创建逻辑中检查未软删除的 `purpose=default` 重复记录。
- 在同一数据库事务中锁定用户行、查询有效 Key、校验并创建记录。
- 添加 Repository 单元测试，覆盖重复拒绝和软删除后重建。

## 范围外

- 数据库字段、索引或唯一约束变更。
- Redis 等外部分布式锁。

## 实现说明

- 仅查询 `deleted_at IS NULL` 的有效记录。
- 通过 `SELECT ... FOR UPDATE` 锁定稳定存在的用户行，使同一用户的创建请求在数据库层串行化，避免两个事务同时查到“不存在”。
- 查询、校验和创建处于同一事务；调用方已有事务时复用该事务。
- SQLite 单元测试方言不支持 `FOR UPDATE`，测试环境跳过行锁但仍验证事务内查重规则；生产 MySQL 路径启用用户行锁。

## 验收标准

- [x] 重复有效平台 Key 和默认 Key 被 Repository 拒绝。
- [x] 软删除后允许创建替代记录，Repository 单元测试通过。

## 验证计划

- `cd backend; go test ./internal/repository -run 'TestAPIKeyRepository_(CreateDuplicateKey|ActivePlatformAndDefaultUniqueness)' -count=1`
- `cd backend; go test ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 代码实现 | `backend/internal/repository/api_key_repo.go` | 在同一事务中锁定用户行，查询并校验有效平台 Key/默认 Key，然后创建记录；支持复用调用方事务。 |
| 单元测试 | `backend/internal/repository/api_key_repo_last_used_unit_test.go` | 覆盖平台 Key、默认 Key 重复拒绝及软删除后替代创建。 |
| 定向验证 | `cd backend; go test ./internal/repository -run 'TestAPIKeyRepository_(CreateDuplicateKey|ActivePlatformAndDefaultUniqueness)' -count=1` | 通过。 |
| 完整验证 | `cd backend; go test ./internal/repository/...` | 通过，`ok github.com/Wei-Shaw/sub2api/internal/repository`。 |

## 执行记录

已完成：撤销 161 号字段与索引 SQL，改为“事务 + 用户行锁 + 查重 + 创建”；该方案由数据库协调多进程、多实例并发。定向测试和完整 Repository 测试均通过。
