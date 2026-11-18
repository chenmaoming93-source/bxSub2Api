package growth

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
)

type CompletionService interface {
	Complete(context.Context, growthservice.CompletionCommand) (growthservice.CompletionResult, error)
}

type completionParams struct {
	TaskID        int64  `json:"task_id"`
	CompletionRef string `json:"completion_ref"`
	RequestID     string `json:"request_id"`
}

func (h *Handler) UCompletion(c *gin.Context) {
	userID, ok := growthSubjectUserID(c)
	if !ok {
		return
	}
	payload, ok := servermiddleware.GetGrowthPayloadFromContext(c)
	if !ok || payload == nil {
		response.Unauthorized(c, "growth payload is unavailable")
		return
	}
	var params completionParams
	if err := decodeGrowthParams(c, &params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	pathTaskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil || pathTaskID <= 0 || params.TaskID != pathTaskID {
		response.BadRequest(c, "task_id path and payload mismatch")
		return
	}
	if params.CompletionRef == "" {
		response.BadRequest(c, "completion_ref is required")
		return
	}
	executeGrowthPayloadIdempotent(
		c,
		"growth.task-completion",
		params,
		payload.IdempotencyKey,
		func(ctx context.Context) (any, error) {
			result, err := h.completions.Complete(ctx, growthservice.CompletionCommand{
				UserID: userID, TaskID: pathTaskID, CompletionRef: params.CompletionRef,
				RequestID: params.RequestID, IdempotencyKey: payload.IdempotencyKey,
			})
			if err != nil {
				return nil, err
			}
			return result, nil
		},
	)
}
