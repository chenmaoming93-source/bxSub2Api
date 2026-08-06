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
	lookup      ExternalTokenUsageDimensionLookup
	reader      ExternalTokenUsageCurrentReader
	projections ExternalTokenUsageProjectionProvider
	location    *time.Location
	now         func() time.Time
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

type ExternalTokenUsagePeriodResult struct {
	PeriodType          tokenstat.PeriodType `json:"period_type"`
	PeriodStart         time.Time            `json:"period_start"`
	PeriodEnd           time.Time            `json:"period_end"`
	DimensionConfigured bool                 `json:"dimension_configured"`
	DataPresent         bool                 `json:"data_present"`
	TotalTokens         *int64               `json:"total_tokens"`
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
	periods := tokenstat.NaturalPeriods(now(), s.location)
	result := ExternalTokenUsageResult{Dimensions: dimensions, Metric: tokenstat.MetricTotalTokens, Timezone: s.location.String()}

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
