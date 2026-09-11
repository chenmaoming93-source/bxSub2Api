package tokenstat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

const defaultIdentityDiscoveryPageSize = 500

var ErrInvalidQuotaResetRequest = errors.New("invalid token quota reset request")

type ResetIdentityDiscoveryRequest struct {
	DimensionValues map[DimensionCode]DimensionValue
	MetricCode      MetricCode
	PeriodType      PeriodType
	IncludeDetails  bool
}

type StatisticIdentity struct {
	Period          Period
	ProjectionID    int64
	ProjectionName  string
	DimensionHash   [16]byte
	DimensionValues map[DimensionCode]DimensionValue
	MetricCode      MetricCode
	MatchedQuotaIDs []int64
}

type IdentityDiscoveryStatus string

const (
	IdentityDiscoveryFound   IdentityDiscoveryStatus = "FOUND"
	IdentityDiscoveryNoQuota IdentityDiscoveryStatus = "NO_QUOTA"
	IdentityDiscoveryNoUsage IdentityDiscoveryStatus = "NO_USAGE"
)

type IdentityDiscoveryResult struct {
	Status            IdentityDiscoveryStatus
	MatchedQuotaCount int
	MatchedQuotas     []QuotaRule
	Identities        []StatisticIdentity
}

type AggregateIdentityCursor struct {
	ProjectionID  int64
	DimensionHash [16]byte
}

type AggregateIdentityQuery struct {
	Period        Period
	ProjectionIDs []int64
	MetricCode    MetricCode
	Filters       map[DimensionCode]DimensionValue
	After         *AggregateIdentityCursor
	Limit         int
}

type ResetScopeProvider interface {
	ActiveProjections() []ProjectionDefinition
	ActiveQuotaRules() []QuotaRule
}

type CurrentDirtyIdentitySource interface {
	ScanCurrent(ctx context.Context, visit func(StatisticIdentity) error) error
}

type AggregateIdentitySource interface {
	ListAggregateIdentities(ctx context.Context, query AggregateIdentityQuery) ([]StatisticIdentity, *AggregateIdentityCursor, error)
}

type ResetIdentityDiscoveryService struct {
	scopes     ResetScopeProvider
	dirty      CurrentDirtyIdentitySource
	aggregates AggregateIdentitySource
	location   *time.Location
	pageSize   int
	now        func() time.Time
}

func NewResetIdentityDiscoveryService(scopes ResetScopeProvider, dirty CurrentDirtyIdentitySource, aggregates AggregateIdentitySource, location *time.Location, pageSize int) *ResetIdentityDiscoveryService {
	if pageSize <= 0 {
		pageSize = defaultIdentityDiscoveryPageSize
	}
	return &ResetIdentityDiscoveryService{
		scopes: scopes, dirty: dirty, aggregates: aggregates,
		location: location, pageSize: pageSize, now: time.Now,
	}
}

func (s *ResetIdentityDiscoveryService) Discover(ctx context.Context, request ResetIdentityDiscoveryRequest) (IdentityDiscoveryResult, error) {
	if err := validateResetIdentityDiscoveryRequest(request); err != nil {
		return IdentityDiscoveryResult{}, fmt.Errorf("%w: %v", ErrInvalidQuotaResetRequest, err)
	}
	if s == nil || s.scopes == nil || s.dirty == nil || s.aggregates == nil || s.location == nil || s.now == nil {
		return IdentityDiscoveryResult{}, errors.New("reset identity discovery dependencies are required")
	}

	now := s.now().In(s.location)
	period, ok := naturalPeriodByType(now, request.PeriodType, s.location)
	if !ok {
		return IdentityDiscoveryResult{}, fmt.Errorf("invalid period type %q", request.PeriodType)
	}

	activeProjections := s.scopes.ActiveProjections()
	projectionIDs := matchingProjectionIDs(activeProjections, request)
	projectionSet := make(map[int64]struct{}, len(projectionIDs))
	projectionNames := make(map[int64]string, len(projectionIDs))
	for _, id := range projectionIDs {
		projectionSet[id] = struct{}{}
	}
	for _, projection := range activeProjections {
		if _, ok := projectionSet[int64(projection.ID)]; ok {
			projectionNames[int64(projection.ID)] = projection.Name
		}
	}
	quotaScopes := matchingQuotaScopes(s.scopes.ActiveQuotaRules(), projectionSet, request, now)
	matchedQuotaDetails := []QuotaRule(nil)
	if request.IncludeDetails {
		matchedQuotaDetails = quotaScopes
	}
	if len(quotaScopes) == 0 {
		return IdentityDiscoveryResult{Status: IdentityDiscoveryNoQuota, MatchedQuotas: matchedQuotaDetails}, nil
	}

	identities := make(map[statisticIdentityKey]StatisticIdentity)
	// This is deliberately the only current-dirty traversal. It must finish
	// before the first MySQL query and is never repeated or locked.
	if err := s.dirty.ScanCurrent(ctx, func(identity StatisticIdentity) error {
		if identityMatchesDiscovery(identity, period, projectionSet, request) {
			identities[newStatisticIdentityKey(identity)] = identity
		}
		return nil
	}); err != nil {
		return IdentityDiscoveryResult{}, fmt.Errorf("scan current dirty token statistics: %w", err)
	}

	var cursor *AggregateIdentityCursor
	for {
		page, next, err := s.aggregates.ListAggregateIdentities(ctx, AggregateIdentityQuery{
			Period: period, ProjectionIDs: projectionIDs, MetricCode: request.MetricCode,
			Filters: request.DimensionValues, After: cursor, Limit: s.pageSize,
		})
		if err != nil {
			return IdentityDiscoveryResult{}, fmt.Errorf("list aggregate token statistic identities: %w", err)
		}
		for _, identity := range page {
			if identityMatchesDiscovery(identity, period, projectionSet, request) {
				identities[newStatisticIdentityKey(identity)] = identity
			}
		}
		if next == nil {
			break
		}
		cursor = next
	}

	result := make([]StatisticIdentity, 0, len(identities))
	for _, identity := range identities {
		matchedQuotaIDs := identityMatchingQuotaIDs(identity, quotaScopes)
		if len(matchedQuotaIDs) > 0 {
			if request.IncludeDetails {
				identity.ProjectionName = projectionNames[identity.ProjectionID]
				identity.MatchedQuotaIDs = matchedQuotaIDs
			}
			result = append(result, identity)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ProjectionID != result[j].ProjectionID {
			return result[i].ProjectionID < result[j].ProjectionID
		}
		return bytes.Compare(result[i].DimensionHash[:], result[j].DimensionHash[:]) < 0
	})
	if len(result) == 0 {
		return IdentityDiscoveryResult{Status: IdentityDiscoveryNoUsage, MatchedQuotaCount: len(quotaScopes), MatchedQuotas: matchedQuotaDetails}, nil
	}
	return IdentityDiscoveryResult{Status: IdentityDiscoveryFound, MatchedQuotaCount: len(quotaScopes), MatchedQuotas: matchedQuotaDetails, Identities: result}, nil
}

func validateResetIdentityDiscoveryRequest(request ResetIdentityDiscoveryRequest) error {
	metric, ok := Metric(request.MetricCode)
	if !ok || !metric.AllowQuota {
		return fmt.Errorf("metric %q does not support quotas", request.MetricCode)
	}
	if request.PeriodType != PeriodDay && request.PeriodType != PeriodWeek && request.PeriodType != PeriodMonth {
		return fmt.Errorf("invalid period type %q", request.PeriodType)
	}
	if len(request.DimensionValues) == 0 {
		return errors.New("at least one dimension value is required")
	}
	for code, value := range request.DimensionValues {
		definition, ok := Dimension(code)
		if !ok {
			return fmt.Errorf("unknown dimension %q", code)
		}
		if value.Type == ValueTypeWildcard {
			return fmt.Errorf("reset dimension %q cannot be wildcard", code)
		}
		if err := validateDimensionValue(definition, value); err != nil {
			return err
		}
	}
	return nil
}

func naturalPeriodByType(at time.Time, periodType PeriodType, location *time.Location) (Period, bool) {
	for _, period := range NaturalPeriods(at, location) {
		if period.Type == periodType {
			return period, true
		}
	}
	return Period{}, false
}

func matchingProjectionIDs(projections []ProjectionDefinition, request ResetIdentityDiscoveryRequest) []int64 {
	result := make([]int64, 0)
	for _, projection := range projections {
		if projection.ID == 0 || !containsMetric(projection.MetricCodes, request.MetricCode) {
			continue
		}
		available := make(map[DimensionCode]struct{}, len(projection.DimensionCodes))
		for _, code := range projection.DimensionCodes {
			available[code] = struct{}{}
		}
		matches := true
		for code := range request.DimensionValues {
			if _, ok := available[code]; !ok {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, int64(projection.ID))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func containsMetric(metrics []MetricCode, wanted MetricCode) bool {
	for _, metric := range metrics {
		if metric == wanted {
			return true
		}
	}
	return false
}

func matchingQuotaScopes(rules []QuotaRule, projectionSet map[int64]struct{}, request ResetIdentityDiscoveryRequest, at time.Time) []QuotaRule {
	result := make([]QuotaRule, 0)
	for _, rule := range rules {
		if _, ok := projectionSet[rule.ProjectionID]; !ok || rule.MetricCode != request.MetricCode || rule.PeriodType != request.PeriodType || !ruleEffective(rule, at) {
			continue
		}
		matches := true
		for code, actual := range request.DimensionValues {
			expected, ok := rule.DimensionValues[code]
			if !ok || (expected.Type != ValueTypeWildcard && expected != actual) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, rule)
		}
	}
	return result
}

func identityMatchesDiscovery(identity StatisticIdentity, period Period, projectionSet map[int64]struct{}, request ResetIdentityDiscoveryRequest) bool {
	if identity.Period.Type != period.Type || !identity.Period.Start.Equal(period.Start) || identity.MetricCode != request.MetricCode {
		return false
	}
	if _, ok := projectionSet[identity.ProjectionID]; !ok {
		return false
	}
	for code, expected := range request.DimensionValues {
		actual, ok := identity.DimensionValues[code]
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func identityMatchingQuotaIDs(identity StatisticIdentity, rules []QuotaRule) []int64 {
	result := make([]int64, 0)
	for _, rule := range rules {
		if rule.ProjectionID == identity.ProjectionID && rule.MetricCode == identity.MetricCode && rule.PeriodType == identity.Period.Type && ruleMatches(rule, identity.DimensionValues) {
			result = append(result, rule.ID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

type statisticIdentityKey struct {
	periodType  PeriodType
	periodStart int64
	projection  int64
	hash        [16]byte
	metric      MetricCode
}

func newStatisticIdentityKey(identity StatisticIdentity) statisticIdentityKey {
	return statisticIdentityKey{
		periodType: identity.Period.Type, periodStart: identity.Period.Start.UnixNano(),
		projection: identity.ProjectionID, hash: identity.DimensionHash, metric: identity.MetricCode,
	}
}
