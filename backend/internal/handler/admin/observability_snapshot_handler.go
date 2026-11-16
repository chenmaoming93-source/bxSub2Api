package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
)

// GetObservabilitySnapshot returns the single first-screen read model.
// GET /api/v1/admin/observability/snapshot
func (h *OpsHandler) GetObservabilitySnapshot(c *gin.Context) {
	if h == nil || h.observabilitySnapshot == nil {
		response.Error(c, http.StatusServiceUnavailable, "Observability snapshot service not available")
		return
	}
	resolved, err := parseObservabilityRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	filter, err := observability.ParseFilter(c.Request.URL.Query())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	limit := observability.DefaultPageSize
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > observability.MaxFilterValues {
			response.BadRequest(c, "limit must be between 1 and 50")
			return
		}
		limit = value
	}
	scope := "anonymous"
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		scope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}
	result, err := h.observabilitySnapshot.Snapshot(c.Request.Context(), service.SnapshotRequest{
		Window: resolved.Window, Bucket: resolved.Bucket, Start: resolved.Start, End: resolved.End,
		Filter: filter, Scope: scope,
		Search: strings.TrimSpace(c.Query("search")), OptionLimit: limit,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("ETag", result.ETag)
	c.Header("Vary", "Authorization, If-None-Match")
	c.Header("Cache-Control", "private, max-age=5")
	if ifNoneMatchMatched(c.GetHeader("If-None-Match"), result.ETag) {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, result.Response)
}
