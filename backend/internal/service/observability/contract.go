// Package observability defines the shared read-only observable dashboard
// contract. Handlers and services must reuse these types instead of defining
// endpoint-specific aliases.
package observability

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Window string

const (
	Window1h        Window = "1h"
	Window6h        Window = "6h"
	Window24h       Window = "24h"
	Window7d        Window = "7d"
	WindowToday     Window = "today"
	WindowYesterday Window = "yesterday"
	WindowCustom    Window = "custom"
)

type Bucket string

const (
	Bucket5m  Bucket = "5m"
	Bucket15m Bucket = "15m"
	Bucket1h  Bucket = "1h"
	Bucket6h  Bucket = "6h"
)

type DimensionCode string

const (
	DimensionUserID     DimensionCode = "user_id"
	DimensionAPIKeyID   DimensionCode = "api_key_id"
	DimensionGroupID    DimensionCode = "group_id"
	DimensionRouteAlias DimensionCode = "route_alias"
	DimensionAccountID  DimensionCode = "account_id"
	DimensionDepartment DimensionCode = "department"
)

const (
	MaxFilterValues = 50
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxRange        = 7 * 24 * time.Hour
	MaxBucketPoints = 288
)

type BucketWindow struct {
	Window Window `json:"window"`
	Bucket Bucket `json:"bucket"`
}

type DimensionDefinition struct {
	Code        DimensionCode `json:"code"`
	DisplayName string        `json:"display_name"`
}

type ContractDescriptor struct {
	Version               string                `json:"version"`
	ReadOnly              bool                  `json:"read_only"`
	BucketWindows         []BucketWindow        `json:"bucket_windows"`
	RangeKinds            []string              `json:"range_kinds"`
	Dimensions            []DimensionDefinition `json:"dimensions"`
	MetricCodes           []string              `json:"metric_codes"`
	Failures              []string              `json:"failure_metrics"`
	MaxFilterValues       int                   `json:"max_filter_values"`
	DefaultPageSize       int                   `json:"default_page_size"`
	MaxPageSize           int                   `json:"max_page_size"`
	RatioScale            string                `json:"ratio_scale"`
	RequestedDimensionsAt int                   `json:"requested_dimension_count"`
}

func Contract() ContractDescriptor {
	return ContractDescriptor{
		Version:  "observability/v1",
		ReadOnly: true,
		BucketWindows: []BucketWindow{
			{Window: Window1h, Bucket: Bucket5m},
			{Window: Window6h, Bucket: Bucket15m},
			{Window: Window24h, Bucket: Bucket1h},
			{Window: Window7d, Bucket: Bucket6h},
		},
		RangeKinds: []string{
			string(Window1h), string(Window6h), string(Window24h), string(Window7d),
			string(WindowToday), string(WindowYesterday), string(WindowCustom),
		},
		Dimensions: []DimensionDefinition{
			{Code: DimensionUserID, DisplayName: "用户"},
			{Code: DimensionGroupID, DisplayName: "分组"},
			{Code: DimensionRouteAlias, DisplayName: "路由别名"},
			{Code: DimensionAccountID, DisplayName: "模型账号"},
		},
		MetricCodes: []string{
			"total_tokens",
			"request_count",
			"avg_duration_ms",
			"avg_first_token_ms",
			"failure_rate",
		},
		Failures: []string{
			"client_failure_rate",
			"business_limited_rate",
			"sla_error_rate",
			"upstream_error_rate",
		},
		MaxFilterValues:       MaxFilterValues,
		DefaultPageSize:       DefaultPageSize,
		MaxPageSize:           MaxPageSize,
		RatioScale:            "0..1",
		RequestedDimensionsAt: 4,
	}
}

func ValidateWindowBucket(window Window, bucket Bucket) error {
	switch window {
	case Window1h:
		if bucket == Bucket5m {
			return nil
		}
	case Window6h:
		if bucket == Bucket15m {
			return nil
		}
	case Window24h:
		if bucket == Bucket1h {
			return nil
		}
	case Window7d:
		if bucket == Bucket6h {
			return nil
		}
	case WindowToday, WindowYesterday, WindowCustom:
		if IsBucket(bucket) {
			return nil
		}
	default:
		return fmt.Errorf("unsupported window %q", window)
	}
	return fmt.Errorf("bucket %q is not allowed for window %q", bucket, window)
}

type ResolvedRange struct {
	Window Window
	Bucket Bucket
	Start  time.Time
	End    time.Time
}

func IsBucket(bucket Bucket) bool {
	switch bucket {
	case Bucket5m, Bucket15m, Bucket1h, Bucket6h:
		return true
	default:
		return false
	}
}

func BucketDuration(bucket Bucket) (time.Duration, error) {
	switch bucket {
	case Bucket5m:
		return 5 * time.Minute, nil
	case Bucket15m:
		return 15 * time.Minute, nil
	case Bucket1h:
		return time.Hour, nil
	case Bucket6h:
		return 6 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported bucket %q", bucket)
	}
}

func DefaultBucketForDuration(duration time.Duration) Bucket {
	switch {
	case duration <= 2*time.Hour:
		return Bucket5m
	case duration <= 12*time.Hour:
		return Bucket15m
	case duration <= 48*time.Hour:
		return Bucket1h
	default:
		return Bucket6h
	}
}

// ResolveRange normalizes fixed, relative, and absolute requests into one
// query range. UI labels remain local to loc while query bounds are UTC.
func ResolveRange(window Window, bucket Bucket, startRaw, endRaw string, now time.Time, loc *time.Location) (ResolvedRange, error) {
	if window == "" {
		window = Window1h
	}
	if loc == nil {
		loc = time.UTC
	}
	now = now.In(loc)
	var start, end time.Time
	switch window {
	case Window1h, Window6h, Window24h, Window7d:
		duration, err := WindowDuration(window)
		if err != nil {
			return ResolvedRange{}, err
		}
		if bucket == "" {
			bucket = defaultBucketForWindow(window)
		}
		if err := ValidateWindowBucket(window, bucket); err != nil {
			return ResolvedRange{}, err
		}
		end = now
		start = end.Add(-duration)
	case WindowToday:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = now
	case WindowYesterday:
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = today
		start = today.AddDate(0, 0, -1)
	case WindowCustom:
		if strings.TrimSpace(startRaw) == "" || strings.TrimSpace(endRaw) == "" {
			return ResolvedRange{}, fmt.Errorf("custom range requires start_at and end_at")
		}
		var err error
		start, err = parseRangeTime(startRaw, loc)
		if err != nil {
			return ResolvedRange{}, fmt.Errorf("invalid start_at: %w", err)
		}
		end, err = parseRangeTime(endRaw, loc)
		if err != nil {
			return ResolvedRange{}, fmt.Errorf("invalid end_at: %w", err)
		}
	default:
		return ResolvedRange{}, fmt.Errorf("unsupported window %q", window)
	}
	if !start.Before(end) {
		return ResolvedRange{}, fmt.Errorf("start_at must be before end_at")
	}
	if end.After(now.UTC()) {
		return ResolvedRange{}, fmt.Errorf("end_at must not be in the future")
	}
	duration := end.Sub(start)
	if duration > MaxRange {
		return ResolvedRange{}, fmt.Errorf("range must not exceed %s", MaxRange)
	}
	if bucket == "" {
		bucket = DefaultBucketForDuration(duration)
	}
	if !IsBucket(bucket) {
		return ResolvedRange{}, fmt.Errorf("unsupported bucket %q", bucket)
	}
	bucketDuration, err := BucketDuration(bucket)
	if err != nil {
		return ResolvedRange{}, err
	}
	if bucketCount := int((duration + bucketDuration - 1) / bucketDuration); bucketCount > MaxBucketPoints {
		return ResolvedRange{}, fmt.Errorf("range produces %d buckets; maximum is %d", bucketCount, MaxBucketPoints)
	}
	return ResolvedRange{Window: window, Bucket: bucket, Start: start.UTC(), End: end.UTC()}, nil
}

func parseRangeTime(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02T15:04", raw, loc)
}

func defaultBucketForWindow(window Window) Bucket {
	switch window {
	case Window1h:
		return Bucket5m
	case Window6h:
		return Bucket15m
	case Window24h:
		return Bucket1h
	case Window7d:
		return Bucket6h
	default:
		return ""
	}
}

func WindowDuration(window Window) (time.Duration, error) {
	switch window {
	case Window1h:
		return time.Hour, nil
	case Window6h:
		return 6 * time.Hour, nil
	case Window24h:
		return 24 * time.Hour, nil
	case Window7d:
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported window %q", window)
	}
}

func RollupWindowSeconds(bucket Bucket) (int, error) {
	switch bucket {
	case Bucket5m:
		return 300, nil
	case Bucket15m:
		// The 15-minute API bucket is composed from persisted 5-minute rows.
		return 300, nil
	case Bucket1h, Bucket6h:
		return 3600, nil
	default:
		return 0, fmt.Errorf("unsupported bucket %q", bucket)
	}
}

type Filter struct {
	UserIDs      []int64
	APIKeyIDs    []int64
	GroupIDs     []int64
	RouteAliases []string
	AccountIDs   []int64
	Department   string
}

func ParseFilter(values url.Values) (Filter, error) {
	userIDs, err := parsePositiveInt64Values("user_id", values["user_id"])
	if err != nil {
		return Filter{}, err
	}
	apiKeyIDs, err := parsePositiveInt64Values("api_key_id", values["api_key_id"])
	if err != nil {
		return Filter{}, err
	}
	groupIDs, err := parsePositiveInt64Values("group_id", values["group_id"])
	if err != nil {
		return Filter{}, err
	}
	accountIDs, err := parsePositiveInt64Values("account_id", values["account_id"])
	if err != nil {
		return Filter{}, err
	}
	routeAliases := normalizeStringValues(values["route_alias"])
	if len(routeAliases) > MaxFilterValues {
		return Filter{}, fmt.Errorf("route_alias accepts at most %d values", MaxFilterValues)
	}
	return Filter{
		UserIDs:      userIDs,
		APIKeyIDs:    apiKeyIDs,
		GroupIDs:     groupIDs,
		RouteAliases: routeAliases,
		AccountIDs:   accountIDs,
		Department:   strings.TrimSpace(values.Get("department")),
	}, nil
}

func parsePositiveInt64Values(name string, values []string) ([]int64, error) {
	if len(values) > MaxFilterValues {
		return nil, fmt.Errorf("%s accepts at most %d values", name, MaxFilterValues)
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("%s must contain positive integers", name)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func normalizeStringValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" || len(raw) > 255 {
			continue
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		result = append(result, raw)
	}
	return result
}

func ValidatePage(page, pageSize int) error {
	if page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		return fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)
	}
	return nil
}

func ValidateRatio(value float64, field string) error {
	if value < 0 || value > 1 {
		return fmt.Errorf("%s must use the 0..1 ratio scale", field)
	}
	return nil
}

type Meta struct {
	AsOf          time.Time `json:"as_of"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Window        Window    `json:"window"`
	Bucket        Bucket    `json:"bucket"`
	Timezone      string    `json:"timezone"`
	Sources       []string  `json:"sources"`
	Consistency   string    `json:"consistency"`
	LagSeconds    int64     `json:"lag_seconds"`
	Complete      bool      `json:"complete"`
	Warnings      []string  `json:"warnings"`
	Coverage      string    `json:"coverage,omitempty"`
	ETag          string    `json:"etag,omitempty"`
	TotalEstimate *int64    `json:"total_estimated,omitempty"`
}

type MetricValue struct {
	Code  string  `json:"code"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type SnapshotResponse struct {
	Meta          Meta                  `json:"meta"`
	Metrics       []MetricValue         `json:"metrics"`
	TopUsers      []DimensionRow        `json:"top_users"`
	TopGroups     []DimensionRow        `json:"top_groups"`
	TopAliases    []DimensionRow        `json:"top_aliases"`
	TopAccounts   []DimensionRow        `json:"top_accounts"`
	TopModels     []DimensionRow        `json:"top_models"`
	Pressure      SnapshotPressure      `json:"pressure"`
	Capacity      CapacitySummary       `json:"capacity"`
	FilterOptions SnapshotFilterOptions `json:"filter_options"`
}

type CapacityLevel struct {
	Name        string  `json:"name,omitempty"`
	Current     int     `json:"current"`
	Limit       int     `json:"limit"`
	Utilization float64 `json:"utilization"`
	Available   bool    `json:"available"`
}

type CapacitySummary struct {
	Global             CapacityLevel `json:"global"`
	User               CapacityLevel `json:"user"`
	Account            CapacityLevel `json:"account"`
	Route              CapacityLevel `json:"route"`
	QueueCurrent       int           `json:"queue_current"`
	AverageWaitMS      *float64      `json:"average_wait_ms,omitempty"`
	TodayRateLimited   int64         `json:"today_rate_limited_count"`
	RateLimitAvailable bool          `json:"rate_limit_available"`
	WaitSamples        int64         `json:"wait_sample_count"`
}

type SnapshotPressure struct {
	Accounts   []PressureRow `json:"accounts"`
	Candidates []PressureRow `json:"candidates"`
}

type PressureRow struct {
	AccountID   int64   `json:"account_id"`
	AccountName string  `json:"account_name,omitempty"`
	GroupID     int64   `json:"group_id,omitempty"`
	GroupName   string  `json:"group_name,omitempty"`
	RouteAlias  string  `json:"route_alias,omitempty"`
	Current     int     `json:"current"`
	Limit       int     `json:"limit"`
	Utilization float64 `json:"utilization"`
	Waiting     int     `json:"waiting"`
}

type SnapshotFilterOptions struct {
	Users    []UserFilterOption    `json:"users"`
	Groups   []NamedFilterOption   `json:"groups"`
	Aliases  []NamedFilterOption   `json:"aliases"`
	Accounts []AccountFilterOption `json:"accounts"`
}

type UserFilterOption struct {
	UserID     int64  `json:"user_id"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

type NamedFilterOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type AccountFilterOption struct {
	AccountID  int64  `json:"account_id"`
	Name       string `json:"name"`
	BoundModel string `json:"bound_model"`
}

type DimensionRow struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type Series struct {
	Key    string    `json:"key"`
	Name   string    `json:"name"`
	Points []float64 `json:"points"`
}

type TimeSeriesResponse struct {
	Meta        Meta     `json:"meta"`
	Labels      []string `json:"labels"`
	Series      []Series `json:"series"`
	Summary     []Series `json:"summary,omitempty"`
	Concurrency []int64  `json:"concurrency,omitempty"`
}

type HeatmapCell struct {
	Row               string     `json:"row"`
	Column            string     `json:"column"`
	Value             float64    `json:"value"`
	Available         bool       `json:"available"`
	ActualConcurrency int        `json:"actual_concurrency,omitempty"`
	ConcurrencyLimit  int        `json:"concurrency_limit,omitempty"`
	SampleAt          *time.Time `json:"sample_at,omitempty"`
}

type HeatmapResponse struct {
	Meta      Meta          `json:"meta"`
	ValueKind string        `json:"value_kind"`
	Available bool          `json:"available"`
	Rows      []string      `json:"rows"`
	Columns   []string      `json:"columns"`
	Cells     []HeatmapCell `json:"cells"`
}

type PathRow struct {
	UserID                      int64      `json:"user_id"`
	UserEmail                   string     `json:"user_email"`
	Department                  string     `json:"department"`
	APIKeyID                    int64      `json:"api_key_id"`
	APIKeyName                  string     `json:"api_key_name"`
	GroupID                     int64      `json:"group_id"`
	GroupName                   string     `json:"group_name"`
	RouteAlias                  string     `json:"route_alias"`
	AccountID                   int64      `json:"account_id"`
	AccountName                 string     `json:"account_name"`
	BoundModel                  string     `json:"bound_model"`
	Tokens                      int64      `json:"tokens"`
	RequestCount                int64      `json:"request_count"`
	SuccessRate                 float64    `json:"success_rate"`
	AverageDurationMS           float64    `json:"average_duration_ms"`
	AverageFirstTokenMS         float64    `json:"average_first_token_ms"`
	DurationSampleCount         int64      `json:"duration_sample_count"`
	TTFTTotalSampleCount        int64      `json:"ttft_sample_count"`
	P95FirstTokenMS             int64      `json:"p95_first_token_ms"`
	DataAt                      *time.Time `json:"data_at,omitempty"`
	ErrorStage                  string     `json:"error_stage,omitempty"`
	PreviousAverageDurationMS   *float64   `json:"previous_average_duration_ms,omitempty"`
	PreviousAverageFirstTokenMS *float64   `json:"previous_average_first_token_ms,omitempty"`
	Current                     int        `json:"current"`
	Limit                       int        `json:"limit"`
	LegacyInferred              bool       `json:"legacy_inferred"`
	Status                      string     `json:"status"`
	ModelAttribution            string     `json:"model_attribution"`
}

type PathsResponse struct {
	Meta  Meta      `json:"meta"`
	Total int       `json:"total"`
	Rows  []PathRow `json:"rows"`
}

type RelationNode struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	Label            string `json:"label"`
	UserID           int64  `json:"user_id,omitempty"`
	UserEmail        string `json:"user_email,omitempty"`
	Department       string `json:"department,omitempty"`
	APIKeyID         int64  `json:"api_key_id,omitempty"`
	GroupID          int64  `json:"group_id,omitempty"`
	RouteAlias       string `json:"route_alias,omitempty"`
	RoutePattern     string `json:"route_pattern,omitempty"`
	AccountID        int64  `json:"account_id,omitempty"`
	BoundModel       string `json:"bound_model,omitempty"`
	ModelAttribution string `json:"model_attribution,omitempty"`
}

type RelationEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	Idle bool   `json:"idle"`
}

type RelationsResponse struct {
	Meta              Meta           `json:"meta"`
	Nodes             []RelationNode `json:"nodes"`
	Edges             []RelationEdge `json:"edges"`
	DataQualityErrors []string       `json:"data_quality_errors"`
}
