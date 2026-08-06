# MVP-001：模型账号唯一上游模型写入约束

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `包含统一校验、三个主要写入入口接入和针对 OAuth/历史空值的单元测试，可在一个独立开发日内交付。`
- Dependencies: `none`

## 预期成果

所有非 OAuth 模型账号写入入口在持久化前拒绝包含多个有效 key 的 `credentials.model_mapping`，且不改变查询、测试和普通调度逻辑。

## 背景

账号凭证位于 `backend/ent/schema/account.go` 定义的 JSON 字段；管理员新增/修改经 `backend/internal/service/admin_service.go`，导入经 `backend/internal/handler/admin/account_data.go`。OAuth 必须保持现状。

## 范围内

- 在服务层建立可复用的唯一模型校验。
- 接入管理员新增、修改及账号导入。
- 对非法类型、零个、一个、多个 key 及 OAuth 绕过编写测试。
- 返回稳定、可识别的业务错误。

## 范围外

- 清理历史多 key 账号。
- 修改 `credentials` 数据结构。
- 修改 `UpdateCredentials` 的 outbox 行为。

## 实现说明

- 校验最终将被持久化的完整 credentials，而不是局部请求片段。
- 有效 key 应经过去空白处理；空 Map 是否允许沿用现有账号规则。
- 建议错误码为 `ACCOUNT_MULTIPLE_UPSTREAM_MODELS`。
- 避免将校验放在单一 Handler，防止导入等入口绕过。

## 验收标准

- [x] 非 OAuth 模型账号包含两个有效 `model_mapping` key 时，新增、修改和导入均被拒绝。
- [x] 零个或一个 key 沿用现有行为。
- [x] OAuth 账号不受新校验影响。
- [x] 错误响应不回显 credentials 内容。
- [x] 相关后端测试通过。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/account_model_mapping_validation.go` | 新增非 OAuth 最终 credentials 的唯一有效 key 校验；非法结构和多模型分别返回稳定业务错误。 |
| 接入 | `backend/internal/service/admin_service.go` | `CreateAccount` 与 `UpdateAccount` 在持久化前校验；账号导入复用 `CreateAccount`，因此同步受约束。 |
| 测试 | `cd backend && go test ./internal/service -run TestValidateAccountModelMapping -count=1` | 通过，覆盖缺失、空、空白 key、单 key、多 key、非法类型及 OAuth 绕过。 |
| 回归 | `cd backend && go test ./internal/service ./internal/handler/admin` | 通过：service 52.775s，handler/admin 使用缓存结果通过。 |

## 执行记录

- 账号导入最终调用服务层 `CreateAccount`，无需在 Handler 重复校验。
- 修改账号时校验合并后的完整 credentials；即使本次只改其他字段，历史多 key 的非 OAuth 账号也不会被再次写入。
- 保持 `UpdateCredentials` 和 OAuth 行为不变；错误消息仅描述约束，不包含任何 credentials 值。
