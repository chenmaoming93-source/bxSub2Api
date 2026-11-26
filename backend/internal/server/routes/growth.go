package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterGrowthRoutes(v1 *gin.RouterGroup, h *handler.Handlers, decryptor middleware.GrowthPayloadDecryptor) {
	growth := v1.Group("/growth")
	growth.Use(decryptor.Middleware())
	growth.POST("/me", h.Growth.UMe)
	growth.POST("/rank", h.Growth.UPointsRank)
	growth.POST("/tasks", h.Growth.UTasks)
	growth.POST("/tasks/:task_id", h.Growth.UTask)
	growth.POST("/tasks/:task_id/completion-requests", h.Growth.UCompletion)
	growth.POST("/ledger", h.Growth.ULedger)
	growth.POST("/products", h.Growth.UProducts)
	growth.POST("/products/:product_id", h.Growth.UProduct)
	growth.POST("/exchanges", h.Growth.UExchanges)
	growth.POST("/exchanges/:order_id", h.Growth.UOrder)
	growth.POST("/checkins", h.Growth.UCheckin)
}
