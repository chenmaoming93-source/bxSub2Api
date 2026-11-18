package growth

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type TaskService interface {
	Create(context.Context, growthservice.TaskCreateInput) (growthservice.TaskListView, error)
	Update(context.Context, int64, growthservice.TaskUpdateInput) (growthservice.TaskListView, error)
	Publish(context.Context, int64, int64, *time.Time, string) (growthservice.TaskListView, error)
	Offline(context.Context, int64, int64, string, bool, string) (growthservice.TaskListView, error)
	Online(context.Context, int64, int64, int64, string) (growthservice.TaskListView, error)
	List(context.Context, growthservice.BudgetListFilter, string) ([]growthservice.TaskListView, int64, error)
	Get(context.Context, int64) (growthservice.TaskListView, error)
	RuleVersions(context.Context, int64) ([]growthservice.TaskRuleView, error)
}

type CreateTaskRequest struct {
	TaskCode          string                      `json:"task_code" binding:"required"`
	Title             string                      `json:"title" binding:"required"`
	Description       string                      `json:"description"`
	Category          string                      `json:"category" binding:"required"`
	TaskType          string                      `json:"task_type" binding:"required"`
	Icon              string                      `json:"icon"`
	SortOrder         int                         `json:"sort_order"`
	InitialAllocation int64                       `json:"initial_allocation" binding:"required,gt=0"`
	Rule              growthservice.TaskRuleInput `json:"rule" binding:"required"`
	RequestID         string                      `json:"request_id"`
}

type UpdateTaskRequest struct {
	Title       string                      `json:"title" binding:"required"`
	Description string                      `json:"description"`
	Category    string                      `json:"category" binding:"required"`
	TaskType    string                      `json:"task_type" binding:"required"`
	Icon        string                      `json:"icon"`
	SortOrder   int                         `json:"sort_order"`
	Rule        growthservice.TaskRuleInput `json:"rule" binding:"required"`
	RequestID   string                      `json:"request_id"`
}

type PublishTaskRequest struct {
	EffectiveFrom *time.Time `json:"effective_from"`
	RequestID     string     `json:"request_id"`
}

type OfflineTaskRequest struct {
	Reason          string `json:"reason" binding:"required"`
	RevokeRemaining bool   `json:"revoke_remaining"`
	RequestID       string `json:"request_id"`
}

type OnlineTaskRequest struct {
	InitialAllocation int64  `json:"initial_allocation" binding:"required,gt=0"`
	RequestID         string `json:"request_id"`
}

func (h *Handler) ListTasks(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.tasks.List(c.Request.Context(), growthservice.BudgetListFilter{
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}, c.Query("category"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) GetTask(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	item, err := h.tasks.Get(c.Request.Context(), taskID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) CreateTask(c *gin.Context) {
	var request CreateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid task request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.tasks.Create(c.Request.Context(), growthservice.TaskCreateInput{
		OperatorID:        operatorID,
		TaskCode:          request.TaskCode,
		Title:             request.Title,
		Description:       request.Description,
		Category:          request.Category,
		TaskType:          request.TaskType,
		Icon:              request.Icon,
		SortOrder:         request.SortOrder,
		InitialAllocation: request.InitialAllocation,
		Rule:              request.Rule,
		RequestID:         requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	var request UpdateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid task request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.tasks.Update(c.Request.Context(), taskID, growthservice.TaskUpdateInput{
		OperatorID:  operatorID,
		Title:       request.Title,
		Description: request.Description,
		Category:    request.Category,
		TaskType:    request.TaskType,
		Icon:        request.Icon,
		SortOrder:   request.SortOrder,
		Rule:        request.Rule,
		RequestID:   requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) PublishTask(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	var request PublishTaskRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			response.BadRequest(c, "invalid publish request: "+err.Error())
			return
		}
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.tasks.Publish(c.Request.Context(), taskID, operatorID, request.EffectiveFrom, requestRequestID(c, request.RequestID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) OfflineTask(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	var request OfflineTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid offline request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.tasks.Offline(c.Request.Context(), taskID, operatorID, request.Reason, request.RevokeRemaining, requestRequestID(c, request.RequestID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) OnlineTask(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	var request OnlineTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid online request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.tasks.Online(c.Request.Context(), taskID, operatorID, request.InitialAllocation, requestRequestID(c, request.RequestID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) TaskRuleVersions(c *gin.Context) {
	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}
	items, err := h.tasks.RuleVersions(c.Request.Context(), taskID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func taskIDParam(c *gin.Context) (int64, bool) {
	taskID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || taskID <= 0 {
		response.BadRequest(c, "invalid task id")
		return 0, false
	}
	return taskID, true
}
