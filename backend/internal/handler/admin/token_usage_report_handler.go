package admin

import (
	"errors"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type TokenUsageReportHandler struct {
	service *service.TokenUsageReportService
}

func NewTokenUsageReportHandler(service *service.TokenUsageReportService) *TokenUsageReportHandler {
	return &TokenUsageReportHandler{service: service}
}

func (h *TokenUsageReportHandler) GetModels(c *gin.Context) {
	query, err := parseModelTokenUsageQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	report, err := h.service.GetModelReport(c.Request.Context(), query)
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	items := make([]gin.H, 0, len(report.Items))
	for _, item := range report.Items {
		items = append(items, gin.H{"usage_date": item.UsageDate.Format("2006-01-02"), "model": item.Model, "used_tokens": item.UsedTokens, "daily_limit_tokens": item.DailyLimitTokens, "usage_rate": item.UsageRate, "status": item.Status})
	}
	response.Success(c, gin.H{"items": items, "summary": gin.H{"used_tokens": report.UsedTokens}, "pagination": gin.H{"page": report.Page, "page_size": report.PageSize, "total": report.Total}})
}

func (h *TokenUsageReportHandler) GetRoutes(c *gin.Context) {
	base, err := parseTokenUsageBase(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	groupID, err := optionalInt64Query(c, "group_id")
	if err != nil {
		response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
		return
	}
	report, err := h.service.GetRouteReport(c.Request.Context(), service.RouteTokenUsageReportQuery{TokenUsageReportQuery: base, GroupID: groupID, RouteAlias: c.Query("route_alias"), UpstreamModel: c.Query("upstream_model")})
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	items := make([]gin.H, 0, len(report.Items))
	for _, x := range report.Items {
		items = append(items, gin.H{"usage_date": x.UsageDate.Format("2006-01-02"), "group_id": x.GroupID, "group_name": x.GroupName, "route_alias": x.RouteAlias, "upstream_model": x.UpstreamModel, "priority": x.Priority, "used_tokens": x.UsedTokens, "daily_limit_tokens": x.DailyLimitTokens, "usage_rate": x.UsageRate, "status": x.Status})
	}
	writeReport(c, items, report.UsedTokens, report.Page, report.PageSize, report.Total)
}

func (h *TokenUsageReportHandler) GetUsers(c *gin.Context) {
	base, err := parseTokenUsageBase(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, err := optionalInt64Query(c, "user_id")
	if err != nil {
		response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
		return
	}
	base.Model = c.Query("model")
	includeDeleted, err := optionalBoolQuery(c, "include_deleted")
	if err != nil {
		response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
		return
	}
	report, err := h.service.GetUserReport(c.Request.Context(), service.UserTokenUsageReportQuery{TokenUsageReportQuery: base, UserID: userID, IncludeDeleted: includeDeleted})
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	items := make([]gin.H, 0, len(report.Items))
	for _, x := range report.Items {
		items = append(items, gin.H{"usage_date": x.UsageDate.Format("2006-01-02"), "user_id": x.UserID, "email": x.Email, "username": x.Username, "user_deleted": x.UserDeleted, "model": x.Model, "used_tokens": x.UsedTokens, "daily_limit_tokens": x.DailyLimitTokens, "usage_rate": x.UsageRate, "status": x.Status})
	}
	writeReport(c, items, report.UsedTokens, report.Page, report.PageSize, report.Total)
}

func (h *TokenUsageReportHandler) GetOptions(c *gin.Context) {
	kind := c.Param("kind")
	parentID := int64(0)
	if raw := c.Param("parent_id"); raw != "" {
		var err error
		parentID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
			return
		}
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil {
			response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
			return
		}
	}
	items, err := h.service.SearchOptions(c.Request.Context(), kind, parentID, c.Query("q"), limit)
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *TokenUsageReportHandler) ModelsOptions(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "kind", Value: "models"})
	h.GetOptions(c)
}
func (h *TokenUsageReportHandler) GroupsOptions(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "kind", Value: "groups"})
	h.GetOptions(c)
}
func (h *TokenUsageReportHandler) RoutesOptions(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "kind", Value: "routes"}, gin.Param{Key: "parent_id", Value: c.Param("group_id")})
	h.GetOptions(c)
}
func (h *TokenUsageReportHandler) UsersOptions(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "kind", Value: "users"})
	h.GetOptions(c)
}
func (h *TokenUsageReportHandler) UserModelsOptions(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "kind", Value: "user_models"}, gin.Param{Key: "parent_id", Value: c.Param("user_id")})
	h.GetOptions(c)
}
func (h *TokenUsageReportHandler) RouteModelsOptions(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
		return
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
			return
		}
	}
	items, err := h.service.SearchOptions(c.Request.Context(), "route_models", groupID, c.Param("route_alias")+"\x00"+c.Query("q"), limit)
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *TokenUsageReportHandler) GetDefaultTarget(c *gin.Context) {
	date := timezone.Today()
	var err error
	if raw := c.Query("date"); raw != "" {
		date, err = timezone.ParseInLocation("2006-01-02", raw)
		if err != nil {
			response.BadRequest(c, service.ErrInvalidTokenUsageReportQuery.Error())
			return
		}
	}
	item, err := h.service.DefaultTarget(c.Request.Context(), c.Query("dimension"), date)
	if errors.Is(err, service.ErrInvalidTokenUsageReportQuery) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, gin.H{"target": item})
}

func writeReport(c *gin.Context, items any, used int64, page, pageSize int, total int64) {
	response.Success(c, gin.H{"items": items, "summary": gin.H{"used_tokens": used}, "pagination": gin.H{"page": page, "page_size": pageSize, "total": total}})
}

func parseModelTokenUsageQuery(c *gin.Context) (service.TokenUsageReportQuery, error) {
	q, err := parseTokenUsageBase(c)
	q.Model = c.Query("model")
	return q, err
}

func parseTokenUsageBase(c *gin.Context) (service.TokenUsageReportQuery, error) {
	startRaw, endRaw := c.Query("start_date"), c.Query("end_date")
	if startRaw == "" || endRaw == "" {
		return service.TokenUsageReportQuery{}, service.ErrInvalidTokenUsageReportQuery
	}
	var err error
	start, err := timezone.ParseInLocation("2006-01-02", startRaw)
	if err != nil {
		return service.TokenUsageReportQuery{}, service.ErrInvalidTokenUsageReportQuery
	}
	end, err := timezone.ParseInLocation("2006-01-02", endRaw)
	if err != nil {
		return service.TokenUsageReportQuery{}, service.ErrInvalidTokenUsageReportQuery
	}
	page, pageSize := 1, 20
	if raw := c.Query("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return service.TokenUsageReportQuery{}, service.ErrInvalidTokenUsageReportQuery
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return service.TokenUsageReportQuery{}, service.ErrInvalidTokenUsageReportQuery
		}
	}
	return service.TokenUsageReportQuery{StartDate: start, EndDate: end, Page: page, PageSize: pageSize, SortBy: defaultString(c.Query("sort_by"), "usage_date"), SortOrder: defaultString(c.Query("sort_order"), "desc")}, nil
}

func optionalInt64Query(c *gin.Context, key string) (int64, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func optionalBoolQuery(c *gin.Context, key string) (bool, error) {
	raw := c.Query(key)
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
