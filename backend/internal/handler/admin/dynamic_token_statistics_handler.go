package admin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/gin-gonic/gin"
)

type DynamicTokenStatisticsHandler struct {
	service    *tokenstat.ProjectionAdminService
	controller *tokenstat.RuntimeController
}

func NewDynamicTokenStatisticsHandler(service *tokenstat.ProjectionAdminService, controller *tokenstat.RuntimeController) *DynamicTokenStatisticsHandler {
	return &DynamicTokenStatisticsHandler{service: service, controller: controller}
}

type runtimeRequest struct {
	Enabled bool `json:"enabled"`
}

type projectionRequest struct {
	Name           string                    `json:"name" binding:"required"`
	DimensionCodes []tokenstat.DimensionCode `json:"dimension_codes" binding:"required"`
	MetricCodes    []tokenstat.MetricCode    `json:"metric_codes" binding:"required"`
}

type quotaRequest struct {
	Name            string                                               `json:"name" binding:"required"`
	DimensionCodes  []tokenstat.DimensionCode                            `json:"dimension_codes" binding:"required"`
	DimensionValues map[tokenstat.DimensionCode]tokenstat.DimensionValue `json:"dimension_values" binding:"required"`
	MetricCode      tokenstat.MetricCode                                 `json:"metric_code" binding:"required"`
	PeriodType      tokenstat.PeriodType                                 `json:"period_type" binding:"required"`
	LimitValue      int64                                                `json:"limit_value" binding:"required"`
	Mode            tokenstat.QuotaMode                                  `json:"mode" binding:"required"`
}

// quotaUpdateRequest deliberately contains only mutable quota fields. Reusing
// quotaRequest here would incorrectly require the immutable projection scope
// and period on every edit.
type quotaUpdateRequest struct {
	Name       string              `json:"name" binding:"required"`
	LimitValue int64               `json:"limit_value" binding:"required"`
	Mode       tokenstat.QuotaMode `json:"mode" binding:"required"`
}

type usageQueryRequest struct {
	ProjectionID int64                                                `json:"projection_id" binding:"required"`
	MetricCode   tokenstat.MetricCode                                 `json:"metric_code" binding:"required"`
	PeriodType   tokenstat.PeriodType                                 `json:"period_type" binding:"required"`
	Start        time.Time                                            `json:"start" binding:"required"`
	End          time.Time                                            `json:"end" binding:"required"`
	Filters      map[tokenstat.DimensionCode]tokenstat.DimensionValue `json:"filters"`
	GroupBy      []tokenstat.DimensionCode                            `json:"group_by"`
	Sort         string                                               `json:"sort"`
	Page         int                                                  `json:"page"`
	PageSize     int                                                  `json:"page_size"`
	Format       string                                               `json:"format"`
}

func (h *DynamicTokenStatisticsHandler) Dimensions(c *gin.Context) {
	response.Success(c, gin.H{"dimensions": tokenstat.ConfigurableDimensions()})
}

func (h *DynamicTokenStatisticsHandler) Metrics(c *gin.Context) {
	response.Success(c, gin.H{"metrics": tokenstat.Metrics()})
}

func (h *DynamicTokenStatisticsHandler) RuntimeState(c *gin.Context) {
	response.Success(c, h.controller.State())
}

func (h *DynamicTokenStatisticsHandler) UpdateRuntimeState(c *gin.Context) {
	var request runtimeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.controller.SetEnabled(c.Request.Context(), request.Enabled); err != nil {
		response.Error(c, 503, err.Error())
		return
	}
	response.Success(c, h.controller.State())
}

func (h *DynamicTokenStatisticsHandler) ListProjections(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, gin.H{"projections": items})
}

func (h *DynamicTokenStatisticsHandler) GetProjection(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, 404, "projection not found")
		return
	}
	response.Success(c, gin.H{"projection": item})
}

func (h *DynamicTokenStatisticsHandler) CreateProjection(c *gin.Context) {
	var request projectionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.Create(c.Request.Context(), tokenstat.ProjectionInput{
		Name: request.Name, DimensionCodes: request.DimensionCodes, MetricCodes: request.MetricCodes,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"projection": item})
}

func (h *DynamicTokenStatisticsHandler) UpdateProjection(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	var request projectionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.UpdateDraft(c.Request.Context(), id, tokenstat.ProjectionInput{
		Name: request.Name, DimensionCodes: request.DimensionCodes, MetricCodes: request.MetricCodes,
	})
	h.projectionResult(c, item, err)
}

func (h *DynamicTokenStatisticsHandler) PublishProjection(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	item, err := h.service.Publish(c.Request.Context(), id)
	h.projectionResult(c, item, err)
}

func (h *DynamicTokenStatisticsHandler) ActivateProjection(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	item, err := h.service.Activate(c.Request.Context(), id)
	h.projectionResult(c, item, err)
}

func (h *DynamicTokenStatisticsHandler) DisableProjection(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	item, err := h.service.Disable(c.Request.Context(), id)
	h.projectionResult(c, item, err)
}

func (h *DynamicTokenStatisticsHandler) ListQuotas(c *gin.Context) {
	items, err := h.service.ListQuotas(c.Request.Context())
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, gin.H{"quotas": items})
}

func (h *DynamicTokenStatisticsHandler) CreateQuota(c *gin.Context) {
	var request quotaRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.CreateQuota(c.Request.Context(), tokenstat.QuotaInput{
		Name: request.Name, DimensionCodes: request.DimensionCodes, DimensionValues: request.DimensionValues,
		MetricCode: request.MetricCode, PeriodType: request.PeriodType, LimitValue: request.LimitValue, Mode: request.Mode,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quota": item})
}

func (h *DynamicTokenStatisticsHandler) UpdateQuota(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	var request quotaUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.UpdateQuota(c.Request.Context(), id, tokenstat.QuotaInput{
		Name: request.Name, LimitValue: request.LimitValue, Mode: request.Mode,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quota": item})
}

func (h *DynamicTokenStatisticsHandler) EnableQuota(c *gin.Context) {
	h.setQuotaStatus(c, true)
}

func (h *DynamicTokenStatisticsHandler) DisableQuota(c *gin.Context) {
	h.setQuotaStatus(c, false)
}

func (h *DynamicTokenStatisticsHandler) DeleteQuota(c *gin.Context) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteQuota(c.Request.Context(), id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *DynamicTokenStatisticsHandler) QueryUsage(c *gin.Context) {
	var request usageQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.QueryUsage(c.Request.Context(), tokenstat.UsageQueryInput{
		ProjectionID: request.ProjectionID, MetricCode: request.MetricCode,
		PeriodType: request.PeriodType, Start: request.Start, End: request.End,
		Filters: request.Filters, GroupBy: request.GroupBy, Sort: request.Sort,
		Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if request.Format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="token-statistics.csv"`)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"period_start", "period_end", "dimensions", "value"})
		for _, row := range result.Rows {
			_ = writer.Write([]string{
				row.PeriodStart.Format(time.RFC3339), row.PeriodEnd.Format(time.RFC3339),
				fmt.Sprint(row.Dimensions), strconv.FormatInt(row.Value, 10),
			})
		}
		writer.Flush()
		return
	}
	response.Success(c, result)
}

func (h *DynamicTokenStatisticsHandler) SyncStatus(c *gin.Context) {
	status, err := h.service.GetSyncStatus(c.Request.Context())
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, status)
}

func (h *DynamicTokenStatisticsHandler) setQuotaStatus(c *gin.Context, enabled bool) {
	id, ok := projectionID(c)
	if !ok {
		return
	}
	item, err := h.service.SetQuotaStatus(c.Request.Context(), id, enabled)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quota": item})
}

func (h *DynamicTokenStatisticsHandler) projectionResult(c *gin.Context, item any, err error) {
	if err != nil {
		if errors.Is(err, tokenstat.ErrInvalidProjectionTransition) {
			response.Error(c, 409, err.Error())
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"projection": item})
}

func projectionID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid projection id")
		return 0, false
	}
	return id, true
}
