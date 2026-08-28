package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterIntegrationRoutes registers the external provisioning routes.
func RegisterIntegrationRoutes(
	v1 *gin.RouterGroup,
	provHandler *handler.ExternalProvisioningHandler,
	tokenUsageHandler *handler.ExternalTokenUsageHandler,
	sceneAccountUsageHandler *handler.ExternalSceneAccountDailyUsageHandler,
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
		integration.POST("/model-routes/list-attributes", provHandler.ListGroupModelRoutesWithAttributes)
		if tokenUsageHandler != nil {
			integration.POST("/token-usage/query", tokenUsageHandler.Query)
			integration.POST("/token-usage/query/group-api-key/daily", tokenUsageHandler.DailyQuery)
			integration.POST("/token-usage/query/group-api-key/daily/csv", tokenUsageHandler.DailyQueryCSV)
		}
		if sceneAccountUsageHandler != nil {
			integration.POST("/token-usage/query/scene-account/daily", sceneAccountUsageHandler.QueryDaily)
		}
	}
}
