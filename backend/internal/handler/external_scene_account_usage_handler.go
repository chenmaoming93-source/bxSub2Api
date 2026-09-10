package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ExternalSceneAccountDailyUsageQuerier interface {
	QuerySceneAccountDailyUsage(ctx context.Context, input service.SceneAccountDailyUsageInput) (*service.SceneAccountDailyUsageResult, error)
}

type ExternalSceneAccountDailyUsageHandler struct {
	service ExternalSceneAccountDailyUsageQuerier
}

func NewExternalSceneAccountDailyUsageHandler(querier *service.SceneAccountDailyUsageService) *ExternalSceneAccountDailyUsageHandler {
	return &ExternalSceneAccountDailyUsageHandler{service: querier}
}

func NewExternalSceneAccountDailyUsageHandlerWithQuerier(querier ExternalSceneAccountDailyUsageQuerier) *ExternalSceneAccountDailyUsageHandler {
	return &ExternalSceneAccountDailyUsageHandler{service: querier}
}

func (h *ExternalSceneAccountDailyUsageHandler) QueryDaily(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "SCENE_USAGE_SERVICE_UNAVAILABLE")
		return
	}
	var input service.SceneAccountDailyUsageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	input.StartDate = strings.TrimSpace(input.StartDate)
	input.EndDate = strings.TrimSpace(input.EndDate)
	input.GroupName = strings.TrimSpace(input.GroupName)
	result, err := h.service.QuerySceneAccountDailyUsage(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		slog.Warn("scene_account_daily_usage_query", "source", "external", "result", "failure", "reason", errors.Reason(err), "group_name", input.GroupName, "source_ip", clientIP(c))
		return
	}
	slog.Info("scene_account_daily_usage_query", "source", "external", "result", "success", "projection_id", result.ProjectionID, "start_date", result.StartDate, "end_date", result.EndDate, "complete", result.Complete, "days", len(result.Days), "source_ip", clientIP(c))
	response.Success(c, result)
}
