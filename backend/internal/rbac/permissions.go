// Package rbac contains the stable permission catalog used by backend route
// declarations, SQL compatibility seeds, and frontend permission metadata.
package rbac

import "sort"

const (
	PermissionAll = "*"

	PermissionProfileSelfRead       = "profile.self.read"
	PermissionProfileSelfUpdate     = "profile.self.update"
	PermissionProfileSelfSecure     = "profile.self.security"
	PermissionAPIKeysSelfRead       = "api_keys.self.read"
	PermissionAPIKeysSelfCreate     = "api_keys.self.create"
	PermissionAPIKeysSelfUpdate     = "api_keys.self.update"
	PermissionAPIKeysSelfDelete     = "api_keys.self.delete"
	PermissionUsageSelfRead         = "usage.self.read"
	PermissionGroupsSelfRead        = "groups.self.read"
	PermissionChannelsSelfRead      = "channels.self.read"
	PermissionRedeemSelfUse         = "redeem.self.use"
	PermissionRedeemSelfRead        = "redeem.self.read"
	PermissionSubscriptionsSelfRead = "subscriptions.self.read"
	PermissionPaymentsSelfRead      = "payments.self.read"
	PermissionPaymentsSelfCreate    = "payments.self.create"
	PermissionPaymentsSelfUpdate    = "payments.self.update"
	PermissionAffiliateSelfRead     = "affiliate.self.read"
	PermissionAffiliateSelfTransfer = "affiliate.self.transfer"
	PermissionAnnouncementsSelfRead = "announcements.self.read"
	PermissionMonitorsSelfRead      = "monitors.self.read"
	PermissionPagesSelfRead         = "pages.self.read"

	PermissionDashboardRead     = "dashboard.read"
	PermissionDashboardBackfill = "dashboard.backfill"

	PermissionUsersRead          = "users.read"
	PermissionUsersCreate        = "users.create"
	PermissionUsersUpdate        = "users.update"
	PermissionUsersDelete        = "users.delete"
	PermissionUsersBalanceAdjust = "users.balance.adjust"
	PermissionUsersAPIKeysRead   = "users.api_keys.read"
	PermissionUsersUsageRead     = "users.usage.read"
	PermissionUsersQuotaRead     = "users.quota.read"
	PermissionUsersQuotaUpdate   = "users.quota.update"
	PermissionUsersRolesRead     = "users.roles.read"
	PermissionUsersRolesAssign   = "users.roles.assign"

	PermissionRolesRead              = "roles.read"
	PermissionRolesCreate            = "roles.create"
	PermissionRolesUpdate            = "roles.update"
	PermissionRolesDelete            = "roles.delete"
	PermissionRolesPermissionsAssign = "roles.permissions.assign"
	PermissionPermissionsRead        = "permissions.read"
	PermissionPermissionsCreate      = "permissions.create"
	PermissionPermissionsUpdate      = "permissions.update"
	PermissionPermissionsDelete      = "permissions.delete"

	PermissionGroupsRead   = "groups.read"
	PermissionGroupsCreate = "groups.create"
	PermissionGroupsUpdate = "groups.update"
	PermissionGroupsDelete = "groups.delete"

	PermissionAccountsRead              = "accounts.read"
	PermissionAccountsCreate            = "accounts.create"
	PermissionAccountsUpdate            = "accounts.update"
	PermissionAccountsDelete            = "accounts.delete"
	PermissionAccountsOperate           = "accounts.operate"
	PermissionAccountsCredentialsRead   = "accounts.credentials.read"
	PermissionAccountsCredentialsUpdate = "accounts.credentials.update"

	PermissionProxiesRead    = "proxies.read"
	PermissionProxiesCreate  = "proxies.create"
	PermissionProxiesUpdate  = "proxies.update"
	PermissionProxiesDelete  = "proxies.delete"
	PermissionProxiesOperate = "proxies.operate"

	PermissionSettingsRead          = "settings.read"
	PermissionSettingsUpdate        = "settings.update"
	PermissionSettingsSecretsManage = "settings.secrets.manage"
	PermissionSystemRead            = "system.read"
	PermissionSystemOperate         = "system.operate"
	PermissionBackupsRead           = "backups.read"
	PermissionBackupsCreate         = "backups.create"
	PermissionBackupsRestore        = "backups.restore"
	PermissionDataManagementRead    = "data_management.read"
	PermissionDataManagementUpdate  = "data_management.update"

	PermissionOpsRead          = "ops.read"
	PermissionOpsUpdate        = "ops.update"
	PermissionOpsLogsManage    = "ops.logs.manage"
	PermissionUsageAdminRead   = "usage.admin.read"
	PermissionUsageAdminManage = "usage.admin.manage"
	PermissionTokenUsageRead   = "token_usage.read"
	PermissionTokenQuotaRead   = "token_quota.read"
	PermissionTokenQuotaUpdate = "token_quota.update"

	PermissionAnnouncementsRead   = "announcements.read"
	PermissionAnnouncementsCreate = "announcements.create"
	PermissionAnnouncementsUpdate = "announcements.update"
	PermissionAnnouncementsDelete = "announcements.delete"

	PermissionChannelsRead   = "channels.read"
	PermissionChannelsCreate = "channels.create"
	PermissionChannelsUpdate = "channels.update"
	PermissionChannelsDelete = "channels.delete"
	PermissionMonitorsRead   = "monitors.read"
	PermissionMonitorsUpdate = "monitors.update"
	PermissionMonitorsRun    = "monitors.run"

	PermissionRiskRead    = "risk.read"
	PermissionRiskUpdate  = "risk.update"
	PermissionRiskOperate = "risk.operate"

	PermissionBillingRead            = "billing.read"
	PermissionBillingOrdersManage    = "billing.orders.manage"
	PermissionBillingPlansManage     = "billing.plans.manage"
	PermissionBillingProvidersManage = "billing.providers.manage"
	PermissionSubscriptionsRead      = "subscriptions.read"
	PermissionSubscriptionsManage    = "subscriptions.manage"
	PermissionRedeemCodesRead        = "redeem_codes.read"
	PermissionRedeemCodesManage      = "redeem_codes.manage"
	PermissionPromoCodesRead         = "promo_codes.read"
	PermissionPromoCodesManage       = "promo_codes.manage"
	PermissionAffiliatesRead         = "affiliates.read"
	PermissionAffiliatesManage       = "affiliates.manage"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type PermissionDefinition struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
	Risk        RiskLevel `json:"risk_level"`
}

var catalog = buildCatalog()

func buildCatalog() []PermissionDefinition {
	var definitions []PermissionDefinition
	add := func(module string, risk RiskLevel, items ...[3]string) {
		for _, item := range items {
			definitions = append(definitions, PermissionDefinition{
				Code: item[0], Name: item[1], Module: module, Description: item[2], Risk: risk,
			})
		}
	}

	add("system", RiskCritical, [3]string{PermissionAll, "全部权限", "内置 admin 与管理员 API Key 专用通配权限"})
	add("self", RiskLow,
		[3]string{PermissionProfileSelfRead, "查看个人资料", "查看当前用户资料"},
		[3]string{PermissionAPIKeysSelfRead, "查看个人 API Key", "查看当前用户拥有的 API Key"},
		[3]string{PermissionUsageSelfRead, "查看个人用量", "查看当前用户用量与错误详情"},
		[3]string{PermissionGroupsSelfRead, "查看个人分组", "查看当前用户可用分组与倍率"},
		[3]string{PermissionChannelsSelfRead, "查看可用渠道", "查看当前用户可用渠道"},
		[3]string{PermissionRedeemSelfRead, "查看兑换记录", "查看当前用户兑换历史"},
		[3]string{PermissionSubscriptionsSelfRead, "查看个人订阅", "查看当前用户订阅"},
		[3]string{PermissionPaymentsSelfRead, "查看个人支付", "查看当前用户订单和支付配置"},
		[3]string{PermissionAffiliateSelfRead, "查看个人返利", "查看当前用户返利信息"},
		[3]string{PermissionAnnouncementsSelfRead, "查看公告", "查看并标记当前用户公告"},
		[3]string{PermissionMonitorsSelfRead, "查看渠道状态", "查看用户可见渠道监控"},
		[3]string{PermissionPagesSelfRead, "查看自定义页面", "查看用户可见自定义页面"},
	)
	add("self", RiskMedium,
		[3]string{PermissionProfileSelfUpdate, "修改个人资料", "修改当前用户资料"},
		[3]string{PermissionProfileSelfSecure, "管理个人安全", "修改密码、身份绑定、通知邮箱和 TOTP"},
		[3]string{PermissionAPIKeysSelfCreate, "创建个人 API Key", "创建当前用户 API Key"},
		[3]string{PermissionAPIKeysSelfUpdate, "修改个人 API Key", "修改当前用户 API Key"},
		[3]string{PermissionAPIKeysSelfDelete, "删除个人 API Key", "删除当前用户 API Key"},
		[3]string{PermissionRedeemSelfUse, "使用兑换码", "当前用户兑换额度或订阅"},
		[3]string{PermissionPaymentsSelfCreate, "创建个人订单", "创建当前用户支付订单"},
		[3]string{PermissionPaymentsSelfUpdate, "操作个人订单", "验证、取消或申请退款"},
		[3]string{PermissionAffiliateSelfTransfer, "转移个人返利", "转移当前用户返利额度"},
	)

	addModulePermissions := func(module, display string, read, create, update, delete string) {
		add(module, RiskLow, [3]string{read, "查看" + display, "查看" + display + "信息"})
		if create != "" {
			add(module, RiskMedium, [3]string{create, "创建" + display, "创建" + display})
		}
		if update != "" {
			add(module, RiskHigh, [3]string{update, "修改" + display, "修改" + display})
		}
		if delete != "" {
			add(module, RiskCritical, [3]string{delete, "删除" + display, "删除" + display})
		}
	}
	addModulePermissions("users", "用户", PermissionUsersRead, PermissionUsersCreate, PermissionUsersUpdate, PermissionUsersDelete)
	addModulePermissions("roles", "角色", PermissionRolesRead, PermissionRolesCreate, PermissionRolesUpdate, PermissionRolesDelete)
	addModulePermissions("groups", "分组", PermissionGroupsRead, PermissionGroupsCreate, PermissionGroupsUpdate, PermissionGroupsDelete)
	addModulePermissions("accounts", "账号", PermissionAccountsRead, PermissionAccountsCreate, PermissionAccountsUpdate, PermissionAccountsDelete)
	addModulePermissions("proxies", "代理", PermissionProxiesRead, PermissionProxiesCreate, PermissionProxiesUpdate, PermissionProxiesDelete)
	addModulePermissions("announcements", "公告", PermissionAnnouncementsRead, PermissionAnnouncementsCreate, PermissionAnnouncementsUpdate, PermissionAnnouncementsDelete)
	addModulePermissions("channels", "渠道", PermissionChannelsRead, PermissionChannelsCreate, PermissionChannelsUpdate, PermissionChannelsDelete)

	add("users", RiskLow,
		[3]string{PermissionUsersAPIKeysRead, "查看用户 API Key", "查看指定用户 API Key"},
		[3]string{PermissionUsersUsageRead, "查看用户用量", "查看指定用户用量"},
		[3]string{PermissionUsersQuotaRead, "查看用户配额", "查看用户平台和模型配额"},
		[3]string{PermissionUsersRolesRead, "查看用户角色", "查看用户角色分配"},
	)
	add("users", RiskCritical,
		[3]string{PermissionUsersBalanceAdjust, "调整用户余额", "增加或扣减用户余额"},
		[3]string{PermissionUsersQuotaUpdate, "修改用户配额", "修改或重置用户配额"},
		[3]string{PermissionUsersRolesAssign, "分配用户角色", "替换用户角色并可能授予管理能力"},
	)
	add("roles", RiskLow, [3]string{PermissionPermissionsRead, "查看权限目录", "查看系统权限定义"})
	add("roles", RiskHigh,
		[3]string{PermissionPermissionsCreate, "创建业务权限", "创建可分配给角色的业务权限编码"},
		[3]string{PermissionPermissionsUpdate, "修改业务权限", "修改或停用非系统权限"},
	)
	add("roles", RiskCritical, [3]string{PermissionPermissionsDelete, "删除业务权限", "删除非系统权限及其角色绑定"})
	add("roles", RiskCritical, [3]string{PermissionRolesPermissionsAssign, "配置角色权限", "全量替换角色权限"})
	add("dashboard", RiskLow, [3]string{PermissionDashboardRead, "查看管理仪表盘", "查看管理统计和趋势"})
	add("dashboard", RiskHigh, [3]string{PermissionDashboardBackfill, "回填仪表盘聚合", "执行聚合数据回填"})
	add("accounts", RiskHigh, [3]string{PermissionAccountsOperate, "操作上游账号", "测试、刷新、恢复或同步账号"})
	add("accounts", RiskCritical,
		[3]string{PermissionAccountsCredentialsRead, "查看账号凭据", "查看敏感上游凭据"},
		[3]string{PermissionAccountsCredentialsUpdate, "修改账号凭据", "导入、交换或批量修改凭据"},
	)
	add("proxies", RiskHigh, [3]string{PermissionProxiesOperate, "操作代理", "测试、质检或批量操作代理"})
	add("settings", RiskLow, [3]string{PermissionSettingsRead, "查看系统设置", "查看系统设置"})
	add("settings", RiskHigh, [3]string{PermissionSettingsUpdate, "修改系统设置", "修改系统运行设置"})
	add("settings", RiskCritical, [3]string{PermissionSettingsSecretsManage, "管理系统密钥", "管理管理员 API Key 等敏感配置"})
	add("system", RiskLow, [3]string{PermissionSystemRead, "查看系统状态", "查看版本和更新状态"})
	add("system", RiskCritical, [3]string{PermissionSystemOperate, "执行系统操作", "更新、回滚或重启系统"})
	add("backups", RiskLow, [3]string{PermissionBackupsRead, "查看备份", "查看和下载备份"})
	add("backups", RiskHigh, [3]string{PermissionBackupsCreate, "创建备份", "创建或配置备份"})
	add("backups", RiskCritical, [3]string{PermissionBackupsRestore, "恢复备份", "从备份恢复系统"})
	add("data_management", RiskLow, [3]string{PermissionDataManagementRead, "查看数据管理", "查看数据源和任务"})
	add("data_management", RiskHigh, [3]string{PermissionDataManagementUpdate, "修改数据管理", "修改数据源或执行任务"})
	add("ops", RiskLow, [3]string{PermissionOpsRead, "查看运维信息", "查看指标、错误和日志"})
	add("ops", RiskHigh,
		[3]string{PermissionOpsUpdate, "修改运维配置", "修改告警、通知和运行时设置"},
		[3]string{PermissionOpsLogsManage, "管理运维日志", "解决错误或清理系统日志"},
	)
	add("usage", RiskLow,
		[3]string{PermissionUsageAdminRead, "查看全局用量", "查看用户和 API Key 用量"},
		[3]string{PermissionTokenUsageRead, "查看 Token 统计", "查看模型、路由和用户 Token 统计"},
		[3]string{PermissionTokenQuotaRead, "查看 Token 配额", "查看模型 Token 配额"},
	)
	add("usage", RiskHigh,
		[3]string{PermissionUsageAdminManage, "管理用量数据", "创建或取消用量清理任务"},
		[3]string{PermissionTokenQuotaUpdate, "修改 Token 配额", "修改全局或用户模型 Token 配额"},
	)
	add("monitors", RiskLow, [3]string{PermissionMonitorsRead, "查看渠道监控", "查看监控与历史"})
	add("monitors", RiskHigh,
		[3]string{PermissionMonitorsUpdate, "修改渠道监控", "修改监控和模板"},
		[3]string{PermissionMonitorsRun, "运行渠道监控", "手动运行监控"},
	)
	add("risk", RiskLow, [3]string{PermissionRiskRead, "查看风控", "查看风控配置、状态和日志"})
	add("risk", RiskHigh, [3]string{PermissionRiskUpdate, "修改风控", "修改风控配置"})
	add("risk", RiskCritical, [3]string{PermissionRiskOperate, "执行风控操作", "解封用户或清理风险哈希"})
	add("billing", RiskLow,
		[3]string{PermissionBillingRead, "查看支付管理", "查看支付仪表盘、订单和配置"},
		[3]string{PermissionSubscriptionsRead, "查看订阅管理", "查看用户订阅"},
		[3]string{PermissionRedeemCodesRead, "查看兑换码", "查看兑换码和统计"},
		[3]string{PermissionPromoCodesRead, "查看优惠码", "查看优惠码和使用记录"},
		[3]string{PermissionAffiliatesRead, "查看返利管理", "查看邀请、返利和转账"},
	)
	add("billing", RiskHigh,
		[3]string{PermissionBillingOrdersManage, "管理支付订单", "取消、重试或退款"},
		[3]string{PermissionBillingPlansManage, "管理支付套餐", "创建、修改或删除套餐"},
		[3]string{PermissionBillingProvidersManage, "管理支付服务商", "修改支付服务商和密钥"},
		[3]string{PermissionSubscriptionsManage, "管理订阅", "分配、延期、重置或删除订阅"},
		[3]string{PermissionRedeemCodesManage, "管理兑换码", "生成、修改或删除兑换码"},
		[3]string{PermissionPromoCodesManage, "管理优惠码", "创建、修改或删除优惠码"},
		[3]string{PermissionAffiliatesManage, "管理返利", "修改专属返利或批量费率"},
	)

	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Code < definitions[j].Code })
	return definitions
}

func Catalog() []PermissionDefinition {
	result := make([]PermissionDefinition, len(catalog))
	copy(result, catalog)
	return result
}

func PermissionExists(code string) bool {
	i := sort.Search(len(catalog), func(i int) bool { return catalog[i].Code >= code })
	return i < len(catalog) && catalog[i].Code == code
}
