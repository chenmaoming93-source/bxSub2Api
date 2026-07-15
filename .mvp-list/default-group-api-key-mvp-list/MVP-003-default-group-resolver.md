# MVP-003：实现默认分组设置与解析器

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `none`

## 预期成果

后端可保存默认分组名，并区分未配置、已配置但缺失、已找到三种状态。

## 背景

沿用 `backend/internal/service/setting_service.go`、`SettingRepository` 和 `GroupRepository`；默认分组按清理后的名称精确匹配。

## 范围内

- 新增 `default_group_name` 设置键、默认值和读写方法。
- 为 `GroupRepository` 增加精确名称查询能力。
- 实现 `DefaultGroupResolver` 及三态结果。
- 单元测试空值、缺失、命中和仓储错误。

## 范围外

- 管理员 HTTP接口。
- 前端设置项。

## 实现说明

- 不得在解析器中隐式创建分组。
- 分组缺失是业务状态，不是基础设施错误。

## 验收标准

- [x] 解析器对三种状态返回稳定、可测试结果。
- [x] 服务单元测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... -run 'DefaultGroup|Setting'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 服务测试 | `cd backend; go test ./internal/service/... -run 'DefaultGroup|Setting' -count=1` | 通过；`internal/service` 与 `internal/service/openai_ws_v2` 均退出码 0。 |
| Repository 测试 | `cd backend; go test ./internal/repository -run 'Group' -count=1` | 通过，退出码 0。 |
| 实现 | `backend/internal/service/default_group_resolver.go` | 已覆盖 `unconfigured`、`missing`、`found` 三态；分组缺失不作为错误，也不会隐式创建。 |
| 设置与查询 | `backend/internal/service/setting_service.go`、`backend/internal/repository/group_repo.go` | 已实现清理后的设置读写及未软删除分组精确名称查询。 |

## 执行记录

实现与验证完成。为避免扩大已有 `GroupRepository` 接口导致大量无关测试桩变更，解析器依赖最小的 `DefaultGroupNameLookup` 接口；具体 `groupRepository` 已提供 `GetByNameExact` 实现，行为与计划一致。
