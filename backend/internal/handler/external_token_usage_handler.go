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

type ExternalTokenUsageQuerier interface {
	QueryCurrentUsage(context.Context, service.ExternalTokenUsageInput) (service.ExternalTokenUsageResult, error)
}

type ExternalTokenUsageHandler struct{ service ExternalTokenUsageQuerier }

func NewExternalTokenUsageHandler(service *service.ExternalTokenUsageService) *ExternalTokenUsageHandler {
	return &ExternalTokenUsageHandler{service: service}
}

func NewExternalTokenUsageHandlerWithQuerier(service ExternalTokenUsageQuerier) *ExternalTokenUsageHandler {
	return &ExternalTokenUsageHandler{service: service}
}

type ExternalTokenUsageRequest struct {
	Username   string `json:"username" binding:"required"`
	GroupName  string `json:"group_name" binding:"required"`
	APIKey     string `json:"api_key" binding:"required"`
	RouteAlias string `json:"route_alias" binding:"required"`
}

// maskAPIKey 对 API Key 明文做脱敏，响应与日志不得回显完整凭证。
func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

type ExternalTokenUsageResponse struct {
	Query              ExternalTokenUsageRequest            `json:"query"`
	ResolvedDimensions service.ExternalTokenUsageDimensions `json:"resolved_dimensions"`
	Metric             string                               `json:"metric"`
	Timezone           string                               `json:"timezone"`
	Periods            struct {
		Day   service.ExternalTokenUsagePeriodResult `json:"day"`
		Week  service.ExternalTokenUsagePeriodResult `json:"week"`
		Month service.ExternalTokenUsagePeriodResult `json:"month"`
	} `json:"periods"`
}

func (h *ExternalTokenUsageHandler) Query(c *gin.Context) {
	var request ExternalTokenUsageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	request.GroupName = strings.TrimSpace(request.GroupName)
	request.APIKey = strings.TrimSpace(request.APIKey)
	request.RouteAlias = strings.TrimSpace(request.RouteAlias)
	if request.Username == "" || len(request.Username) > 255 || request.GroupName == "" || len(request.GroupName) > 100 || request.APIKey == "" || len(request.APIKey) > 128 || request.RouteAlias == "" || len(request.RouteAlias) > 255 {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "TOKEN_USAGE_UNAVAILABLE")
		return
	}
	result, err := h.service.QueryCurrentUsage(c.Request.Context(), service.ExternalTokenUsageInput{Username: request.Username, GroupName: request.GroupName, APIKey: request.APIKey, RouteAlias: request.RouteAlias})
	if err != nil {
		if errors.Is(err, service.ErrTokenUsageUnavailable) {
			response.Error(c, http.StatusServiceUnavailable, "TOKEN_USAGE_UNAVAILABLE")
		} else {
			response.ErrorFrom(c, err)
		}
		slog.Warn("integration_token_usage_query", "event", "integration.token_usage_query", "source_ip", clientIP(c), "result", "failure", "reason", err.Error())
		return
	}
	echo := request
	echo.APIKey = maskAPIKey(echo.APIKey)
	out := ExternalTokenUsageResponse{Query: echo, ResolvedDimensions: result.Dimensions, Metric: string(result.Metric), Timezone: result.Timezone}
	out.Periods.Day, out.Periods.Week, out.Periods.Month = result.Day, result.Week, result.Month
	slog.Info("integration_token_usage_query", "event", "integration.token_usage_query", "source_ip", clientIP(c), "user_id", result.Dimensions.UserID, "group_id", result.Dimensions.GroupID, "api_key_id", result.Dimensions.APIKeyID, "route_alias", result.Dimensions.RouteAlias, "result", "success")
	response.Success(c, out)
}
