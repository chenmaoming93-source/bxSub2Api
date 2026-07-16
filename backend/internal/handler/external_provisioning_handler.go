package handler

import (
	"context"
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
	svc interface {
		EnsurePlatformKey(ctx context.Context, input service.EnsurePlatformKeyInput) (*service.EnsurePlatformKeyResult, error)
		ListGroupModelRoutes(ctx context.Context, input service.ListGroupModelRoutesInput) ([]service.GroupModelRouteProjection, error)
	}
}

// NewExternalProvisioningHandler creates an ExternalProvisioningHandler.
func NewExternalProvisioningHandler(svc interface {
	EnsurePlatformKey(ctx context.Context, input service.EnsurePlatformKeyInput) (*service.EnsurePlatformKeyResult, error)
	ListGroupModelRoutes(ctx context.Context, input service.ListGroupModelRoutesInput) ([]service.GroupModelRouteProjection, error)
}) *ExternalProvisioningHandler {
	return &ExternalProvisioningHandler{svc: svc}
}

type ListGroupModelRoutesRequest struct {
	GroupName string `json:"group_name" binding:"required"`
}

type GroupModelRouteResponse struct {
	RouteAlias     string   `json:"route_alias"`
	UpstreamModels []string `json:"upstream_models"`
}

type ListGroupModelRoutesResponse struct {
	GroupName string                    `json:"group_name"`
	Routes    []GroupModelRouteResponse `json:"routes"`
}

func (h *ExternalProvisioningHandler) ListGroupModelRoutes(c *gin.Context) {
	var req ListGroupModelRoutesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	req.GroupName = strings.TrimSpace(req.GroupName)
	if req.GroupName == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}

	routes, err := h.svc.ListGroupModelRoutes(c.Request.Context(), service.ListGroupModelRoutesInput{GroupName: req.GroupName})
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			auditGroupRoutes(c, req.GroupName, "failure", "group_not_found")
			response.ErrorFrom(c, service.ErrGroupNotFound)
			return
		}
		auditGroupRoutes(c, req.GroupName, "failure", "internal_error")
		response.InternalError(c, "failed to list group model routes")
		return
	}

	items := make([]GroupModelRouteResponse, 0, len(routes))
	for _, route := range routes {
		items = append(items, GroupModelRouteResponse{RouteAlias: route.RouteAlias, UpstreamModels: route.UpstreamModels})
	}
	auditGroupRoutes(c, req.GroupName, "success", "")
	response.Success(c, ListGroupModelRoutesResponse{GroupName: req.GroupName, Routes: items})
}

func auditGroupRoutes(c *gin.Context, groupName, result, reason string) {
	attrs := []any{
		slog.String("event", "provisioning.group_model_routes_list"),
		slog.String("group_name", groupName),
		slog.String("source_ip", clientIP(c)),
		slog.String("result", result),
	}
	if reason != "" {
		attrs = append(attrs, slog.String("reason", reason))
	}
	slog.Info("provisioning_group_model_routes_list", attrs...)
}

// EnsureAPIKeyRequest is the request body for POST /api/v1/integrations/api-keys/ensure.
type EnsureAPIKeyRequest struct {
	User      string `json:"user" binding:"required"`
	Platform  string `json:"platform" binding:"required"`
	GroupName string `json:"group_name" binding:"required"`
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
	GroupID     int64  `json:"group_id"`
	GroupName   string `json:"group_name"`
}

// EnsureAPIKey handles POST /api/v1/integrations/api-keys/ensure.
func (h *ExternalProvisioningHandler) EnsureAPIKey(c *gin.Context) {
	var req EnsureAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}

	req.User = strings.TrimSpace(req.User)
	req.Platform = strings.TrimSpace(req.Platform)
	req.GroupName = strings.TrimSpace(req.GroupName)
	if req.GroupName == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}

	result, err := h.svc.EnsurePlatformKey(c.Request.Context(), service.EnsurePlatformKeyInput{
		User:      req.User,
		Platform:  req.Platform,
		GroupName: req.GroupName,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			auditFailure(c, req.Platform, req.GroupName, "user_not_found")
			response.Error(c, http.StatusNotFound, "USER_NOT_FOUND")
			return
		}
		if errors.Is(err, service.ErrGroupNotFound) || errors.Is(err, service.ErrProvisioningGroupInactive) || errors.Is(err, service.ErrProvisioningSubscriptionGroup) || errors.Is(err, service.ErrProvisioningGroupNotAllowed) {
			auditFailure(c, req.Platform, req.GroupName, "group_rejected")
			response.ErrorFrom(c, err)
			return
		}
		auditFailure(c, req.Platform, req.GroupName, "internal_error")
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
		GroupID:     result.Group.ID,
		GroupName:   result.Group.Name,
	}

	auditSuccess(c, result.User.ID, req.Platform, result.Group.ID, result.Group.Name, result.UserCreated, result.KeyCreated)

	if result.UserCreated || result.KeyCreated {
		response.Created(c, resp)
		return
	}
	response.Success(c, resp)
}

func auditSuccess(c *gin.Context, userID int64, platform string, groupID int64, groupName string, userCreated, keyCreated bool) {
	slog.Info("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.Int64("user_id", userID),
		slog.String("platform", platform),
		slog.Int64("group_id", groupID),
		slog.String("group_name", groupName),
		slog.String("source_ip", clientIP(c)),
		slog.Bool("user_created", userCreated),
		slog.Bool("key_created", keyCreated),
		slog.String("result", "success"),
	)
}

func auditFailure(c *gin.Context, platform, groupName, reason string) {
	slog.Warn("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.String("platform", platform),
		slog.String("group_name", groupName),
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
