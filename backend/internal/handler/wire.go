package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	growthhandler "github.com/Wei-Shaw/sub2api/internal/handler/growth"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/service"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"

	"github.com/google/wire"
)

func ProvideAuthHandler(cfg *config.Config, authService *service.AuthService, userService *service.UserService, settingService *service.SettingService, promoService *service.PromoService, redeemService *service.RedeemService, totpService *service.TotpService, userAttributeService *service.UserAttributeService, permissionService *rbac.PermissionService) *AuthHandler {
	h := NewAuthHandler(cfg, authService, userService, settingService, promoService, redeemService, totpService, userAttributeService)
	h.rbacPermissionService = permissionService
	return h
}

func ProvideAdminUserHandler(
	adminService service.AdminService,
	concurrencyService *service.ConcurrencyService,
	userPlatformQuotaRepo service.UserPlatformQuotaRepository,
	billingCache service.BillingCache,
	ldapUserSync *service.LDAPUserSyncService,
	permissionService *rbac.PermissionService,
) *admin.UserHandler {
	h := admin.NewUserHandlerWithLDAPSync(
		adminService,
		concurrencyService,
		userPlatformQuotaRepo,
		billingCache,
		ldapUserSync,
	)
	h.SetSuperAdminUserProvider(permissionService)
	return h
}

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	accountHandler *admin.AccountHandler,
	announcementHandler *admin.AnnouncementHandler,
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	oauthHandler *admin.OAuthHandler,
	openaiOAuthHandler *admin.OpenAIOAuthHandler,
	geminiOAuthHandler *admin.GeminiOAuthHandler,
	antigravityOAuthHandler *admin.AntigravityOAuthHandler,
	proxyHandler *admin.ProxyHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	opsHandler *admin.OpsHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	usageHandler *admin.UsageHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	errorPassthroughHandler *admin.ErrorPassthroughHandler,
	tlsFingerprintProfileHandler *admin.TLSFingerprintProfileHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	scheduledTestHandler *admin.ScheduledTestHandler,
	channelHandler *admin.ChannelHandler,
	channelMonitorHandler *admin.ChannelMonitorHandler,
	channelMonitorTemplateHandler *admin.ChannelMonitorRequestTemplateHandler,
	contentModerationHandler *admin.ContentModerationHandler,
	paymentHandler *admin.PaymentHandler,
	affiliateHandler *admin.AffiliateHandler,
	complianceHandler *admin.ComplianceHandler,
	dynamicTokenStatisticsHandler *admin.DynamicTokenStatisticsHandler,
	sceneAccountUsageHandler *admin.SceneAccountDailyUsageHandler,
	rbacHandler *admin.RBACHandler,
) *AdminHandlers {
	return &AdminHandlers{
		Dashboard:              dashboardHandler,
		User:                   userHandler,
		Group:                  groupHandler,
		Account:                accountHandler,
		Announcement:           announcementHandler,
		DataManagement:         dataManagementHandler,
		Backup:                 backupHandler,
		OAuth:                  oauthHandler,
		OpenAIOAuth:            openaiOAuthHandler,
		GeminiOAuth:            geminiOAuthHandler,
		AntigravityOAuth:       antigravityOAuthHandler,
		Proxy:                  proxyHandler,
		Redeem:                 redeemHandler,
		Promo:                  promoHandler,
		Setting:                settingHandler,
		Ops:                    opsHandler,
		System:                 systemHandler,
		Subscription:           subscriptionHandler,
		Usage:                  usageHandler,
		UserAttribute:          userAttributeHandler,
		ErrorPassthrough:       errorPassthroughHandler,
		TLSFingerprintProfile:  tlsFingerprintProfileHandler,
		APIKey:                 apiKeyHandler,
		ScheduledTest:          scheduledTestHandler,
		Channel:                channelHandler,
		ChannelMonitor:         channelMonitorHandler,
		ChannelMonitorTemplate: channelMonitorTemplateHandler,
		ContentModeration:      contentModerationHandler,
		Payment:                paymentHandler,
		Affiliate:              affiliateHandler,
		Compliance:             complianceHandler,
		DynamicTokenStatistics: dynamicTokenStatisticsHandler,
		SceneAccountUsage:      sceneAccountUsageHandler,
		RBAC:                   rbacHandler,
	}
}

// ProvideAdminGroupHandler wires the background schedule refresher into the
// existing group handler without changing direct test constructors.
func ProvideAdminGroupHandler(
	adminService service.AdminService,
	dashboardService *service.DashboardService,
	groupCapacityService *service.GroupCapacityService,
	refresher *service.ModelRouteConcurrencyScheduleRefresher,
	settingService *service.SettingService,
) *admin.GroupHandler {
	h := admin.NewGroupHandler(adminService, dashboardService, groupCapacityService)
	h.SetModelRouteConcurrencyScheduleRefresher(refresher)
	h.SetSecurityCheckSettingService(settingService)
	return h
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo, notificationEmailService *service.NotificationEmailService) *SettingHandler {
	h := NewSettingHandler(settingService, buildInfo.Version)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvideAdminSettingHandler creates admin.SettingHandler with notification template APIs.
func ProvideAdminSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, opsService *service.OpsService, paymentConfigService *service.PaymentConfigService, paymentService *service.PaymentService, userAttributeService *service.UserAttributeService, notificationEmailService *service.NotificationEmailService) *admin.SettingHandler {
	h := admin.NewSettingHandler(settingService, emailService, turnstileService, opsService, paymentConfigService, paymentService, userAttributeService)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	growthHandler *growthhandler.Handler,
	announcementHandler *AnnouncementHandler,
	channelMonitorUserHandler *ChannelMonitorUserHandler,
	adminHandlers *AdminHandlers,
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	securityCheckService *service.SecurityCheckService,
	securityConfigProvider *service.SecurityConfigProvider,
	securityCheckCollector *service.SecurityCheckCollector,
	securityCheckLogStore service.SecurityCheckLogStore,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	paymentHandler *PaymentHandler,
	paymentWebhookHandler *PaymentWebhookHandler,
	availableChannelHandler *AvailableChannelHandler,
	externalProvisioningHandler *ExternalProvisioningHandler,
	externalTokenUsageHandler *ExternalTokenUsageHandler,
	externalSceneAccountUsageHandler *ExternalSceneAccountDailyUsageHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
) *Handlers {
	gatewayHandler.SetSecurityCheckDependencies(securityCheckService, securityConfigProvider, securityCheckCollector)
	openaiGatewayHandler.SetSecurityCheckDependencies(securityCheckService, securityConfigProvider, securityCheckCollector)
	adminHandlers.Group.SetSecurityCheckLogDependencies(securityCheckLogStore, securityCheckCollector)
	return &Handlers{
		Auth:                      authHandler,
		User:                      userHandler,
		APIKey:                    apiKeyHandler,
		Usage:                     usageHandler,
		Redeem:                    redeemHandler,
		Subscription:              subscriptionHandler,
		Growth:                    growthHandler,
		Announcement:              announcementHandler,
		ChannelMonitor:            channelMonitorUserHandler,
		Admin:                     adminHandlers,
		Gateway:                   gatewayHandler,
		OpenAIGateway:             openaiGatewayHandler,
		Setting:                   settingHandler,
		Totp:                      totpHandler,
		Payment:                   paymentHandler,
		PaymentWebhook:            paymentWebhookHandler,
		AvailableChannel:          availableChannelHandler,
		ExternalProvisioning:      externalProvisioningHandler,
		ExternalTokenUsage:        externalTokenUsageHandler,
		ExternalSceneAccountUsage: externalSceneAccountUsageHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	ProvideAuthHandler,
	NewUserHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	NewChannelMonitorUserHandler,
	growthhandler.NewHandler,
	growthservice.NewBudgetService,
	growthservice.NewTaskService,
	growthservice.NewProductService,
	growthservice.NewReadService,
	growthservice.NewTransferService,
	growthservice.NewCheckinService,
	growthservice.NewCompletionService,
	growthservice.NewOrderService,
	growthservice.NewFulfillmentService,
	growthservice.NewAdminGrowthService,
	wire.Bind(new(growthhandler.BudgetService), new(*growthservice.BudgetService)),
	wire.Bind(new(growthhandler.TaskService), new(*growthservice.TaskService)),
	wire.Bind(new(growthhandler.ProductService), new(*growthservice.ProductService)),
	wire.Bind(new(growthhandler.ReadService), new(*growthservice.ReadService)),
	wire.Bind(new(growthhandler.CheckinService), new(*growthservice.CheckinService)),
	wire.Bind(new(growthhandler.CompletionService), new(*growthservice.CompletionService)),
	wire.Bind(new(growthhandler.OrderService), new(*growthservice.OrderService)),
	wire.Bind(new(growthhandler.FulfillmentService), new(*growthservice.FulfillmentService)),
	wire.Bind(new(growthhandler.AdminGrowthService), new(*growthservice.AdminGrowthService)),
	NewGatewayHandler,
	NewOpenAIGatewayHandler,
	NewTotpHandler,
	ProvideSettingHandler,
	NewPaymentHandler,
	NewPaymentWebhookHandler,
	NewAvailableChannelHandler,
	NewExternalTokenUsageHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	ProvideAdminUserHandler,
	ProvideAdminGroupHandler,
	admin.NewAccountHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewOAuthHandler,
	admin.NewOpenAIOAuthHandler,
	admin.NewGeminiOAuthHandler,
	admin.NewAntigravityOAuthHandler,
	admin.NewProxyHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	ProvideAdminSettingHandler,
	admin.ProvideOpsHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUsageHandlerWithTokenStats,
	admin.NewUserAttributeHandler,
	admin.NewErrorPassthroughHandler,
	admin.NewTLSFingerprintProfileHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewScheduledTestHandler,
	admin.NewChannelHandler,
	admin.NewChannelMonitorHandler,
	admin.NewRBACHandler,
	admin.NewChannelMonitorRequestTemplateHandler,
	admin.NewContentModerationHandler,
	admin.NewPaymentHandler,
	admin.NewAffiliateHandler,
	admin.NewComplianceHandler,
	admin.NewDynamicTokenStatisticsHandler,
	admin.NewSceneAccountDailyUsageHandler,
	NewExternalSceneAccountDailyUsageHandler,
	tokenstat.NewRuntimeController,
	tokenstat.NewProjectionAdminService,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
