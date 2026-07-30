package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	rbacRoutes *rbac.RouteRegistrar,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.Use(middleware.PrincipalFromAuthenticatedSubject())
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			rbacRoutes.GET(user, "/profile", rbac.PermissionProfileSelfRead, h.User.GetProfile)
			rbacRoutes.PUT(user, "/password", rbac.PermissionProfileSelfSecure, h.User.ChangePassword)
			rbacRoutes.PUT(user, "", rbac.PermissionProfileSelfUpdate, h.User.UpdateProfile)
			rbacRoutes.GET(user, "/aff", rbac.PermissionAffiliateSelfRead, h.User.GetAffiliate)
			rbacRoutes.POST(user, "/aff/transfer", rbac.PermissionAffiliateSelfTransfer, h.User.TransferAffiliateQuota)
			rbacRoutes.POST(user, "/account-bindings/email/send-code", rbac.PermissionProfileSelfSecure, h.User.SendEmailBindingCode)
			rbacRoutes.POST(user, "/account-bindings/email", rbac.PermissionProfileSelfSecure, h.User.BindEmailIdentity)
			rbacRoutes.DELETE(user, "/account-bindings/:provider", rbac.PermissionProfileSelfSecure, h.User.UnbindIdentity)
			rbacRoutes.POST(user, "/auth-identities/bind/start", rbac.PermissionProfileSelfSecure, h.User.StartIdentityBinding)
			rbacRoutes.GET(user, "/api-keys/:id/usage/daily", rbac.PermissionUsageSelfRead, h.Usage.GetMyAPIKeyDailyUsage)
			rbacRoutes.GET(user, "/platform-quotas", rbac.PermissionUsageSelfRead, h.User.GetMyPlatformQuotas)

			// 通知邮箱管理
			notifyEmail := user.Group("/notify-email")
			{
				rbacRoutes.POST(notifyEmail, "/send-code", rbac.PermissionProfileSelfSecure, h.User.SendNotifyEmailCode)
				rbacRoutes.POST(notifyEmail, "/verify", rbac.PermissionProfileSelfSecure, h.User.VerifyNotifyEmail)
				rbacRoutes.PUT(notifyEmail, "/toggle", rbac.PermissionProfileSelfSecure, h.User.ToggleNotifyEmail)
				rbacRoutes.DELETE(notifyEmail, "", rbac.PermissionProfileSelfSecure, h.User.RemoveNotifyEmail)
			}

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				rbacRoutes.GET(totp, "/status", rbac.PermissionProfileSelfSecure, h.Totp.GetStatus)
				rbacRoutes.GET(totp, "/verification-method", rbac.PermissionProfileSelfSecure, h.Totp.GetVerificationMethod)
				rbacRoutes.POST(totp, "/send-code", rbac.PermissionProfileSelfSecure, h.Totp.SendVerifyCode)
				rbacRoutes.POST(totp, "/setup", rbac.PermissionProfileSelfSecure, h.Totp.InitiateSetup)
				rbacRoutes.POST(totp, "/enable", rbac.PermissionProfileSelfSecure, h.Totp.Enable)
				rbacRoutes.POST(totp, "/disable", rbac.PermissionProfileSelfSecure, h.Totp.Disable)
			}
		}

		// API Key管理
		keys := authenticated.Group("/keys")
		{
			rbacRoutes.GET(keys, "", rbac.PermissionAPIKeysSelfRead, h.APIKey.List)
			rbacRoutes.GET(keys, "/:id", rbac.PermissionAPIKeysSelfRead, h.APIKey.GetByID)
			rbacRoutes.POST(keys, "", rbac.PermissionAPIKeysSelfCreate, h.APIKey.Create)
			rbacRoutes.PUT(keys, "/:id", rbac.PermissionAPIKeysSelfUpdate, h.APIKey.Update)
			rbacRoutes.DELETE(keys, "/:id", rbac.PermissionAPIKeysSelfDelete, h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		{
			rbacRoutes.GET(groups, "/available", rbac.PermissionGroupsSelfRead, h.APIKey.GetAvailableGroups)
			rbacRoutes.GET(groups, "/rates", rbac.PermissionGroupsSelfRead, h.APIKey.GetUserGroupRates)
		}

		// 用户可用渠道（非管理员接口）
		channels := authenticated.Group("/channels")
		{
			rbacRoutes.GET(channels, "/available", rbac.PermissionChannelsSelfRead, h.AvailableChannel.List)
		}

		// 使用记录
		usage := authenticated.Group("/usage")
		{
			rbacRoutes.GET(usage, "", rbac.PermissionUsageSelfRead, h.Usage.List)
			rbacRoutes.GET(usage, "/errors", rbac.PermissionUsageSelfRead, h.Usage.ListErrors)
			rbacRoutes.GET(usage, "/errors/:id", rbac.PermissionUsageSelfRead, h.Usage.GetErrorDetail)
			rbacRoutes.GET(usage, "/:id", rbac.PermissionUsageSelfRead, h.Usage.GetByID)
			rbacRoutes.GET(usage, "/stats", rbac.PermissionUsageSelfRead, h.Usage.Stats)
			// User dashboard endpoints
			rbacRoutes.GET(usage, "/dashboard/stats", rbac.PermissionUsageSelfRead, h.Usage.DashboardStats)
			rbacRoutes.GET(usage, "/dashboard/trend", rbac.PermissionUsageSelfRead, h.Usage.DashboardTrend)
			rbacRoutes.GET(usage, "/dashboard/models", rbac.PermissionUsageSelfRead, h.Usage.DashboardModels)
			rbacRoutes.POST(usage, "/dashboard/api-keys-usage", rbac.PermissionUsageSelfRead, h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			rbacRoutes.GET(announcements, "", rbac.PermissionAnnouncementsSelfRead, h.Announcement.List)
			rbacRoutes.POST(announcements, "/:id/read", rbac.PermissionAnnouncementsSelfRead, h.Announcement.MarkRead)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			rbacRoutes.POST(redeem, "", rbac.PermissionRedeemSelfUse, h.Redeem.Redeem)
			rbacRoutes.GET(redeem, "/history", rbac.PermissionRedeemSelfRead, h.Redeem.GetHistory)
		}

		// 用户订阅
		subscriptions := authenticated.Group("/subscriptions")
		{
			rbacRoutes.GET(subscriptions, "", rbac.PermissionSubscriptionsSelfRead, h.Subscription.List)
			rbacRoutes.GET(subscriptions, "/active", rbac.PermissionSubscriptionsSelfRead, h.Subscription.GetActive)
			rbacRoutes.GET(subscriptions, "/progress", rbac.PermissionSubscriptionsSelfRead, h.Subscription.GetProgress)
			rbacRoutes.GET(subscriptions, "/summary", rbac.PermissionSubscriptionsSelfRead, h.Subscription.GetSummary)
		}

		// 渠道监控（用户只读）
		monitors := authenticated.Group("/channel-monitors")
		{
			rbacRoutes.GET(monitors, "", rbac.PermissionMonitorsSelfRead, h.ChannelMonitor.List)
			rbacRoutes.GET(monitors, "/:id/status", rbac.PermissionMonitorsSelfRead, h.ChannelMonitor.GetStatus)
		}
	}
}
