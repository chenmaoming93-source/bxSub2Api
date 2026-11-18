package growth

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type ProductService interface {
	Create(context.Context, growthservice.ProductCreateInput) (growthservice.ProductListView, error)
	Update(context.Context, int64, growthservice.ProductUpdateInput) (growthservice.ProductListView, error)
	Publish(context.Context, int64, int64, string) (growthservice.ProductListView, error)
	AdjustStock(context.Context, int64, growthservice.StockAdjustment) (growthservice.ProductListView, error)
	Offline(context.Context, int64, int64, string, string) (growthservice.ProductListView, error)
	List(context.Context, growthservice.BudgetListFilter, string) ([]growthservice.ProductListView, int64, error)
	Get(context.Context, int64) (growthservice.ProductListView, error)
	Versions(context.Context, int64) ([]growthservice.ProductVersionView, error)
}

type CreateProductRequest struct {
	ProductCode     string                            `json:"product_code" binding:"required"`
	Title           string                            `json:"title" binding:"required"`
	ProductType     string                            `json:"product_type" binding:"required"`
	Icon            string                            `json:"icon"`
	FulfillmentMode string                            `json:"fulfillment_mode" binding:"required"`
	SortOrder       int                               `json:"sort_order"`
	StockMode       string                            `json:"stock_mode" binding:"required"`
	InitialStock    int64                             `json:"initial_stock"`
	Version         growthservice.ProductVersionInput `json:"version" binding:"required"`
	RequestID       string                            `json:"request_id"`
}

type UpdateProductRequest struct {
	Title           string                            `json:"title" binding:"required"`
	Icon            string                            `json:"icon"`
	FulfillmentMode string                            `json:"fulfillment_mode" binding:"required"`
	SortOrder       int                               `json:"sort_order"`
	Version         growthservice.ProductVersionInput `json:"version" binding:"required"`
	RequestID       string                            `json:"request_id"`
}

type StockAdjustmentRequest struct {
	Delta     int64  `json:"delta" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

type OfflineProductRequest struct {
	Reason    string `json:"reason" binding:"required"`
	RequestID string `json:"request_id"`
}

func (h *Handler) ListProducts(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.products.List(c.Request.Context(), growthservice.BudgetListFilter{
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}, c.Query("product_type"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) GetProduct(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	item, err := h.products.Get(c.Request.Context(), productID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var request CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid product request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.products.Create(c.Request.Context(), growthservice.ProductCreateInput{
		OperatorID:      operatorID,
		ProductCode:     request.ProductCode,
		Title:           request.Title,
		ProductType:     request.ProductType,
		Icon:            request.Icon,
		FulfillmentMode: request.FulfillmentMode,
		SortOrder:       request.SortOrder,
		StockMode:       request.StockMode,
		InitialStock:    request.InitialStock,
		Version:         request.Version,
		RequestID:       requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	var request UpdateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid product request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.products.Update(c.Request.Context(), productID, growthservice.ProductUpdateInput{
		OperatorID:      operatorID,
		Title:           request.Title,
		Icon:            request.Icon,
		FulfillmentMode: request.FulfillmentMode,
		SortOrder:       request.SortOrder,
		Version:         request.Version,
		RequestID:       requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) PublishProduct(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.products.Publish(c.Request.Context(), productID, operatorID, requestRequestID(c, ""))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) AdjustProductStock(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	var request StockAdjustmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid stock request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.products.AdjustStock(c.Request.Context(), productID, growthservice.StockAdjustment{
		OperatorID: operatorID,
		Delta:      request.Delta,
		Reason:     request.Reason,
		RequestID:  requestRequestID(c, request.RequestID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) OfflineProduct(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	var request OfflineProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid offline request: "+err.Error())
		return
	}
	operatorID, ok := currentOperatorID(c)
	if !ok {
		response.Unauthorized(c, "administrator identity is required")
		return
	}
	item, err := h.products.Offline(c.Request.Context(), productID, operatorID, request.Reason, requestRequestID(c, request.RequestID))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) ProductVersions(c *gin.Context) {
	productID, ok := productIDParam(c)
	if !ok {
		return
	}
	items, err := h.products.Versions(c.Request.Context(), productID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func productIDParam(c *gin.Context) (int64, bool) {
	value, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || value <= 0 {
		response.BadRequest(c, "invalid product id")
		return 0, false
	}
	return value, true
}
