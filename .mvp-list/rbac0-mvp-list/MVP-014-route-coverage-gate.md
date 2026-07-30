# MVP-014：建立路由权限闭合门禁

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `集中实现启动校验、CI 测试和 SQL/代码一致性，是一个独立质量门禁。`
- Dependencies: `MVP-007, MVP-008, MVP-009, MVP-010, MVP-011, MVP-012, MVP-013`

## 预期成果

新增或修改 Gin 路由时，任何未声明权限或排除原因的接口都会在启动验证或 CI 中失败。

## 背景

当前基线为 `522 = 427 + 95`，未来数字可变，但闭合公式必须成立。

## 范围内

- 枚举实际 Gin 路由。
- Registry 受控/排除分类核对。
- 权限代码目录、数据库 Seed 与路由声明一致性检查。
- 启动 readiness 校验和 CI 测试。
- 可读的差异报告。

## 范围外

- 修改业务权限定义。
- 前端路由门禁。

## 实现说明

- 实际 Gin 路由是最终事实来源，Markdown 清单仅为迁移基线。
- 测试必须输出 Method、Path 和缺失原因，便于修复。

## 验收标准

- [x] 当前所有路由闭合，无未分类项。
- [x] 新增未登记测试路由会稳定导致测试失败。
- [x] 未知权限编码和 SQL 缺失权限均能被发现。
- [x] 公开、网关、集成和 webhook 排除理由完整。

## 验证计划

- `cd backend && go test ./internal/server/... -run RBACRouteCoverage`
- `cd backend && go test ./internal/rbac/... -run CatalogConsistency`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/rbac/coverage.go` | 以 `gin.Engine.Routes()` 为事实来源核对 Registry，输出总数、受控数、排除数和 Method/Path 缺口；仅已知公开信任边界可自动登记排除理由。 |
| 启动门禁 | `backend/internal/server/router.go` | 所有路由注册完成后执行排除分类与闭合校验；出现未登记路由时启动立即失败。 |
| 测试 | `go test ./internal/rbac/... -run 'CatalogConsistency|RBACRouteCoverage' -count=1` | 通过；遗漏路由稳定失败，排除理由完整，目录权限逐项存在于 MySQL/GoldenDB 兼容 Seed。 |
| 测试 | `go test ./internal/server/... -run RBACRouteCoverage -count=1` | 通过。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过。 |

## 执行记录

最终数字不再硬编码：每次启动从实际 Gin 路由动态计算 `Total = Controlled + Excluded`。原 `522 = 427 + 95` 仅保留为迁移清单基线；当前代码删掉了清单中的一个历史用户接口，因此不能用旧数字掩盖实际差异。
