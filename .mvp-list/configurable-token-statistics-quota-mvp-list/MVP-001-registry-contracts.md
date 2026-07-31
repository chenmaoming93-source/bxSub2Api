# MVP-001：建立动态统计注册表与领域契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `可在一个纵向切片内完成注册表、领域类型、周期计算、独立配置和隔离测试。`
- Dependencies: `none`

## 预期成果

后端拥有独立于旧体系的动态维度、指标、投影和用量事件领域契约，并能稳定计算自然日、周、月及规范维度签名。

## 背景

现有固定统计类型不能被新体系复用。新代码需使用独立命名和配置，首期注册 `user_id`、`api_key_id`、`group_id`、`route_alias`、`account_id`、`upstream_model` 与 `total_tokens`。

## 范围内

- 新维度和指标注册表。
- 动态值类型、投影定义、用量事件、周期类型。
- Asia/Shanghai 自然周期计算。
- 维度排序、签名和规范编码/哈希契约。
- `gateway.dynamic_token_statistics` 独立配置与校验。
- 新旧体系依赖隔离测试。

## 范围外

- MySQL 表。
- Redis 写入。
- 管理 API 和前端。

## 实现说明

- 优先使用独立 `tokenstat` 包或 `DynamicTokenStat*` 前缀。
- 注册表必须拒绝未知代码、类型错误、重复维度和语义不稳定配置。
- 规范编码需要版本化并保存可验证的原始内容。

## 验收标准

- [x] 六个首期维度和 `total_tokens` 已注册且可枚举。
- [x] 自然日、周、月边界测试覆盖月末、年末和周一。
- [x] 不同选择顺序生成相同维度签名。
- [x] 规范编码和 128 位哈希具有稳定测试向量。
- [x] 独立配置默认值和非法值校验通过。
- [x] 新领域包不引用旧 `TokenStatisticsType` 和旧固定限额接口。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/config/...`
- `rg -n "TokenStatisticsType|DailyTokenQuota" backend/internal/service/tokenstat backend/internal/repository/tokenstat`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/tokenstat` | 六维注册表、指标、领域类型、自然周期及版本化规范编码已实现 |
| 测试 | `cd backend && go test ./internal/service/... ./internal/config/...` | 通过 |
| 隔离 | `rg -n "TokenStatisticsType\|DailyTokenQuota" backend/internal/service/tokenstat` | 无匹配；新包未引用旧固定统计/限额 |

## 执行记录

2026-07-30：完成注册表、领域类型、Asia/Shanghai 自然周期、稳定 128 位哈希向量及独立配置校验；全部验收项验证通过。
