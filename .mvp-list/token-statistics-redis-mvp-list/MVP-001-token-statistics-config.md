# MVP-001：建立 Token 统计配置契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 配置结构、默认值、校验和聚焦单测可在一个小闭环内完成。
- Dependencies: `none`

## 预期成果

应用能够加载并校验 Redis Token 日统计所需配置，旧 Key 默认保留 2 天，非法值返回包含配置路径和值的具体错误。

## 背景

配置入口位于 `backend/internal/config/config.go`，现有配置测试集中在 `backend/internal/config/config_test.go`。

## 范围内

- 增加 `gateway.token_statistics` 配置结构。
- 设置 Plan 中六个默认值。
- 校验正整数和开关语义。
- 更新 `deploy/config.example.yaml`。
- 增加默认值与非法值测试。

## 范围外

- Redis Repository、同步 worker 和业务路径接入。

## 实现说明

- 错误需包含完整配置路径、实际值和有效范围。
- `redis_retention_days` 默认值必须为 `2`。

## 验收标准

- [x] 所有配置字段均可通过 Viper/mapstructure 加载。
- [x] 默认值与最终 Plan 一致。
- [x] 非法值启动校验失败且错误具体可查。
- [x] 配置单测通过。

## 验证计划

- `cd backend; go test ./internal/config -run "TokenStatistics|Config"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/config/config.go` | 新增 `GatewayTokenStatisticsConfig`、六项默认值及携带路径和值的边界校验。 |
| 示例 | `deploy/config.example.yaml` | 新增 `gateway.token_statistics` 六项配置示例。 |
| 测试 | `cd backend; go test ./internal/config -run "TokenStatistics|Config"` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/config 2.462s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：实现并验证配置加载、默认值、覆盖值及非法值错误契约；未进入 Redis Repository 或业务 wiring 范围。

