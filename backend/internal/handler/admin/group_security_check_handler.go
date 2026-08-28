package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// UpdateSecurityCheck updates only the group-level SingGuard policy.
// PUT /api/v1/admin/groups/:id/security-check
func (h *GroupHandler) SetSecurityCheckLogDependencies(store service.SecurityCheckLogStore, collector *service.SecurityCheckCollector) {
	h.securityCheckLogStore = store
	h.securityCheckCollector = collector
}

func (h *GroupHandler) ListSecurityCheckLogs(c *gin.Context) {
	if h.securityCheckLogStore == nil {
		response.Error(c, http.StatusServiceUnavailable, "security log store unavailable")
		return
	}
	page := 1
	pageSize := 20
	if value, err := strconv.Atoi(c.Query("page")); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(c.Query("page_size")); err == nil && value > 0 {
		pageSize = value
	}
	filter := service.SecurityCheckLogFilter{Page: page, PageSize: pageSize, Decision: c.Query("decision"), Status: c.Query("status")}
	if value, err := strconv.ParseInt(c.Query("group_id"), 10, 64); err == nil {
		filter.GroupID = value
	}
	if value, err := time.Parse(time.RFC3339, c.Query("from")); err == nil {
		filter.From = value
	}
	if value, err := time.Parse(time.RFC3339, c.Query("to")); err == nil {
		filter.To = value
	}
	result, err := h.securityCheckLogStore.ListSecurityCheckLogs(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GroupHandler) GetSecurityCheckLog(c *gin.Context) {
	if h.securityCheckLogStore == nil {
		response.Error(c, http.StatusServiceUnavailable, "security log store unavailable")
		return
	}
	id, err := strconv.ParseInt(c.Param("log_id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid log ID")
		return
	}
	result, err := h.securityCheckLogStore.GetSecurityCheckLog(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GroupHandler) SecurityCheckCollectionStatus(c *gin.Context) {
	if h.securityCheckCollector == nil {
		response.Success(c, map[string]any{"circuit_open": false, "failure_count": 0})
		return
	}
	response.Success(c, h.securityCheckCollector.SecurityCheckCollectionStatus())
}

func (h *GroupHandler) ReopenSecurityCheckCollection(c *gin.Context) {
	if h.securityCheckCollector != nil {
		h.securityCheckCollector.Reopen()
	}
	response.Success(c, map[string]any{"reopened": true})
}

func (h *GroupHandler) UpdateSecurityCheck(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	var config domain.SecurityCheckConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "Invalid security check config: "+err.Error())
		return
	}
	group, err := h.adminService.UpdateGroup(c.Request.Context(), groupID, &service.UpdateGroupInput{
		SecurityCheckConfig: &config,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.GroupFromServiceAdmin(group))
}
