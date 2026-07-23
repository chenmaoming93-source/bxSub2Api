# MVP-004：供应服务执行分组解析与访问校验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `在既有供应编排中加入一个集中准入步骤，并用 service stub 覆盖全部分支。`
- Dependencies: `MVP-001`

## 预期成果

外部供应服务在创建 Key 前按 `group_name` 解析分组，并严格执行 active、standard、公开/专属授权规则；任何校验失败都不调用 Key 创建服务。

## 背景

当前 `backend/internal/service/external_provisioning_service.go` 只解析用户并调用平台 Key 服务。用户对象已经包含 `AllowedGroups`，领域方法 `User.CanBindGroup` 已定义公开和专属标准分组的授权语义。

## 范围内

- 在供应输入中加入 `GroupName`。
- 注入最小化的按名称分组查询接口。
- 对 group name 做 Trim 后精确查询。
- 拒绝不存在、inactive 和 subscription 分组。
- 公开标准分组允许，专属标准分组复用 `User.CanBindGroup`。
- 将解析出的 group ID 传给平台 Key 服务。
- 定义可供 Handler 映射的稳定领域错误。
- 补充本地用户和 LDAP 新用户场景测试。

## 范围外

- 不实现 HTTP 请求/响应字段。
- 不比较请求 platform 与 `groups.platform`。
- 不实现订阅有效性检查。
- 不修改 `user_allowed_groups`。

## 实现说明

- 分组查询或数据库异常必须 fail closed。
- LDAP 新创建用户通常没有专属授权，因此专属分组应被拒绝；公开分组可继续创建。
- 测试 stub 应记录平台 Key 服务是否被调用，以证明失败分支没有副作用。

## 验收标准

- [x] 公开 active standard 分组允许创建。
- [x] 已授权专属 active standard 分组允许创建。
- [x] 未授权专属分组返回 `GROUP_NOT_ALLOWED` 对应领域错误。
- [x] subscription 分组返回不支持错误。
- [x] inactive 和不存在分组返回各自领域错误。
- [x] platform 与分组平台不同时仍允许。
- [x] 所有拒绝分支均未创建 Key。
- [x] 供应服务定向测试通过。

## 验证计划

- `cd backend && go test ./internal/service -run 'TestEnsurePlatformKey'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/external_provisioning_service.go` | Trim 后按名称精确查询，fail closed；校验 active、standard 与 `User.CanBindGroup`，并把解析出的 group ID 传给平台 Key 服务。 |
| 错误契约 | `backend/internal/service/external_provisioning_service.go` | 定义 `GROUP_INACTIVE`、`SUBSCRIPTION_GROUP_NOT_SUPPORTED`、`GROUP_NOT_ALLOWED` 稳定领域错误；不存在复用 `GROUP_NOT_FOUND`。 |
| 测试 | `backend/internal/service/external_provisioning_service_test.go` | 覆盖公开/专属授权、未授权、inactive、subscription、不存在、跨 platform、本地用户及 LDAP 新用户，拒绝分支断言未创建 Key。 |
| 验证 | `cd backend && go test ./internal/service -run 'TestEnsurePlatformKey'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.852s`。 |

## 执行记录

2026-07-16：完成供应服务分组解析、准入校验和稳定领域错误，并通过定向测试。
