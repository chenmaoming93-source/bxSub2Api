package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ExternalTokenUsageQuerier interface {
	QueryCurrentUsage(context.Context, service.ExternalTokenUsageInput) (service.ExternalTokenUsageResult, error)
	QueryDailyUsage(context.Context, service.ExternalDailyUsageInput) (service.ExternalDailyUsageResult, error)
	QueryDailyUsageFilled(context.Context, service.ExternalDailyUsageInput) (service.ExternalDailyUsageResult, error)
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

// maxDailyQueryDays matches tokenstat.maxQueryDays: custom day-level ranges
// cannot exceed 366 days.
const maxDailyQueryDays = 366

// ExternalDailyUsageRequest is the day-level range query input. Dates use
// YYYY-MM-DD and are interpreted as day boundaries in the statistics timezone.
type ExternalDailyUsageRequest struct {
	GroupName string `json:"group_name" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type ExternalDailyUsageResponse struct {
	Query               ExternalDailyUsageRequest       `json:"query"`
	ResolvedDimensions  map[string]int64                `json:"resolved_dimensions"`
	ProjectionID        int64                           `json:"projection_id"`
	Metric              string                          `json:"metric"`
	Timezone            string                          `json:"timezone"`
	DimensionConfigured bool                            `json:"dimension_configured"`
	ProjectionEnabledAt *time.Time                      `json:"projection_enabled_at,omitempty"`
	LastSyncedAt        *time.Time                      `json:"last_synced_at,omitempty"`
	Complete            bool                            `json:"complete"`
	Message             string                          `json:"message"`
	Days                []service.ExternalDailyUsageDay `json:"days"`
}

// parseDailyUsageRequest binds and validates the shared daily-range request.
// On failure it writes the error response and returns ok=false.
func parseDailyUsageRequest(c *gin.Context) (request ExternalDailyUsageRequest, start, end time.Time, ok bool) {
	var parsed ExternalDailyUsageRequest
	if err := c.ShouldBindJSON(&parsed); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return parsed, time.Time{}, time.Time{}, false
	}
	parsed.GroupName = strings.TrimSpace(parsed.GroupName)
	parsed.APIKey = strings.TrimSpace(parsed.APIKey)
	parsed.StartDate = strings.TrimSpace(parsed.StartDate)
	parsed.EndDate = strings.TrimSpace(parsed.EndDate)
	start, startErr := time.Parse("2006-01-02", parsed.StartDate)
	end, endErr := time.Parse("2006-01-02", parsed.EndDate)
	if parsed.GroupName == "" || len(parsed.GroupName) > 100 || parsed.APIKey == "" || len(parsed.APIKey) > 128 ||
		startErr != nil || endErr != nil || end.Before(start) || end.Sub(start) > maxDailyQueryDays*24*time.Hour {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return parsed, time.Time{}, time.Time{}, false
	}
	return parsed, start, end, true
}

func (h *ExternalTokenUsageHandler) DailyQuery(c *gin.Context) {
	request, start, end, ok := parseDailyUsageRequest(c)
	if !ok {
		return
	}
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "TOKEN_USAGE_UNAVAILABLE")
		return
	}
	result, err := h.service.QueryDailyUsage(c.Request.Context(), service.ExternalDailyUsageInput{GroupName: request.GroupName, APIKey: request.APIKey, StartDate: start, EndDate: end})
	if err != nil {
		response.ErrorFrom(c, err)
		slog.Warn("integration_token_usage_daily", "event", "integration.token_usage_daily", "source_ip", clientIP(c), "result", "failure", "reason", err.Error())
		return
	}
	echo := request
	echo.APIKey = maskAPIKey(echo.APIKey)
	out := ExternalDailyUsageResponse{
		Query:               echo,
		ResolvedDimensions:  map[string]int64{"group_id": result.GroupID, "api_key_id": result.APIKeyID},
		ProjectionID:        result.ProjectionID,
		Metric:              string(result.Metric),
		Timezone:            result.Timezone,
		DimensionConfigured: result.DimensionConfigured,
		ProjectionEnabledAt: result.ProjectionEnabledAt,
		LastSyncedAt:        result.LastSyncedAt,
		Complete:            result.Complete,
		Message:             result.Message,
		Days:                result.Days,
	}
	slog.Info("integration_token_usage_daily", "event", "integration.token_usage_daily", "source_ip", clientIP(c), "group_id", result.GroupID, "api_key_id", result.APIKeyID, "projection_id", result.ProjectionID, "dimension_configured", result.DimensionConfigured, "days", len(result.Days), "result", "success")
	response.Success(c, out)
}

// DailyQueryCSV downloads the same daily consumption as a zero-filled CSV
// (one row per day in the range, missing days as 0). The "统计维度未配置" state
// cannot be expressed as data rows, so it is returned as a 409 conflict
// instead of an all-zero download.
func (h *ExternalTokenUsageHandler) DailyQueryCSV(c *gin.Context) {
	request, start, end, ok := parseDailyUsageRequest(c)
	if !ok {
		return
	}
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "TOKEN_USAGE_UNAVAILABLE")
		return
	}
	result, err := h.service.QueryDailyUsageFilled(c.Request.Context(), service.ExternalDailyUsageInput{GroupName: request.GroupName, APIKey: request.APIKey, StartDate: start, EndDate: end})
	if err != nil {
		response.ErrorFrom(c, err)
		slog.Warn("integration_token_usage_daily_csv", "event", "integration.token_usage_daily_csv", "source_ip", clientIP(c), "result", "failure", "reason", err.Error())
		return
	}
	if !result.DimensionConfigured {
		response.ErrorFrom(c, service.ErrDailyStatisticsNotConfigured)
		slog.Warn("integration_token_usage_daily_csv", "event", "integration.token_usage_daily_csv", "source_ip", clientIP(c), "group_id", result.GroupID, "api_key_id", result.APIKeyID, "result", "statistics_not_configured")
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="token-usage-group-api-key-daily.csv"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"date", "total_tokens"})
	for _, day := range result.Days {
		_ = writer.Write([]string{day.Date, strconv.FormatInt(day.TotalTokens, 10)})
	}
	writer.Flush()
	slog.Info("integration_token_usage_daily_csv", "event", "integration.token_usage_daily_csv", "source_ip", clientIP(c), "group_id", result.GroupID, "api_key_id", result.APIKeyID, "projection_id", result.ProjectionID, "days", len(result.Days), "result", "success")
}
