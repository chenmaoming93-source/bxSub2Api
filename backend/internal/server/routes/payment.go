package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPaymentRoutes registers all payment-related routes:
// user-facing endpoints, webhook endpoints, and admin endpoints.
func RegisterPaymentRoutes(
	v1 *gin.RouterGroup,
	paymentHandler *handler.PaymentHandler,
	webhookHandler *handler.PaymentWebhookHandler,
	adminPaymentHandler *admin.PaymentHandler,
	jwtAuth middleware.JWTAuthMiddleware,
	adminIdentityAuth middleware.AdminIdentityAuthMiddleware,
	settingService *service.SettingService,
	rbacRoutes *rbac.RouteRegistrar,
	rbacRegistry *rbac.Registry,
) {
	// --- User-facing payment endpoints (authenticated) ---
	authenticated := v1.Group("/payment")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.Use(middleware.PrincipalFromAuthenticatedSubject())
	{
		rbacRoutes.GET(authenticated, "/config", rbac.PermissionPaymentsSelfRead, paymentHandler.GetPaymentConfig)
		rbacRoutes.GET(authenticated, "/checkout-info", rbac.PermissionPaymentsSelfRead, paymentHandler.GetCheckoutInfo)
		rbacRoutes.GET(authenticated, "/plans", rbac.PermissionPaymentsSelfRead, paymentHandler.GetPlans)
		rbacRoutes.GET(authenticated, "/channels", rbac.PermissionPaymentsSelfRead, paymentHandler.GetChannels)
		rbacRoutes.GET(authenticated, "/limits", rbac.PermissionPaymentsSelfRead, paymentHandler.GetLimits)

		orders := authenticated.Group("/orders")
		{
			rbacRoutes.POST(orders, "", rbac.PermissionPaymentsSelfCreate, paymentHandler.CreateOrder)
			rbacRoutes.POST(orders, "/verify", rbac.PermissionPaymentsSelfUpdate, paymentHandler.VerifyOrder)
			rbacRoutes.GET(orders, "/my", rbac.PermissionPaymentsSelfRead, paymentHandler.GetMyOrders)
			rbacRoutes.GET(orders, "/:id", rbac.PermissionPaymentsSelfRead, paymentHandler.GetOrder)
			rbacRoutes.POST(orders, "/:id/cancel", rbac.PermissionPaymentsSelfUpdate, paymentHandler.CancelOrder)
			rbacRoutes.POST(orders, "/:id/refund-request", rbac.PermissionPaymentsSelfUpdate, paymentHandler.RequestRefund)
			rbacRoutes.GET(orders, "/refund-eligible-providers", rbac.PermissionPaymentsSelfRead, paymentHandler.GetRefundEligibleProviders)
		}
	}

	// --- Public payment endpoints (no auth) ---
	// Signed resume-token recovery is the preferred public lookup path.
	// The legacy anonymous out_trade_no verify endpoint remains available as a
	// persisted-state compatibility path for staggered upgrades.
	public := v1.Group("/payment/public")
	{
		public.POST("/orders/verify", paymentHandler.VerifyOrderPublic)
		public.POST("/orders/resolve", paymentHandler.ResolveOrderPublicByResumeToken)
	}

	// --- Webhook endpoints (no auth) ---
	webhook := v1.Group("/payment/webhook")
	{
		// EasyPay sends GET callbacks with query params
		webhook.GET("/easypay", webhookHandler.EasyPayNotify)
		webhook.POST("/easypay", webhookHandler.EasyPayNotify)
		webhook.POST("/alipay", webhookHandler.AlipayNotify)
		webhook.POST("/wxpay", webhookHandler.WxpayNotify)
		webhook.POST("/stripe", webhookHandler.StripeWebhook)
		webhook.POST("/airwallex", webhookHandler.AirwallexWebhook)
	}

	// --- Admin payment endpoints (admin auth) ---
	adminGroup := v1.Group("/admin/payment")
	adminGroup.Use(gin.HandlerFunc(adminIdentityAuth))
	adminGroup.Use(middleware.PrincipalFromAuthenticatedSubject())
	adminGroup.Use(middleware.RequireLegacyAdminForUnregistered(rbacRegistry))
	adminGroup.Use(middleware.AdminComplianceGuard(settingService))
	{
		// Dashboard
		rbacRoutes.GET(adminGroup, "/dashboard", rbac.PermissionBillingRead, adminPaymentHandler.GetDashboard)

		// Config
		rbacRoutes.GET(adminGroup, "/config", rbac.PermissionBillingRead, adminPaymentHandler.GetConfig)
		rbacRoutes.PUT(adminGroup, "/config", rbac.PermissionBillingProvidersManage, adminPaymentHandler.UpdateConfig)

		// Orders
		adminOrders := adminGroup.Group("/orders")
		{
			rbacRoutes.GET(adminOrders, "", rbac.PermissionBillingRead, adminPaymentHandler.ListOrders)
			rbacRoutes.GET(adminOrders, "/:id", rbac.PermissionBillingRead, adminPaymentHandler.GetOrderDetail)
			rbacRoutes.POST(adminOrders, "/:id/cancel", rbac.PermissionBillingOrdersManage, adminPaymentHandler.CancelOrder)
			rbacRoutes.POST(adminOrders, "/:id/retry", rbac.PermissionBillingOrdersManage, adminPaymentHandler.RetryFulfillment)
			rbacRoutes.POST(adminOrders, "/:id/refund", rbac.PermissionBillingOrdersManage, adminPaymentHandler.ProcessRefund)
		}

		// Subscription Plans
		plans := adminGroup.Group("/plans")
		{
			rbacRoutes.GET(plans, "", rbac.PermissionBillingRead, adminPaymentHandler.ListPlans)
			rbacRoutes.POST(plans, "", rbac.PermissionBillingPlansManage, adminPaymentHandler.CreatePlan)
			rbacRoutes.PUT(plans, "/:id", rbac.PermissionBillingPlansManage, adminPaymentHandler.UpdatePlan)
			rbacRoutes.DELETE(plans, "/:id", rbac.PermissionBillingPlansManage, adminPaymentHandler.DeletePlan)
		}

		// Provider Instances
		providers := adminGroup.Group("/providers")
		{
			rbacRoutes.GET(providers, "", rbac.PermissionBillingRead, adminPaymentHandler.ListProviders)
			rbacRoutes.POST(providers, "", rbac.PermissionBillingProvidersManage, adminPaymentHandler.CreateProvider)
			rbacRoutes.PUT(providers, "/:id", rbac.PermissionBillingProvidersManage, adminPaymentHandler.UpdateProvider)
			rbacRoutes.DELETE(providers, "/:id", rbac.PermissionBillingProvidersManage, adminPaymentHandler.DeleteProvider)
		}
	}
}
