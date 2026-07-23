# MVP-006：实现分组路由只读投影 Service

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `复用现有精确分组查询和路由解析，仅增加只读投影、去重排序及单元测试，不涉及 HTTP 或持久化变更。`
- Dependencies: `none`

## 预期成果

Service 可按分组名读取现有 `model_routing`，生成稳定的“路由别名 → 上游模型列表”只读结果。

## 背景

`ExternalProvisioningService` 已依赖 `GetByNameExact`，分组领域对象也已有模型路由解析能力。本需求必须复用现有数据，不新增表或 SQL。

## 范围内

- 定义 Service 输入、结果及路由项类型。
- 清理并校验分组名。
- 复用 `GetByNameExact` 查询分组。
- 解析现有候选结构，提取非空模型。
- 对模型去重，对别名和模型稳定排序。
- 无路由配置时返回非 `nil` 空数组语义。
- 增加 Service 单元测试。

## 范围外

- 不新增 Handler 或路由。
- 不新增 Repository 方法、SQL 或数据库迁移。
- 不返回账号、优先级和限额。

## 实现说明

- 查询配置内容时不依赖 `model_routing_enabled` 是否开启。
- 分组不存在沿用 `ErrGroupNotFound`。
- 解析异常包装为内部错误，避免泄漏配置细节。

## 验收标准

- [x] 精确分组名可返回全部路由别名及去重模型。
- [x] 返回顺序稳定且空配置返回空列表。
- [x] 分组不存在和 Repository 错误可被上层区分。
- [x] 结果不含账号、优先级和限额。
- [x] 未产生 Schema、Migration 或新 SQL。

## 验证计划

- `cd backend && go test ./internal/service -run 'TestExternalProvisioningService.*Group.*Route'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/external_provisioning_service.go` | 新增只读路由投影，复用 `GetByNameExact` 与领域解析，按别名和模型稳定排序。 |
| 测试 | `cd backend && go test ./internal/service -run 'TestExternalProvisioningService.*Group.*Route'` | 通过：service 包退出码 0。 |
| 结构检查 | 启动前工作区对照与 `git diff --name-only -- backend/ent backend/migrations backend/sqlArchiving ...` | 本 MVP 未新增或修改 Schema、Migration、SQL、配置；检查中列出的相关改动均为执行前已存在的用户改动。 |

## 执行记录

已覆盖禁用路由标志下的读取、去重排序、非 `nil` 空数组、未找到和 Repository 错误传播。
