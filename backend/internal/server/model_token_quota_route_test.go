package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelTokenQuotaAdminRouteRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			ModelTokenQuota:     adminhandler.NewModelTokenQuotaHandler(nil),
			UserModelTokenQuota: adminhandler.NewUserModelTokenQuotaHandler(nil, nil),
		},
	}
	adminAuth := middleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	routes.RegisterAdminRoutes(v1, handlers, adminAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-token-quotas", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserModelTokenQuotaAdminRouteRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			ModelTokenQuota:     adminhandler.NewModelTokenQuotaHandler(nil),
			UserModelTokenQuota: adminhandler.NewUserModelTokenQuotaHandler(nil, nil),
		},
	}
	adminAuth := middleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	routes.RegisterAdminRoutes(v1, handlers, adminAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1/model-token-quotas", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTokenUsageReportAdminRouteRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		ModelTokenQuota:     adminhandler.NewModelTokenQuotaHandler(nil),
		UserModelTokenQuota: adminhandler.NewUserModelTokenQuotaHandler(nil, nil),
		TokenUsageReport:    adminhandler.NewTokenUsageReportHandler(nil),
	}}
	adminAuth := middleware.AdminAuthMiddleware(func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) })
	routes.RegisterAdminRoutes(v1, handlers, adminAuth, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/token-usage/models?model=gpt", nil))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
