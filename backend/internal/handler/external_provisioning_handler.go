package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ExternalProvisioningHandler handles the external API-key provisioning endpoint.
type ExternalProvisioningHandler struct {
	svc *service.ExternalProvisioningService
}

// NewExternalProvisioningHandler creates an ExternalProvisioningHandler.
func NewExternalProvisioningHandler(svc *service.ExternalProvisioningService) *ExternalProvisioningHandler {
	return &ExternalProvisioningHandler{svc: svc}
}

// EnsureAPIKeyRequest is the request body for POST /api/v1/integrations/api-keys/ensure.
type EnsureAPIKeyRequest struct {
	User     string `json:"user" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

// EnsureAPIKeyResponse is the response body.
type EnsureAPIKeyResponse struct {
	APIKey      string `json:"api_key"`
	UserID      int64  `json:"user_id"`
	UserEmail   string `json:"user"`
	Username    string `json:"username"`
	Platform    string `json:"platform"`
	UserCreated bool   `json:"user_created"`
	KeyCreated  bool   `json:"key_created"`
}

// EnsureAPIKey handles POST /api/v1/integrations/api-keys/ensure.
func (h *ExternalProvisioningHandler) EnsureAPIKey(c *gin.Context) {
	var req EnsureAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	req.User = strings.TrimSpace(req.User)
	req.Platform = strings.TrimSpace(req.Platform)

	result, err := h.svc.EnsurePlatformKey(c.Request.Context(), service.EnsurePlatformKeyInput{
		User:     req.User,
		Platform:  req.Platform,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			auditFailure(c, req.Platform, "user_not_found")
			response.Error(c, http.StatusNotFound, "USER_NOT_FOUND")
			return
		}
		auditFailure(c, req.Platform, "internal_error")
		response.InternalError(c, "failed to ensure api key")
		return
	}

	// Cache-Control headers.
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	resp := EnsureAPIKeyResponse{
		APIKey:      result.APIKey.Key,
		UserID:      result.User.ID,
		UserEmail:   result.User.Email,
		Username:    result.User.Username,
		Platform:    req.Platform,
		UserCreated: result.UserCreated,
		KeyCreated:  result.KeyCreated,
	}

	auditSuccess(c, result.User.ID, req.Platform, result.UserCreated, result.KeyCreated)

	if result.UserCreated || result.KeyCreated {
		response.Created(c, resp)
		return
	}
	response.Success(c, resp)
}

func auditSuccess(c *gin.Context, userID int64, platform string, userCreated, keyCreated bool) {
	slog.Info("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.Int64("user_id", userID),
		slog.String("platform", platform),
		slog.String("source_ip", clientIP(c)),
		slog.Bool("user_created", userCreated),
		slog.Bool("key_created", keyCreated),
		slog.String("result", "success"),
	)
}

func auditFailure(c *gin.Context, platform, reason string) {
	slog.Warn("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.String("platform", platform),
		slog.String("source_ip", clientIP(c)),
		slog.String("result", "failure"),
		slog.String("reason", reason),
	)
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		return "unknown"
	}
	return ip
}
