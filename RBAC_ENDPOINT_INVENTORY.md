# 现有后端接口、访问范围与前端页面清单

> 生成依据：当前工作区的 Gin 路由注册代码与 Vue Router 页面。此文档描述的是**当前实现**，不是未来 RBAC 权限方案。

## 口径说明

- “管理员接口”表示当前经过 `AdminAuth`，事实上只有 `role == admin` 可以调用。
- “用户个人接口”表示经过 JWT；管理员账号通常也能登录调用，但接口应只处理当前身份或校验资源归属。
- “公开”不代表没有业务校验；OAuth state、临时令牌、验证码、支付签名等仍属于接口自身安全条件。
- 模型网关使用 API Key，外部集成使用独立密钥，支付 webhook 使用服务商签名；它们不应直接纳入后台用户 RBAC。
- “对应页面”按当前前端业务模块归并。标记为“无页面”的接口是外部调用、服务器回调或兼容协议端点。

## 数量汇总

| 分类 | 数量 |
|---|---:|
| 管理员接口 | 362 |
| 用户个人接口 | 65 |
| 公开及登录流程接口 | 54 |
| 支付回调接口 | 6 |
| 外部集成接口 | 3 |
| 模型网关接口 | 32 |
| **合计** | **522** |

## 管理员接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| GET | `/api/v1/admin/accounts` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:323` |
| POST | `/api/v1/admin/accounts` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:326` |
| DELETE | `/api/v1/admin/accounts/:id` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:332` |
| GET | `/api/v1/admin/accounts/:id` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:324` |
| PUT | `/api/v1/admin/accounts/:id` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:331` |
| POST | `/api/v1/admin/accounts/:id/apply-oauth-credentials` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:336` |
| POST | `/api/v1/admin/accounts/:id/clear-error` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:340` |
| POST | `/api/v1/admin/accounts/:id/clear-rate-limit` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:345` |
| GET | `/api/v1/admin/accounts/:id/credentials` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:325` |
| GET | `/api/v1/admin/accounts/:id/models` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:351` |
| POST | `/api/v1/admin/accounts/:id/models/sync-upstream` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:352` |
| POST | `/api/v1/admin/accounts/:id/recover-state` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:334` |
| POST | `/api/v1/admin/accounts/:id/refresh` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:335` |
| POST | `/api/v1/admin/accounts/:id/refresh-tier` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:338` |
| POST | `/api/v1/admin/accounts/:id/reset-quota` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:346` |
| POST | `/api/v1/admin/accounts/:id/revert-proxy-fallback` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:341` |
| POST | `/api/v1/admin/accounts/:id/schedulable` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:349` |
| GET | `/api/v1/admin/accounts/:id/scheduled-test-plans` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:622` |
| POST | `/api/v1/admin/accounts/:id/set-privacy` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:337` |
| GET | `/api/v1/admin/accounts/:id/stats` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:339` |
| DELETE | `/api/v1/admin/accounts/:id/temp-unschedulable` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:348` |
| GET | `/api/v1/admin/accounts/:id/temp-unschedulable` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:347` |
| POST | `/api/v1/admin/accounts/:id/test` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:333` |
| GET | `/api/v1/admin/accounts/:id/today-stats` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:343` |
| GET | `/api/v1/admin/accounts/:id/usage` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:342` |
| GET | `/api/v1/admin/accounts/antigravity/default-model-mapping` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:363` |
| POST | `/api/v1/admin/accounts/batch` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:353` |
| POST | `/api/v1/admin/accounts/batch-clear-error` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:359` |
| POST | `/api/v1/admin/accounts/batch-refresh` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:360` |
| POST | `/api/v1/admin/accounts/batch-refresh-tier` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:357` |
| POST | `/api/v1/admin/accounts/batch-update-credentials` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:356` |
| POST | `/api/v1/admin/accounts/bulk-update` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:358` |
| POST | `/api/v1/admin/accounts/check-mixed-channel` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:327` |
| POST | `/api/v1/admin/accounts/cookie-auth` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:370` |
| GET | `/api/v1/admin/accounts/data` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:354` |
| POST | `/api/v1/admin/accounts/data` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:355` |
| POST | `/api/v1/admin/accounts/exchange-code` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:368` |
| POST | `/api/v1/admin/accounts/exchange-setup-token-code` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:369` |
| POST | `/api/v1/admin/accounts/generate-auth-url` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:366` |
| POST | `/api/v1/admin/accounts/generate-setup-token-url` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:367` |
| POST | `/api/v1/admin/accounts/import/codex-session` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:328` |
| POST | `/api/v1/admin/accounts/models/sync-upstream-preview` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:350` |
| POST | `/api/v1/admin/accounts/setup-token-cookie-auth` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:371` |
| POST | `/api/v1/admin/accounts/sync/crs` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:329` |
| POST | `/api/v1/admin/accounts/sync/crs/preview` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:330` |
| POST | `/api/v1/admin/accounts/today-stats/batch` | 管理员（当前 AdminAuth） | `/admin/accounts` | `admin.go:344` |
| GET | `/api/v1/admin/affiliates/invites` | 管理员（当前 AdminAuth） | `/admin/affiliates/invites` | `admin.go:688` |
| GET | `/api/v1/admin/affiliates/rebates` | 管理员（当前 AdminAuth） | `/admin/affiliates/rebates` | `admin.go:689` |
| GET | `/api/v1/admin/affiliates/transfers` | 管理员（当前 AdminAuth） | `/admin/affiliates/transfers` | `admin.go:690` |
| GET | `/api/v1/admin/affiliates/users` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:694` |
| DELETE | `/api/v1/admin/affiliates/users/:user_id` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:699` |
| PUT | `/api/v1/admin/affiliates/users/:user_id` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:698` |
| GET | `/api/v1/admin/affiliates/users/:user_id/overview` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:697` |
| POST | `/api/v1/admin/affiliates/users/batch-rate` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:696` |
| GET | `/api/v1/admin/affiliates/users/lookup` | 管理员（当前 AdminAuth） | 管理员返利页面 | `admin.go:695` |
| GET | `/api/v1/admin/announcements` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:378` |
| POST | `/api/v1/admin/announcements` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:379` |
| DELETE | `/api/v1/admin/announcements/:id` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:382` |
| GET | `/api/v1/admin/announcements/:id` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:380` |
| PUT | `/api/v1/admin/announcements/:id` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:381` |
| GET | `/api/v1/admin/announcements/:id/read-status` | 管理员（当前 AdminAuth） | `/admin/announcements` | `admin.go:383` |
| POST | `/api/v1/admin/antigravity/oauth/auth-url` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:412` |
| POST | `/api/v1/admin/antigravity/oauth/exchange-code` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:413` |
| POST | `/api/v1/admin/antigravity/oauth/refresh-token` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:414` |
| PUT | `/api/v1/admin/api-keys/:id` | 管理员（当前 AdminAuth） | `/admin/users`（API Key 弹窗） | `admin.go:137` |
| GET | `/api/v1/admin/backups` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:547` |
| POST | `/api/v1/admin/backups` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:546` |
| DELETE | `/api/v1/admin/backups/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:549` |
| GET | `/api/v1/admin/backups/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:548` |
| GET | `/api/v1/admin/backups/:id/download-url` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:550` |
| POST | `/api/v1/admin/backups/:id/restore` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:553` |
| GET | `/api/v1/admin/backups/s3-config` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:537` |
| PUT | `/api/v1/admin/backups/s3-config` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:538` |
| POST | `/api/v1/admin/backups/s3-config/test` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:539` |
| GET | `/api/v1/admin/backups/schedule` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:542` |
| PUT | `/api/v1/admin/backups/schedule` | 管理员（当前 AdminAuth） | `/admin/settings`（备份配置/操作） | `admin.go:543` |
| GET | `/api/v1/admin/channel-monitor-templates` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:674` |
| POST | `/api/v1/admin/channel-monitor-templates` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:675` |
| DELETE | `/api/v1/admin/channel-monitor-templates/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:678` |
| GET | `/api/v1/admin/channel-monitor-templates/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:676` |
| PUT | `/api/v1/admin/channel-monitor-templates/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:677` |
| POST | `/api/v1/admin/channel-monitor-templates/:id/apply` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:680` |
| GET | `/api/v1/admin/channel-monitor-templates/:id/monitors` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:679` |
| GET | `/api/v1/admin/channel-monitors` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:663` |
| POST | `/api/v1/admin/channel-monitors` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:664` |
| DELETE | `/api/v1/admin/channel-monitors/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:667` |
| GET | `/api/v1/admin/channel-monitors/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:665` |
| PUT | `/api/v1/admin/channel-monitors/:id` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:666` |
| GET | `/api/v1/admin/channel-monitors/:id/history` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:669` |
| POST | `/api/v1/admin/channel-monitors/:id/run` | 管理员（当前 AdminAuth） | `/admin/channels/monitor` | `admin.go:668` |
| GET | `/api/v1/admin/channels` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:650` |
| POST | `/api/v1/admin/channels` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:654` |
| DELETE | `/api/v1/admin/channels/:id` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:656` |
| GET | `/api/v1/admin/channels/:id` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:653` |
| PUT | `/api/v1/admin/channels/:id` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:655` |
| GET | `/api/v1/admin/channels/model-pricing` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:651` |
| GET | `/api/v1/admin/channels/pricing/sync-models` | 管理员（当前 AdminAuth） | `/admin/channels/pricing` | `admin.go:652` |
| GET | `/api/v1/admin/compliance` | 管理员（当前 AdminAuth） | 全局管理端合规弹窗 | `admin.go:115` |
| POST | `/api/v1/admin/compliance/accept` | 管理员（当前 AdminAuth） | 全局管理端合规弹窗 | `admin.go:116` |
| POST | `/api/v1/admin/dashboard/aggregation/backfill` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:264` |
| GET | `/api/v1/admin/dashboard/api-keys-trend` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:258` |
| POST | `/api/v1/admin/dashboard/api-keys-usage` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:262` |
| GET | `/api/v1/admin/dashboard/groups` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:257` |
| GET | `/api/v1/admin/dashboard/models` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:256` |
| GET | `/api/v1/admin/dashboard/realtime` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:254` |
| GET | `/api/v1/admin/dashboard/snapshot-v2` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:252` |
| GET | `/api/v1/admin/dashboard/stats` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:253` |
| GET | `/api/v1/admin/dashboard/trend` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:255` |
| GET | `/api/v1/admin/dashboard/user-breakdown` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:263` |
| GET | `/api/v1/admin/dashboard/users-ranking` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:260` |
| GET | `/api/v1/admin/dashboard/users-trend` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:259` |
| POST | `/api/v1/admin/dashboard/users-usage` | 管理员（当前 AdminAuth） | `/admin/dashboard` | `admin.go:261` |
| GET | `/api/v1/admin/data-management/agent/health` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:513` |
| GET | `/api/v1/admin/data-management/backups` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:528` |
| POST | `/api/v1/admin/data-management/backups` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:527` |
| GET | `/api/v1/admin/data-management/backups/:job_id` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:529` |
| GET | `/api/v1/admin/data-management/config` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:514` |
| PUT | `/api/v1/admin/data-management/config` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:515` |
| GET | `/api/v1/admin/data-management/s3/profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:522` |
| POST | `/api/v1/admin/data-management/s3/profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:523` |
| DELETE | `/api/v1/admin/data-management/s3/profiles/:profile_id` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:525` |
| PUT | `/api/v1/admin/data-management/s3/profiles/:profile_id` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:524` |
| POST | `/api/v1/admin/data-management/s3/profiles/:profile_id/activate` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:526` |
| POST | `/api/v1/admin/data-management/s3/test` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:521` |
| GET | `/api/v1/admin/data-management/sources/:source_type/profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:516` |
| POST | `/api/v1/admin/data-management/sources/:source_type/profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:517` |
| DELETE | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:519` |
| PUT | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:518` |
| POST | `/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id/activate` | 管理员（当前 AdminAuth） | `/admin/settings`（数据管理配置） | `admin.go:520` |
| GET | `/api/v1/admin/default-group` | 管理员（当前 AdminAuth） | `/admin/default-group-routing` | `admin.go:467` |
| GET | `/api/v1/admin/error-passthrough-rules` | 管理员（当前 AdminAuth） | `/admin/settings`（错误透传配置） | `admin.go:628` |
| POST | `/api/v1/admin/error-passthrough-rules` | 管理员（当前 AdminAuth） | `/admin/settings`（错误透传配置） | `admin.go:630` |
| DELETE | `/api/v1/admin/error-passthrough-rules/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（错误透传配置） | `admin.go:632` |
| GET | `/api/v1/admin/error-passthrough-rules/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（错误透传配置） | `admin.go:629` |
| PUT | `/api/v1/admin/error-passthrough-rules/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（错误透传配置） | `admin.go:631` |
| POST | `/api/v1/admin/gemini/oauth/auth-url` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:403` |
| GET | `/api/v1/admin/gemini/oauth/capabilities` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:405` |
| POST | `/api/v1/admin/gemini/oauth/exchange-code` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:404` |
| GET | `/api/v1/admin/groups` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:300` |
| POST | `/api/v1/admin/groups` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:307` |
| DELETE | `/api/v1/admin/groups/:id` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:309` |
| GET | `/api/v1/admin/groups/:id` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:306` |
| PUT | `/api/v1/admin/groups/:id` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:308` |
| GET | `/api/v1/admin/groups/:id/api-keys` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:316` |
| GET | `/api/v1/admin/groups/:id/models-list-candidates` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:305` |
| DELETE | `/api/v1/admin/groups/:id/rate-multipliers` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:313` |
| GET | `/api/v1/admin/groups/:id/rate-multipliers` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:311` |
| PUT | `/api/v1/admin/groups/:id/rate-multipliers` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:312` |
| DELETE | `/api/v1/admin/groups/:id/rpm-overrides` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:315` |
| PUT | `/api/v1/admin/groups/:id/rpm-overrides` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:314` |
| GET | `/api/v1/admin/groups/:id/stats` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:310` |
| GET | `/api/v1/admin/groups/:id/subscriptions` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:582` |
| GET | `/api/v1/admin/groups/all` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:301` |
| GET | `/api/v1/admin/groups/capacity-summary` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:303` |
| PUT | `/api/v1/admin/groups/sort-order` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:304` |
| GET | `/api/v1/admin/groups/usage-summary` | 管理员（当前 AdminAuth） | `/admin/groups`；部分用于 `/admin/default-group-routing` | `admin.go:302` |
| GET | `/api/v1/admin/model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/groups`（模型限额弹窗） | `admin.go:144` |
| PUT | `/api/v1/admin/model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/groups`（模型限额弹窗） | `admin.go:145` |
| GET | `/api/v1/admin/openai/accounts/:id/quota` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:395` |
| POST | `/api/v1/admin/openai/accounts/:id/refresh` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:393` |
| POST | `/api/v1/admin/openai/accounts/:id/reset-quota` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:396` |
| POST | `/api/v1/admin/openai/create-from-oauth` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:394` |
| POST | `/api/v1/admin/openai/exchange-code` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:391` |
| POST | `/api/v1/admin/openai/generate-auth-url` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:390` |
| POST | `/api/v1/admin/openai/refresh-token` | 管理员（当前 AdminAuth） | `/admin/accounts`（OAuth 操作） | `admin.go:392` |
| GET | `/api/v1/admin/ops/account-availability` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:170` |
| GET | `/api/v1/admin/ops/advanced-settings` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:198` |
| PUT | `/api/v1/admin/ops/advanced-settings` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:199` |
| GET | `/api/v1/admin/ops/alert-events` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:178` |
| GET | `/api/v1/admin/ops/alert-events/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:179` |
| PUT | `/api/v1/admin/ops/alert-events/:id/status` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:180` |
| GET | `/api/v1/admin/ops/alert-rules` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:174` |
| POST | `/api/v1/admin/ops/alert-rules` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:175` |
| DELETE | `/api/v1/admin/ops/alert-rules/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:177` |
| PUT | `/api/v1/admin/ops/alert-rules/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:176` |
| POST | `/api/v1/admin/ops/alert-silences` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:181` |
| GET | `/api/v1/admin/ops/concurrency` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:168` |
| GET | `/api/v1/admin/ops/dashboard/error-distribution` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:244` |
| GET | `/api/v1/admin/ops/dashboard/error-trend` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:243` |
| GET | `/api/v1/admin/ops/dashboard/latency-histogram` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:242` |
| GET | `/api/v1/admin/ops/dashboard/openai-token-stats` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:245` |
| GET | `/api/v1/admin/ops/dashboard/overview` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:240` |
| GET | `/api/v1/admin/ops/dashboard/snapshot-v2` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:239` |
| GET | `/api/v1/admin/ops/dashboard/throughput-trend` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:241` |
| GET | `/api/v1/admin/ops/email-notification/config` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:184` |
| PUT | `/api/v1/admin/ops/email-notification/config` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:185` |
| GET | `/api/v1/admin/ops/errors` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:215` |
| GET | `/api/v1/admin/ops/errors/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:216` |
| PUT | `/api/v1/admin/ops/errors/:id/resolve` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:217` |
| GET | `/api/v1/admin/ops/realtime-traffic` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:171` |
| GET | `/api/v1/admin/ops/request-errors` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:220` |
| GET | `/api/v1/admin/ops/request-errors/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:221` |
| PUT | `/api/v1/admin/ops/request-errors/:id/resolve` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:223` |
| GET | `/api/v1/admin/ops/request-errors/:id/upstream-errors` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:222` |
| GET | `/api/v1/admin/ops/requests` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:231` |
| GET | `/api/v1/admin/ops/runtime/alert` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:190` |
| PUT | `/api/v1/admin/ops/runtime/alert` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:191` |
| GET | `/api/v1/admin/ops/runtime/logging` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:192` |
| PUT | `/api/v1/admin/ops/runtime/logging` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:193` |
| POST | `/api/v1/admin/ops/runtime/logging/reset` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:194` |
| GET | `/api/v1/admin/ops/settings/metric-thresholds` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:204` |
| PUT | `/api/v1/admin/ops/settings/metric-thresholds` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:205` |
| GET | `/api/v1/admin/ops/system-logs` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:234` |
| POST | `/api/v1/admin/ops/system-logs/cleanup` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:235` |
| GET | `/api/v1/admin/ops/system-logs/health` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:236` |
| GET | `/api/v1/admin/ops/upstream-errors` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:226` |
| GET | `/api/v1/admin/ops/upstream-errors/:id` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:227` |
| PUT | `/api/v1/admin/ops/upstream-errors/:id/resolve` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:228` |
| GET | `/api/v1/admin/ops/user-concurrency` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:169` |
| GET | `/api/v1/admin/ops/ws/qps` | 管理员（当前 AdminAuth） | `/admin/ops` | `admin.go:211` |
| GET | `/api/v1/admin/payment/config` | 管理员（当前 AdminAuth） | `/admin/orders/dashboard` 或支付管理配置 | `payment.go:77` |
| PUT | `/api/v1/admin/payment/config` | 管理员（当前 AdminAuth） | `/admin/orders/dashboard` 或支付管理配置 | `payment.go:78` |
| GET | `/api/v1/admin/payment/dashboard` | 管理员（当前 AdminAuth） | `/admin/orders/dashboard` | `payment.go:74` |
| GET | `/api/v1/admin/payment/orders` | 管理员（当前 AdminAuth） | `/admin/orders` | `payment.go:83` |
| GET | `/api/v1/admin/payment/orders/:id` | 管理员（当前 AdminAuth） | `/admin/orders` | `payment.go:84` |
| POST | `/api/v1/admin/payment/orders/:id/cancel` | 管理员（当前 AdminAuth） | `/admin/orders` | `payment.go:85` |
| POST | `/api/v1/admin/payment/orders/:id/refund` | 管理员（当前 AdminAuth） | `/admin/orders` | `payment.go:87` |
| POST | `/api/v1/admin/payment/orders/:id/retry` | 管理员（当前 AdminAuth） | `/admin/orders` | `payment.go:86` |
| GET | `/api/v1/admin/payment/plans` | 管理员（当前 AdminAuth） | `/admin/orders/plans` | `payment.go:93` |
| POST | `/api/v1/admin/payment/plans` | 管理员（当前 AdminAuth） | `/admin/orders/plans` | `payment.go:94` |
| DELETE | `/api/v1/admin/payment/plans/:id` | 管理员（当前 AdminAuth） | `/admin/orders/plans` | `payment.go:96` |
| PUT | `/api/v1/admin/payment/plans/:id` | 管理员（当前 AdminAuth） | `/admin/orders/plans` | `payment.go:95` |
| GET | `/api/v1/admin/payment/providers` | 管理员（当前 AdminAuth） | `/admin/orders/plans`（服务商管理） | `payment.go:102` |
| POST | `/api/v1/admin/payment/providers` | 管理员（当前 AdminAuth） | `/admin/orders/plans`（服务商管理） | `payment.go:103` |
| DELETE | `/api/v1/admin/payment/providers/:id` | 管理员（当前 AdminAuth） | `/admin/orders/plans`（服务商管理） | `payment.go:105` |
| PUT | `/api/v1/admin/payment/providers/:id` | 管理员（当前 AdminAuth） | `/admin/orders/plans`（服务商管理） | `payment.go:104` |
| GET | `/api/v1/admin/promo-codes` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:457` |
| POST | `/api/v1/admin/promo-codes` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:459` |
| DELETE | `/api/v1/admin/promo-codes/:id` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:461` |
| GET | `/api/v1/admin/promo-codes/:id` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:458` |
| PUT | `/api/v1/admin/promo-codes/:id` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:460` |
| GET | `/api/v1/admin/promo-codes/:id/usages` | 管理员（当前 AdminAuth） | `/admin/promo-codes` | `admin.go:462` |
| GET | `/api/v1/admin/proxies` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:421` |
| POST | `/api/v1/admin/proxies` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:426` |
| DELETE | `/api/v1/admin/proxies/:id` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:428` |
| GET | `/api/v1/admin/proxies/:id` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:425` |
| PUT | `/api/v1/admin/proxies/:id` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:427` |
| GET | `/api/v1/admin/proxies/:id/accounts` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:432` |
| POST | `/api/v1/admin/proxies/:id/quality-check` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:430` |
| GET | `/api/v1/admin/proxies/:id/stats` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:431` |
| POST | `/api/v1/admin/proxies/:id/test` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:429` |
| GET | `/api/v1/admin/proxies/all` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:422` |
| POST | `/api/v1/admin/proxies/batch` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:434` |
| POST | `/api/v1/admin/proxies/batch-delete` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:433` |
| GET | `/api/v1/admin/proxies/data` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:423` |
| POST | `/api/v1/admin/proxies/data` | 管理员（当前 AdminAuth） | `/admin/proxies` | `admin.go:424` |
| GET | `/api/v1/admin/redeem-codes` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:441` |
| DELETE | `/api/v1/admin/redeem-codes/:id` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:447` |
| GET | `/api/v1/admin/redeem-codes/:id` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:444` |
| POST | `/api/v1/admin/redeem-codes/:id/expire` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:450` |
| POST | `/api/v1/admin/redeem-codes/batch-delete` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:448` |
| POST | `/api/v1/admin/redeem-codes/batch-update` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:449` |
| POST | `/api/v1/admin/redeem-codes/create-and-redeem` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:445` |
| GET | `/api/v1/admin/redeem-codes/export` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:443` |
| POST | `/api/v1/admin/redeem-codes/generate` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:446` |
| GET | `/api/v1/admin/redeem-codes/stats` | 管理员（当前 AdminAuth） | `/admin/redeem` | `admin.go:442` |
| POST | `/api/v1/admin/risk-control/api-keys/test` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:125` |
| GET | `/api/v1/admin/risk-control/config` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:123` |
| PUT | `/api/v1/admin/risk-control/config` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:124` |
| DELETE | `/api/v1/admin/risk-control/hashes` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:129` |
| DELETE | `/api/v1/admin/risk-control/hashes/all` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:130` |
| GET | `/api/v1/admin/risk-control/logs` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:127` |
| GET | `/api/v1/admin/risk-control/status` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:126` |
| POST | `/api/v1/admin/risk-control/users/:user_id/unban` | 管理员（当前 AdminAuth） | `/admin/risk-control` | `admin.go:128` |
| POST | `/api/v1/admin/scheduled-test-plans` | 管理员（当前 AdminAuth） | `/admin/accounts`（定时测试） | `admin.go:616` |
| DELETE | `/api/v1/admin/scheduled-test-plans/:id` | 管理员（当前 AdminAuth） | `/admin/accounts`（定时测试） | `admin.go:618` |
| PUT | `/api/v1/admin/scheduled-test-plans/:id` | 管理员（当前 AdminAuth） | `/admin/accounts`（定时测试） | `admin.go:617` |
| GET | `/api/v1/admin/scheduled-test-plans/:id/results` | 管理员（当前 AdminAuth） | `/admin/accounts`（定时测试） | `admin.go:619` |
| GET | `/api/v1/admin/settings` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:470` |
| PUT | `/api/v1/admin/settings` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:471` |
| DELETE | `/api/v1/admin/settings/admin-api-key` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:483` |
| GET | `/api/v1/admin/settings/admin-api-key` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:481` |
| POST | `/api/v1/admin/settings/admin-api-key/regenerate` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:482` |
| GET | `/api/v1/admin/settings/beta-policy` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:497` |
| PUT | `/api/v1/admin/settings/beta-policy` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:498` |
| PUT | `/api/v1/admin/settings/default-group` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:472` |
| GET | `/api/v1/admin/settings/default-model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:505` |
| PUT | `/api/v1/admin/settings/default-model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:506` |
| POST | `/api/v1/admin/settings/email-template-preview` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:476` |
| GET | `/api/v1/admin/settings/email-templates` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:475` |
| GET | `/api/v1/admin/settings/email-templates/:event/:locale` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:477` |
| PUT | `/api/v1/admin/settings/email-templates/:event/:locale` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:478` |
| POST | `/api/v1/admin/settings/email-templates/:event/:locale/restore-official` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:479` |
| GET | `/api/v1/admin/settings/overload-cooldown` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:485` |
| PUT | `/api/v1/admin/settings/overload-cooldown` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:486` |
| GET | `/api/v1/admin/settings/rate-limit-429-cooldown` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:488` |
| PUT | `/api/v1/admin/settings/rate-limit-429-cooldown` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:489` |
| GET | `/api/v1/admin/settings/rectifier` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:494` |
| PUT | `/api/v1/admin/settings/rectifier` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:495` |
| POST | `/api/v1/admin/settings/send-test-email` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:474` |
| GET | `/api/v1/admin/settings/stream-timeout` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:491` |
| PUT | `/api/v1/admin/settings/stream-timeout` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:492` |
| POST | `/api/v1/admin/settings/test-smtp` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:473` |
| GET | `/api/v1/admin/settings/web-search-emulation` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:500` |
| PUT | `/api/v1/admin/settings/web-search-emulation` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:501` |
| POST | `/api/v1/admin/settings/web-search-emulation/reset-usage` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:503` |
| POST | `/api/v1/admin/settings/web-search-emulation/test` | 管理员（当前 AdminAuth） | `/admin/settings` | `admin.go:502` |
| GET | `/api/v1/admin/subscriptions` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:571` |
| DELETE | `/api/v1/admin/subscriptions/:id` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:578` |
| GET | `/api/v1/admin/subscriptions/:id` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:572` |
| POST | `/api/v1/admin/subscriptions/:id/extend` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:576` |
| GET | `/api/v1/admin/subscriptions/:id/progress` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:573` |
| POST | `/api/v1/admin/subscriptions/:id/reset-quota` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:577` |
| POST | `/api/v1/admin/subscriptions/assign` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:574` |
| POST | `/api/v1/admin/subscriptions/bulk-assign` | 管理员（当前 AdminAuth） | `/admin/subscriptions` | `admin.go:575` |
| GET | `/api/v1/admin/system/check-updates` | 管理员（当前 AdminAuth） | 全局版本组件；`/admin/settings` | `admin.go:561` |
| POST | `/api/v1/admin/system/restart` | 管理员（当前 AdminAuth） | 全局版本组件；`/admin/settings` | `admin.go:564` |
| POST | `/api/v1/admin/system/rollback` | 管理员（当前 AdminAuth） | 全局版本组件；`/admin/settings` | `admin.go:563` |
| POST | `/api/v1/admin/system/update` | 管理员（当前 AdminAuth） | 全局版本组件；`/admin/settings` | `admin.go:562` |
| GET | `/api/v1/admin/system/version` | 管理员（当前 AdminAuth） | 全局版本组件；`/admin/settings` | `admin.go:560` |
| GET | `/api/v1/admin/tls-fingerprint-profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（TLS 指纹配置） | `admin.go:639` |
| POST | `/api/v1/admin/tls-fingerprint-profiles` | 管理员（当前 AdminAuth） | `/admin/settings`（TLS 指纹配置） | `admin.go:641` |
| DELETE | `/api/v1/admin/tls-fingerprint-profiles/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（TLS 指纹配置） | `admin.go:643` |
| GET | `/api/v1/admin/tls-fingerprint-profiles/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（TLS 指纹配置） | `admin.go:640` |
| PUT | `/api/v1/admin/tls-fingerprint-profiles/:id` | 管理员（当前 AdminAuth） | `/admin/settings`（TLS 指纹配置） | `admin.go:642` |
| GET | `/api/v1/admin/token-usage/default-target` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:159` |
| GET | `/api/v1/admin/token-usage/models` | 管理员（当前 AdminAuth） | `/admin/token-usage/models` | `admin.go:150` |
| GET | `/api/v1/admin/token-usage/options/groups` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:154` |
| GET | `/api/v1/admin/token-usage/options/groups/:group_id/routes` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:155` |
| GET | `/api/v1/admin/token-usage/options/groups/:group_id/routes/:route_alias/models` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:156` |
| GET | `/api/v1/admin/token-usage/options/models` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:153` |
| GET | `/api/v1/admin/token-usage/options/users` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:157` |
| GET | `/api/v1/admin/token-usage/options/users/:user_id/models` | 管理员（当前 AdminAuth） | Token 统计页面的筛选器/公共调用 | `admin.go:158` |
| GET | `/api/v1/admin/token-usage/routes` | 管理员（当前 AdminAuth） | `/admin/token-usage/routes` | `admin.go:151` |
| GET | `/api/v1/admin/token-usage/user-group-model-daily` | 管理员（当前 AdminAuth） | `/admin/token-usage/user-group-model-daily` | `admin.go:160` |
| GET | `/api/v1/admin/token-usage/users` | 管理员（当前 AdminAuth） | `/admin/token-usage/users` | `admin.go:152` |
| GET | `/api/v1/admin/usage` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:591` |
| GET | `/api/v1/admin/usage/cleanup-tasks` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:595` |
| POST | `/api/v1/admin/usage/cleanup-tasks` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:596` |
| POST | `/api/v1/admin/usage/cleanup-tasks/:id/cancel` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:597` |
| GET | `/api/v1/admin/usage/search-api-keys` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:594` |
| GET | `/api/v1/admin/usage/search-users` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:593` |
| GET | `/api/v1/admin/usage/stats` | 管理员（当前 AdminAuth） | `/admin/usage` | `admin.go:592` |
| GET | `/api/v1/admin/user-attributes` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:604` |
| POST | `/api/v1/admin/user-attributes` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:605` |
| DELETE | `/api/v1/admin/user-attributes/:id` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:609` |
| PUT | `/api/v1/admin/user-attributes/:id` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:608` |
| POST | `/api/v1/admin/user-attributes/batch` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:606` |
| PUT | `/api/v1/admin/user-attributes/reorder` | 管理员（当前 AdminAuth） | `/admin/users`（用户属性配置） | `admin.go:607` |
| GET | `/api/v1/admin/users` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:271` |
| POST | `/api/v1/admin/users` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:274` |
| DELETE | `/api/v1/admin/users/:id` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:276` |
| GET | `/api/v1/admin/users/:id` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:272` |
| PUT | `/api/v1/admin/users/:id` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:275` |
| GET | `/api/v1/admin/users/:id/api-keys` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:278` |
| GET | `/api/v1/admin/users/:id/attributes` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:292` |
| PUT | `/api/v1/admin/users/:id/attributes` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:293` |
| POST | `/api/v1/admin/users/:id/auth-identities` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:273` |
| POST | `/api/v1/admin/users/:id/balance` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:277` |
| GET | `/api/v1/admin/users/:id/balance-history` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:280` |
| GET | `/api/v1/admin/users/:id/model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:287` |
| PUT | `/api/v1/admin/users/:id/model-token-quotas` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:288` |
| GET | `/api/v1/admin/users/:id/platform-quotas` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:284` |
| PUT | `/api/v1/admin/users/:id/platform-quotas` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:285` |
| POST | `/api/v1/admin/users/:id/platform-quotas/reset` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:286` |
| POST | `/api/v1/admin/users/:id/replace-group` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:281` |
| GET | `/api/v1/admin/users/:id/rpm-status` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:282` |
| GET | `/api/v1/admin/users/:id/subscriptions` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:585` |
| GET | `/api/v1/admin/users/:id/usage` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:279` |
| POST | `/api/v1/admin/users/batch-concurrency` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:283` |
| POST | `/api/v1/admin/users/model-token-quotas/batch` | 管理员（当前 AdminAuth） | `/admin/users` | `admin.go:289` |
| GET | `/api/v1/pages` | 管理员（当前 AdminAuth） | 管理端自定义菜单配置 | `handler/page_handler.go:282` |

## 用户个人接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| GET | `/api/v1/auth/me` | 登录用户本人（JWT） | 登录态初始化；所有登录后页面 | `auth.go:225` |
| POST | `/api/v1/auth/oauth/bind-token` | 登录用户本人（JWT） | `/profile`（第三方账号绑定） | `auth.go:228` |
| POST | `/api/v1/auth/revoke-all-sessions` | 登录用户本人（JWT） | `/profile` | `auth.go:227` |
| GET | `/api/v1/payment/channels` | 登录用户本人（JWT，订单归属校验） | `/purchase` 及各支付流程页面 | `payment.go:31` |
| GET | `/api/v1/payment/checkout-info` | 登录用户本人（JWT，订单归属校验） | `/purchase` 及各支付流程页面 | `payment.go:29` |
| GET | `/api/v1/payment/config` | 登录用户本人（JWT，订单归属校验） | `/purchase` 及各支付流程页面 | `payment.go:28` |
| GET | `/api/v1/payment/limits` | 登录用户本人（JWT，订单归属校验） | `/purchase` 及各支付流程页面 | `payment.go:32` |
| POST | `/api/v1/payment/orders` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:36` |
| GET | `/api/v1/payment/orders/:id` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:39` |
| POST | `/api/v1/payment/orders/:id/cancel` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:40` |
| POST | `/api/v1/payment/orders/:id/refund-request` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:41` |
| GET | `/api/v1/payment/orders/my` | 登录用户本人（JWT，订单归属校验） | `/orders` | `payment.go:38` |
| GET | `/api/v1/payment/orders/refund-eligible-providers` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:42` |
| POST | `/api/v1/payment/orders/verify` | 登录用户本人（JWT，订单归属校验） | `/purchase`、支付结果页、`/orders` | `payment.go:37` |
| GET | `/api/v1/payment/plans` | 登录用户本人（JWT，订单归属校验） | `/purchase` 及各支付流程页面 | `payment.go:30` |
| GET | `/api/v1/announcements` | 登录用户本人（JWT，资源需按 user_id 隔离） | 全局布局中的公告组件 | `user.go:100` |
| POST | `/api/v1/announcements/:id/read` | 登录用户本人（JWT，资源需按 user_id 隔离） | 全局布局中的公告组件 | `user.go:101` |
| GET | `/api/v1/channel-monitors` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/monitor` | `user.go:123` |
| GET | `/api/v1/channel-monitors/:id/status` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/monitor` | `user.go:124` |
| GET | `/api/v1/channels/available` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/available-channels` | `user.go:78` |
| GET | `/api/v1/groups/available` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys`、`/usage/group-model-daily` 等筛选器 | `user.go:71` |
| GET | `/api/v1/groups/rates` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys`、`/usage/group-model-daily` 等筛选器 | `user.go:72` |
| GET | `/api/v1/keys` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys` | `user.go:61` |
| POST | `/api/v1/keys` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys` | `user.go:63` |
| DELETE | `/api/v1/keys/:id` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys` | `user.go:65` |
| GET | `/api/v1/keys/:id` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys` | `user.go:62` |
| PUT | `/api/v1/keys/:id` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/keys` | `user.go:64` |
| POST | `/api/v1/redeem` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/redeem` | `user.go:107` |
| GET | `/api/v1/redeem/history` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/redeem` | `user.go:108` |
| GET | `/api/v1/subscriptions` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/subscriptions`；摘要也用于 `/dashboard` | `user.go:114` |
| GET | `/api/v1/subscriptions/active` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/subscriptions`；摘要也用于 `/dashboard` | `user.go:115` |
| GET | `/api/v1/subscriptions/progress` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/subscriptions`；摘要也用于 `/dashboard` | `user.go:116` |
| GET | `/api/v1/subscriptions/summary` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/subscriptions`；摘要也用于 `/dashboard` | `user.go:117` |
| GET | `/api/v1/usage` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage`；详情也用于 `/key-usage` | `user.go:85` |
| GET | `/api/v1/usage/:id` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage`；详情也用于 `/key-usage` | `user.go:88` |
| POST | `/api/v1/usage/dashboard/api-keys-usage` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/dashboard` | `user.go:94` |
| GET | `/api/v1/usage/dashboard/models` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/dashboard` | `user.go:93` |
| GET | `/api/v1/usage/dashboard/stats` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/dashboard` | `user.go:91` |
| GET | `/api/v1/usage/dashboard/trend` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/dashboard` | `user.go:92` |
| GET | `/api/v1/usage/errors` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage`；详情也用于 `/key-usage` | `user.go:86` |
| GET | `/api/v1/usage/errors/:id` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage`；详情也用于 `/key-usage` | `user.go:87` |
| GET | `/api/v1/usage/group-model-daily` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage/group-model-daily` | `user.go:84` |
| GET | `/api/v1/usage/stats` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/usage`；详情也用于 `/key-usage` | `user.go:89` |
| PUT | `/api/v1/user` | 登录用户本人（JWT，资源需按 user_id 隔离） | 未发现独立页面（可能由公共组件调用） | `user.go:27` |
| DELETE | `/api/v1/user/account-bindings/:provider` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:32` |
| POST | `/api/v1/user/account-bindings/email` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:31` |
| POST | `/api/v1/user/account-bindings/email/send-code` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:30` |
| GET | `/api/v1/user/aff` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/affiliate` | `user.go:28` |
| POST | `/api/v1/user/aff/transfer` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/affiliate` | `user.go:29` |
| GET | `/api/v1/user/api-keys/:id/usage/daily` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/key-usage` | `user.go:34` |
| POST | `/api/v1/user/auth-identities/bind/start` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:33` |
| DELETE | `/api/v1/user/notify-email` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:43` |
| POST | `/api/v1/user/notify-email/send-code` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:40` |
| PUT | `/api/v1/user/notify-email/toggle` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:42` |
| POST | `/api/v1/user/notify-email/verify` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:41` |
| PUT | `/api/v1/user/password` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:26` |
| GET | `/api/v1/user/platform-quotas` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/dashboard`（配额展示） | `user.go:35` |
| GET | `/api/v1/user/profile` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:25` |
| POST | `/api/v1/user/totp/disable` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:54` |
| POST | `/api/v1/user/totp/enable` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:53` |
| POST | `/api/v1/user/totp/send-code` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:51` |
| POST | `/api/v1/user/totp/setup` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:52` |
| GET | `/api/v1/user/totp/status` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:49` |
| GET | `/api/v1/user/totp/verification-method` | 登录用户本人（JWT，资源需按 user_id 隔离） | `/profile` | `user.go:50` |
| GET | `/api/v1/pages/:slug` | 登录用户（JWT；处理器再检查页面可见范围） | `/custom/:id` | `handler/page_handler.go:268` |

## 公开及登录流程接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| POST | `/api/event_logging/batch` | 公开 | 无页面（Claude Code 遥测兼容） | `common.go:17` |
| GET | `/api/v1/settings/email-unsubscribe` | 公开 | 邮件退订链接 | `auth.go:217` |
| GET | `/api/v1/settings/public` | 公开 | 首页、登录页及全局设置初始化 | `auth.go:216` |
| GET | `/health` | 公开 | 无页面（健康检查） | `common.go:12` |
| GET | `/setup/status` | 公开 | `/setup` 启动探测 | `common.go:23` |
| GET | `/api/v1/pages/:slug/images/*filename` | 公开 URL；处理器检查页面可见范围 | `/custom/:id` | `handler/page_handler.go:274` |
| POST | `/api/v1/auth/forgot-password` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:58` |
| POST | `/api/v1/auth/login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:34` |
| POST | `/api/v1/auth/login/2fa` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:37` |
| POST | `/api/v1/auth/logout` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:48` |
| POST | `/api/v1/auth/oauth/dingtalk/bind-login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:199` |
| GET | `/api/v1/auth/oauth/dingtalk/bind/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:186` |
| GET | `/api/v1/auth/oauth/dingtalk/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:192` |
| POST | `/api/v1/auth/oauth/dingtalk/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:193` |
| POST | `/api/v1/auth/oauth/dingtalk/create-account` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:205` |
| GET | `/api/v1/auth/oauth/dingtalk/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:185` |
| GET | `/api/v1/auth/oauth/github/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:67` |
| POST | `/api/v1/auth/oauth/github/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:68` |
| GET | `/api/v1/auth/oauth/github/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:66` |
| GET | `/api/v1/auth/oauth/google/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:75` |
| POST | `/api/v1/auth/oauth/google/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:76` |
| GET | `/api/v1/auth/oauth/google/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:74` |
| POST | `/api/v1/auth/oauth/linuxdo/bind-login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:129` |
| GET | `/api/v1/auth/oauth/linuxdo/bind/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:82` |
| GET | `/api/v1/auth/oauth/linuxdo/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:88` |
| POST | `/api/v1/auth/oauth/linuxdo/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:123` |
| POST | `/api/v1/auth/oauth/linuxdo/create-account` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:135` |
| GET | `/api/v1/auth/oauth/linuxdo/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:65` |
| POST | `/api/v1/auth/oauth/oidc/bind-login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:173` |
| GET | `/api/v1/auth/oauth/oidc/bind/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:160` |
| GET | `/api/v1/auth/oauth/oidc/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:166` |
| POST | `/api/v1/auth/oauth/oidc/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:167` |
| POST | `/api/v1/auth/oauth/oidc/create-account` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:179` |
| GET | `/api/v1/auth/oauth/oidc/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:159` |
| POST | `/api/v1/auth/oauth/pending/bind-login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:117` |
| POST | `/api/v1/auth/oauth/pending/create-account` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:111` |
| POST | `/api/v1/auth/oauth/pending/exchange` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:99` |
| POST | `/api/v1/auth/oauth/pending/send-verify-code` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:105` |
| POST | `/api/v1/auth/oauth/wechat/bind-login` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:147` |
| GET | `/api/v1/auth/oauth/wechat/bind/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:90` |
| GET | `/api/v1/auth/oauth/wechat/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:96` |
| POST | `/api/v1/auth/oauth/wechat/complete-registration` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:141` |
| POST | `/api/v1/auth/oauth/wechat/create-account` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:153` |
| GET | `/api/v1/auth/oauth/wechat/payment/callback` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:98` |
| GET | `/api/v1/auth/oauth/wechat/payment/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:97` |
| GET | `/api/v1/auth/oauth/wechat/start` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:89` |
| POST | `/api/v1/auth/refresh` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:44` |
| POST | `/api/v1/auth/register` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:31` |
| POST | `/api/v1/auth/reset-password` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:62` |
| POST | `/api/v1/auth/send-verify-code` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:40` |
| POST | `/api/v1/auth/validate-invitation-code` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:54` |
| POST | `/api/v1/auth/validate-promo-code` | 公开/登录流程（部分携带临时凭证） | 登录、注册、找回密码或 OAuth 回调页面 | `auth.go:50` |
| POST | `/api/v1/payment/public/orders/resolve` | 公开（订单恢复令牌/持久状态校验） | 支付结果恢复流程 | `payment.go:53` |
| POST | `/api/v1/payment/public/orders/verify` | 公开（订单恢复令牌/持久状态校验） | 支付结果恢复流程 | `payment.go:52` |

## 支付回调接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| POST | `/api/v1/payment/webhook/airwallex` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:65` |
| POST | `/api/v1/payment/webhook/alipay` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:62` |
| GET | `/api/v1/payment/webhook/easypay` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:60` |
| POST | `/api/v1/payment/webhook/easypay` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:61` |
| POST | `/api/v1/payment/webhook/stripe` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:64` |
| POST | `/api/v1/payment/webhook/wxpay` | 支付服务商回调（签名校验，不是用户 RBAC） | 无页面（服务商服务器回调） | `payment.go:63` |

## 外部集成接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| POST | `/api/v1/integrations/api-keys/getOrCreate` | 外部集成密钥（独立于用户 RBAC） | 无站内页面（外部系统调用） | `integrations.go:27` |
| POST | `/api/v1/integrations/model-routes/list` | 外部集成密钥（独立于用户 RBAC） | 无站内页面（外部系统调用） | `integrations.go:28` |
| POST | `/api/v1/integrations/token-usage/user-group-model/daily` | 外部集成密钥（独立于用户 RBAC） | 无站内页面（外部系统调用） | `integrations.go:30` |

## 模型网关接口

| 方法 | 接口 | 当前访问范围 | 对应前端页面/用途 | 源码 |
|---|---|---|---|---|
| GET | `/antigravity/models` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:215` |
| POST | `/antigravity/v1/messages` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:227` |
| POST | `/antigravity/v1/messages/count_tokens` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:228` |
| GET | `/antigravity/v1/models` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:229` |
| GET | `/antigravity/v1/usage` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:230` |
| GET | `/antigravity/v1beta/models` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:242` |
| POST | `/antigravity/v1beta/models/*modelAction` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:244` |
| GET | `/antigravity/v1beta/models/:model` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:243` |
| GET | `/backend-api/codex/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:164` |
| POST | `/backend-api/codex/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:162` |
| POST | `/backend-api/codex/responses/*subpath` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:163` |
| POST | `/chat/completions` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:167` |
| POST | `/embeddings` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:174` |
| POST | `/images/edits` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:200` |
| POST | `/images/generations` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:187` |
| GET | `/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:158` |
| POST | `/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:156` |
| POST | `/responses/*subpath` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:157` |
| POST | `/v1/chat/completions` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:85` |
| POST | `/v1/embeddings` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:92` |
| POST | `/v1/images/edits` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:118` |
| POST | `/v1/images/generations` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:105` |
| POST | `/v1/messages` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:44` |
| POST | `/v1/messages/count_tokens` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:52` |
| GET | `/v1/models` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:66` |
| GET | `/v1/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:83` |
| POST | `/v1/responses` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:69` |
| POST | `/v1/responses/*subpath` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:76` |
| GET | `/v1/usage` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:67` |
| GET | `/v1beta/models` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:142` |
| POST | `/v1beta/models/*modelAction` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:145` |
| GET | `/v1beta/models/:model` | API Key 调用方（分组/订阅约束，不是后台页面） | 无站内页面（Claude/OpenAI/Gemini 兼容 API） | `gateway.go:143` |

## 前端页面总览

### 用户侧

`/dashboard`、`/keys`、`/key-usage`、`/usage`、`/usage/group-model-daily`、`/redeem`、`/affiliate`、`/available-channels`、`/profile`、`/subscriptions`、`/purchase`、`/orders`、各支付结果页、`/monitor`、`/custom/:id`。

### 管理侧

`/admin/dashboard`、`/admin/ops`、`/admin/users`、`/admin/groups`、`/admin/default-group-routing`、`/admin/accounts`、`/admin/announcements`、`/admin/proxies`、`/admin/redeem`、`/admin/promo-codes`、`/admin/settings`、`/admin/risk-control`、`/admin/usage`、四个 Token 统计页面、三个返利页面、支付仪表盘/订单/套餐页面、渠道定价与监控页面。

## RBAC 落地时的直接用法

这份清单可作为权限登记底稿，但不建议一条 URL 对应一个权限。下一步应按业务动作合并，例如用户列表和用户详情共同使用 `users.read`，修改资料使用 `users.update`，调整余额单独使用 `users.balance.adjust`。模型网关、支付 webhook 和外部集成继续保留各自认证方式，不接后台角色权限。
