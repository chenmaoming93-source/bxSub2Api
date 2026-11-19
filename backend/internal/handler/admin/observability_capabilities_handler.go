package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
)

// GetObservabilityCapabilities returns the read-only endpoint contract.
// GET /api/v1/admin/observability/capabilities
func (h *OpsHandler) GetObservabilityCapabilities(c *gin.Context) {
	payload := observability.Contract()
	etag := buildETagFromAny(payload)
	c.Header("ETag", etag)
	c.Header("Vary", "If-None-Match")
	if ifNoneMatchMatched(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, payload)
}
