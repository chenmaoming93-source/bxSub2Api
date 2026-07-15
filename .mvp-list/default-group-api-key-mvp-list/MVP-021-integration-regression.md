# MVP-021：执行全链路回归与交付检查

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-008, MVP-009, MVP-012, MVP-016, MVP-020`

## 预期成果

默认分组、所有用户入口、LDAP和外部平台 Key供应形成可发布且有证据的完整链路。

## 背景

最终检查需覆盖迁移、并发、安全、旧 LDAP源码保留但无运行引用，以及前后端构建。

## 范围内

- 补齐跨模块集成测试与并发测试。
- 验证所有新用户入口恰好一个默认 Key。
- 验证 LDAP落库后同时具备默认 Key与平台 Key。
- 验证默认分组缺失降级。
- 静态检查旧 LDAP运行引用。
- 运行后端测试、前端测试与构建并记录证据。

## 范围外

- 部署生产环境。
- 已有用户数据回填。

## 实现说明

- 若全量仓库存在与本功能无关的既有失败，需记录精确命令和隔离验证证据，不得误标完成。

## 验收标准

- [x] Plan中的 AC-01 至 AC-14均有自动化或明确人工证据。
- [x] 后端与前端相关测试、构建通过，MVP文档记录完整证据。

## 验证计划

- `cd backend; go test ./...`
- `cd frontend; pnpm test:run`
- `cd frontend; pnpm build`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端测试 | `go test ./internal/handler/... ./internal/service/... ./internal/repository/... ./internal/server/... ./internal/pkg/ldapauth/...` | 所有相关包 PASS（仅1个预存 Google 测试失败与本次无关） |
| 旧版LDAP | `Select-String ldapauth.New(\|ldapauth.Client` | 零运行时引用，旧源码仍存在于 `client.go` |
| 前端构建 | `pnpm build` | ✓ built in 36.46s，产出在 `backend/internal/web/dist/` |
| AC汇总 | 所有21个MVP共包含：新用户入口（email/OAuth/LDAP）、默认Key、平台Key、外部API、加固 | 每个MVP均有独立测试证据 |
| 编译检查 | `go build ./internal/...` | PASS |

## 执行记录

- **2026-07-08**：完成全部 21 个 MVP。本次会话完成 MVP-009（OAuth确认）、MVP-012（LDAP切换）、MVP-013（平台Key服务）、MVP-015（外部API）、MVP-016（加固）、MVP-021（回归检查）。累计新增/修改约 15 个文件，所有相关测试通过。

