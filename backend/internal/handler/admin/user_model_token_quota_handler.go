package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UserModelTokenQuotaHandler struct {
	adminService service.AdminService
	service      *service.UserModelTokenQuotaAdminService
}

func NewUserModelTokenQuotaHandler(adminService service.AdminService, service *service.UserModelTokenQuotaAdminService) *UserModelTokenQuotaHandler {
	return &UserModelTokenQuotaHandler{adminService: adminService, service: service}
}

type userModelTokenQuotaResponse struct {
	UserID           int64  `json:"user_id"`
	Model            string `json:"model"`
	UsageDate        string `json:"usage_date"`
	UsedTokens       int64  `json:"used_tokens"`
	DailyLimitTokens *int64 `json:"daily_limit_tokens"`
}

type updateUserModelTokenQuotasRequest struct {
	Quotas []userModelTokenQuotaInput `json:"quotas" binding:"required"`
}

type userModelTokenQuotaInput struct {
	Model            string `json:"model" binding:"required"`
	DailyLimitTokens *int64 `json:"daily_limit_tokens"`
}

func (h *UserModelTokenQuotaHandler) List(c *gin.Context) {
	userID, ok := parseUserModelQuotaUserID(c)
	if !ok {
		return
	}
	if !h.ensureUserExists(c, userID) {
		return
	}
	records, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quotas": userModelTokenQuotaResponses(records)})
}

func (h *UserModelTokenQuotaHandler) Update(c *gin.Context) {
	userID, ok := parseUserModelQuotaUserID(c)
	if !ok {
		return
	}
	if !h.ensureUserExists(c, userID) {
		return
	}
	var req updateUserModelTokenQuotasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inputs := make([]service.UserModelDailyTokenQuotaInput, 0, len(req.Quotas))
	for _, quota := range req.Quotas {
		inputs = append(inputs, service.UserModelDailyTokenQuotaInput{
			Model:            quota.Model,
			DailyLimitTokens: quota.DailyLimitTokens,
		})
	}
	records, err := h.service.Upsert(c.Request.Context(), userID, inputs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"quotas": userModelTokenQuotaResponses(records)})
}

func (h *UserModelTokenQuotaHandler) ensureUserExists(c *gin.Context, userID int64) bool {
	if h == nil || h.service == nil {
		response.Error(c, 503, "user model token quota service not available")
		return false
	}
	if h.adminService == nil {
		response.Error(c, 503, "admin service not available")
		return false
	}
	if _, err := h.adminService.GetUser(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	return true
}

func parseUserModelQuotaUserID(c *gin.Context) (int64, bool) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return 0, false
	}
	return userID, true
}

func userModelTokenQuotaResponses(records []service.UserModelDailyTokenQuotaRecord) []userModelTokenQuotaResponse {
	out := make([]userModelTokenQuotaResponse, 0, len(records))
	for _, record := range records {
		out = append(out, userModelTokenQuotaResponse{
			UserID:           record.UserID,
			Model:            record.Model,
			UsageDate:        record.UsageDate.Format("2006-01-02"),
			UsedTokens:       record.UsedTokens,
			DailyLimitTokens: record.DailyLimitTokens,
		})
	}
	return out
}
