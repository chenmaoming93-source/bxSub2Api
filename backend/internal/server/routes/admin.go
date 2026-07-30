// Package routes provides HTTP route registration and handlers.
package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AdminRBACOptions struct {
	IdentityAuth middleware.AdminIdentityAuthMiddleware
	Registrar    *rbac.RouteRegistrar
	Registry     *rbac.Registry
}

func adminGET(routes *rbac.RouteRegistrar, group *gin.RouterGroup, path, permission string, handler gin.HandlerFunc) {
	if routes == nil {
		group.GET(path, handler)
		return
	}
	routes.GET(group, path, permission, handler)
}

func adminPOST(routes *rbac.RouteRegistrar, group *gin.RouterGroup, path, permission string, handler gin.HandlerFunc) {
	if routes == nil {
		group.POST(path, handler)
		return
	}
	routes.POST(group, path, permission, handler)
}

func adminPUT(routes *rbac.RouteRegistrar, group *gin.RouterGroup, path, permission string, handler gin.HandlerFunc) {
	if routes == nil {
		group.PUT(path, handler)
		return
	}
	routes.PUT(group, path, permission, handler)
}

func adminDELETE(routes *rbac.RouteRegistrar, group *gin.RouterGroup, path, permission string, handler gin.HandlerFunc) {
	if routes == nil {
		group.DELETE(path, handler)
		return
	}
	routes.DELETE(group, path, permission, handler)
}

// RegisterAdminRoutes 注册管理员路由
func RegisterAdminRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
	settingService *service.SettingService,
	rbacOptions ...AdminRBACOptions,
) {
	admin := v1.Group("/admin")
	var rbacRoutes *rbac.RouteRegistrar
	if len(rbacOptions) > 0 && rbacOptions[0].Registrar != nil && rbacOptions[0].Registry != nil {
		option := rbacOptions[0]
		rbacRoutes = option.Registrar
		admin.Use(gin.HandlerFunc(option.IdentityAuth))
		admin.Use(middleware.PrincipalFromAuthenticatedSubject())
		admin.Use(middleware.RequireLegacyAdminForUnregistered(option.Registry))
	} else {
		admin.Use(gin.HandlerFunc(adminAuth))
	}
	admin.Use(middleware.AdminComplianceGuard(settingService))
	{
		// 部署与运营合规确认
		registerAdminComplianceRoutes(admin, h, rbacRoutes)

		// 仪表盘
		registerDashboardRoutes(admin, h, rbacRoutes)

		// 用户管理
		registerUserManagementRoutes(admin, h, rbacRoutes)

		// 分组管理
		registerGroupRoutes(admin, h, rbacRoutes)

		// 账号管理
		registerAccountRoutes(admin, h, rbacRoutes)

		// 公告管理
		registerAnnouncementRoutes(admin, h, rbacRoutes)

		// OpenAI OAuth
		registerOpenAIOAuthRoutes(admin, h, rbacRoutes)

		// Gemini OAuth
		registerGeminiOAuthRoutes(admin, h, rbacRoutes)

		// Antigravity OAuth
		registerAntigravityOAuthRoutes(admin, h, rbacRoutes)

		// 代理管理
		registerProxyRoutes(admin, h, rbacRoutes)

		// 卡密管理
		registerRedeemCodeRoutes(admin, h, rbacRoutes)

		// 优惠码管理
		registerPromoCodeRoutes(admin, h, rbacRoutes)

		// 系统设置
		registerSettingsRoutes(admin, h, rbacRoutes)

		// 数据管理
		registerDataManagementRoutes(admin, h, rbacRoutes)

		// 数据库备份恢复
		registerBackupRoutes(admin, h, rbacRoutes)

		// 运维监控（Ops）
		registerOpsRoutes(admin, h, rbacRoutes)

		// 系统管理
		registerSystemRoutes(admin, h, rbacRoutes)

		// 订阅管理
		registerSubscriptionRoutes(admin, h, rbacRoutes)

		// 使用记录管理
		registerUsageRoutes(admin, h, rbacRoutes)

		// 用户属性管理
		registerUserAttributeRoutes(admin, h, rbacRoutes)

		// 错误透传规则管理
		registerErrorPassthroughRoutes(admin, h, rbacRoutes)

		// TLS 指纹模板管理
		registerTLSFingerprintProfileRoutes(admin, h, rbacRoutes)

		// API Key 管理
		registerAdminAPIKeyRoutes(admin, h, rbacRoutes)

		// 定时测试计划
		registerScheduledTestRoutes(admin, h, rbacRoutes)

		// 渠道管理
		registerChannelRoutes(admin, h, rbacRoutes)

		// 渠道监控
		registerChannelMonitorRoutes(admin, h, rbacRoutes)

		// 风控中心
		registerContentModerationRoutes(admin, h, rbacRoutes)

		// 邀请返利（专属用户管理）
		registerAffiliateRoutes(admin, h, rbacRoutes)

		// 全局模型 Token 配额
		registerModelTokenQuotaRoutes(admin, h, rbacRoutes)
		registerRBACManagementRoutes(admin, h, rbacRoutes)
	}
}

func registerRBACManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	group := admin.Group("/rbac")
	adminGET(routes, group, "/roles", rbac.PermissionRolesRead, h.Admin.RBAC.ListRoles)
	adminPOST(routes, group, "/roles", rbac.PermissionRolesCreate, h.Admin.RBAC.CreateRole)
	adminPUT(routes, group, "/roles/:id", rbac.PermissionRolesUpdate, h.Admin.RBAC.UpdateRole)
	adminDELETE(routes, group, "/roles/:id", rbac.PermissionRolesDelete, h.Admin.RBAC.DeleteRole)
	adminGET(routes, group, "/permissions", rbac.PermissionPermissionsRead, h.Admin.RBAC.ListPermissions)
	adminPOST(routes, group, "/permissions", rbac.PermissionPermissionsCreate, h.Admin.RBAC.CreatePermission)
	adminPUT(routes, group, "/permissions/:id", rbac.PermissionPermissionsUpdate, h.Admin.RBAC.UpdatePermission)
	adminDELETE(routes, group, "/permissions/:id", rbac.PermissionPermissionsDelete, h.Admin.RBAC.DeletePermission)
	adminGET(routes, group, "/roles/:id/permissions", rbac.PermissionRolesRead, h.Admin.RBAC.GetRolePermissions)
	adminPUT(routes, group, "/roles/:id/permissions", rbac.PermissionRolesPermissionsAssign, h.Admin.RBAC.ReplaceRolePermissions)
	adminGET(routes, admin, "/users/:id/roles", rbac.PermissionUsersRolesRead, h.Admin.RBAC.GetUserRoles)
	adminPUT(routes, admin, "/users/:id/roles", rbac.PermissionUsersRolesAssign, h.Admin.RBAC.ReplaceUserRoles)
}

func registerAdminComplianceRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	compliance := admin.Group("/compliance")
	{
		adminGET(routes, compliance, "", rbac.PermissionSettingsRead, h.Admin.Compliance.GetStatus)
		adminPOST(routes, compliance, "/accept", rbac.PermissionSettingsUpdate, h.Admin.Compliance.Accept)
	}
}

func registerContentModerationRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	risk := admin.Group("/risk-control")
	{
		adminGET(routes, risk, "/config", rbac.PermissionRiskRead, h.Admin.ContentModeration.GetConfig)
		adminPUT(routes, risk, "/config", rbac.PermissionRiskUpdate, h.Admin.ContentModeration.UpdateConfig)
		adminPOST(routes, risk, "/api-keys/test", rbac.PermissionRiskUpdate, h.Admin.ContentModeration.TestAPIKeys)
		adminGET(routes, risk, "/status", rbac.PermissionRiskRead, h.Admin.ContentModeration.GetStatus)
		adminGET(routes, risk, "/logs", rbac.PermissionRiskRead, h.Admin.ContentModeration.ListLogs)
		adminPOST(routes, risk, "/users/:user_id/unban", rbac.PermissionRiskOperate, h.Admin.ContentModeration.UnbanUser)
		adminDELETE(routes, risk, "/hashes", rbac.PermissionRiskOperate, h.Admin.ContentModeration.DeleteFlaggedHash)
		adminDELETE(routes, risk, "/hashes/all", rbac.PermissionRiskOperate, h.Admin.ContentModeration.ClearFlaggedHashes)
	}
}

func registerAdminAPIKeyRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	apiKeys := admin.Group("/api-keys")
	{
		adminPUT(routes, apiKeys, "/:id", rbac.PermissionUsersUpdate, h.Admin.APIKey.UpdateGroup)
	}
}

func registerModelTokenQuotaRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	quotas := admin.Group("/model-token-quotas")
	{
		adminGET(routes, quotas, "", rbac.PermissionTokenQuotaRead, h.Admin.ModelTokenQuota.List)
		adminPUT(routes, quotas, "", rbac.PermissionTokenQuotaUpdate, h.Admin.ModelTokenQuota.Update)
	}

	tokenUsage := admin.Group("/token-usage")
	{
		adminGET(routes, tokenUsage, "/models", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.GetModels)
		adminGET(routes, tokenUsage, "/routes", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.GetRoutes)
		adminGET(routes, tokenUsage, "/users", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.GetUsers)
		adminGET(routes, tokenUsage, "/options/models", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.ModelsOptions)
		adminGET(routes, tokenUsage, "/options/groups", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.GroupsOptions)
		adminGET(routes, tokenUsage, "/options/groups/:group_id/routes", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.RoutesOptions)
		adminGET(routes, tokenUsage, "/options/groups/:group_id/routes/:route_alias/models", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.RouteModelsOptions)
		adminGET(routes, tokenUsage, "/options/users", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.UsersOptions)
		adminGET(routes, tokenUsage, "/options/users/:user_id/models", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.UserModelsOptions)
		adminGET(routes, tokenUsage, "/default-target", rbac.PermissionTokenUsageRead, h.Admin.TokenUsageReport.GetDefaultTarget)
	}
}

func registerOpsRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	ops := admin.Group("/ops")
	{
		// Realtime ops signals
		adminGET(routes, ops, "/concurrency", rbac.PermissionOpsRead, h.Admin.Ops.GetConcurrencyStats)
		adminGET(routes, ops, "/user-concurrency", rbac.PermissionOpsRead, h.Admin.Ops.GetUserConcurrencyStats)
		adminGET(routes, ops, "/account-availability", rbac.PermissionOpsRead, h.Admin.Ops.GetAccountAvailability)
		adminGET(routes, ops, "/realtime-traffic", rbac.PermissionOpsRead, h.Admin.Ops.GetRealtimeTrafficSummary)

		// Alerts (rules + events)
		adminGET(routes, ops, "/alert-rules", rbac.PermissionOpsRead, h.Admin.Ops.ListAlertRules)
		adminPOST(routes, ops, "/alert-rules", rbac.PermissionOpsUpdate, h.Admin.Ops.CreateAlertRule)
		adminPUT(routes, ops, "/alert-rules/:id", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateAlertRule)
		adminDELETE(routes, ops, "/alert-rules/:id", rbac.PermissionOpsUpdate, h.Admin.Ops.DeleteAlertRule)
		adminGET(routes, ops, "/alert-events", rbac.PermissionOpsRead, h.Admin.Ops.ListAlertEvents)
		adminGET(routes, ops, "/alert-events/:id", rbac.PermissionOpsRead, h.Admin.Ops.GetAlertEvent)
		adminPUT(routes, ops, "/alert-events/:id/status", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateAlertEventStatus)
		adminPOST(routes, ops, "/alert-silences", rbac.PermissionOpsUpdate, h.Admin.Ops.CreateAlertSilence)

		// Email notification config (DB-backed)
		adminGET(routes, ops, "/email-notification/config", rbac.PermissionOpsRead, h.Admin.Ops.GetEmailNotificationConfig)
		adminPUT(routes, ops, "/email-notification/config", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateEmailNotificationConfig)

		// Runtime settings (DB-backed)
		runtime := ops.Group("/runtime")
		{
			adminGET(routes, runtime, "/alert", rbac.PermissionOpsRead, h.Admin.Ops.GetAlertRuntimeSettings)
			adminPUT(routes, runtime, "/alert", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateAlertRuntimeSettings)
			adminGET(routes, runtime, "/logging", rbac.PermissionOpsRead, h.Admin.Ops.GetRuntimeLogConfig)
			adminPUT(routes, runtime, "/logging", rbac.PermissionOpsLogsManage, h.Admin.Ops.UpdateRuntimeLogConfig)
			adminPOST(routes, runtime, "/logging/reset", rbac.PermissionOpsLogsManage, h.Admin.Ops.ResetRuntimeLogConfig)
		}

		// Advanced settings (DB-backed)
		adminGET(routes, ops, "/advanced-settings", rbac.PermissionOpsRead, h.Admin.Ops.GetAdvancedSettings)
		adminPUT(routes, ops, "/advanced-settings", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateAdvancedSettings)

		// Settings group (DB-backed)
		settings := ops.Group("/settings")
		{
			adminGET(routes, settings, "/metric-thresholds", rbac.PermissionOpsRead, h.Admin.Ops.GetMetricThresholds)
			adminPUT(routes, settings, "/metric-thresholds", rbac.PermissionOpsUpdate, h.Admin.Ops.UpdateMetricThresholds)
		}

		// WebSocket realtime (QPS/TPS)
		ws := ops.Group("/ws")
		{
			adminGET(routes, ws, "/qps", rbac.PermissionOpsRead, h.Admin.Ops.QPSWSHandler)
		}

		// Error logs (legacy)
		adminGET(routes, ops, "/errors", rbac.PermissionOpsRead, h.Admin.Ops.GetErrorLogs)
		adminGET(routes, ops, "/errors/:id", rbac.PermissionOpsRead, h.Admin.Ops.GetErrorLogByID)
		adminPUT(routes, ops, "/errors/:id/resolve", rbac.PermissionOpsLogsManage, h.Admin.Ops.UpdateErrorResolution)

		// Request errors (client-visible failures)
		adminGET(routes, ops, "/request-errors", rbac.PermissionOpsRead, h.Admin.Ops.ListRequestErrors)
		adminGET(routes, ops, "/request-errors/:id", rbac.PermissionOpsRead, h.Admin.Ops.GetRequestError)
		adminGET(routes, ops, "/request-errors/:id/upstream-errors", rbac.PermissionOpsRead, h.Admin.Ops.ListRequestErrorUpstreamErrors)
		adminPUT(routes, ops, "/request-errors/:id/resolve", rbac.PermissionOpsLogsManage, h.Admin.Ops.ResolveRequestError)

		// Upstream errors (independent upstream failures)
		adminGET(routes, ops, "/upstream-errors", rbac.PermissionOpsRead, h.Admin.Ops.ListUpstreamErrors)
		adminGET(routes, ops, "/upstream-errors/:id", rbac.PermissionOpsRead, h.Admin.Ops.GetUpstreamError)
		adminPUT(routes, ops, "/upstream-errors/:id/resolve", rbac.PermissionOpsLogsManage, h.Admin.Ops.ResolveUpstreamError)

		// Request drilldown (success + error)
		adminGET(routes, ops, "/requests", rbac.PermissionOpsRead, h.Admin.Ops.ListRequestDetails)

		// Indexed system logs
		adminGET(routes, ops, "/system-logs", rbac.PermissionOpsRead, h.Admin.Ops.ListSystemLogs)
		adminPOST(routes, ops, "/system-logs/cleanup", rbac.PermissionOpsLogsManage, h.Admin.Ops.CleanupSystemLogs)
		adminGET(routes, ops, "/system-logs/health", rbac.PermissionOpsRead, h.Admin.Ops.GetSystemLogIngestionHealth)

		// Dashboard (vNext - raw path for MVP)
		adminGET(routes, ops, "/dashboard/snapshot-v2", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardSnapshotV2)
		adminGET(routes, ops, "/dashboard/overview", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardOverview)
		adminGET(routes, ops, "/dashboard/throughput-trend", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardThroughputTrend)
		adminGET(routes, ops, "/dashboard/latency-histogram", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardLatencyHistogram)
		adminGET(routes, ops, "/dashboard/error-trend", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardErrorTrend)
		adminGET(routes, ops, "/dashboard/error-distribution", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardErrorDistribution)
		adminGET(routes, ops, "/dashboard/openai-token-stats", rbac.PermissionOpsRead, h.Admin.Ops.GetDashboardOpenAITokenStats)
	}
}

func registerDashboardRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	dashboard := admin.Group("/dashboard")
	{
		adminGET(routes, dashboard, "/snapshot-v2", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetSnapshotV2)
		adminGET(routes, dashboard, "/stats", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetStats)
		adminGET(routes, dashboard, "/realtime", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetRealtimeMetrics)
		adminGET(routes, dashboard, "/trend", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetUsageTrend)
		adminGET(routes, dashboard, "/models", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetModelStats)
		adminGET(routes, dashboard, "/groups", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetGroupStats)
		adminGET(routes, dashboard, "/api-keys-trend", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetAPIKeyUsageTrend)
		adminGET(routes, dashboard, "/users-trend", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetUserUsageTrend)
		adminGET(routes, dashboard, "/users-ranking", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetUserSpendingRanking)
		adminPOST(routes, dashboard, "/users-usage", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetBatchUsersUsage)
		adminPOST(routes, dashboard, "/api-keys-usage", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetBatchAPIKeysUsage)
		adminGET(routes, dashboard, "/user-breakdown", rbac.PermissionDashboardRead, h.Admin.Dashboard.GetUserBreakdown)
		adminPOST(routes, dashboard, "/aggregation/backfill", rbac.PermissionDashboardBackfill, h.Admin.Dashboard.BackfillAggregation)
	}
}

func registerUserManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	users := admin.Group("/users")
	{
		adminGET(routes, users, "", rbac.PermissionUsersRead, h.Admin.User.List)
		adminGET(routes, users, "/:id", rbac.PermissionUsersRead, h.Admin.User.GetByID)
		adminPOST(routes, users, "/:id/auth-identities", rbac.PermissionUsersUpdate, h.Admin.User.BindAuthIdentity)
		adminPOST(routes, users, "", rbac.PermissionUsersCreate, h.Admin.User.Create)
		adminPUT(routes, users, "/:id", rbac.PermissionUsersUpdate, h.Admin.User.Update)
		adminDELETE(routes, users, "/:id", rbac.PermissionUsersDelete, h.Admin.User.Delete)
		adminPOST(routes, users, "/:id/balance", rbac.PermissionUsersBalanceAdjust, h.Admin.User.UpdateBalance)
		adminGET(routes, users, "/:id/api-keys", rbac.PermissionUsersAPIKeysRead, h.Admin.User.GetUserAPIKeys)
		adminGET(routes, users, "/:id/usage", rbac.PermissionUsersUsageRead, h.Admin.User.GetUserUsage)
		adminGET(routes, users, "/:id/balance-history", rbac.PermissionUsersRead, h.Admin.User.GetBalanceHistory)
		adminPOST(routes, users, "/:id/replace-group", rbac.PermissionUsersUpdate, h.Admin.User.ReplaceGroup)
		adminGET(routes, users, "/:id/rpm-status", rbac.PermissionUsersRead, h.Admin.User.GetUserRPMStatus)
		adminPOST(routes, users, "/batch-concurrency", rbac.PermissionUsersUpdate, h.Admin.User.BatchUpdateConcurrency)
		adminGET(routes, users, "/:id/platform-quotas", rbac.PermissionUsersQuotaRead, h.Admin.User.GetUserPlatformQuotas)
		adminPUT(routes, users, "/:id/platform-quotas", rbac.PermissionUsersQuotaUpdate, h.Admin.User.UpdateUserPlatformQuotas)
		adminPOST(routes, users, "/:id/platform-quotas/reset", rbac.PermissionUsersQuotaUpdate, h.Admin.User.ResetUserPlatformQuotaWindow)
		adminGET(routes, users, "/:id/model-token-quotas", rbac.PermissionUsersQuotaRead, h.Admin.UserModelTokenQuota.List)
		adminPUT(routes, users, "/:id/model-token-quotas", rbac.PermissionUsersQuotaUpdate, h.Admin.UserModelTokenQuota.Update)
		adminPOST(routes, users, "/model-token-quotas/batch", rbac.PermissionUsersQuotaUpdate, h.Admin.UserModelTokenQuota.Batch)

		// User attribute values
		adminGET(routes, users, "/:id/attributes", rbac.PermissionUsersRead, h.Admin.UserAttribute.GetUserAttributes)
		adminPUT(routes, users, "/:id/attributes", rbac.PermissionUsersUpdate, h.Admin.UserAttribute.UpdateUserAttributes)
	}
}

func registerGroupRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	groups := admin.Group("/groups")
	{
		adminGET(routes, groups, "", rbac.PermissionGroupsRead, h.Admin.Group.List)
		adminGET(routes, groups, "/all", rbac.PermissionGroupsRead, h.Admin.Group.GetAll)
		adminGET(routes, groups, "/usage-summary", rbac.PermissionGroupsRead, h.Admin.Group.GetUsageSummary)
		adminGET(routes, groups, "/capacity-summary", rbac.PermissionGroupsRead, h.Admin.Group.GetCapacitySummary)
		adminPUT(routes, groups, "/sort-order", rbac.PermissionGroupsUpdate, h.Admin.Group.UpdateSortOrder)
		adminGET(routes, groups, "/:id/models-list-candidates", rbac.PermissionGroupsRead, h.Admin.Group.GetModelsListCandidates)
		adminGET(routes, groups, "/:id", rbac.PermissionGroupsRead, h.Admin.Group.GetByID)
		adminPOST(routes, groups, "", rbac.PermissionGroupsCreate, h.Admin.Group.Create)
		adminPUT(routes, groups, "/:id", rbac.PermissionGroupsUpdate, h.Admin.Group.Update)
		adminDELETE(routes, groups, "/:id", rbac.PermissionGroupsDelete, h.Admin.Group.Delete)
		adminGET(routes, groups, "/:id/stats", rbac.PermissionGroupsRead, h.Admin.Group.GetStats)
		adminGET(routes, groups, "/:id/rate-multipliers", rbac.PermissionGroupsRead, h.Admin.Group.GetGroupRateMultipliers)
		adminPUT(routes, groups, "/:id/rate-multipliers", rbac.PermissionGroupsUpdate, h.Admin.Group.BatchSetGroupRateMultipliers)
		adminDELETE(routes, groups, "/:id/rate-multipliers", rbac.PermissionGroupsUpdate, h.Admin.Group.ClearGroupRateMultipliers)
		adminPUT(routes, groups, "/:id/rpm-overrides", rbac.PermissionGroupsUpdate, h.Admin.Group.BatchSetGroupRPMOverrides)
		adminDELETE(routes, groups, "/:id/rpm-overrides", rbac.PermissionGroupsUpdate, h.Admin.Group.ClearGroupRPMOverrides)
		adminGET(routes, groups, "/:id/api-keys", rbac.PermissionGroupsRead, h.Admin.Group.GetGroupAPIKeys)
	}
}

func registerAccountRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	accounts := admin.Group("/accounts")
	{
		adminGET(routes, accounts, "", rbac.PermissionAccountsRead, h.Admin.Account.List)
		adminGET(routes, accounts, "/:id", rbac.PermissionAccountsRead, h.Admin.Account.GetByID)
		adminGET(routes, accounts, "/:id/credentials", rbac.PermissionAccountsCredentialsRead, h.Admin.Account.GetCredentials)
		adminPOST(routes, accounts, "", rbac.PermissionAccountsCreate, h.Admin.Account.Create)
		adminPOST(routes, accounts, "/check-mixed-channel", rbac.PermissionAccountsRead, h.Admin.Account.CheckMixedChannel)
		adminPOST(routes, accounts, "/import/codex-session", rbac.PermissionAccountsCredentialsUpdate, h.Admin.Account.ImportCodexSession)
		adminPOST(routes, accounts, "/sync/crs", rbac.PermissionAccountsCredentialsUpdate, h.Admin.Account.SyncFromCRS)
		adminPOST(routes, accounts, "/sync/crs/preview", rbac.PermissionAccountsCredentialsRead, h.Admin.Account.PreviewFromCRS)
		adminPUT(routes, accounts, "/:id", rbac.PermissionAccountsUpdate, h.Admin.Account.Update)
		adminDELETE(routes, accounts, "/:id", rbac.PermissionAccountsDelete, h.Admin.Account.Delete)
		adminPOST(routes, accounts, "/:id/test", rbac.PermissionAccountsOperate, h.Admin.Account.Test)
		adminPOST(routes, accounts, "/:id/recover-state", rbac.PermissionAccountsOperate, h.Admin.Account.RecoverState)
		adminPOST(routes, accounts, "/:id/refresh", rbac.PermissionAccountsOperate, h.Admin.Account.Refresh)
		adminPOST(routes, accounts, "/:id/apply-oauth-credentials", rbac.PermissionAccountsCredentialsUpdate, h.Admin.Account.ApplyOAuthCredentials)
		adminPOST(routes, accounts, "/:id/set-privacy", rbac.PermissionAccountsUpdate, h.Admin.Account.SetPrivacy)
		adminPOST(routes, accounts, "/:id/refresh-tier", rbac.PermissionAccountsOperate, h.Admin.Account.RefreshTier)
		adminGET(routes, accounts, "/:id/stats", rbac.PermissionAccountsRead, h.Admin.Account.GetStats)
		adminPOST(routes, accounts, "/:id/clear-error", rbac.PermissionAccountsOperate, h.Admin.Account.ClearError)
		adminPOST(routes, accounts, "/:id/revert-proxy-fallback", rbac.PermissionAccountsOperate, h.Admin.Account.RevertProxyFallback)
		adminGET(routes, accounts, "/:id/usage", rbac.PermissionAccountsRead, h.Admin.Account.GetUsage)
		adminGET(routes, accounts, "/:id/today-stats", rbac.PermissionAccountsRead, h.Admin.Account.GetTodayStats)
		adminPOST(routes, accounts, "/today-stats/batch", rbac.PermissionAccountsRead, h.Admin.Account.GetBatchTodayStats)
		adminPOST(routes, accounts, "/:id/clear-rate-limit", rbac.PermissionAccountsOperate, h.Admin.Account.ClearRateLimit)
		adminPOST(routes, accounts, "/:id/reset-quota", rbac.PermissionAccountsOperate, h.Admin.Account.ResetQuota)
		adminGET(routes, accounts, "/:id/temp-unschedulable", rbac.PermissionAccountsRead, h.Admin.Account.GetTempUnschedulable)
		adminDELETE(routes, accounts, "/:id/temp-unschedulable", rbac.PermissionAccountsOperate, h.Admin.Account.ClearTempUnschedulable)
		adminPOST(routes, accounts, "/:id/schedulable", rbac.PermissionAccountsOperate, h.Admin.Account.SetSchedulable)
		adminPOST(routes, accounts, "/models/sync-upstream-preview", rbac.PermissionAccountsRead, h.Admin.Account.SyncUpstreamModelsPreview)
		adminGET(routes, accounts, "/:id/models", rbac.PermissionAccountsRead, h.Admin.Account.GetAvailableModels)
		adminPOST(routes, accounts, "/:id/models/sync-upstream", rbac.PermissionAccountsOperate, h.Admin.Account.SyncUpstreamModels)
		adminPOST(routes, accounts, "/batch", rbac.PermissionAccountsCreate, h.Admin.Account.BatchCreate)
		adminGET(routes, accounts, "/data", rbac.PermissionAccountsCredentialsRead, h.Admin.Account.ExportData)
		adminPOST(routes, accounts, "/data", rbac.PermissionAccountsCredentialsUpdate, h.Admin.Account.ImportData)
		adminPOST(routes, accounts, "/batch-update-credentials", rbac.PermissionAccountsCredentialsUpdate, h.Admin.Account.BatchUpdateCredentials)
		adminPOST(routes, accounts, "/batch-refresh-tier", rbac.PermissionAccountsOperate, h.Admin.Account.BatchRefreshTier)
		adminPOST(routes, accounts, "/bulk-update", rbac.PermissionAccountsUpdate, h.Admin.Account.BulkUpdate)
		adminPOST(routes, accounts, "/batch-clear-error", rbac.PermissionAccountsOperate, h.Admin.Account.BatchClearError)
		adminPOST(routes, accounts, "/batch-refresh", rbac.PermissionAccountsOperate, h.Admin.Account.BatchRefresh)

		// Antigravity 默认模型映射
		adminGET(routes, accounts, "/antigravity/default-model-mapping", rbac.PermissionAccountsRead, h.Admin.Account.GetAntigravityDefaultModelMapping)

		// Claude OAuth routes
		adminPOST(routes, accounts, "/generate-auth-url", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.GenerateAuthURL)
		adminPOST(routes, accounts, "/generate-setup-token-url", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.GenerateSetupTokenURL)
		adminPOST(routes, accounts, "/exchange-code", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.ExchangeCode)
		adminPOST(routes, accounts, "/exchange-setup-token-code", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.ExchangeSetupTokenCode)
		adminPOST(routes, accounts, "/cookie-auth", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.CookieAuth)
		adminPOST(routes, accounts, "/setup-token-cookie-auth", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OAuth.SetupTokenCookieAuth)
	}
}

func registerAnnouncementRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	announcements := admin.Group("/announcements")
	{
		adminGET(routes, announcements, "", rbac.PermissionAnnouncementsRead, h.Admin.Announcement.List)
		adminPOST(routes, announcements, "", rbac.PermissionAnnouncementsCreate, h.Admin.Announcement.Create)
		adminGET(routes, announcements, "/:id", rbac.PermissionAnnouncementsRead, h.Admin.Announcement.GetByID)
		adminPUT(routes, announcements, "/:id", rbac.PermissionAnnouncementsUpdate, h.Admin.Announcement.Update)
		adminDELETE(routes, announcements, "/:id", rbac.PermissionAnnouncementsDelete, h.Admin.Announcement.Delete)
		adminGET(routes, announcements, "/:id/read-status", rbac.PermissionAnnouncementsRead, h.Admin.Announcement.ListReadStatus)
	}
}

func registerOpenAIOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	openai := admin.Group("/openai")
	{
		adminPOST(routes, openai, "/generate-auth-url", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OpenAIOAuth.GenerateAuthURL)
		adminPOST(routes, openai, "/exchange-code", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OpenAIOAuth.ExchangeCode)
		adminPOST(routes, openai, "/refresh-token", rbac.PermissionAccountsCredentialsUpdate, h.Admin.OpenAIOAuth.RefreshToken)
		adminPOST(routes, openai, "/accounts/:id/refresh", rbac.PermissionAccountsOperate, h.Admin.OpenAIOAuth.RefreshAccountToken)
		adminPOST(routes, openai, "/create-from-oauth", rbac.PermissionAccountsCreate, h.Admin.OpenAIOAuth.CreateAccountFromOAuth)
		adminGET(routes, openai, "/accounts/:id/quota", rbac.PermissionAccountsRead, h.Admin.OpenAIOAuth.QueryQuota)
		adminPOST(routes, openai, "/accounts/:id/reset-quota", rbac.PermissionAccountsOperate, h.Admin.OpenAIOAuth.ResetQuota)
	}
}

func registerGeminiOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	gemini := admin.Group("/gemini")
	{
		adminPOST(routes, gemini, "/oauth/auth-url", rbac.PermissionAccountsCredentialsUpdate, h.Admin.GeminiOAuth.GenerateAuthURL)
		adminPOST(routes, gemini, "/oauth/exchange-code", rbac.PermissionAccountsCredentialsUpdate, h.Admin.GeminiOAuth.ExchangeCode)
		adminGET(routes, gemini, "/oauth/capabilities", rbac.PermissionAccountsRead, h.Admin.GeminiOAuth.GetCapabilities)
	}
}

func registerAntigravityOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	antigravity := admin.Group("/antigravity")
	{
		adminPOST(routes, antigravity, "/oauth/auth-url", rbac.PermissionAccountsCredentialsUpdate, h.Admin.AntigravityOAuth.GenerateAuthURL)
		adminPOST(routes, antigravity, "/oauth/exchange-code", rbac.PermissionAccountsCredentialsUpdate, h.Admin.AntigravityOAuth.ExchangeCode)
		adminPOST(routes, antigravity, "/oauth/refresh-token", rbac.PermissionAccountsCredentialsUpdate, h.Admin.AntigravityOAuth.RefreshToken)
	}
}

func registerProxyRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	proxies := admin.Group("/proxies")
	{
		adminGET(routes, proxies, "", rbac.PermissionProxiesRead, h.Admin.Proxy.List)
		adminGET(routes, proxies, "/all", rbac.PermissionProxiesRead, h.Admin.Proxy.GetAll)
		adminGET(routes, proxies, "/data", rbac.PermissionProxiesRead, h.Admin.Proxy.ExportData)
		adminPOST(routes, proxies, "/data", rbac.PermissionProxiesUpdate, h.Admin.Proxy.ImportData)
		adminGET(routes, proxies, "/:id", rbac.PermissionProxiesRead, h.Admin.Proxy.GetByID)
		adminPOST(routes, proxies, "", rbac.PermissionProxiesCreate, h.Admin.Proxy.Create)
		adminPUT(routes, proxies, "/:id", rbac.PermissionProxiesUpdate, h.Admin.Proxy.Update)
		adminDELETE(routes, proxies, "/:id", rbac.PermissionProxiesDelete, h.Admin.Proxy.Delete)
		adminPOST(routes, proxies, "/:id/test", rbac.PermissionProxiesOperate, h.Admin.Proxy.Test)
		adminPOST(routes, proxies, "/:id/quality-check", rbac.PermissionProxiesOperate, h.Admin.Proxy.CheckQuality)
		adminGET(routes, proxies, "/:id/stats", rbac.PermissionProxiesRead, h.Admin.Proxy.GetStats)
		adminGET(routes, proxies, "/:id/accounts", rbac.PermissionProxiesRead, h.Admin.Proxy.GetProxyAccounts)
		adminPOST(routes, proxies, "/batch-delete", rbac.PermissionProxiesDelete, h.Admin.Proxy.BatchDelete)
		adminPOST(routes, proxies, "/batch", rbac.PermissionProxiesCreate, h.Admin.Proxy.BatchCreate)
	}
}

func registerRedeemCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	codes := admin.Group("/redeem-codes")
	{
		adminGET(routes, codes, "", rbac.PermissionRedeemCodesRead, h.Admin.Redeem.List)
		adminGET(routes, codes, "/stats", rbac.PermissionRedeemCodesRead, h.Admin.Redeem.GetStats)
		adminGET(routes, codes, "/export", rbac.PermissionRedeemCodesRead, h.Admin.Redeem.Export)
		adminGET(routes, codes, "/:id", rbac.PermissionRedeemCodesRead, h.Admin.Redeem.GetByID)
		adminPOST(routes, codes, "/create-and-redeem", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.CreateAndRedeem)
		adminPOST(routes, codes, "/generate", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.Generate)
		adminDELETE(routes, codes, "/:id", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.Delete)
		adminPOST(routes, codes, "/batch-delete", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.BatchDelete)
		adminPOST(routes, codes, "/batch-update", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.BatchUpdate)
		adminPOST(routes, codes, "/:id/expire", rbac.PermissionRedeemCodesManage, h.Admin.Redeem.Expire)
	}
}

func registerPromoCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	promoCodes := admin.Group("/promo-codes")
	{
		adminGET(routes, promoCodes, "", rbac.PermissionPromoCodesRead, h.Admin.Promo.List)
		adminGET(routes, promoCodes, "/:id", rbac.PermissionPromoCodesRead, h.Admin.Promo.GetByID)
		adminPOST(routes, promoCodes, "", rbac.PermissionPromoCodesManage, h.Admin.Promo.Create)
		adminPUT(routes, promoCodes, "/:id", rbac.PermissionPromoCodesManage, h.Admin.Promo.Update)
		adminDELETE(routes, promoCodes, "/:id", rbac.PermissionPromoCodesManage, h.Admin.Promo.Delete)
		adminGET(routes, promoCodes, "/:id/usages", rbac.PermissionPromoCodesRead, h.Admin.Promo.GetUsages)
	}
}

func registerSettingsRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	adminGET(routes, admin, "/default-group", rbac.PermissionSettingsRead, h.Admin.Setting.GetDefaultGroup)
	adminSettings := admin.Group("/settings")
	{
		adminGET(routes, adminSettings, "", rbac.PermissionSettingsRead, h.Admin.Setting.GetSettings)
		adminPUT(routes, adminSettings, "", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateSettings)
		adminPUT(routes, adminSettings, "/default-group", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateDefaultGroup)
		adminPOST(routes, adminSettings, "/test-smtp", rbac.PermissionSettingsUpdate, h.Admin.Setting.TestSMTPConnection)
		adminPOST(routes, adminSettings, "/send-test-email", rbac.PermissionSettingsUpdate, h.Admin.Setting.SendTestEmail)
		adminGET(routes, adminSettings, "/email-templates", rbac.PermissionSettingsRead, h.Admin.Setting.ListEmailTemplates)
		adminPOST(routes, adminSettings, "/email-template-preview", rbac.PermissionSettingsRead, h.Admin.Setting.PreviewEmailTemplate)
		adminGET(routes, adminSettings, "/email-templates/:event/:locale", rbac.PermissionSettingsRead, h.Admin.Setting.GetEmailTemplate)
		adminPUT(routes, adminSettings, "/email-templates/:event/:locale", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateEmailTemplate)
		adminPOST(routes, adminSettings, "/email-templates/:event/:locale/restore-official", rbac.PermissionSettingsUpdate, h.Admin.Setting.RestoreOfficialEmailTemplate)
		// Admin API Key 管理
		adminGET(routes, adminSettings, "/admin-api-key", rbac.PermissionSettingsSecretsManage, h.Admin.Setting.GetAdminAPIKey)
		adminPOST(routes, adminSettings, "/admin-api-key/regenerate", rbac.PermissionSettingsSecretsManage, h.Admin.Setting.RegenerateAdminAPIKey)
		adminDELETE(routes, adminSettings, "/admin-api-key", rbac.PermissionSettingsSecretsManage, h.Admin.Setting.DeleteAdminAPIKey)
		// 529过载冷却配置
		adminGET(routes, adminSettings, "/overload-cooldown", rbac.PermissionSettingsRead, h.Admin.Setting.GetOverloadCooldownSettings)
		adminPUT(routes, adminSettings, "/overload-cooldown", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateOverloadCooldownSettings)
		// 429默认回避配置
		adminGET(routes, adminSettings, "/rate-limit-429-cooldown", rbac.PermissionSettingsRead, h.Admin.Setting.GetRateLimit429CooldownSettings)
		adminPUT(routes, adminSettings, "/rate-limit-429-cooldown", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateRateLimit429CooldownSettings)
		// 流超时处理配置
		adminGET(routes, adminSettings, "/stream-timeout", rbac.PermissionSettingsRead, h.Admin.Setting.GetStreamTimeoutSettings)
		adminPUT(routes, adminSettings, "/stream-timeout", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateStreamTimeoutSettings)
		// 请求整流器配置
		adminGET(routes, adminSettings, "/rectifier", rbac.PermissionSettingsRead, h.Admin.Setting.GetRectifierSettings)
		adminPUT(routes, adminSettings, "/rectifier", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateRectifierSettings)
		// Beta 策略配置
		adminGET(routes, adminSettings, "/beta-policy", rbac.PermissionSettingsRead, h.Admin.Setting.GetBetaPolicySettings)
		adminPUT(routes, adminSettings, "/beta-policy", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateBetaPolicySettings)
		// Web Search 模拟配置
		adminGET(routes, adminSettings, "/web-search-emulation", rbac.PermissionSettingsRead, h.Admin.Setting.GetWebSearchEmulationConfig)
		adminPUT(routes, adminSettings, "/web-search-emulation", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateWebSearchEmulationConfig)
		adminPOST(routes, adminSettings, "/web-search-emulation/test", rbac.PermissionSettingsUpdate, h.Admin.Setting.TestWebSearchEmulation)
		adminPOST(routes, adminSettings, "/web-search-emulation/reset-usage", rbac.PermissionSettingsUpdate, h.Admin.Setting.ResetWebSearchUsage)
		// 新用户默认模型 Token 限额
		adminGET(routes, adminSettings, "/default-model-token-quotas", rbac.PermissionSettingsRead, h.Admin.Setting.GetDefaultModelTokenQuotas)
		adminPUT(routes, adminSettings, "/default-model-token-quotas", rbac.PermissionSettingsUpdate, h.Admin.Setting.UpdateDefaultModelTokenQuotas)
	}
}

func registerDataManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	dataManagement := admin.Group("/data-management")
	{
		adminGET(routes, dataManagement, "/agent/health", rbac.PermissionDataManagementRead, h.Admin.DataManagement.GetAgentHealth)
		adminGET(routes, dataManagement, "/config", rbac.PermissionDataManagementRead, h.Admin.DataManagement.GetConfig)
		adminPUT(routes, dataManagement, "/config", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.UpdateConfig)
		adminGET(routes, dataManagement, "/sources/:source_type/profiles", rbac.PermissionDataManagementRead, h.Admin.DataManagement.ListSourceProfiles)
		adminPOST(routes, dataManagement, "/sources/:source_type/profiles", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.CreateSourceProfile)
		adminPUT(routes, dataManagement, "/sources/:source_type/profiles/:profile_id", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.UpdateSourceProfile)
		adminDELETE(routes, dataManagement, "/sources/:source_type/profiles/:profile_id", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.DeleteSourceProfile)
		adminPOST(routes, dataManagement, "/sources/:source_type/profiles/:profile_id/activate", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.SetActiveSourceProfile)
		adminPOST(routes, dataManagement, "/s3/test", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.TestS3)
		adminGET(routes, dataManagement, "/s3/profiles", rbac.PermissionDataManagementRead, h.Admin.DataManagement.ListS3Profiles)
		adminPOST(routes, dataManagement, "/s3/profiles", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.CreateS3Profile)
		adminPUT(routes, dataManagement, "/s3/profiles/:profile_id", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.UpdateS3Profile)
		adminDELETE(routes, dataManagement, "/s3/profiles/:profile_id", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.DeleteS3Profile)
		adminPOST(routes, dataManagement, "/s3/profiles/:profile_id/activate", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.SetActiveS3Profile)
		adminPOST(routes, dataManagement, "/backups", rbac.PermissionDataManagementUpdate, h.Admin.DataManagement.CreateBackupJob)
		adminGET(routes, dataManagement, "/backups", rbac.PermissionDataManagementRead, h.Admin.DataManagement.ListBackupJobs)
		adminGET(routes, dataManagement, "/backups/:job_id", rbac.PermissionDataManagementRead, h.Admin.DataManagement.GetBackupJob)
	}
}

func registerBackupRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	backup := admin.Group("/backups")
	{
		// S3 存储配置
		adminGET(routes, backup, "/s3-config", rbac.PermissionBackupsRead, h.Admin.Backup.GetS3Config)
		adminPUT(routes, backup, "/s3-config", rbac.PermissionBackupsCreate, h.Admin.Backup.UpdateS3Config)
		adminPOST(routes, backup, "/s3-config/test", rbac.PermissionBackupsCreate, h.Admin.Backup.TestS3Connection)

		// 定时备份配置
		adminGET(routes, backup, "/schedule", rbac.PermissionBackupsRead, h.Admin.Backup.GetSchedule)
		adminPUT(routes, backup, "/schedule", rbac.PermissionBackupsCreate, h.Admin.Backup.UpdateSchedule)

		// 备份操作
		adminPOST(routes, backup, "", rbac.PermissionBackupsCreate, h.Admin.Backup.CreateBackup)
		adminGET(routes, backup, "", rbac.PermissionBackupsRead, h.Admin.Backup.ListBackups)
		adminGET(routes, backup, "/:id", rbac.PermissionBackupsRead, h.Admin.Backup.GetBackup)
		adminDELETE(routes, backup, "/:id", rbac.PermissionBackupsCreate, h.Admin.Backup.DeleteBackup)
		adminGET(routes, backup, "/:id/download-url", rbac.PermissionBackupsRead, h.Admin.Backup.GetDownloadURL)

		// 恢复操作
		adminPOST(routes, backup, "/:id/restore", rbac.PermissionBackupsRestore, h.Admin.Backup.RestoreBackup)
	}
}

func registerSystemRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	system := admin.Group("/system")
	{
		adminGET(routes, system, "/version", rbac.PermissionSystemRead, h.Admin.System.GetVersion)
		adminGET(routes, system, "/check-updates", rbac.PermissionSystemRead, h.Admin.System.CheckUpdates)
		adminPOST(routes, system, "/update", rbac.PermissionSystemOperate, h.Admin.System.PerformUpdate)
		adminPOST(routes, system, "/rollback", rbac.PermissionSystemOperate, h.Admin.System.Rollback)
		adminPOST(routes, system, "/restart", rbac.PermissionSystemOperate, h.Admin.System.RestartService)
	}
}

func registerSubscriptionRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	subscriptions := admin.Group("/subscriptions")
	{
		adminGET(routes, subscriptions, "", rbac.PermissionSubscriptionsRead, h.Admin.Subscription.List)
		adminGET(routes, subscriptions, "/:id", rbac.PermissionSubscriptionsRead, h.Admin.Subscription.GetByID)
		adminGET(routes, subscriptions, "/:id/progress", rbac.PermissionSubscriptionsRead, h.Admin.Subscription.GetProgress)
		adminPOST(routes, subscriptions, "/assign", rbac.PermissionSubscriptionsManage, h.Admin.Subscription.Assign)
		adminPOST(routes, subscriptions, "/bulk-assign", rbac.PermissionSubscriptionsManage, h.Admin.Subscription.BulkAssign)
		adminPOST(routes, subscriptions, "/:id/extend", rbac.PermissionSubscriptionsManage, h.Admin.Subscription.Extend)
		adminPOST(routes, subscriptions, "/:id/reset-quota", rbac.PermissionSubscriptionsManage, h.Admin.Subscription.ResetQuota)
		adminDELETE(routes, subscriptions, "/:id", rbac.PermissionSubscriptionsManage, h.Admin.Subscription.Revoke)
	}

	// 分组下的订阅列表
	adminGET(routes, admin, "/groups/:id/subscriptions", rbac.PermissionSubscriptionsRead, h.Admin.Subscription.ListByGroup)

	// 用户下的订阅列表
	adminGET(routes, admin, "/users/:id/subscriptions", rbac.PermissionSubscriptionsRead, h.Admin.Subscription.ListByUser)
}

func registerUsageRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	usage := admin.Group("/usage")
	{
		adminGET(routes, usage, "", rbac.PermissionUsageAdminRead, h.Admin.Usage.List)
		adminGET(routes, usage, "/stats", rbac.PermissionUsageAdminRead, h.Admin.Usage.Stats)
		adminGET(routes, usage, "/search-users", rbac.PermissionUsageAdminRead, h.Admin.Usage.SearchUsers)
		adminGET(routes, usage, "/search-api-keys", rbac.PermissionUsageAdminRead, h.Admin.Usage.SearchAPIKeys)
		adminGET(routes, usage, "/cleanup-tasks", rbac.PermissionUsageAdminRead, h.Admin.Usage.ListCleanupTasks)
		adminPOST(routes, usage, "/cleanup-tasks", rbac.PermissionUsageAdminManage, h.Admin.Usage.CreateCleanupTask)
		adminPOST(routes, usage, "/cleanup-tasks/:id/cancel", rbac.PermissionUsageAdminManage, h.Admin.Usage.CancelCleanupTask)
	}
}

func registerUserAttributeRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	attrs := admin.Group("/user-attributes")
	{
		adminGET(routes, attrs, "", rbac.PermissionUsersRead, h.Admin.UserAttribute.ListDefinitions)
		adminPOST(routes, attrs, "", rbac.PermissionUsersUpdate, h.Admin.UserAttribute.CreateDefinition)
		adminPOST(routes, attrs, "/batch", rbac.PermissionUsersRead, h.Admin.UserAttribute.GetBatchUserAttributes)
		adminPUT(routes, attrs, "/reorder", rbac.PermissionUsersUpdate, h.Admin.UserAttribute.ReorderDefinitions)
		adminPUT(routes, attrs, "/:id", rbac.PermissionUsersUpdate, h.Admin.UserAttribute.UpdateDefinition)
		adminDELETE(routes, attrs, "/:id", rbac.PermissionUsersUpdate, h.Admin.UserAttribute.DeleteDefinition)
	}
}

func registerScheduledTestRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	plans := admin.Group("/scheduled-test-plans")
	{
		adminPOST(routes, plans, "", rbac.PermissionMonitorsUpdate, h.Admin.ScheduledTest.Create)
		adminPUT(routes, plans, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ScheduledTest.Update)
		adminDELETE(routes, plans, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ScheduledTest.Delete)
		adminGET(routes, plans, "/:id/results", rbac.PermissionMonitorsRead, h.Admin.ScheduledTest.ListResults)
	}
	// Nested under accounts
	adminGET(routes, admin, "/accounts/:id/scheduled-test-plans", rbac.PermissionMonitorsRead, h.Admin.ScheduledTest.ListByAccount)
}

func registerErrorPassthroughRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	rules := admin.Group("/error-passthrough-rules")
	{
		adminGET(routes, rules, "", rbac.PermissionSettingsRead, h.Admin.ErrorPassthrough.List)
		adminGET(routes, rules, "/:id", rbac.PermissionSettingsRead, h.Admin.ErrorPassthrough.GetByID)
		adminPOST(routes, rules, "", rbac.PermissionSettingsUpdate, h.Admin.ErrorPassthrough.Create)
		adminPUT(routes, rules, "/:id", rbac.PermissionSettingsUpdate, h.Admin.ErrorPassthrough.Update)
		adminDELETE(routes, rules, "/:id", rbac.PermissionSettingsUpdate, h.Admin.ErrorPassthrough.Delete)
	}
}

func registerTLSFingerprintProfileRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	profiles := admin.Group("/tls-fingerprint-profiles")
	{
		adminGET(routes, profiles, "", rbac.PermissionSettingsRead, h.Admin.TLSFingerprintProfile.List)
		adminGET(routes, profiles, "/:id", rbac.PermissionSettingsRead, h.Admin.TLSFingerprintProfile.GetByID)
		adminPOST(routes, profiles, "", rbac.PermissionSettingsUpdate, h.Admin.TLSFingerprintProfile.Create)
		adminPUT(routes, profiles, "/:id", rbac.PermissionSettingsUpdate, h.Admin.TLSFingerprintProfile.Update)
		adminDELETE(routes, profiles, "/:id", rbac.PermissionSettingsUpdate, h.Admin.TLSFingerprintProfile.Delete)
	}
}

func registerChannelRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	channels := admin.Group("/channels")
	{
		adminGET(routes, channels, "", rbac.PermissionChannelsRead, h.Admin.Channel.List)
		adminGET(routes, channels, "/model-pricing", rbac.PermissionChannelsRead, h.Admin.Channel.GetModelDefaultPricing)
		adminGET(routes, channels, "/pricing/sync-models", rbac.PermissionChannelsRead, h.Admin.Channel.SyncPricingModels)
		adminGET(routes, channels, "/:id", rbac.PermissionChannelsRead, h.Admin.Channel.GetByID)
		adminPOST(routes, channels, "", rbac.PermissionChannelsCreate, h.Admin.Channel.Create)
		adminPUT(routes, channels, "/:id", rbac.PermissionChannelsUpdate, h.Admin.Channel.Update)
		adminDELETE(routes, channels, "/:id", rbac.PermissionChannelsDelete, h.Admin.Channel.Delete)
	}
}

func registerChannelMonitorRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	monitors := admin.Group("/channel-monitors")
	{
		adminGET(routes, monitors, "", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitor.List)
		adminPOST(routes, monitors, "", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitor.Create)
		adminGET(routes, monitors, "/:id", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitor.Get)
		adminPUT(routes, monitors, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitor.Update)
		adminDELETE(routes, monitors, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitor.Delete)
		adminPOST(routes, monitors, "/:id/run", rbac.PermissionMonitorsRun, h.Admin.ChannelMonitor.Run)
		adminGET(routes, monitors, "/:id/history", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitor.History)
	}

	templates := admin.Group("/channel-monitor-templates")
	{
		adminGET(routes, templates, "", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitorTemplate.List)
		adminPOST(routes, templates, "", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitorTemplate.Create)
		adminGET(routes, templates, "/:id", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitorTemplate.Get)
		adminPUT(routes, templates, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitorTemplate.Update)
		adminDELETE(routes, templates, "/:id", rbac.PermissionMonitorsUpdate, h.Admin.ChannelMonitorTemplate.Delete)
		adminGET(routes, templates, "/:id/monitors", rbac.PermissionMonitorsRead, h.Admin.ChannelMonitorTemplate.AssociatedMonitors)
		adminPOST(routes, templates, "/:id/apply", rbac.PermissionMonitorsRun, h.Admin.ChannelMonitorTemplate.Apply)
	}
}

// registerAffiliateRoutes 注册邀请返利的管理端路由（专属用户配置）
func registerAffiliateRoutes(admin *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	affiliates := admin.Group("/affiliates")
	{
		adminGET(routes, affiliates, "/invites", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.ListInviteRecords)
		adminGET(routes, affiliates, "/rebates", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.ListRebateRecords)
		adminGET(routes, affiliates, "/transfers", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.ListTransferRecords)

		users := affiliates.Group("/users")
		{
			adminGET(routes, users, "", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.ListUsers)
			adminGET(routes, users, "/lookup", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.LookupUsers)
			adminPOST(routes, users, "/batch-rate", rbac.PermissionAffiliatesManage, h.Admin.Affiliate.BatchSetRate)
			adminGET(routes, users, "/:user_id/overview", rbac.PermissionAffiliatesRead, h.Admin.Affiliate.GetUserOverview)
			adminPUT(routes, users, "/:user_id", rbac.PermissionAffiliatesManage, h.Admin.Affiliate.UpdateUserSettings)
			adminDELETE(routes, users, "/:user_id", rbac.PermissionAffiliatesManage, h.Admin.Affiliate.ClearUserSettings)
		}
	}
}
