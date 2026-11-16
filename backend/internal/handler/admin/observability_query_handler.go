package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
)

func (h *OpsHandler) GetObservabilityTimeseries(c *gin.Context) {
	h.handleObservabilityQuery(c, "timeseries")
}

func (h *OpsHandler) GetObservabilityLatencyTimeseries(c *gin.Context) {
	h.handleObservabilityQuery(c, "latency")
}

func (h *OpsHandler) GetObservabilityHeatmap(c *gin.Context) {
	h.handleObservabilityQuery(c, "heatmap")
}

func (h *OpsHandler) handleObservabilityQuery(c *gin.Context, kind string) {
	if h == nil || h.observabilityQuery == nil {
		response.Error(c, http.StatusServiceUnavailable, "Observability query service not available")
		return
	}
	resolved, err := parseObservabilityRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	filter, err := observability.ParseFilter(c.Request.URL.Query())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	scope := "anonymous"
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		scope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}
	request := service.ObservabilityQueryRequest{
		Window: resolved.Window, Bucket: resolved.Bucket, Start: resolved.Start, End: resolved.End,
		Filter: filter, Scope: scope,
		GroupBy:   strings.TrimSpace(c.Query("group_by")),
		ValueKind: strings.TrimSpace(c.Query("value_kind")),
	}
	if request.GroupBy != "" && !validObservabilityGroupBy(request.GroupBy, kind) {
		response.BadRequest(c, "unsupported group_by")
		return
	}
	if kind == "heatmap" {
		if request.ValueKind == "" {
			request.ValueKind = "sampled_concurrency"
		}
		if request.ValueKind != "sampled_concurrency" && request.ValueKind != "token_rate" && request.ValueKind != "error_rate" {
			response.BadRequest(c, "unsupported value_kind")
			return
		}
	}
	c.Header("Cache-Control", "private, max-age=5")
	switch kind {
	case "timeseries":
		payload, queryErr := h.observabilityQuery.Timeseries(c.Request.Context(), request)
		if queryErr != nil {
			h.observabilityQueryError(c, queryErr)
			return
		}
		response.Success(c, payload)
	case "latency":
		payload, queryErr := h.observabilityQuery.LatencyTimeseries(c.Request.Context(), request)
		if queryErr != nil {
			h.observabilityQueryError(c, queryErr)
			return
		}
		response.Success(c, payload)
	case "heatmap":
		payload, queryErr := h.observabilityQuery.Heatmap(c.Request.Context(), request)
		if queryErr != nil {
			h.observabilityQueryError(c, queryErr)
			return
		}
		response.Success(c, payload)
	default:
		response.BadRequest(c, "Unsupported observability query")
	}
}

func parseObservabilityRange(c *gin.Context) (observability.ResolvedRange, error) {
	return observability.ResolveRange(
		observability.Window(strings.TrimSpace(c.DefaultQuery("window", string(observability.Window1h)))),
		observability.Bucket(strings.TrimSpace(c.Query("bucket"))),
		strings.TrimSpace(c.Query("start_at")),
		strings.TrimSpace(c.Query("end_at")),
		time.Now().UTC(),
		timezone.Location(),
	)
}

func (h *OpsHandler) observabilityQueryError(c *gin.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		response.Error(c, http.StatusGatewayTimeout, "OBSERVABILITY_QUERY_TIMEOUT")
		return
	}
	response.ErrorFrom(c, err)
}

func validObservabilityGroupBy(value, kind string) bool {
	switch value {
	case "user_id", "group_id", "route_alias", "account_id":
	default:
		return false
	}
	return true
}
