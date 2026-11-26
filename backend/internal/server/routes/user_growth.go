package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/rbac"

	"github.com/gin-gonic/gin"
)

func registerGrowthUserRoutes(user *gin.RouterGroup, h *handler.Handlers, routes *rbac.RouteRegistrar) {
	growth := user.Group("/growth")
	routes.GET(growth, "/overview", rbac.PermissionGrowthSelfRead, h.Growth.WOverview)
	routes.GET(growth, "/tasks", rbac.PermissionGrowthSelfRead, h.Growth.WTasks)
	routes.GET(growth, "/products", rbac.PermissionGrowthSelfRead, h.Growth.WProducts)
	routes.GET(growth, "/products/:id", rbac.PermissionGrowthSelfRead, h.Growth.WProduct)
	routes.GET(growth, "/exchanges", rbac.PermissionGrowthSelfRead, h.Growth.WExchanges)
	routes.POST(growth, "/exchanges", rbac.PermissionGrowthSelfExchange, h.Growth.WCreateExchange)
	routes.GET(growth, "/exchanges/:id", rbac.PermissionGrowthSelfRead, h.Growth.WOrder)
	routes.GET(growth, "/ledger", rbac.PermissionGrowthSelfRead, h.Growth.WLedger)
}
