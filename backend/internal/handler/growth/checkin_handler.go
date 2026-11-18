package growth

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type CheckinService interface {
	Checkin(context.Context, int64, string) (growthservice.CheckinResult, error)
}

type checkinParams struct {
	RequestID string `json:"request_id"`
}

func (h *Handler) UCheckin(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	payload, ok := servermiddleware.GetGrowthPayloadFromContext(c)
	if !ok || payload == nil {
		response.Unauthorized(c, "growth payload is unavailable")
		return
	}
	if len(payload.Params) == 0 {
		payload.Params = json.RawMessage(`{}`)
	}
	var params checkinParams
	if err := json.Unmarshal(payload.Params, &params); err != nil {
		response.BadRequest(c, "invalid checkin params")
		return
	}
	executeGrowthPayloadIdempotent(
		c,
		"growth.checkin",
		map[string]any{"request_id": params.RequestID},
		payload.IdempotencyKey,
		func(ctx context.Context) (any, error) {
			return h.checkins.Checkin(ctx, userID, payload.IdempotencyKey)
		},
	)
}
