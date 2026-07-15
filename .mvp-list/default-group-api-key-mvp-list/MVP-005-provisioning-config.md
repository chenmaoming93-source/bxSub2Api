# MVP-005：增加外部供应安全配置

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `none`

## 预期成果

服务可通过配置文件或环境变量安全启停外部 API Key供应能力。

## 背景

配置定义和默认值位于 `backend/internal/config/config.go`；Token不得进入管理员设置响应。

## 范围内

- 新增 `ExternalAPIKeyProvisioningConfig`。
- 支持 enabled 与 access_token 的 Viper/环境变量绑定。
- 增加启用状态下 Token非空及强度校验。
- 增加配置解析与校验测试。

## 范围外

- HTTP认证中间件。
- Token轮换管理 UI。

## 实现说明

- 默认关闭。
- 测试不得打印或断言暴露完整生产式 Token。

## 验收标准

- [x] 环境变量能覆盖配置文件且默认关闭。
- [x] 启用但 Token不合格时启动校验失败。

## 验证计划

- `cd backend; go test ./internal/config/... -run 'ExternalAPIKeyProvisioning|Provisioning'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 定向测试 | `cd backend; go test ./internal/config/... -run 'ExternalAPIKeyProvisioning|Provisioning' -count=1` | 通过；覆盖默认关闭、环境变量优先级及 Token 校验。 |
| 完整回归 | `cd backend; go test ./internal/config/... -count=1` | 通过，退出码 0。 |
| 配置实现 | `backend/internal/config/config.go` | 新增 `external_api_key_provisioning.enabled/access_token`，支持 Viper 配置文件和同名大写下划线环境变量。 |
| 安全校验 | `backend/internal/config/external_api_key_provisioning_test.go` | 启用时要求 Token 至少 32 bytes 且不含空白；测试和错误信息均不输出完整 Token。 |

## 执行记录

实现与验证完成。该配置不进入管理员设置模型或响应，仅由配置文件/环境变量控制；默认关闭。
