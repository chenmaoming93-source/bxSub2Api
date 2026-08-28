package admin

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

type SceneAccountDailyUsageQuerier interface {
	QuerySceneAccountDailyUsage(ctx context.Context, input service.SceneAccountDailyUsageInput) (*service.SceneAccountDailyUsageResult, error)
}

type SceneAccountDailyUsageHandler struct {
	service SceneAccountDailyUsageQuerier
}

func NewSceneAccountDailyUsageHandler(querier *service.SceneAccountDailyUsageService) *SceneAccountDailyUsageHandler {
	return &SceneAccountDailyUsageHandler{service: querier}
}

func NewSceneAccountDailyUsageHandlerWithQuerier(querier SceneAccountDailyUsageQuerier) *SceneAccountDailyUsageHandler {
	return &SceneAccountDailyUsageHandler{service: querier}
}

func (h *SceneAccountDailyUsageHandler) QueryDaily(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "SCENE_USAGE_SERVICE_UNAVAILABLE")
		return
	}
	input := service.SceneAccountDailyUsageInput{
		StartDate: strings.TrimSpace(c.Query("start_date")),
		EndDate:   strings.TrimSpace(c.Query("end_date")),
		GroupName: strings.TrimSpace(c.Query("group_name")),
	}
	result, err := h.service.QuerySceneAccountDailyUsage(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		slog.Warn("scene_account_daily_usage_query", "source", "admin", "result", "failure", "reason", errors.Reason(err), "group_name", input.GroupName)
		return
	}
	slog.Info("scene_account_daily_usage_query", "source", "admin", "result", "success", "projection_id", result.ProjectionID, "start_date", result.StartDate, "end_date", result.EndDate, "complete", result.Complete, "days", len(result.Days))
	response.Success(c, result)
}
