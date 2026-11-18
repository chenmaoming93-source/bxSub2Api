package growth

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type ReadService interface {
	UOverview(context.Context, int64) (growthservice.UOverviewView, error)
	WOverview(context.Context, int64) (growthservice.WOverviewView, error)
	UTasks(context.Context, int64, int, int, string) ([]growthservice.UTaskItemView, int64, error)
	UTask(context.Context, int64, int64) (growthservice.UTaskItemView, error)
	WTasks(context.Context, int64) ([]growthservice.WTaskItemView, error)
	UProducts(context.Context, int64, int, int) ([]growthservice.UProductItemView, int64, error)
	UProduct(context.Context, int64, int64) (growthservice.UProductDetailView, error)
	WProducts(context.Context, int64) ([]growthservice.WProductItemView, error)
	WProduct(context.Context, int64, int64) (growthservice.WProductItemView, error)
	UExchanges(context.Context, int64, int, int, string) ([]growthservice.UExchangeItemView, int64, error)
	WExchanges(context.Context, int64, int, int, string) ([]growthservice.WExchangeItemView, int64, error)
	Ledger(context.Context, int64, int64, int, string) (growthservice.LedgerPage, error)
}

type pageParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type taskListParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Status   string `json:"status"`
}

type taskParams struct {
	TaskID int64 `json:"task_id"`
}

type productParams struct {
	ProductID       int64 `json:"product_id"`
	ExpectedVersion *int  `json:"expected_version,omitempty"`
}

type ledgerParams struct {
	BeforeEntryID int64  `json:"before_entry_id"`
	Limit         int    `json:"limit"`
	Type          string `json:"type"`
}

type exchangeListParams struct {
	Page            int    `json:"page"`
	PageSize        int    `json:"page_size"`
	Status          string `json:"status"`
	ProductID       int64  `json:"product_id"`
	ExpectedVersion *int   `json:"expected_version"`
	RequestID       string `json:"request_id"`
}

func (h *Handler) UMe(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	item, err := h.reads.UOverview(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UTasks(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	var params taskListParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	page, pageSize := normalizeReadPage(params.Page, params.PageSize)
	items, total, err := h.reads.UTasks(c.Request.Context(), userID, page, pageSize, params.Status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) UTask(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.BadRequest(c, "invalid task id")
		return
	}
	var params taskParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if params.TaskID != taskID {
		response.BadRequest(c, "task_id path and payload mismatch")
		return
	}
	item, err := h.reads.UTask(c.Request.Context(), userID, taskID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UProducts(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	var params pageParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	page, pageSize := normalizeReadPage(params.Page, params.PageSize)
	items, total, err := h.reads.UProducts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) UProduct(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		response.BadRequest(c, "invalid product id")
		return
	}
	var params productParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if params.ProductID != productID {
		response.BadRequest(c, "product_id path and payload mismatch")
		return
	}
	item, err := h.reads.UProduct(c.Request.Context(), userID, productID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) ULedger(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	var params ledgerParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	page, err := h.reads.Ledger(c.Request.Context(), userID, params.BeforeEntryID, params.Limit, params.Type)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

func (h *Handler) UExchanges(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	var params exchangeListParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if params.ProductID > 0 {
		payload, ok := servermiddleware.GetGrowthPayloadFromContext(c)
		if !ok || payload == nil {
			response.Unauthorized(c, "growth payload is unavailable")
			return
		}
		executeGrowthPayloadIdempotent(
			c,
			"growth.exchange",
			params,
			payload.IdempotencyKey,
			func(ctx context.Context) (any, error) {
				return h.orders.Create(ctx, growthservice.ExchangeCommand{
					UserID: userID, ProductID: params.ProductID,
					ExpectedVersion: params.ExpectedVersion,
					RequestID:       params.RequestID, IdempotencyKey: payload.IdempotencyKey,
				})
			},
		)
		return
	}
	page, pageSize := normalizeReadPage(params.Page, params.PageSize)
	items, total, err := h.reads.UExchanges(c.Request.Context(), userID, page, pageSize, params.Status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) WOverview(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	item, err := h.reads.WOverview(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) WTasks(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	items, err := h.reads.WTasks(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) WProducts(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	items, err := h.reads.WProducts(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) WProduct(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID <= 0 {
		response.BadRequest(c, "invalid product id")
		return
	}
	item, err := h.reads.WProduct(c.Request.Context(), userID, productID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) WExchanges(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
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

func (h *Handler) WLedger(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
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

func growthSubjectUserID(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authenticated user is required")
		return 0, false
	}
	return subject.UserID, true
}

func decodeGrowthParams(c *gin.Context, target any) error {
	payload, ok := servermiddleware.GetGrowthPayloadFromContext(c)
	if !ok || payload == nil {
		return fmt.Errorf("growth payload is unavailable")
	}
	if len(payload.Params) == 0 || string(payload.Params) == "null" {
		payload.Params = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(payload.Params, target); err != nil {
		return fmt.Errorf("invalid growth params: %w", err)
	}
	return nil
}

func normalizeReadPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
