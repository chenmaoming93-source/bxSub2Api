package growth

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type AdminGrowthService interface {
	ListUsers(context.Context, int, int, string, int64, int64) ([]growthservice.AdminUserPoints, int64, error)
	Adjust(context.Context, growthservice.AdjustmentCommand) (growthservice.AdjustmentResult, error)
	Reconcile(context.Context) (growthservice.ReconciliationReport, error)
	AuditLogs(context.Context, int, int, string) ([]*ent.GrowthAdminAudit, int64, error)
}

type adjustmentRequest struct {
	Direction string `json:"direction" binding:"required,oneof=INCREASE DECREASE"`
	Points    int64  `json:"points" binding:"required,gt=0"`
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

func (h *Handler) AdminListUserPoints(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	minPoints, _ := strconv.ParseInt(c.Query("min_points"), 10, 64)
	maxPoints, _ := strconv.ParseInt(c.Query("max_points"), 10, 64)
	items, total, err := h.adminPoints.ListUsers(
		c.Request.Context(), page, pageSize, c.Query("search"), minPoints, maxPoints,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) AdminUserLedger(c *gin.Context) {
	userID, ok := adminUserIDParam(c)
	if !ok {
		return
	}
	beforeID, _ := strconv.ParseInt(c.Query("before_entry_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := h.reads.Ledger(c.Request.Context(), userID, beforeID, limit, c.Query("type"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

func (h *Handler) AdminUserExchanges(c *gin.Context) {
	userID, ok := adminUserIDParam(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.reads.WExchanges(c.Request.Context(), userID, page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) AdminAdjustPoints(c *gin.Context) {
	userID, ok := adminUserIDParam(c)
	if !ok {
		return
	}
	var request adjustmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid adjustment request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	result, err := h.adminPoints.Adjust(c.Request.Context(), growthservice.AdjustmentCommand{
		OperatorID: operatorID, UserID: userID, Direction: request.Direction,
		Points: request.Points, Reason: request.Reason,
		RequestID: requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) AdminReconciliation(c *gin.Context) {
	report, err := h.adminPoints.Reconcile(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, report)
}

func (h *Handler) AdminAuditLogs(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.adminPoints.AuditLogs(c.Request.Context(), page, pageSize, c.Query("action"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func adminUserIDParam(c *gin.Context) (int64, bool) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user id")
		return 0, false
	}
	return userID, true
}
