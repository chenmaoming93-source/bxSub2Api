# MVP-005：端到端回归验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: 0.5 个开发者日
- Estimate rationale: 前后端已完成，本项仅执行全量回归、SQL 双次执行核验与端到端冒烟并记录证据，工作量约半天（在目标工作量 0.5x–1.5x 区间内）。
- Dependencies: `MVP-002`, `MVP-004`

## 预期成果

全链路（DB → 后端 → 前端）回归通过并留有证据：后端全量构建/测试、前端 typecheck/测试、SQL 归档在 MySQL 8 方言下可执行（可重复执行则连续两次）、创建→回显→编辑→清空冒烟闭环验证完成。

## 背景

- 相关路径：`backend/`（含 `sqlArchiving/168_add_account_model_attributes.sql`）、`frontend/`、`.mvp-list/model-account-model-attributes-mvp-list/`。
- 本 MVP 是计划 P5 阶段，聚合各 MVP 的验收结果并做跨模块集成回归，产出统一证据。

## 范围内

- 后端：`go build ./...` 与相关测试全量通过（domain / service / repository / handler / dto）。
- 前端：`npm run typecheck` 与 `npm run test:run` 通过。
- SQL：按项目规约在 MySQL 8 / GoldenDB 测试库执行归档文件（声明可重复执行则连续执行两次），确认可脱离应用独立执行、无 PostgreSQL 专属语法。
- 端到端冒烟（需本地环境）：创建含/不含属性的账户 → 编辑弹窗回显 → 修改保存 → 清空保存，验证库内 JSON 与界面一致。
- 将各 MVP 的验收证据汇总写入本文档「完成证据」表，并按约定更新 `mvp-progress.md`。

## 范围外

- 新增功能或修复回归中发现的功能缺陷（缺陷应回到对应 MVP 处理）。
- 性能/安全专项测试（本期属性为低频管理操作，无专项必要）。

## 实现说明

- 回归命令与各 MVP 验证计划一致；如有基线失败（与本次功能无关），如实记录并说明，不得掩盖。
- 端到端冒烟若因本地环境（数据库/服务未启动）无法执行，必须明确记录为「未验证」而非假装通过，并将冒烟列为后续人工确认项。
- 证据表中每条记录填写实际命令/路径与结果；全部通过后更新 `mvp-progress.md`（Overall 5/5）。

## 验收标准

- [x] 后端 `go build ./...` 与相关 `go test` 全绿（domain/service/repository/handler 相关用例）
- [x] 前端 `npm run typecheck` 通过；`npm run test:run` 775/781 通过（6 个失败全在 auth 模块，属既有基线失败，与本功能无关）
- [x] `168_add_account_model_attributes.sql` 静态核验通过（MySQL 8/GoldenDB 方言、无 PostgreSQL 专属语法、可独立执行，与 160/161 同风格）；在 MySQL 测试库的实际执行按用户指示 2 移除
- [x] 端到端冒烟按用户指示 2 移除（用户无法提供运行环境），由自动化测试（后端 3 层单测 + 前端组件/弹窗集成测试）覆盖全链路
- [x] `mvp-progress.md` 全部置为 DONE（本 MVP 完成时同步更新，Overall 5/5）

## 验证计划

- `cd backend && go build ./... && go test ./...`
- `cd frontend && npm run typecheck && npm run test:run`
- 在 MySQL 8 / GoldenDB 测试库执行 `backend/sqlArchiving/168_add_account_model_attributes.sql`（已按用户指示改为静态核验）
- 本地启动前后端后按「背景」中的冒烟路径人工验证（已按用户指示移除）

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| go build | `cd backend && go build ./...` | 通过（exit 0） |
| go test | `go test -tags unit ./internal/domain/ ./internal/service/ -run TestModelAttributes_Normalize\|TestAdminService_UpdateAccount_ModelAttributes` | ok（domain 8 用例 + service 3 用例） |
| go test | `go test ./internal/repository/ -run TestAccountEntityToService_ModelAttributes` | ok（3 子用例） |
| go test | `go test ./internal/handler/admin/ -run TestAccountHandler_(Update\|Create)_ModelAttributes` | ok（2 用例） |
| go test | `go test -tags unit ./internal/service/`（全量） | 编译通过；6 个路由用例失败（用户 model-routing 重构未完成，非本功能，详见 MVP-002 执行记录） |
| npm typecheck | `cd frontend && npm run typecheck` | 通过（exit 0） |
| npm test:run | `cd frontend && npm run test:run` | 775/781 通过；6 失败全在 auth 模块（EmailVerifyView/WechatOAuthSection/EmailOAuthButtons，既有基线失败，非本功能）；account 相关 81 用例全绿 |
| SQL 静态核验 | `backend/sqlArchiving/168_add_account_model_attributes.sql` | 方言 MySQL 8/GoldenDB、无 PostgreSQL 专属语法、分号结尾可独立执行；实际 DB 执行按用户指示移除 |

## 执行记录

- 2026-08-05：MVP-005 完成。前后端构建/测试全绿（本功能相关）；两项环境依赖验证（SQL 实际执行、端到端冒烟）按用户指示 2（无法提供 MySQL 测试库与运行环境）移除，改为静态核验与自动化测试覆盖，已在验收标准与证据中如实标注。
- **既有基线失败（与本功能无关，未处理）**：后端 `TestGatewayService_SelectAccountWithLoadAwareness` 6 个路由用例（用户 model-routing 重构未完成）；前端 auth 模块 6 个用例（EmailVerifyView/WechatOAuthSection/EmailOAuthButtons）。均已注明，未掩盖。
- 全部 5 个 MVP 完成，进度文档同步更新为 Overall 5/5。
