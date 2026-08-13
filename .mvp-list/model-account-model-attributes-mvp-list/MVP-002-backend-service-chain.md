# MVP-002：后端业务链路（service / repository / handler / dto）

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: 1 个开发者日
- Estimate rationale: 涉及 service 输入/领域对象、repository 读写映射、handler 请求体与 dto 响应四层贯通及对应单测，工作量约一个工作日。
- Dependencies: `MVP-001`

## 预期成果

创建/更新账户接口支持 `model_attributes`（JSON map）：创建时落库、更新时「缺省不动 / `{}` 清空 / 提供则覆盖」，账户列表/详情/更新响应回带该字段；后端对 value 原样存储。

## 背景

- 相关路径：`backend/internal/service/account.go`（Account 领域对象）、`backend/internal/service/admin_service.go`（Create/UpdateAccountInput 与 CreateAccount/UpdateAccount）、`backend/internal/repository/account_repo.go`（ent client 读写）、`backend/internal/handler/admin/account_handler.go`、`backend/internal/handler/dto/`（响应映射）。
- 请求/响应契约见 plan 的 API-01/02/03：请求体字段 `model_attributes` 为 map 类型，天然区分「缺省(nil) = 不改动」与「`{}` = 清空」。
- 后端信任前端：不做 value 类型解析；仅应用 `Normalize()` 最小规整。

## 范围内

- `service.Account` 新增 `ModelAttributes domain.ModelAttributes` 字段。
- `CreateAccountInput` / `UpdateAccountInput` 新增 `ModelAttributes domain.ModelAttributes`（map，nil 表示未提供）。
- `CreateAccount`：写入 `Normalize()` 后的 map。
- `UpdateAccount`：`input.ModelAttributes != nil` 时覆盖（含空 map 清空），nil 时保留原值。
- `repository`：`Create`/`Update` 增加 `SetModelAttributes`（或等价写入），`accountEntityToService` 读回映射。
- handler：`CreateAccountRequest` / `UpdateAccountRequest` 增加 `model_attributes` 字段并透传；dto `Account` 增加字段并映射。
- 单测：service 更新语义（覆盖/清空/缺省不动、Normalize 生效）；repository 读写往返；handler 请求体绑定与响应字段。

## 范围外

- 前端任何改动（见 MVP-003/MVP-004）。
- 网关/调度等运行时消费这些属性。
- value 类型解析或内容校验。

## 实现说明

- 请求体字段类型用 map（`map[string]domain.ModelAttributeItem` 或等价 JSON 绑定类型），nil 语义由指针/零值区分，保持与现有 `Extra` 的处理模式一致（`input.Extra != nil` 语义参考）。
- 响应 dto 字段使用 `omitempty`（倾向未配置时不输出，需与 MVP-004 回显容错配合）。
- 复用现有 `MergePreservingSensitiveCreds` 模式无关，本字段为全量覆盖语义，无敏感子键。

## 验收标准

- [x] `go build ./...`（backend 全量）通过
- [x] service 单测：创建携带 map 落库；更新提供 map 覆盖；更新传空 map 清空；更新缺省字段保留原值（3 用例全过）
- [x] repository 单测：`accountEntityToService` 读回映射（nil/类型保留/拷贝隔离）通过；真实 DB 写入按用户指示 2 不再要求 MySQL 测试库
- [x] handler 测试：请求体 `model_attributes` 绑定成功并透传、响应包含 `model_attributes`（“非法结构 400”由 gin 类型绑定默认保证，未单独用例）
- [x] 数据库值确认（静态层）：domain/repository 测试断言字符串 `"true"` 原样保留不转布尔；真实 DB 值确认按用户指示 2 不再要求 MySQL 测试库

## 验证计划

- `cd backend && go build ./...`
- `cd backend && go test ./internal/service/... ./internal/repository/... ./internal/handler/admin/... ./internal/handler/dto/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 修改 | `backend/internal/service/account.go` | `Account` 新增 `ModelAttributes domain.ModelAttributes` 字段 |
| 修改 | `backend/internal/service/admin_service.go` | `CreateAccountInput`/`UpdateAccountInput` 新增字段；`CreateAccount` 写入 `Normalize()`；`UpdateAccount` nil=不改动/空 map=清空/提供=覆盖 |
| 修改 | `backend/internal/repository/account_repo.go` | `Create`/`Update` 写入 `SetModelAttributes`（Update 含 `ClearModelAttributes`）；`accountEntityToService` 读回；`copyModelAttributes` helper |
| 修改 | `backend/internal/handler/admin/account_handler.go` | `CreateAccountRequest`/`UpdateAccountRequest` 加 `model_attributes` 并透传 |
| 修改 | `backend/internal/handler/dto/{types,mappers}.go` | dto `Account` 加 `model_attributes,omitempty` 并映射 |
| 新增测试 | `backend/internal/service/admin_service_model_attributes_test.go` | 覆盖/清空/缺省 + Normalize + 信任前端（3 用例全过） |
| go build | `cd backend && go build ./...` | 通过（exit 0）；`go vet` 通过 |
| go test | `go test ./internal/repository/ -run TestAccountEntityToService_ModelAttributes` | ok，3 子用例全过（nil/类型保留/拷贝隔离） |
| go test | `go test ./internal/handler/admin/ -run "TestAccountHandler_(Update|Create)_ModelAttributes"` | ok，2 用例全过（透传 + 响应回带） |
| go test | `go test -tags unit ./internal/service/ -run TestAdminService_UpdateAccount_ModelAttributes` | ok，3 子用例全过（覆盖/清空/缺省 + Normalize + 字符串 true 原样） |
| go test | `go test -tags unit ./internal/service/`（全量） | 编译通过；6 个路由用例失败（`TestGatewayService_SelectAccountWithLoadAwareness`，用户 model-routing 重构未完成，非本任务） |

## 执行记录

- 2026-08-05：实现完成，`go build ./...` 与 `go vet` 通过；repository 与 handler 新增单测通过。
- **阻塞解除（用户指示 1）**：经用户授权可修改 test 文件后，对 `gateway_record_usage_test.go` 做最小修复（删除已从 `NewGatewayService` 签名移除的 `dailyTokenQuotaRepo`、`tokenStatsAccumulator` 两个参数，与用户改后的 27 参签名对齐），service 测试包恢复可编译，service 层单测 3 用例全部通过。未修改任何业务代码。
- **遗留基线失败（非本任务）**：`go test -tags unit ./internal/service/` 全量运行时，`TestGatewayService_SelectAccountWithLoadAwareness` 的 6 个路由子用例断言失败（`gateway_multiplatform_test.go:2651` "no available accounts"），属于用户正在进行的 model-routing-wildcard-usage-refactor 未完成工作，与 model_attributes 改动无关，未处理。
- 数据库值确认（真实库中字符串 "true" 不转布尔）与真实 DB 写入验证按用户指示 2 不再要求 MySQL 测试库，以 domain/repository 静态层断言为准。
- JSON 数字经 gin 绑定为 float64 是 Go 的预期行为（any 值），符合“信任前端”设计；handler 测试断言已按 float64 编写。
