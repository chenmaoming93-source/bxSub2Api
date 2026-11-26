package growth

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GrowthEndpointPending is the temporary handler used while the U-face contract
// gateway is verified independently from its business read/command slices.
func (h *Handler) GrowthEndpointPending(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, "growth endpoint is not implemented yet")
}
