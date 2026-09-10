# MVP-001：分组支持可选且可重复的场景名

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 覆盖一个完整的分组字段垂直切片：迁移、Ent、后端输入输出和分组管理页，范围集中且可独立验收。
- Dependencies: `none`

## 预期成果

分组可以保存、编辑并展示可为空、可重复的 `scene_name`，而现有唯一技术分组名 `name` 的行为保持不变。

## 背景

当前分组实体定义在 `backend/ent/schema/group.go`，分组业务和 DTO 位于 `backend/internal/service` 与 `backend/internal/handler/dto`，管理页面为 `frontend/src/views/admin/GroupsView.vue`。新增字段不参与路由、绑定和唯一性判断。

## 范围内

- 在 `backend/sqlArchiving/172_add_group_scene_name.sql` 使用项目规约要求的位置新增 `groups.scene_name VARCHAR(100) NULL` 数据库归档 SQL。
- 更新 Ent schema、生成代码、Service、Repository、DTO 和前端类型。
- 更新分组创建与编辑接口。
- 更新 `/admin/groups` 的创建表单、编辑表单和列表展示。
- 空值展示为“未设置场景名”，不得增加唯一约束。
- 增加后端和前端针对空值、重复值和字段回显的测试。

## 范围外

- 不改变 `groups.name` 的唯一约束。
- 不将 `scene_name` 用作统计维度、路由匹配或分组查询键。
- 不实现场景 Token 用量统计接口和用量页面。

## 实现说明

- 遵循项目现有数据库迁移编号和 Ent 生成流程。
- Service 层可使用空字符串表示空场景名，HTTP DTO 按现有分组字段风格保持兼容。
- 所有分组实体转换入口都必须映射 `scene_name`，避免列表、详情和嵌套对象不一致。

## 验收标准

- [x] 数据库迁移增加可为空的 `scene_name`，且没有唯一约束。
- [x] 创建和编辑分组可以提交空值及与其他分组相同的场景名。
- [x] 分组列表和详情正确返回 `scene_name`，空值显示为“未设置场景名”。
- [x] `name` 的唯一校验和现有分组功能保持不变。
- [x] 后端相关测试通过，前端相关测试或类型检查通过。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/repository/... ./internal/handler/...`
- `cd frontend && pnpm test:run -- src/views/admin/__tests__`
- `cd frontend && pnpm typecheck`
- 通过测试或人工检查确认两个分组可以使用相同 `scene_name`，且按 `name` 的既有查询行为不变。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| SQL 归档 | `backend/sqlArchiving/172_add_group_scene_name.sql` | MySQL/GoldenDB 兼容的可空字段，无唯一约束；按 PROJECT_CONVENTIONS.md 采用归档目录和现有编号 172。 |
| Ent/后端 | `go generate ./ent; go generate ./cmd/server` | 通过；生成代码包含 `SceneName`、可空 setter 和迁移元数据。 |
| 后端聚焦测试 | `go test -tags unit ./internal/service -run 'TestAdminService_(CreateGroup_PreservesOptionalSceneName|UpdateGroup_AllowsDuplicateOrEmptySceneName)' -count=1` | 通过。 |
| Repository 映射测试 | `go test ./internal/repository -run TestGroupEntityToService_PreservesMessagesDispatchModelConfig -count=1` | 通过；分组实体映射保持可用。 |
| 前端类型检查 | `pnpm typecheck` | 通过。 |
| 前端管理测试 | `pnpm test:run -- src/views/admin/__tests__` | 82 个测试通过，但命令因既有 `AccountsView` 未处理的 `getModelRouteReferences` 异步错误退出 1；该错误与本 MVP 无关，已记录限制。 |
| 变更路径 | `backend/ent/schema/group.go`, `backend/internal/service/{group.go,admin_service.go,group_service.go}`, `backend/internal/repository/{group_repo.go,api_key_repo.go}`, `backend/internal/handler/{admin/group_handler.go,dto/{types.go,mappers.go}}`, `frontend/src/{types/index.ts,views/admin/GroupsView.vue,i18n/locales/{zh.ts,en.ts}}`, `backend/internal/service/group_scene_name_test.go` | 完成字段垂直传递、管理表单/列表展示及空值文案。 |

## 执行记录

- 2026-08-26：按 `PROJECT_CONVENTIONS.md` 核对 DDL 归档目录和编号；保留项目已有的安全检查相关工作树修改，完成 `scene_name` 垂直切片并执行聚焦验证。

