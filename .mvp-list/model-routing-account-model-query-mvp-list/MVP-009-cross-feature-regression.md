# MVP-009：完成跨功能回归与无迁移交付验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `仅执行聚焦回归、构建和结构检查，并修复本需求直接引入的小范围问题；不承担新的功能开发。`
- Dependencies: `MVP-005, MVP-008`

## 预期成果

两个需求在同一工作树中通过前后端聚焦回归，确认接口契约、历史路由兼容性和无数据库迁移约束均满足，可进入交付。

## 背景

前后端分支可独立完成，但最终需要共同验证模型路由提交结构、`getOrCreate` 回归、新查询接口和构建结果。本 MVP 是可审计的集成交付门槛。

## 范围内

- 运行模型路由编辑器及 i18n 聚焦测试。
- 运行 External Provisioning Service、Handler、路由和鉴权测试。
- 运行前端构建和相关 Go 包测试。
- 检查 `model_routing` 结构未变。
- 检查没有新增 Ent Schema、Migration 和配置字段。
- 记录实际命令与结果；仅修复本需求直接导致的回归。

## 范围外

- 不增加新的产品功能。
- 不处理与本需求无关的既有失败。
- 不执行生产部署。

## 实现说明

- 若全量测试耗时超出 40 分钟，优先完成列出的聚焦测试和构建，并把既有全量失败明确记录为外部证据。
- 使用 `git diff -- backend/ent backend/config/config.yaml deploy/config.example.yaml` 等只读检查确认无非预期数据或配置变化。

## 验收标准

- [x] 前端聚焦测试和构建通过。
- [x] 后端 Service、Handler、路由及鉴权测试通过。
- [x] `getOrCreate` 现有成功和错误契约保持通过。
- [x] 模型路由持久化结构与历史配置兼容。
- [x] 没有新增数据库 Migration、Ent Schema 或后端配置字段。
- [x] 所有完成证据已写回各 MVP 文档和 `mvp-progress.md`。

## 验证计划

- `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/i18n/__tests__/modelTokenQuotaLocales.spec.ts`
- `pnpm --dir frontend run build`
- `cd backend && go test ./internal/service ./internal/handler ./internal/server/...`
- `git diff --check`
- `git diff -- backend/ent backend/config/config.yaml deploy/config.example.yaml`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 前端回归 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/i18n/__tests__/modelTokenQuotaLocales.spec.ts` | 通过：2 个测试文件、13 个测试。 |
| 前端构建 | `pnpm --dir frontend run build` | 通过：895 个模块转换并生成 production bundle；仅有既有构建警告。 |
| 后端回归 | `cd backend && go test ./internal/service ./internal/handler ./internal/server/...` | 通过：Service、Handler、Server、Middleware、Routes 包全部成功。 |
| 格式检查 | `git diff --check` | 通过：退出码 0。 |
| 结构与配置检查 | `git status`、`git diff -- backend/ent backend/config/config.yaml deploy/config.example.yaml` | 本计划未改动模型路由持久化结构、Ent Schema、Migration、SQL 或配置；工作区中相关文件均为执行前已存在的其他用户改动。 |
| 文档审计 | 手工检查计划目录全部 Markdown | 9 个 MVP 均为 `mvp-list/v1`、`VERIFIED`，验收项与证据完整。 |

## 执行记录

跨功能回归完成。新前端仍序列化原有 `model`、`account_ids`、`priority`、`daily_token_limit`；新后端接口只读取现有 `model_routing`，未改变持久化格式。
