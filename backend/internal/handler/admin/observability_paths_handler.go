package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

func (h *OpsHandler) GetObservabilityPaths(c *gin.Context) {
	if h == nil || h.observabilityPath == nil {
		response.Error(c, http.StatusServiceUnavailable, "Observability path service not available")
		return
	}
	query, err := parseObservabilityPathQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	payload, err := h.observabilityPath.Paths(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *OpsHandler) GetObservabilityRelations(c *gin.Context) {
	if h == nil || h.observabilityPath == nil {
		response.Error(c, http.StatusServiceUnavailable, "Observability path service not available")
		return
	}
	query, err := parseObservabilityPathQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	payload, err := h.observabilityPath.Relations(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func parseObservabilityPathQuery(c *gin.Context) (service.ObservabilityPathQuery, error) {
	resolved, err := parseObservabilityRange(c)
	if err != nil {
		return service.ObservabilityPathQuery{}, err
	}
	filter, err := observability.ParseFilter(c.Request.URL.Query())
	if err != nil {
		return service.ObservabilityPathQuery{}, err
	}
	page, pageSize := response.ParsePagination(c)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > observability.MaxPageSize {
		pageSize = observability.DefaultPageSize
	}
	if resolved.End.Sub(resolved.Start) >= observability.MaxRange {
		page = 1
		if pageSize > 100 {
			pageSize = 100
		}
	}
	includeIdle, _ := strconv.ParseBool(c.DefaultQuery("include_idle", "false"))
	return service.ObservabilityPathQuery{
		Start: resolved.Start, End: resolved.End, Window: resolved.Window, Bucket: resolved.Bucket, Filter: filter,
		Search: strings.TrimSpace(c.Query("search")), Status: strings.TrimSpace(c.Query("status")),
		Sort: strings.TrimSpace(c.Query("sort")), Page: page, PageSize: pageSize, IncludeIdle: includeIdle,
	}, nil
}
