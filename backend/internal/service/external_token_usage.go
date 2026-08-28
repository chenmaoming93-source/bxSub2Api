package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

var ErrRouteAliasNotFound = infraerrors.NotFound("ROUTE_ALIAS_NOT_FOUND", "route alias not found")

// ErrAPIKeyMismatch 表示 API Key 值存在，但不属于请求中的用户或分组。
// 与 ErrAPIKeyNotFound（值不存在或已删除）区分，便于调用方定位参数组合问题；
// 消息不泄露该 Key 实际归属。
var ErrAPIKeyMismatch = infraerrors.BadRequest("API_KEY_MISMATCH", "api key does not belong to the given user or group")
var ErrTokenUsageUnavailable = errors.New("token usage unavailable")

type ExternalTokenUsageInput struct {
	Username   string
	GroupName  string
	APIKey     string
	RouteAlias string
}

type ExternalTokenUsageDimensions struct {
	UserID     int64  `json:"user_id"`
	GroupID    int64  `json:"group_id"`
	APIKeyID   int64  `json:"api_key_id"`
	RouteAlias string `json:"route_alias"`
}

type ExternalTokenUsageDimensionLookup interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindGroupByName(ctx context.Context, name string) (*Group, error)
	FindAPIKeyByKey(ctx context.Context, key string) (*APIKey, error)
}

type ExternalTokenUsageService struct {
	lookup       ExternalTokenUsageDimensionLookup
	reader       ExternalTokenUsageCurrentReader
	projections  ExternalTokenUsageProjectionProvider
	quotaRules   ExternalTokenUsageQuotaProvider
	historyQuery ExternalTokenUsageHistoryQuerier
	location     *time.Location
	now          func() time.Time
}

func NewExternalTokenUsageService(lookup ExternalTokenUsageDimensionLookup) *ExternalTokenUsageService {
	return &ExternalTokenUsageService{lookup: lookup, now: time.Now}
}

type ExternalTokenUsageCurrentValue struct {
	Exists bool
	Value  int64
}
type ExternalTokenUsageCurrentReader interface {
	Read(context.Context, tokenstat.Period, int64, [16]byte, tokenstat.MetricCode) (ExternalTokenUsageCurrentValue, error)
}
type ExternalTokenUsageProjectionProvider interface {
	ActiveProjections() []tokenstat.ProjectionDefinition
}

type ExternalTokenUsageQuotaProvider interface {
	ActiveQuotaRules() []tokenstat.QuotaRule
}

// ExternalTokenUsageHistoryQuerier queries MySQL token_stat_aggregates through
// the configurable token statistics projection service.
type ExternalTokenUsageHistoryQuerier interface {
	QueryUsage(ctx context.Context, input tokenstat.UsageQueryInput) (*tokenstat.UsageQueryResult, error)
}

type ExternalTokenUsagePeriodResult struct {
	PeriodType          tokenstat.PeriodType `json:"period_type"`
	PeriodStart         time.Time            `json:"period_start"`
	PeriodEnd           time.Time            `json:"period_end"`
	DimensionConfigured bool                 `json:"dimension_configured"`
	DataPresent         bool                 `json:"data_present"`
	TotalTokens         *int64               `json:"total_tokens"`
	EnforcedLimit       *int64               `json:"enforced_limit"`
	Message             string               `json:"message"`
}

type ExternalTokenUsageResult struct {
	Dimensions ExternalTokenUsageDimensions   `json:"resolved_dimensions"`
	Metric     tokenstat.MetricCode           `json:"metric"`
	Timezone   string                         `json:"timezone"`
	Day        ExternalTokenUsagePeriodResult `json:"day"`
	Week       ExternalTokenUsagePeriodResult `json:"week"`
	Month      ExternalTokenUsagePeriodResult `json:"month"`
}

func (s *ExternalTokenUsageService) ConfigureCurrentUsage(reader ExternalTokenUsageCurrentReader, projections ExternalTokenUsageProjectionProvider, location *time.Location) {
	s.reader, s.projections, s.location = reader, projections, location
}

func (s *ExternalTokenUsageService) ConfigureHistoryQuery(queryer ExternalTokenUsageHistoryQuerier) {
	s.historyQuery = queryer
}

func (s *ExternalTokenUsageService) ConfigureQuotaRules(provider ExternalTokenUsageQuotaProvider) {
	s.quotaRules = provider
}

func (s *ExternalTokenUsageService) SetClockForTest(now func() time.Time) { s.now = now }

func (s *ExternalTokenUsageService) ResolveDimensions(ctx context.Context, input ExternalTokenUsageInput) (ExternalTokenUsageDimensions, error) {
	if s == nil || s.lookup == nil {
		return ExternalTokenUsageDimensions{}, fmt.Errorf("external token usage dimension lookup is required")
	}
	email := strings.TrimSpace(input.Username)
	groupName := strings.TrimSpace(input.GroupName)
	apiKeyValue := strings.TrimSpace(input.APIKey)
	routeAlias := strings.TrimSpace(input.RouteAlias)

	user, err := s.lookup.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ExternalTokenUsageDimensions{}, ErrUserNotFound
		}
		return ExternalTokenUsageDimensions{}, fmt.Errorf("find token usage user: %w", err)
	}
	group, err := s.lookup.FindGroupByName(ctx, groupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return ExternalTokenUsageDimensions{}, ErrGroupNotFound
		}
		return ExternalTokenUsageDimensions{}, fmt.Errorf("find token usage group: %w", err)
	}
	// API Key 值在 api_keys.key 上具有数据库唯一约束，直接按值解析。
	// 仍校验归属：密钥必须属于目标用户和分组，否则视为不存在。
	key, err := s.lookup.FindAPIKeyByKey(ctx, apiKeyValue)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return ExternalTokenUsageDimensions{}, ErrAPIKeyNotFound
		}
		return ExternalTokenUsageDimensions{}, fmt.Errorf("find token usage api key: %w", err)
	}
	if key.UserID != user.ID || key.GroupID == nil || *key.GroupID != group.ID {
		return ExternalTokenUsageDimensions{}, ErrAPIKeyMismatch
	}
	foundAlias := false
	for _, candidate := range group.ModelRoutingRuleNames() {
		if candidate == routeAlias {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		return ExternalTokenUsageDimensions{}, ErrRouteAliasNotFound
	}
	return ExternalTokenUsageDimensions{UserID: user.ID, GroupID: group.ID, APIKeyID: key.ID, RouteAlias: routeAlias}, nil
}

func (s *ExternalTokenUsageService) QueryCurrentUsage(ctx context.Context, input ExternalTokenUsageInput) (ExternalTokenUsageResult, error) {
	dimensions, err := s.ResolveDimensions(ctx, input)
	if err != nil {
		return ExternalTokenUsageResult{}, err
	}
	if s.reader == nil || s.projections == nil || s.location == nil {
		return ExternalTokenUsageResult{}, fmt.Errorf("external token usage service is not configured")
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	at := now()
	periods := tokenstat.NaturalPeriods(at, s.location)
	result := ExternalTokenUsageResult{Dimensions: dimensions, Metric: tokenstat.MetricTotalTokens, Timezone: s.location.String()}
	available := map[tokenstat.DimensionCode]tokenstat.DimensionValue{
		tokenstat.DimensionUserID:     tokenstat.Int64Value(dimensions.UserID),
		tokenstat.DimensionAPIKeyID:   tokenstat.Int64Value(dimensions.APIKeyID),
		tokenstat.DimensionGroupID:    tokenstat.Int64Value(dimensions.GroupID),
		tokenstat.DimensionRouteAlias: tokenstat.StringValue(dimensions.RouteAlias),
	}
	var quotaRules []tokenstat.QuotaRule
	if s.quotaRules != nil && tokenstat.RuntimeEnabled() {
		quotaRules = s.quotaRules.ActiveQuotaRules()
	}
	enforcedLimit := func(periodType tokenstat.PeriodType) *int64 {
		limit, ok := tokenstat.MinEnforcedQuotaLimit(at, periodType, tokenstat.MetricTotalTokens, quotaRules, available)
		if !ok {
			return nil
		}
		return &limit
	}

	target := []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionRouteAlias}
	targetSignature, _ := tokenstat.DimensionSignature(target)
	var projection *tokenstat.ProjectionDefinition
	for _, candidate := range s.projections.ActiveProjections() {
		signature, sigErr := tokenstat.DimensionSignature(candidate.DimensionCodes)
		if sigErr == nil && signature == targetSignature && containsMetric(candidate.MetricCodes, tokenstat.MetricTotalTokens) {
			copy := candidate
			projection = &copy
			break
		}
	}
	if projection == nil {
		assignExternalPeriods(&result, periods, func(period tokenstat.Period) ExternalTokenUsagePeriodResult {
			return ExternalTokenUsagePeriodResult{PeriodType: period.Type, PeriodStart: period.Start, PeriodEnd: period.End, Message: "统计维度未配置"}
		})
		result.Day.EnforcedLimit = enforcedLimit(result.Day.PeriodType)
		result.Week.EnforcedLimit = enforcedLimit(result.Week.PeriodType)
		result.Month.EnforcedLimit = enforcedLimit(result.Month.PeriodType)
		return result, nil
	}
	identity, err := tokenstat.BuildDimensionIdentity(projection.DimensionCodes, map[tokenstat.DimensionCode]tokenstat.DimensionValue{
		tokenstat.DimensionUserID: tokenstat.Int64Value(dimensions.UserID), tokenstat.DimensionAPIKeyID: tokenstat.Int64Value(dimensions.APIKeyID),
		tokenstat.DimensionGroupID: tokenstat.Int64Value(dimensions.GroupID), tokenstat.DimensionRouteAlias: tokenstat.StringValue(dimensions.RouteAlias),
	})
	if err != nil {
		return ExternalTokenUsageResult{}, fmt.Errorf("build token usage identity: %w", err)
	}
	values := make([]ExternalTokenUsagePeriodResult, len(periods))
	for i, period := range periods {
		value, readErr := s.reader.Read(ctx, period, int64(projection.ID), identity.Hash, tokenstat.MetricTotalTokens)
		if readErr != nil {
			return ExternalTokenUsageResult{}, fmt.Errorf("%w: %v", ErrTokenUsageUnavailable, readErr)
		}
		tokens := value.Value
		values[i] = ExternalTokenUsagePeriodResult{PeriodType: period.Type, PeriodStart: period.Start, PeriodEnd: period.End, DimensionConfigured: true, DataPresent: value.Exists, TotalTokens: &tokens}
		values[i].EnforcedLimit = enforcedLimit(period.Type)
		if !value.Exists {
			values[i].Message = "统计维度已配置，当前周期暂无数据"
		}
	}
	result.Day, result.Week, result.Month = values[0], values[1], values[2]
	return result, nil
}

func containsMetric(codes []tokenstat.MetricCode, target tokenstat.MetricCode) bool {
	for _, code := range codes {
		if code == target {
			return true
		}
	}
	return false
}
func assignExternalPeriods(result *ExternalTokenUsageResult, periods []tokenstat.Period, build func(tokenstat.Period) ExternalTokenUsagePeriodResult) {
	result.Day, result.Week, result.Month = build(periods[0]), build(periods[1]), build(periods[2])
}

// ExternalDailyUsageInput is the daily-range token usage query. StartDate and
// EndDate carry only the parsed YYYY-MM-DD components; the service interprets
// them as day boundaries in the statistics timezone.
type ExternalDailyUsageInput struct {
	GroupName string
	APIKey    string
	StartDate time.Time
	EndDate   time.Time
}

// ExternalDailyUsageDay is one day with recorded token consumption. Days
// without data are omitted (never zero-filled).
type ExternalDailyUsageDay struct {
	Date        string `json:"date"`
	TotalTokens int64  `json:"total_tokens"`
}

// ExternalDailyUsageResult follows a three-state contract:
//   - projection not configured: DimensionConfigured=false, Days=[],
//     Message="统计维度未配置" (distinct from an empty data list);
//   - configured but no data in range: DimensionConfigured=true, Days=[];
//   - configured with data: Days contains only days that have rows, ascending.
type ExternalDailyUsageResult struct {
	GroupID             int64                   `json:"group_id"`
	APIKeyID            int64                   `json:"api_key_id"`
	ProjectionID        int64                   `json:"projection_id"`
	Metric              tokenstat.MetricCode    `json:"metric"`
	Timezone            string                  `json:"timezone"`
	DimensionConfigured bool                    `json:"dimension_configured"`
	ProjectionEnabledAt *time.Time              `json:"projection_enabled_at,omitempty"`
	LastSyncedAt        *time.Time              `json:"last_synced_at,omitempty"`
	Complete            bool                    `json:"complete"`
	Message             string                  `json:"message"`
	Days                []ExternalDailyUsageDay `json:"days"`
}

// ErrDailyStatisticsNotConfigured indicates that no projection with the exact
// api_key_id,group_id signature is active. The JSON endpoint represents this
// as dimension_configured=false; the CSV endpoint cannot express it as data
// rows, so it surfaces it as a 409 conflict instead of an all-zero download.
var ErrDailyStatisticsNotConfigured = infraerrors.Conflict("STATISTICS_NOT_CONFIGURED", "统计维度未配置")

// QueryDailyUsage returns per-day total_tokens consumption of one API Key
// within one group over a day-level range, reading MySQL token_stat_aggregates
// through the configurable statistics projection service. Days without data
// are omitted (never zero-filled).
func (s *ExternalTokenUsageService) QueryDailyUsage(ctx context.Context, input ExternalDailyUsageInput) (ExternalDailyUsageResult, error) {
	core, err := s.runDailyQuery(ctx, input)
	if err != nil {
		return ExternalDailyUsageResult{}, err
	}
	return core.result(false), nil
}

// QueryDailyUsageFilled behaves like QueryDailyUsage but returns one entry for
// every day in the range, zero-filling days without recorded data. Used by the
// CSV download endpoint.
func (s *ExternalTokenUsageService) QueryDailyUsageFilled(ctx context.Context, input ExternalDailyUsageInput) (ExternalDailyUsageResult, error) {
	core, err := s.runDailyQuery(ctx, input)
	if err != nil {
		return ExternalDailyUsageResult{}, err
	}
	return core.result(true), nil
}

// dailyQueryCore is the shared resolution + projection + query result used by
// both the unfilled JSON endpoint and the zero-filled CSV endpoint.
type dailyQueryCore struct {
	groupID, apiKeyID    int64
	projectionID         int64
	metric               tokenstat.MetricCode
	timezone             string
	location             *time.Location
	dimensionConfigured  bool
	projectionEnabledAt  *time.Time
	lastSyncedAt         *time.Time
	complete             bool
	message              string
	rows                 []tokenstat.UsageQueryRow
	rangeStart, rangeEnd time.Time // day boundaries in the statistics location
}

func (c *dailyQueryCore) result(fill bool) ExternalDailyUsageResult {
	result := ExternalDailyUsageResult{
		GroupID: c.groupID, APIKeyID: c.apiKeyID, ProjectionID: c.projectionID,
		Metric: c.metric, Timezone: c.timezone, DimensionConfigured: c.dimensionConfigured,
		ProjectionEnabledAt: c.projectionEnabledAt, LastSyncedAt: c.lastSyncedAt,
		Complete: c.complete, Message: c.message, Days: []ExternalDailyUsageDay{},
	}
	if !c.dimensionConfigured {
		return result
	}
	if fill {
		byDay := make(map[string]int64, len(c.rows))
		for _, row := range c.rows {
			byDay[row.PeriodStart.In(c.location).Format("2006-01-02")] = row.Value
		}
		days := make([]ExternalDailyUsageDay, 0)
		for day := c.rangeStart; day.Before(c.rangeEnd); day = day.AddDate(0, 0, 1) {
			date := day.Format("2006-01-02")
			days = append(days, ExternalDailyUsageDay{Date: date, TotalTokens: byDay[date]})
		}
		result.Days = days
		return result
	}
	days := make([]ExternalDailyUsageDay, 0, len(c.rows))
	for _, row := range c.rows {
		days = append(days, ExternalDailyUsageDay{
			Date:        row.PeriodStart.In(c.location).Format("2006-01-02"),
			TotalTokens: row.Value,
		})
	}
	result.Days = days
	return result
}

func (s *ExternalTokenUsageService) runDailyQuery(ctx context.Context, input ExternalDailyUsageInput) (*dailyQueryCore, error) {
	if s == nil || s.lookup == nil {
		return nil, fmt.Errorf("external token usage dimension lookup is required")
	}
	group, err := s.lookup.FindGroupByName(ctx, input.GroupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("find token usage group: %w", err)
	}
	key, err := s.lookup.FindAPIKeyByKey(ctx, input.APIKey)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("find token usage api key: %w", err)
	}
	if key.GroupID == nil || *key.GroupID != group.ID {
		return nil, ErrAPIKeyMismatch
	}
	if s.historyQuery == nil || s.projections == nil || s.location == nil {
		return nil, fmt.Errorf("external token usage service is not configured")
	}

	core := &dailyQueryCore{
		groupID: group.ID, apiKeyID: key.ID,
		metric: tokenstat.MetricTotalTokens, timezone: s.location.String(), location: s.location,
	}
	projection := findDailyProjection(s.projections.ActiveProjections())
	if projection == nil {
		core.message = "统计维度未配置"
		return core, nil
	}
	core.projectionID = int64(projection.ID)
	core.dimensionConfigured = true

	core.rangeStart = time.Date(input.StartDate.Year(), input.StartDate.Month(), input.StartDate.Day(), 0, 0, 0, 0, s.location)
	core.rangeEnd = time.Date(input.EndDate.Year(), input.EndDate.Month(), input.EndDate.Day(), 0, 0, 0, 0, s.location).AddDate(0, 0, 1)

	queryResult, err := s.historyQuery.QueryUsage(ctx, tokenstat.UsageQueryInput{
		ProjectionID: core.projectionID,
		MetricCode:   tokenstat.MetricTotalTokens,
		PeriodType:   tokenstat.PeriodDay,
		Start:        core.rangeStart,
		End:          core.rangeEnd,
		Filters: map[tokenstat.DimensionCode]tokenstat.DimensionValue{
			tokenstat.DimensionAPIKeyID: tokenstat.Int64Value(key.ID),
			tokenstat.DimensionGroupID:  tokenstat.Int64Value(group.ID),
		},
		Sort:     "time_asc",
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, fmt.Errorf("query daily token usage: %w", err)
	}
	core.projectionEnabledAt = queryResult.ProjectionEnabledAt
	core.lastSyncedAt = queryResult.LastSyncedAt
	core.complete = queryResult.Complete
	core.rows = queryResult.Rows
	return core, nil
}

// findDailyProjection returns the ACTIVE projection whose canonical dimension
// signature is exactly api_key_id,group_id and which collects total_tokens.
//
// Superset projections (for example user_id,api_key_id,group_id,route_alias)
// are deliberately NOT accepted: the async pipeline skips an event for a
// projection whenever any required dimension is missing from the event
// (async_pipeline.go operations()), so a superset projection can silently
// undercount. Requiring the exact two-dimension projection guarantees every
// event that has a group and an API Key is counted.
func findDailyProjection(projections []tokenstat.ProjectionDefinition) *tokenstat.ProjectionDefinition {
	signature, _ := tokenstat.DimensionSignature([]tokenstat.DimensionCode{
		tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID,
	})
	for i := range projections {
		candidate := &projections[i]
		if !containsMetric(candidate.MetricCodes, tokenstat.MetricTotalTokens) {
			continue
		}
		candidateSignature, err := tokenstat.DimensionSignature(candidate.DimensionCodes)
		if err != nil {
			continue
		}
		if candidateSignature == signature {
			return candidate
		}
	}
	return nil
}
