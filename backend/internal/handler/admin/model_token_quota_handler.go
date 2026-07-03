package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelTokenQuotaHandler struct {
	service *service.ModelTokenQuotaAdminService
}

func NewModelTokenQuotaHandler(service *service.ModelTokenQuotaAdminService) *ModelTokenQuotaHandler {
	return &ModelTokenQuotaHandler{service: service}
}

type modelTokenQuotaResponse struct {
	Model            string `json:"model"`
	UsageDate        string `json:"usage_date"`
	UsedTokens       int64  `json:"used_tokens"`
	DailyLimitTokens *int64 `json:"daily_limit_tokens"`
}

type updateModelTokenQuotaRequest struct {
	Model            string `json:"model" binding:"required"`
	DailyLimitTokens *int64 `json:"daily_limit_tokens"`
}

func (h *ModelTokenQuotaHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, 503, "model token quota service not available")
		return
	}
	records, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	out := make([]modelTokenQuotaResponse, 0, len(records))
	for _, record := range records {
		out = append(out, modelTokenQuotaResponseFromRecord(record))
	}
	response.Success(c, gin.H{"quotas": out})
}

func (h *ModelTokenQuotaHandler) Update(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, 503, "model token quota service not available")
		return
	}
	var req updateModelTokenQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	record, err := h.service.Set(c.Request.Context(), strings.TrimSpace(req.Model), req.DailyLimitTokens)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quota": modelTokenQuotaResponseFromRecord(record)})
}

func modelTokenQuotaResponseFromRecord(record service.ModelDailyTokenQuotaRecord) modelTokenQuotaResponse {
	return modelTokenQuotaResponse{
		Model:            record.Model,
		UsageDate:        record.UsageDate.Format("2006-01-02"),
		UsedTokens:       record.UsedTokens,
		DailyLimitTokens: record.DailyLimitTokens,
	}
}
