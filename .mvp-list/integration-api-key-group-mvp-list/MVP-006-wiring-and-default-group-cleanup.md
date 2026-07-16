# MVP-006：完成依赖注入并移除默认分组供应链路

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `集中处理构造函数和 Wire 生成代码的连锁编译修改，以全后端编译和定向测试作为独立交付。`
- Dependencies: `MVP-002, MVP-004, MVP-005`

## 预期成果

生产依赖注入完整连接分组查询、三元组平台 Key 服务和外部供应 Handler，外部供应创建路径不再解析或使用系统默认分组，后端所有包可编译。

## 背景

构造函数变更会影响 `backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go` 以及测试 stub。该工作需要在各局部能力完成后统一收口，避免主分支持续处于不可编译状态。

## 范围内

- 更新平台 Key 服务、供应服务及相关 Provider 的构造调用。
- 注入现有 GroupRepository 或最小查询适配器。
- 移除外部供应路径中的默认分组 resolver 参数。
- 更新 Wire 声明及生成结果，遵循项目现有生成方式。
- 修复受接口签名影响的编译 stub 和测试构造器。
- 执行全后端编译/测试编译检查。

## 范围外

- 不改变普通默认 API Key 的默认分组行为。
- 不删除全系统通用 `DefaultGroupResolver`。
- 不修改配置文件。
- 不新增业务规则。

## 实现说明

- 仅移除 `PlatformAPIKeyService` 在外部平台 Key 场景的默认分组依赖。
- `APIKeyService.GetOrCreateDefault` 等普通默认 Key 流程必须保持原行为。
- Wire 生成代码必须与声明一致，不允许只手工修改生成文件。

## 验收标准

- [x] 生产路由可构造新的供应服务。
- [x] 外部平台 Key 路径不调用 `ResolveDefaultGroup`。
- [x] 普通默认 Key 流程相关测试保持通过。
- [x] 所有受影响 stub 和构造调用已同步。
- [x] 后端全包能够编译，定向 service/handler 测试通过。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler ./internal/server/routes`
- `cd backend && go test ./internal/service -run 'TestGetOrCreateDefault|TestEnsurePlatformKey|TestGetOrCreatePlatformKey'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 生产装配 | `backend/cmd/server/wire_gen.go` | 现有手工供应链路注入具体 GroupRepository，并使用无默认分组 resolver 的平台 Key 服务构造器。 |
| 仓储 Provider | `backend/internal/repository/group_repo.go` | 构造器保留具体仓储类型，使 Wire 装配可同时满足通用仓储与精确名称查询窄接口。 |
| 定向验证 | `cd backend && go test ./internal/service -run 'TestGetOrCreateDefault|TestEnsurePlatformKey|TestGetOrCreatePlatformKey'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 6.202s`。 |
| 包验证 | `cd backend && go test ./internal/service ./internal/handler ./internal/server/routes` | 全部通过：service、handler、routes。 |
| 生产入口 | `cd backend && go test ./cmd/server ./internal/service ./internal/handler ./internal/server/routes` | 全部通过；`cmd/server` 编译与测试通过。 |
| 配置检查 | `git diff -- backend/config/config.yaml deploy/config.example.yaml` | 无输出；本功能未变更配置。 |

## 执行记录

2026-07-16：完成构造器、生产装配、仓储具体类型与测试 stub 收口；确认外部路径无 `ResolveDefaultGroup`，普通默认 Key 回归和生产入口编译通过。
