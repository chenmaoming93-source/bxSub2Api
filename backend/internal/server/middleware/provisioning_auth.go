package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

// ExternalProvisioningAuth protects only the external API-key provisioning routes.
func ExternalProvisioningAuth(cfg config.ExternalAPIKeyProvisioningConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(cfg.AccessToken)
		if !cfg.Enabled || expected == "" {
			AbortWithError(c, http.StatusNotFound, "NOT_FOUND", "Not found")
			return
		}

		header := c.GetHeader("Authorization")
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" ||
			subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
			AbortWithError(c, http.StatusUnauthorized, "INVALID_ACCESS_TOKEN", "Invalid access token")
			return
		}
		c.Next()
	}
}
