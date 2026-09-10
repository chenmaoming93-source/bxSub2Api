package service

import (
	"context"
	"errors"
	"testing"
	"time"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/stretchr/testify/require"
)

type externalHistoryQueryStub struct {
	result *tokenstat.UsageQueryResult
	err    error
	calls  int
	inputs []tokenstat.UsageQueryInput
}

func (s *externalHistoryQueryStub) QueryUsage(_ context.Context, input tokenstat.UsageQueryInput) (*tokenstat.UsageQueryResult, error) {
	s.calls++
	s.inputs = append(s.inputs, input)
	return s.result, s.err
}

func dailyUsageService(history ExternalTokenUsageHistoryQuerier, projections []tokenstat.ProjectionDefinition, location *time.Location) *ExternalTokenUsageService {
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	svc.ConfigureCurrentUsage(&externalCurrentReaderStub{}, externalProjectionStub(projections), location)
	svc.ConfigureHistoryQuery(history)
	return svc
}

func dailyInput() ExternalDailyUsageInput {
	return ExternalDailyUsageInput{
		GroupName: "public",
		APIKey:    "sk-ldap-key-0123456789",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	}
}

func TestExternalDailyUsageQueryReturnsDataDays(t *testing.T) {
	location := time.UTC
	enabledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, location)
	syncedAt := time.Date(2026, 7, 31, 23, 55, 0, 0, location)
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{
		Rows: []tokenstat.UsageQueryRow{
			{PeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, location), PeriodEnd: time.Date(2026, 7, 2, 0, 0, 0, 0, location), Value: 12500},
			{PeriodStart: time.Date(2026, 7, 3, 0, 0, 0, 0, location), PeriodEnd: time.Date(2026, 7, 4, 0, 0, 0, 0, location), Value: 9800},
		},
		ProjectionEnabledAt: &enabledAt, LastSyncedAt: &syncedAt, Complete: true,
	}}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, location)

	got, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.NoError(t, err)
	require.True(t, got.DimensionConfigured)
	require.Equal(t, int64(7), got.ProjectionID)
	require.Equal(t, tokenstat.MetricTotalTokens, got.Metric)
	require.Equal(t, []ExternalDailyUsageDay{
		{Date: "2026-07-01", TotalTokens: 12500},
		{Date: "2026-07-03", TotalTokens: 9800},
	}, got.Days)
	require.True(t, got.Complete)
	require.Equal(t, &enabledAt, got.ProjectionEnabledAt)
	require.Equal(t, &syncedAt, got.LastSyncedAt)
	require.Equal(t, 1, history.calls)

	input := history.inputs[0]
	require.Equal(t, int64(7), input.ProjectionID)
	require.Equal(t, tokenstat.PeriodDay, input.PeriodType)
	require.Equal(t, tokenstat.MetricTotalTokens, input.MetricCode)
	require.Equal(t, tokenstat.Int64Value(4), input.Filters[tokenstat.DimensionAPIKeyID])
	require.Equal(t, tokenstat.Int64Value(2), input.Filters[tokenstat.DimensionGroupID])
	require.Equal(t, "time_asc", input.Sort)
	require.Equal(t, 1000, input.PageSize)
}

func TestExternalDailyUsageRangeBoundariesInStatsTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 1, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, location)

	input := dailyInput()
	input.StartDate = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	input.EndDate = time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	_, err = svc.QueryDailyUsage(context.Background(), input)
	require.NoError(t, err)

	start := history.inputs[0].Start
	end := history.inputs[0].End
	require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, location), start)
	require.Equal(t, time.Date(2026, 7, 4, 0, 0, 0, 0, location), end)
	require.Equal(t, "2026-07-01", start.In(location).Format("2006-01-02"))
	require.Equal(t, "2026-07-04", end.In(location).Format("2006-01-02"))
}

func TestExternalDailyUsageUnconfiguredProjection(t *testing.T) {
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	svc := dailyUsageService(history, nil, time.UTC)

	got, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.NoError(t, err)
	require.False(t, got.DimensionConfigured)
	require.Equal(t, "统计维度未配置", got.Message)
	require.Empty(t, got.Days)
	require.NotNil(t, got.Days)
	require.Zero(t, history.calls)

	// 部分维度或非 total_tokens 投影同样视为未配置。
	partial := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 1, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, time.UTC)
	got, err = partial.QueryDailyUsage(context.Background(), dailyInput())
	require.NoError(t, err)
	require.False(t, got.DimensionConfigured)
}

func TestExternalDailyUsageConfiguredButNoData(t *testing.T) {
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, time.UTC)

	got, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.NoError(t, err)
	require.True(t, got.DimensionConfigured)
	require.Empty(t, got.Days)
	require.NotNil(t, got.Days)
	require.Equal(t, 1, history.calls)
}

func TestExternalDailyUsageRequiresExactProjection(t *testing.T) {
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	exact := tokenstat.ProjectionDefinition{ID: 2, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}
	superset4 := tokenstat.ProjectionDefinition{ID: 1, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionUserID, tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionRouteAlias}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}
	superset3 := tokenstat.ProjectionDefinition{ID: 3, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID, tokenstat.DimensionUpstreamModel}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}}

	// 精确签名 api_key_id,group_id 存在时选中它（即使也有超集投影）。
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{superset4, exact, superset3}, time.UTC)
	got, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.NoError(t, err)
	require.True(t, got.DimensionConfigured)
	require.Equal(t, int64(2), got.ProjectionID)

	// 只有超集投影（无论维度多少）一律视为未配置，不得回退计算。
	for _, projections := range [][]tokenstat.ProjectionDefinition{
		{superset4},
		{superset3},
		{superset4, superset3},
	} {
		perCase := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
		svc = dailyUsageService(perCase, projections, time.UTC)
		got, err = svc.QueryDailyUsage(context.Background(), dailyInput())
		require.NoError(t, err)
		require.False(t, got.DimensionConfigured, "projections=%v", projections)
		require.Equal(t, "统计维度未配置", got.Message)
		require.Zero(t, perCase.calls)
	}
}

func TestExternalDailyUsageFilledZeroFillsMissingDays(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{
		Rows: []tokenstat.UsageQueryRow{
			{PeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, location), PeriodEnd: time.Date(2026, 7, 2, 0, 0, 0, 0, location), Value: 12500},
			{PeriodStart: time.Date(2026, 7, 3, 0, 0, 0, 0, location), PeriodEnd: time.Date(2026, 7, 4, 0, 0, 0, 0, location), Value: 9800},
		},
		Complete: true,
	}}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, location)

	input := dailyInput()
	input.StartDate = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	input.EndDate = time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	got, err := svc.QueryDailyUsageFilled(context.Background(), input)
	require.NoError(t, err)
	require.True(t, got.DimensionConfigured)
	require.Equal(t, []ExternalDailyUsageDay{
		{Date: "2026-07-01", TotalTokens: 12500},
		{Date: "2026-07-02", TotalTokens: 0},
		{Date: "2026-07-03", TotalTokens: 9800},
		{Date: "2026-07-04", TotalTokens: 0},
		{Date: "2026-07-05", TotalTokens: 0},
	}, got.Days)
}

func TestExternalDailyUsageFilledConfiguredButNoData(t *testing.T) {
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, time.UTC)

	input := dailyInput()
	input.StartDate = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	input.EndDate = time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	got, err := svc.QueryDailyUsageFilled(context.Background(), input)
	require.NoError(t, err)
	require.True(t, got.DimensionConfigured)
	require.Len(t, got.Days, 3)
	for _, day := range got.Days {
		require.Zero(t, day.TotalTokens)
	}
	require.Equal(t, "2026-07-01", got.Days[0].Date)
	require.Equal(t, "2026-07-03", got.Days[2].Date)
}

func TestExternalDailyUsageFilledUnconfigured(t *testing.T) {
	history := &externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}}
	svc := dailyUsageService(history, nil, time.UTC)
	got, err := svc.QueryDailyUsageFilled(context.Background(), dailyInput())
	require.NoError(t, err)
	require.False(t, got.DimensionConfigured)
	require.Equal(t, "统计维度未配置", got.Message)
	require.Empty(t, got.Days)
	require.Zero(t, history.calls)
}

func TestExternalDailyUsageGroupAndAPIKeyErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*externalTokenDimensionLookupStub)
		want   error
	}{
		{"group not found", func(s *externalTokenDimensionLookupStub) { s.groupErr = ErrGroupNotFound }, ErrGroupNotFound},
		{"api key not found", func(s *externalTokenDimensionLookupStub) { s.keyErr = ErrAPIKeyNotFound }, ErrAPIKeyNotFound},
		{"api key belongs to another group", func(s *externalTokenDimensionLookupStub) { other := int64(99); s.key.GroupID = &other }, ErrAPIKeyMismatch},
		{"api key has no group", func(s *externalTokenDimensionLookupStub) { s.key.GroupID = nil }, ErrAPIKeyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := validExternalTokenLookup()
			tt.mutate(lookup)
			svc := NewExternalTokenUsageService(lookup)
			svc.ConfigureHistoryQuery(&externalHistoryQueryStub{result: &tokenstat.UsageQueryResult{}})
			_, err := svc.QueryDailyUsage(context.Background(), dailyInput())
			require.True(t, errors.Is(err, tt.want), "error=%v", err)
		})
	}
}

func TestExternalDailyUsageHistoryError(t *testing.T) {
	history := &externalHistoryQueryStub{err: errors.New("mysql down")}
	svc := dailyUsageService(history, []tokenstat.ProjectionDefinition{
		{ID: 7, DimensionCodes: []tokenstat.DimensionCode{tokenstat.DimensionAPIKeyID, tokenstat.DimensionGroupID}, MetricCodes: []tokenstat.MetricCode{tokenstat.MetricTotalTokens}},
	}, time.UTC)
	_, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.Error(t, err)
	require.Contains(t, err.Error(), "mysql down")
}

func TestExternalDailyUsageNotConfigured(t *testing.T) {
	svc := NewExternalTokenUsageService(validExternalTokenLookup())
	_, err := svc.QueryDailyUsage(context.Background(), dailyInput())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not configured")
}
