# MVP-006：实现内部默认 API Key 创建能力

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-001, MVP-002, MVP-003`

## 预期成果

受信任的内部服务可为用户幂等创建 `purpose=default` 的默认 Key，并按默认分组状态绑定。

## 背景

现有 `APIKeyService.Create` 带用户侧分组权限校验；初始化路径需要明确的内部入口，而不是绕过公开接口校验。

## 范围内

- 新增内部默认 Key创建方法。
- 生成 `Default API Key`，`platform=NULL`、`purpose=default`。
- 默认分组缺失时写入 `group_id=NULL`。
- 处理唯一冲突并回读已有默认 Key。
- 添加服务单元测试。

## 范围外

- 用户记录创建事务。
- 平台专属 Key。

## 实现说明

- 复用现有安全随机 Key生成逻辑。
- 内部入口不可直接暴露为用户 HTTP API。

## 验收标准

- [x] 重复调用返回同一默认 Key，不生成重复记录。
- [x] 默认分组存在/缺失场景测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... -run 'DefaultAPIKey|DefaultKey'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 服务实现 | `backend/internal/service/api_key_service.go` | 新增内部 `EnsureDefaultAPIKey`：优先回读，按默认分组状态创建，唯一冲突后回读已有默认 Key。 |
| Repository | `backend/internal/repository/api_key_repo.go` | 新增仅查询未软删除 `purpose=default` Key 的 `GetDefaultByUserID`。 |
| 单元测试 | `backend/internal/service/api_key_service_default_test.go` | 覆盖幂等、默认分组存在、默认分组缺失和唯一冲突回读。 |
| 定向验证 | `cd backend; go test ./internal/service/... -run 'DefaultAPIKey|DefaultKey' -count=1` | 通过。 |
| 构建标签验证 | `cd backend; go test -tags=unit ./internal/service -run 'DefaultAPIKey|DefaultKey' -count=1` | 通过。 |
| Repository 回归 | `cd backend; go test ./internal/repository/...` | 通过。 |

## 执行记录

已完成：内部入口不暴露为 HTTP API；默认分组存在时绑定其 ID，未配置或缺失时写入 `group_id=NULL`，并通过唯一冲突回读保证幂等结果。
