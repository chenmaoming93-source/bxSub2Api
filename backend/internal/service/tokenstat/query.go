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
	"github.com/Wei-Shaw/sub2api/ent/tokenstatprojection"
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

type DepartmentUsageRow struct {
	Department    string  `json:"department"`
	TotalTokens   int64   `json:"total_tokens"`
	UserCount     int64   `json:"user_count"`
	AverageTokens float64 `json:"average_tokens"`
	Percentage    float64 `json:"percentage"`
}

type DepartmentUsageResult struct {
	Rows         []DepartmentUsageRow `json:"rows"`
	Total        int                  `json:"total"`
	Summary      int64                `json:"summary"`
	Complete     bool                 `json:"complete"`
	LastSyncedAt *time.Time           `json:"last_synced_at,omitempty"`
	Consistency  string               `json:"consistency"`
}

type DepartmentUserUsageRow struct {
	UserID      int64   `json:"user_id"`
	Email       string  `json:"email"`
	Username    string  `json:"username"`
	TotalTokens int64   `json:"total_tokens"`
	Percentage  float64 `json:"percentage"`
}

type DepartmentUserUsageResult struct {
	Department            string                   `json:"department"`
	DepartmentTotalTokens int64                    `json:"department_total_tokens"`
	Rows                  []DepartmentUserUsageRow `json:"rows"`
	Total                 int                      `json:"total"`
	Page                  int                      `json:"page"`
	PageSize              int                      `json:"page_size"`
	Complete              bool                     `json:"complete"`
	LastSyncedAt          *time.Time               `json:"last_synced_at,omitempty"`
	Consistency           string                   `json:"consistency"`
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

// QueryDepartmentUsage aggregates user-level Token data by each user's current
// department. The department stored on an aggregate row is intentionally not
// used, so moving a user changes how the selected date range is reported.
func (s *ProjectionAdminService) QueryDepartmentUsage(ctx context.Context, start, end time.Time) (*DepartmentUsageResult, error) {
	if err := validateDepartmentUsageRange(start, end); err != nil {
		return nil, err
	}
	projection, err := s.activeUserTokenProjection(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTotalTokensMetric(ctx, projection.ID); err != nil {
		return nil, err
	}
	users, err := s.queryCurrentUserTokenUsage(ctx, projection.ID, start, end, nil)
	if err != nil {
		return nil, err
	}

	byDepartment := make(map[string]*DepartmentUsageRow)
	var summary int64
	var lastSyncedAt *time.Time
	complete := true
	now := time.Now()
	for _, user := range users {
		department := normalizeDepartment(user.Department)
		row := byDepartment[department]
		if row == nil {
			row = &DepartmentUsageRow{Department: department}
			byDepartment[department] = row
		}
		row.UserCount++
		row.TotalTokens += user.TotalTokens
		summary += user.TotalTokens
		if user.LastSyncedAt != nil {
			if lastSyncedAt == nil || user.LastSyncedAt.After(*lastSyncedAt) {
				value := *user.LastSyncedAt
				lastSyncedAt = &value
			}
			if end.Before(now) && user.LastSyncedAt.Before(end) {
				complete = false
			}
		}
	}
	rows := make([]DepartmentUsageRow, 0, len(byDepartment))
	for _, row := range byDepartment {
		if row.UserCount > 0 {
			row.AverageTokens = float64(row.TotalTokens) / float64(row.UserCount)
		}
		if summary > 0 {
			row.Percentage = float64(row.TotalTokens) * 100 / float64(summary)
		}
		rows = append(rows, *row)
	}
	sortDepartmentUsageRows(rows)
	return &DepartmentUsageResult{
		Rows: rows, Total: len(rows), Summary: summary, Complete: complete,
		LastSyncedAt: lastSyncedAt, Consistency: "mysql_eventual",
	}, nil
}

// QueryDepartmentUserUsage returns all current users in a department, including
// users with no Token rows. Pagination is applied after the current-department
// join and in-memory ordering.
func (s *ProjectionAdminService) QueryDepartmentUserUsage(ctx context.Context, start, end time.Time, department string, page, pageSize int) (*DepartmentUserUsageResult, error) {
	if err := validateDepartmentUsageRange(start, end); err != nil {
		return nil, err
	}
	projection, err := s.activeUserTokenProjection(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTotalTokensMetric(ctx, projection.ID); err != nil {
		return nil, err
	}
	department = normalizeDepartment(department)
	users, err := s.queryCurrentUserTokenUsage(ctx, projection.ID, start, end, &department)
	if err != nil {
		return nil, err
	}
	var totalTokens int64
	var lastSyncedAt *time.Time
	complete := true
	now := time.Now()
	rows := make([]DepartmentUserUsageRow, 0, len(users))
	for _, user := range users {
		totalTokens += user.TotalTokens
		rows = append(rows, DepartmentUserUsageRow{
			UserID: user.UserID, Email: user.Email, Username: user.Username, TotalTokens: user.TotalTokens,
		})
		if user.LastSyncedAt != nil {
			if lastSyncedAt == nil || user.LastSyncedAt.After(*lastSyncedAt) {
				value := *user.LastSyncedAt
				lastSyncedAt = &value
			}
			if end.Before(now) && user.LastSyncedAt.Before(end) {
				complete = false
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].TotalTokens == rows[j].TotalTokens {
			return rows[i].UserID < rows[j].UserID
		}
		return rows[i].TotalTokens > rows[j].TotalTokens
	})
	for i := range rows {
		if totalTokens > 0 {
			rows[i].Percentage = float64(rows[i].TotalTokens) * 100 / float64(totalTokens)
		}
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	startIndex := (page - 1) * pageSize
	if startIndex > len(rows) {
		startIndex = len(rows)
	}
	endIndex := startIndex + pageSize
	if endIndex > len(rows) {
		endIndex = len(rows)
	}
	return &DepartmentUserUsageResult{
		Department: department, DepartmentTotalTokens: totalTokens,
		Rows: rows[startIndex:endIndex], Total: len(rows), Page: page, PageSize: pageSize,
		Complete: complete, LastSyncedAt: lastSyncedAt, Consistency: "mysql_eventual",
	}, nil
}

type currentUserTokenUsage struct {
	UserID       int64
	Email        string
	Username     string
	Department   string
	TotalTokens  int64
	LastSyncedAt *time.Time
}

func (s *ProjectionAdminService) activeUserTokenProjection(ctx context.Context) (*ent.TokenStatProjection, error) {
	projection, err := s.client.TokenStatProjection.Query().Where(
		tokenstatprojection.DimensionSignatureEQ(string(DimensionUserID)),
		tokenstatprojection.StatusEQ(ProjectionStatusActive),
	).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user token projection is not active: %w", err)
	}
	return projection, nil
}

func (s *ProjectionAdminService) ensureTotalTokensMetric(ctx context.Context, projectionID int64) error {
	metricAllowed, err := s.client.TokenStatProjectionMetric.Query().Where(
		tokenstatprojectionmetric.ProjectionIDEQ(projectionID),
		tokenstatprojectionmetric.MetricCodeEQ(string(MetricTotalTokens)),
		tokenstatprojectionmetric.StatusEQ(ProjectionStatusActive),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if !metricAllowed {
		return fmt.Errorf("user token projection does not collect total_tokens")
	}
	return nil
}

func (s *ProjectionAdminService) queryCurrentUserTokenUsage(ctx context.Context, projectionID int64, start, end time.Time, department *string) ([]currentUserTokenUsage, error) {
	query := "" +
		"SELECT u.id, u.email, u.username, u.department, COALESCE(SUM(a.metric_value), 0), MAX(a.last_synced_at) " +
		"FROM users AS u " +
		"LEFT JOIN token_stat_aggregates AS a ON a.user_id = u.id " +
		"AND a.projection_id = ? AND a.metric_code = ? AND a.period_type = ? " +
		"AND a.period_start >= ? AND a.period_start < ? " +
		"WHERE u.deleted_at IS NULL"
	args := []any{projectionID, string(MetricTotalTokens), string(PeriodDay), start, end}
	if department != nil {
		query += " AND COALESCE(NULLIF(TRIM(u.department), ''), '未设置') = ?"
		args = append(args, *department)
	}
	query += " GROUP BY u.id, u.email, u.username, u.department ORDER BY u.id"

	var rows sql.Rows
	if err := s.client.Driver().Query(ctx, query, args, &rows); err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]currentUserTokenUsage, 0)
	for rows.Next() {
		var item currentUserTokenUsage
		var lastSynced any
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.Department, &item.TotalTokens, &lastSynced); err != nil {
			return nil, err
		}
		value, err := parseNullableTime(lastSynced)
		if err != nil {
			return nil, err
		}
		item.LastSyncedAt = value
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func validateDepartmentUsageRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return errors.New("start and end must define a non-empty range")
	}
	if end.Sub(start) > maxQueryDays*24*time.Hour {
		return fmt.Errorf("query range exceeds %d days", maxQueryDays)
	}
	return nil
}

func parseNullableTime(raw any) (*time.Time, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case time.Time:
		return &value, nil
	case string:
		return parseTimeString(value)
	case []byte:
		return parseTimeString(string(value))
	default:
		return nil, fmt.Errorf("unsupported timestamp type %T", raw)
	}
}

func parseTimeString(value string) (*time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05 -0700 MST", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid timestamp %q", value)
}

func normalizeDepartment(department string) string {
	department = strings.TrimSpace(department)
	if department == "" {
		return "未设置"
	}
	return department
}

func sortDepartmentUsageRows(rows []DepartmentUsageRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalTokens == rows[j].TotalTokens {
			return rows[i].Department < rows[j].Department
		}
		return rows[i].TotalTokens > rows[j].TotalTokens
	})
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
	case DimensionDepartment:
		return tokenstataggregate.DepartmentEQ(value.String), nil
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
	case DimensionDepartment:
		if row.Department != nil {
			return StringValue(*row.Department), true
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
