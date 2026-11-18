package growth

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type FulfillmentService interface {
	ListAdminOrders(context.Context, int, int, string, string) ([]growthservice.ExchangeOrderDetail, int64, error)
	AdminDetail(context.Context, int64) (growthservice.ExchangeOrderDetail, error)
	Retry(context.Context, int64, int64, string) (growthservice.ExchangeOrderDetail, error)
	ManualComplete(context.Context, int64, int64, string, string) (growthservice.ExchangeOrderDetail, error)
	AdminRefund(context.Context, int64, int64, string, string) (growthservice.ExchangeOrderDetail, error)
}

type orderActionRequest struct {
	Reason    string `json:"reason"`
	RequestID string `json:"request_id"`
}

func (h *Handler) AdminListOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.fulfillment.ListAdminOrders(
		c.Request.Context(), page, pageSize, c.Query("status"), c.Query("search"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) AdminOrderDetail(c *gin.Context) {
	orderID, ok := orderActionID(c)
	if !ok {
		return
	}
	item, err := h.fulfillment.AdminDetail(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) AdminRetryOrder(c *gin.Context) {
	orderID, ok := orderActionID(c)
	if !ok {
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.fulfillment.Retry(c.Request.Context(), orderID, operatorID, requestRequestID(c, ""))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) AdminManualCompleteOrder(c *gin.Context) {
	orderID, ok := orderActionID(c)
	if !ok {
		return
	}
	var request orderActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid manual completion request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.fulfillment.ManualComplete(
		c.Request.Context(), orderID, operatorID, request.Reason, requestRequestID(c, request.RequestID),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) AdminRefundOrder(c *gin.Context) {
	orderID, ok := orderActionID(c)
	if !ok {
		return
	}
	var request orderActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid refund request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.fulfillment.AdminRefund(
		c.Request.Context(), orderID, operatorID, request.Reason, requestRequestID(c, request.RequestID),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func orderActionID(c *gin.Context) (int64, bool) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.BadRequest(c, "invalid order id")
		return 0, false
	}
	return orderID, true
}
