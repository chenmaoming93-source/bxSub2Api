package server

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const frameSrcRefreshTimeout = 5 * time.Second

// SetupRouter 配置路由器中间件和路由
func SetupRouter(
	r *gin.Engine,
	handlers *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	adminIdentityAuth middleware2.AdminIdentityAuthMiddleware,
	apiKeyAuth middleware2.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
	rbacPermissionService *rbac.PermissionService,
	rbacRegistry *rbac.Registry,
) *gin.Engine {
	// 缓存 iframe 页面的 origin 列表，用于动态注入 CSP frame-src
	var cachedFrameOrigins atomic.Pointer[[]string]
	emptyOrigins := []string{}
	cachedFrameOrigins.Store(&emptyOrigins)

	refreshFrameOrigins := func() {
		ctx, cancel := context.WithTimeout(context.Background(), frameSrcRefreshTimeout)
		defer cancel()
		origins, err := settingService.GetFrameSrcOrigins(ctx)
		if err != nil {
			// 获取失败时保留已有缓存，避免 frame-src 被意外清空
			return
		}
		cachedFrameOrigins.Store(&origins)
	}
	refreshFrameOrigins() // 启动时初始化

	// 应用中间件
	r.Use(middleware2.RequestLogger())
	r.Use(middleware2.Logger())
	r.Use(middleware2.CORS(cfg.CORS))
	r.Use(middleware2.SecurityHeaders(cfg.Security.CSP, func() []string {
		if p := cachedFrameOrigins.Load(); p != nil {
			return *p
		}
		return nil
	}))

	// Serve embedded frontend with settings injection if available
	if web.HasEmbeddedFrontend() {
		frontendServer, err := web.NewFrontendServer(settingService)
		if err != nil {
			log.Printf("Warning: Failed to create frontend server with settings injection: %v, using legacy mode", err)
			r.Use(web.ServeEmbeddedFrontend())
			settingService.SetOnUpdateCallback(refreshFrameOrigins)
		} else {
			// Register combined callback: invalidate HTML cache + refresh frame origins
			settingService.SetOnUpdateCallback(func() {
				frontendServer.InvalidateCache()
				refreshFrameOrigins()
			})
			r.Use(frontendServer.Middleware())
		}
	} else {
		settingService.SetOnUpdateCallback(refreshFrameOrigins)
	}

	// 注册路由
	registerRoutes(r, handlers, jwtAuth, adminAuth, adminIdentityAuth, apiKeyAuth, apiKeyService, subscriptionService, opsService, settingService, cfg, redisClient, rbacPermissionService, rbacRegistry)

	return r
}

// registerRoutes 注册所有 HTTP 路由
func registerRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	adminIdentityAuth middleware2.AdminIdentityAuthMiddleware,
	apiKeyAuth middleware2.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
	redisClient *redis.Client,
	rbacPermissionService *rbac.PermissionService,
	rbacRegistry *rbac.Registry,
) {
	// 通用路由（健康检查、状态等）
	routes.RegisterCommonRoutes(r)

	// API v1
	v1 := r.Group("/api/v1")
	rbacRegistrar := rbac.NewRouteRegistrar(
		rbacRegistry,
		middleware2.RequirePermission(
			rbacPermissionService,
			middleware2.ParseRBACMode(cfg.RBAC.Mode),
			middleware2.NewRBACAuthorizationAuditHook(cfg.RBAC.AuditDenials),
		),
	)

	// 注册各模块路由
	routes.RegisterAuthRoutes(v1, h, jwtAuth, redisClient, settingService, rbacRegistrar)
	routes.RegisterUserRoutes(v1, h, jwtAuth, settingService, rbacRegistrar)
	routes.RegisterAdminRoutes(v1, h, adminAuth, settingService, routes.AdminRBACOptions{
		IdentityAuth: adminIdentityAuth,
		Registrar:    rbacRegistrar,
		Registry:     rbacRegistry,
	})
	routes.RegisterGatewayRoutes(r, h, apiKeyAuth, apiKeyService, subscriptionService, opsService, settingService, cfg)
	routes.RegisterPaymentRoutes(v1, h.Payment, h.PaymentWebhook, h.Admin.Payment, jwtAuth, adminIdentityAuth, settingService, rbacRegistrar, rbacRegistry)

	handler.RegisterPageRoutes(v1, cfg.Pricing.DataDir, gin.HandlerFunc(jwtAuth), gin.HandlerFunc(adminIdentityAuth), settingService, rbacRegistrar)

	// 外部供应接口
	pc := cfg.ExternalAPIKeyProvisioning
	bizLimit := pc.RateLimitBizPerMinute
	authLimit := pc.RateLimitAuthPerMinute
	var provLimiter *middleware2.ProvisioningRateLimiter
	if bizLimit != -1 || authLimit != -1 {
		if bizLimit <= 0 {
			bizLimit = 60
		}
		if authLimit <= 0 {
			authLimit = 10
		}
		provLimiter = middleware2.NewProvisioningRateLimiter(time.Minute, time.Minute, authLimit, bizLimit)
		provLimiter.StartCleanup(5 * time.Minute)
	}
	provHardening := middleware2.NewProvisioningHardening(provLimiter, nil)
	provAuth := middleware2.ExternalProvisioningAuth(pc)
	routes.RegisterIntegrationRoutes(v1, h.ExternalProvisioning, provAuth, provHardening.Middleware())

	if err := rbac.RegisterKnownExclusions(r.Routes(), rbacRegistry); err != nil {
		panic(err)
	}
	if err := rbac.ValidateRouteCoverage(r.Routes(), rbacRegistry).Err(); err != nil {
		panic(err)
	}
}
