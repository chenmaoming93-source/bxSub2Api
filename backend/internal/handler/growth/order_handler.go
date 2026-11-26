package growth

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type OrderService interface {
	Create(context.Context, growthservice.ExchangeCommand) (growthservice.ExchangeCreateResult, error)
	UDetail(context.Context, int64, int64) (growthservice.ExchangeOrderDetail, error)
	WDetail(context.Context, int64, int64) (growthservice.ExchangeOrderDetail, error)
}

type WCreateExchangeRequest struct {
	ProductID       int64  `json:"product_id" binding:"required,gt=0"`
	ExpectedVersion *int   `json:"expected_version"`
	RequestID       string `json:"request_id"`
}

func (h *Handler) UOrder(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.BadRequest(c, "invalid order id")
		return
	}
	payload, ok := servermiddleware.GetGrowthPayloadFromContext(c)
	if !ok || payload == nil {
		response.Unauthorized(c, "growth payload is unavailable")
		return
	}
	var params struct {
		OrderID int64 `json:"order_id"`
	}
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if params.OrderID != orderID {
		response.BadRequest(c, "order_id path and payload mismatch")
		return
	}
	item, err := h.orders.UDetail(c.Request.Context(), userID, orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) WCreateExchange(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	var request WCreateExchangeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid exchange request: "+err.Error())
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	executeGrowthPayloadIdempotent(
		c,
		"growth.exchange",
		request,
		key,
		func(ctx context.Context) (any, error) {
			return h.orders.Create(ctx, growthservice.ExchangeCommand{
				UserID: userID, ProductID: request.ProductID,
				ExpectedVersion: request.ExpectedVersion,
				RequestID:       request.RequestID, IdempotencyKey: key,
			})
		},
	)
}

func (h *Handler) WOrder(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.BadRequest(c, "invalid order id")
		return
	}
	item, err := h.orders.WDetail(c.Request.Context(), userID, orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
