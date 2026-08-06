package tokenstat

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/tokenstataggregate"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatperiodstate"
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojectionmetric"
)

const (
	maxQueryDays = 366
	maxQueryRows = 10000
)

type UsageQueryInput struct {
	ProjectionID int64
	MetricCode   MetricCode
	PeriodType   PeriodType
	Start        time.Time
	End          time.Time
	Filters      map[DimensionCode]DimensionValue
	GroupBy      []DimensionCode
	Sort         string
	Page         int
	PageSize     int
}

type UsageQueryRow struct {
	PeriodStart time.Time                        `json:"period_start"`
	PeriodEnd   time.Time                        `json:"period_end"`
	Dimensions  map[DimensionCode]DimensionValue `json:"dimensions"`
	Value       int64                            `json:"value"`
}

type UsageQueryResult struct {
	Rows                []UsageQueryRow `json:"rows"`
	Total               int             `json:"total"`
	Summary             int64           `json:"summary"`
	ProjectionEnabledAt *time.Time      `json:"projection_enabled_at,omitempty"`
	LastSyncedAt        *time.Time      `json:"last_synced_at,omitempty"`
	Complete            bool            `json:"complete"`
	Consistency         string          `json:"consistency"`
}

type SyncStatus struct {
	Periods      []*ent.TokenStatPeriodState `json:"periods"`
	LastSyncedAt *time.Time                  `json:"last_synced_at,omitempty"`
	Metrics      ObservabilitySnapshot       `json:"metrics"`
}

func (s *ProjectionAdminService) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	periods, err := s.client.TokenStatPeriodState.Query().
		Order(tokenstatperiodstate.ByPeriodStart()).
		Limit(30).All(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := s.client.TokenStatAggregate.Query().
		Order(tokenstataggregate.ByLastSyncedAt(sql.OrderDesc())).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	var lastSynced *time.Time
	if latest != nil {
		value := latest.LastSyncedAt
		lastSynced = &value
	}
	return &SyncStatus{Periods: periods, LastSyncedAt: lastSynced, Metrics: MetricsSnapshot()}, nil
}

type queryBucket struct {
	start      time.Time
	end        time.Time
	dimensions map[DimensionCode]DimensionValue
	value      int64
}

func (s *ProjectionAdminService) QueryUsage(ctx context.Context, input UsageQueryInput) (*UsageQueryResult, error) {
	projection, err := s.Get(ctx, input.ProjectionID)
	if err != nil {
		return nil, fmt.Errorf("projection not found: %w", err)
	}
	if err := validateUsageQuery(projection, &input); err != nil {
		return nil, err
	}
	metricAllowed, err := s.client.TokenStatProjectionMetric.Query().Where(
		tokenstatprojectionmetric.ProjectionIDEQ(input.ProjectionID),
		tokenstatprojectionmetric.MetricCodeEQ(string(input.MetricCode)),
		tokenstatprojectionmetric.StatusEQ(ProjectionStatusActive),
	).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !metricAllowed {
		return nil, fmt.Errorf("metric %q is not collected by projection", input.MetricCode)
	}

	predicates := []predicate.TokenStatAggregate{
		tokenstataggregate.ProjectionIDEQ(input.ProjectionID),
		tokenstataggregate.MetricCodeEQ(string(input.MetricCode)),
		tokenstataggregate.PeriodTypeEQ(string(input.PeriodType)),
		tokenstataggregate.PeriodStartGTE(input.Start),
		tokenstataggregate.PeriodStartLT(input.End),
	}
	for code, value := range input.Filters {
		predicate, err := aggregateDimensionPredicate(code, value)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, predicate)
	}
	rows, err := s.client.TokenStatAggregate.Query().Where(predicates...).
		Order(tokenstataggregate.ByPeriodStart()).
		Limit(maxQueryRows + 1).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) > maxQueryRows {
		return nil, fmt.Errorf("query result exceeds %d source rows; narrow the range or add filters", maxQueryRows)
	}

	buckets := make(map[string]*queryBucket)
	var summary int64
	var lastSynced *time.Time
	complete := true
	now := time.Now()
	for _, row := range rows {
		key, dimensions, err := queryGroupKey(row, input.GroupBy)
		if err != nil {
			return nil, err
		}
		key = row.PeriodStart.UTC().Format(time.RFC3339Nano) + "|" + key
		bucket := buckets[key]
		if bucket == nil {
			bucket = &queryBucket{start: row.PeriodStart, end: row.PeriodEnd, dimensions: dimensions}
			buckets[key] = bucket
		}
		bucket.value += row.MetricValue
		summary += row.MetricValue
		if lastSynced == nil || row.LastSyncedAt.After(*lastSynced) {
			value := row.LastSyncedAt
			lastSynced = &value
		}
		if row.PeriodEnd.Before(now) && row.LastSyncedAt.Before(row.PeriodEnd) {
			complete = false
		}
	}
	resultRows := make([]UsageQueryRow, 0, len(buckets))
	for _, bucket := range buckets {
		resultRows = append(resultRows, UsageQueryRow{
			PeriodStart: bucket.start, PeriodEnd: bucket.end,
			Dimensions: bucket.dimensions, Value: bucket.value,
		})
	}
	sortUsageQueryRows(resultRows, input.Sort)
	total := len(resultRows)
	startIndex := (input.Page - 1) * input.PageSize
	if startIndex > total {
		startIndex = total
	}
	endIndex := startIndex + input.PageSize
	if endIndex > total {
		endIndex = total
	}
	return &UsageQueryResult{
		Rows: resultRows[startIndex:endIndex], Total: total, Summary: summary,
		ProjectionEnabledAt: projection.EnabledAt, LastSyncedAt: lastSynced,
		Complete: complete, Consistency: "mysql_eventual",
	}, nil
}

func validateUsageQuery(projection *ent.TokenStatProjection, input *UsageQueryInput) error {
	if input.Start.IsZero() || input.End.IsZero() || !input.Start.Before(input.End) {
		return errors.New("start and end must define a non-empty range")
	}
	if input.End.Sub(input.Start) > maxQueryDays*24*time.Hour {
		return fmt.Errorf("query range exceeds %d days", maxQueryDays)
	}
	if input.PeriodType != PeriodDay && input.PeriodType != PeriodWeek && input.PeriodType != PeriodMonth {
		return errors.New("invalid period_type")
	}
	if input.PeriodType != PeriodDay && !isNaturalRange(input.Start, input.End, input.PeriodType) {
		return errors.New("custom date ranges must use daily period_type")
	}
	allowedDimensions := make(map[DimensionCode]bool, len(projection.DimensionCodes))
	for _, raw := range projection.DimensionCodes {
		allowedDimensions[DimensionCode(raw)] = true
	}
	for code, value := range input.Filters {
		if !allowedDimensions[code] {
			return fmt.Errorf("filter dimension %q is not in projection", code)
		}
		definition, ok := Dimension(code)
		if !ok || validateDimensionValue(definition, value) != nil {
			return fmt.Errorf("invalid filter dimension %q", code)
		}
	}
	for _, code := range input.GroupBy {
		if !allowedDimensions[code] {
			return fmt.Errorf("group_by dimension %q is not in projection", code)
		}
	}
	if input.Sort == "" {
		input.Sort = "time_asc"
	}
	if input.Sort != "time_asc" && input.Sort != "time_desc" && input.Sort != "value_asc" && input.Sort != "value_desc" {
		return errors.New("invalid sort")
	}
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 50
	}
	if input.PageSize > 1000 {
		return errors.New("page_size exceeds 1000")
	}
	return nil
}

func isNaturalRange(start, end time.Time, periodType PeriodType) bool {
	periods := NaturalPeriods(start, start.Location())
	var alignedStart bool
	for _, period := range periods {
		if period.Type == periodType {
			alignedStart = start.Equal(period.Start)
			break
		}
	}
	if !alignedStart {
		return false
	}
	endPeriods := NaturalPeriods(end, end.Location())
	for _, period := range endPeriods {
		if period.Type == periodType {
			return end.Equal(period.Start)
		}
	}
	return false
}

func aggregateDimensionPredicate(code DimensionCode, value DimensionValue) (predicate.TokenStatAggregate, error) {
	switch code {
	case DimensionUserID:
		return tokenstataggregate.UserIDEQ(value.Int64), nil
	case DimensionAPIKeyID:
		return tokenstataggregate.APIKeyIDEQ(value.Int64), nil
	case DimensionGroupID:
		return tokenstataggregate.GroupIDEQ(value.Int64), nil
	case DimensionRouteAlias:
		return tokenstataggregate.RouteAliasEQ(value.String), nil
	case DimensionAccountID:
		return tokenstataggregate.AccountIDEQ(value.Int64), nil
	case DimensionUpstreamModel:
		return tokenstataggregate.UpstreamModelEQ(value.String), nil
	default:
		return nil, fmt.Errorf("unsupported filter dimension %q", code)
	}
}

func queryGroupKey(row *ent.TokenStatAggregate, groupBy []DimensionCode) (string, map[DimensionCode]DimensionValue, error) {
	parts := make([]string, 0, len(groupBy))
	values := make(map[DimensionCode]DimensionValue, len(groupBy))
	for _, code := range groupBy {
		value, ok := aggregateDimensionValue(row, code)
		if !ok {
			return "", nil, fmt.Errorf("aggregate row is missing dimension %q", code)
		}
		values[code] = value
		if value.Type == ValueTypeInt64 {
			parts = append(parts, fmt.Sprintf("%s=%d", code, value.Int64))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", code, value.String))
		}
	}
	return strings.Join(parts, "\x1f"), values, nil
}

func aggregateDimensionValue(row *ent.TokenStatAggregate, code DimensionCode) (DimensionValue, bool) {
	switch code {
	case DimensionUserID:
		if row.UserID != nil {
			return Int64Value(*row.UserID), true
		}
	case DimensionAPIKeyID:
		if row.APIKeyID != nil {
			return Int64Value(*row.APIKeyID), true
		}
	case DimensionGroupID:
		if row.GroupID != nil {
			return Int64Value(*row.GroupID), true
		}
	case DimensionRouteAlias:
		if row.RouteAlias != nil {
			return StringValue(*row.RouteAlias), true
		}
	case DimensionAccountID:
		if row.AccountID != nil {
			return Int64Value(*row.AccountID), true
		}
	case DimensionUpstreamModel:
		if row.UpstreamModel != nil {
			return StringValue(*row.UpstreamModel), true
		}
	}
	return DimensionValue{}, false
}

func sortUsageQueryRows(rows []UsageQueryRow, order string) {
	sort.SliceStable(rows, func(i, j int) bool {
		switch order {
		case "time_desc":
			return rows[i].PeriodStart.After(rows[j].PeriodStart)
		case "value_asc":
			return rows[i].Value < rows[j].Value
		case "value_desc":
			return rows[i].Value > rows[j].Value
		default:
			return rows[i].PeriodStart.Before(rows[j].PeriodStart)
		}
	})
}
