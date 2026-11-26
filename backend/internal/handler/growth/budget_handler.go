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

type BudgetService interface {
	GetAccount(context.Context) (growthservice.BudgetAccount, error)
	TopUp(context.Context, growthservice.BudgetMutation) (growthservice.BudgetAccount, error)
	Reduce(context.Context, growthservice.BudgetMutation) (growthservice.BudgetAccount, error)
	Allocate(context.Context, int64, growthservice.BudgetMutation) (growthservice.BudgetAccount, error)
	Revoke(context.Context, int64, growthservice.BudgetMutation) (growthservice.BudgetAccount, error)
	ListEntries(context.Context, growthservice.BudgetListFilter) ([]growthservice.BudgetEntry, int64, error)
	ListAllocations(context.Context, growthservice.BudgetListFilter) ([]growthservice.TaskAllocation, int64, error)
}

type Handler struct {
	budget      BudgetService
	tasks       TaskService
	products    ProductService
	reads       ReadService
	checkins    CheckinService
	completions CompletionService
	orders      OrderService
	fulfillment FulfillmentService
	adminPoints AdminGrowthService
}

func NewHandler(
	budgetBudgetService BudgetService,
	taskService TaskService,
	productService ProductService,
	readService ReadService,
	checkinService CheckinService,
	completionService CompletionService,
	orderService OrderService,
	fulfillmentService FulfillmentService,
	adminGrowthService AdminGrowthService,
) *Handler {
	return &Handler{
		budget: budgetBudgetService, tasks: taskService, products: productService,
		reads: readService, checkins: checkinService, completions: completionService,
		orders: orderService, fulfillment: fulfillmentService, adminPoints: adminGrowthService,
	}
}

type BudgetEntryRequest struct {
	Direction string `json:"direction" binding:"required,oneof=INCREASE DECREASE"`
	Points    int64  `json:"points" binding:"required,gt=0"`
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

type AllocationRequest struct {
	TaskID    int64  `json:"task_id" binding:"required,gt=0"`
	Points    int64  `json:"points" binding:"required,gt=0"`
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

type RevokeRequest struct {
	Points    int64  `json:"points" binding:"required,gt=0"`
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

func (h *Handler) GetOverview(c *gin.Context) {
	h.getAccount(c)
}

func (h *Handler) GetBudget(c *gin.Context) {
	h.getAccount(c)
}

func (h *Handler) getAccount(c *gin.Context) {
	account, err := h.budget.GetAccount(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *Handler) ListBudgetEntries(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	taskID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("task_id")), 10, 64)
	entries, total, err := h.budget.ListEntries(c.Request.Context(), growthservice.BudgetListFilter{
		EntryType: c.Query("type"),
		TaskID:    taskID,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}

func (h *Handler) CreateBudgetEntry(c *gin.Context) {
	var request BudgetEntryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid budget entry request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	mutation := growthservice.BudgetMutation{
		OperatorID: operatorID,
		Points:     request.Points,
		Reason:     request.Reason,
		RequestID:  requestRequestID(c, request.RequestID),
	}
	var (
		account growthservice.BudgetAccount
		err     error
	)
	if request.Direction == "INCREASE" {
		account, err = h.budget.TopUp(c.Request.Context(), mutation)
	} else {
		account, err = h.budget.Reduce(c.Request.Context(), mutation)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *Handler) ListAllocations(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.budget.ListAllocations(c.Request.Context(), growthservice.BudgetListFilter{
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) Allocate(c *gin.Context) {
	var request AllocationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid allocation request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	account, err := h.budget.Allocate(c.Request.Context(), request.TaskID, growthservice.BudgetMutation{
		OperatorID: operatorID,
		Points:     request.Points,
		Reason:     request.Reason,
		RequestID:  requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *Handler) Revoke(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.BadRequest(c, "invalid task id")
		return
	}
	var request RevokeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid revoke request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	account, err := h.budget.Revoke(c.Request.Context(), taskID, growthservice.BudgetMutation{
		OperatorID: operatorID,
		Points:     request.Points,
		Reason:     request.Reason,
		RequestID:  requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func currentOperatorID(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	return subject.UserID, ok && subject.UserID > 0
}

func requestRequestID(c *gin.Context, fallback string) string {
	if value := strings.TrimSpace(c.GetHeader("X-Request-Id")); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
