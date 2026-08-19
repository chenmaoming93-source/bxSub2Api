# MVP-001：分组路由账号关系表与保存同步

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 完成一张表、迁移、解析复用和分组保存事务同步。
- Dependencies: `none`

## 预期成果

新增 `group_model_route_accounts` 表，并让分组模型路由保存时同步维护关系表，同时保留已有并发配置。

## 范围内

- 新增迁移、唯一约束和查询索引。
- 复用现有模型路由 JSON 解析逻辑。
- 接入分组新增、修改、删除关系的事务流程。
- 为关系同步增加后端测试。

## 范围外

- 重建接口、账号页面和 Redis 限流。

## 验收标准

- [x] 迁移创建关系表及约束索引。
- [x] 新增、修改、删除分组路由后关系表同步入口已接入。
- [x] 已有关系的 `max_concurrency` 不被普通同步清空。
- [x] 关键同步失败时数据库事务回滚。
- [x] 相关定向编译和测试通过。

## 验证计划

- 运行相关 Go 测试和迁移测试。
- 检查 `git diff` 与测试输出。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 迁移 | `backend/sqlArchiving/169_create_group_model_route_accounts.sql` | 已新增关系表、唯一约束、查询索引和外键 |
| Schema | `backend/ent/schema/group_model_route_account.go` | Ent schema 已生成 |
| 代码 | `backend/internal/repository/group_model_route_account_repo.go` | 支持解析 JSON、保留已有并发配置并重建关系 |
| 测试 | `GOCACHE=.gocache go test ./internal/repository/...` | 通过 |

## 执行记录

- 2026-08-10：将分组保存和关系同步收紧到同一 Ent/SQL 事务，定向仓储测试通过。
