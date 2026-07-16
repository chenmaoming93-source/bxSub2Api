package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterIntegrationRoutes registers the external provisioning routes.
func RegisterIntegrationRoutes(
	v1 *gin.RouterGroup,
	provHandler *handler.ExternalProvisioningHandler,
	provAuth gin.HandlerFunc,
	provHardening gin.HandlerFunc,
) {
	if provHandler == nil {
		return
	}

	integration := v1.Group("/integrations")
	integration.Use(provAuth, provHardening)
	{
		integration.POST("/api-keys/getOrCreate", provHandler.EnsureAPIKey)
		integration.POST("/model-routes/list", provHandler.ListGroupModelRoutes)
	}
}
