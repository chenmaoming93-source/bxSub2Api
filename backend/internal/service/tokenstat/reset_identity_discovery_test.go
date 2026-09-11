package tokenstat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type resetScopeStub struct {
	projections []ProjectionDefinition
	rules       []QuotaRule
	calls       int
}

func (s *resetScopeStub) ActiveProjections() []ProjectionDefinition { s.calls++; return s.projections }
func (s *resetScopeStub) ActiveQuotaRules() []QuotaRule             { s.calls++; return s.rules }

type dirtyIdentitySourceStub struct {
	identities []StatisticIdentity
	calls      int
	order      *[]string
	err        error
}

func (s *dirtyIdentitySourceStub) ScanCurrent(_ context.Context, visit func(StatisticIdentity) error) error {
	s.calls++
	if s.order != nil {
		*s.order = append(*s.order, "dirty")
	}
	if s.err != nil {
		return s.err
	}
	for _, identity := range s.identities {
		if err := visit(identity); err != nil {
			return err
		}
	}
	return nil
}

type aggregateIdentitySourceStub struct {
	pages [][]StatisticIdentity
	calls int
	order *[]string
}

func (s *aggregateIdentitySourceStub) ListAggregateIdentities(_ context.Context, query AggregateIdentityQuery) ([]StatisticIdentity, *AggregateIdentityCursor, error) {
	s.calls++
	if s.order != nil {
		*s.order = append(*s.order, "mysql")
	}
	pageIndex := s.calls - 1
	if pageIndex >= len(s.pages) {
		return nil, nil, nil
	}
	page := s.pages[pageIndex]
	if pageIndex == len(s.pages)-1 {
		return page, nil, nil
	}
	last := page[len(page)-1]
	return page, &AggregateIdentityCursor{ProjectionID: last.ProjectionID, DimensionHash: last.DimensionHash}, nil
}

func TestResetIdentityDiscoveryMatchesProjectionSupersetsAndConcreteQuota(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	period := NaturalPeriods(at, location)[0]
	requestValues := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42), DimensionGroupID: Int64Value(7)}
	deepseekValues := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42), DimensionGroupID: Int64Value(7), DimensionUpstreamModel: StringValue("deepseek")}
	claudeValues := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42), DimensionGroupID: Int64Value(7), DimensionUpstreamModel: StringValue("claude")}
	scopes := &resetScopeStub{
		projections: []ProjectionDefinition{
			{ID: 1, Name: "user-group", DimensionCodes: []DimensionCode{DimensionUserID, DimensionGroupID}, MetricCodes: []MetricCode{MetricTotalTokens}},
			{ID: 2, Name: "user-group-model", DimensionCodes: []DimensionCode{DimensionUserID, DimensionGroupID, DimensionUpstreamModel}, MetricCodes: []MetricCode{MetricTotalTokens}},
			{ID: 3, DimensionCodes: []DimensionCode{DimensionGroupID, DimensionUpstreamModel}, MetricCodes: []MetricCode{MetricTotalTokens}},
		},
		rules: []QuotaRule{
			{ID: 1, Name: "daily user group", ProjectionID: 1, DimensionCodes: []DimensionCode{DimensionUserID, DimensionGroupID}, DimensionValues: requestValues, MetricCode: MetricTotalTokens, PeriodType: PeriodDay},
			{ID: 2, Name: "daily model", ProjectionID: 2, DimensionCodes: []DimensionCode{DimensionUserID, DimensionGroupID, DimensionUpstreamModel}, DimensionValues: deepseekValues, MetricCode: MetricTotalTokens, PeriodType: PeriodDay},
		},
	}
	identityOne := StatisticIdentity{Period: period, ProjectionID: 1, DimensionHash: [16]byte{1}, DimensionValues: requestValues, MetricCode: MetricTotalTokens}
	identityDeepseek := StatisticIdentity{Period: period, ProjectionID: 2, DimensionHash: [16]byte{2}, DimensionValues: deepseekValues, MetricCode: MetricTotalTokens}
	identityClaude := StatisticIdentity{Period: period, ProjectionID: 2, DimensionHash: [16]byte{3}, DimensionValues: claudeValues, MetricCode: MetricTotalTokens}
	wrongGroup := StatisticIdentity{Period: period, ProjectionID: 1, DimensionHash: [16]byte{4}, DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42), DimensionGroupID: Int64Value(8)}, MetricCode: MetricTotalTokens}
	wrongMetric := StatisticIdentity{Period: period, ProjectionID: 1, DimensionHash: [16]byte{5}, DimensionValues: requestValues, MetricCode: MetricCode("other")}
	previousPeriod := StatisticIdentity{Period: Period{Type: PeriodDay, Start: period.Start.AddDate(0, 0, -1), End: period.Start}, ProjectionID: 1, DimensionHash: [16]byte{6}, DimensionValues: requestValues, MetricCode: MetricTotalTokens}
	missingRequestDimension := StatisticIdentity{Period: period, ProjectionID: 3, DimensionHash: [16]byte{7}, DimensionValues: map[DimensionCode]DimensionValue{DimensionGroupID: Int64Value(7), DimensionUpstreamModel: StringValue("deepseek")}, MetricCode: MetricTotalTokens}
	order := []string{}
	dirty := &dirtyIdentitySourceStub{identities: []StatisticIdentity{identityOne, identityDeepseek, wrongGroup, wrongMetric, previousPeriod, missingRequestDimension}, order: &order}
	aggregates := &aggregateIdentitySourceStub{pages: [][]StatisticIdentity{{identityOne}, {identityClaude}}, order: &order}
	service := NewResetIdentityDiscoveryService(scopes, dirty, aggregates, location, 1)
	service.now = func() time.Time { return at }

	result, err := service.Discover(context.Background(), ResetIdentityDiscoveryRequest{DimensionValues: requestValues, MetricCode: MetricTotalTokens, PeriodType: PeriodDay, IncludeDetails: true})
	require.NoError(t, err)
	require.Equal(t, IdentityDiscoveryFound, result.Status)
	require.Equal(t, 2, result.MatchedQuotaCount)
	require.Len(t, result.Identities, 2)
	require.Equal(t, int64(1), result.Identities[0].ProjectionID)
	require.Equal(t, int64(2), result.Identities[1].ProjectionID)
	require.Equal(t, "user-group", result.Identities[0].ProjectionName)
	require.Equal(t, []int64{1}, result.Identities[0].MatchedQuotaIDs)
	require.Equal(t, "user-group-model", result.Identities[1].ProjectionName)
	require.Equal(t, []int64{2}, result.Identities[1].MatchedQuotaIDs)
	require.Equal(t, []string{"daily user group", "daily model"}, []string{result.MatchedQuotas[0].Name, result.MatchedQuotas[1].Name})
	require.Equal(t, []string{"dirty", "mysql", "mysql"}, order)
	require.Equal(t, 1, dirty.calls, "current dirty must be traversed exactly once")
	require.Equal(t, 2, scopes.calls, "scope snapshots must not be re-read")
}

func TestResetIdentityDiscoveryDistinguishesNoQuotaAndNoUsage(t *testing.T) {
	location := time.UTC
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	request := ResetIdentityDiscoveryRequest{DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(42)}, MetricCode: MetricTotalTokens, PeriodType: PeriodDay}
	projection := ProjectionDefinition{ID: 1, DimensionCodes: []DimensionCode{DimensionUserID}, MetricCodes: []MetricCode{MetricTotalTokens}}

	t.Run("no quota avoids both sources", func(t *testing.T) {
		dirty := &dirtyIdentitySourceStub{}
		aggregates := &aggregateIdentitySourceStub{}
		service := NewResetIdentityDiscoveryService(&resetScopeStub{projections: []ProjectionDefinition{projection}}, dirty, aggregates, location, 10)
		service.now = func() time.Time { return at }
		result, err := service.Discover(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, IdentityDiscoveryNoQuota, result.Status)
		require.Zero(t, dirty.calls)
		require.Zero(t, aggregates.calls)
	})

	t.Run("quota without identities", func(t *testing.T) {
		rule := QuotaRule{ProjectionID: 1, DimensionCodes: []DimensionCode{DimensionUserID}, DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: WildcardValue()}, MetricCode: MetricTotalTokens, PeriodType: PeriodDay}
		dirty := &dirtyIdentitySourceStub{}
		aggregates := &aggregateIdentitySourceStub{}
		service := NewResetIdentityDiscoveryService(&resetScopeStub{projections: []ProjectionDefinition{projection}, rules: []QuotaRule{rule}}, dirty, aggregates, location, 10)
		service.now = func() time.Time { return at }
		result, err := service.Discover(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, IdentityDiscoveryNoUsage, result.Status)
		require.Equal(t, 1, result.MatchedQuotaCount)
		require.Equal(t, 1, dirty.calls)
		require.Equal(t, 1, aggregates.calls)
	})
}

func TestResetIdentityDiscoveryValidatesRequestAndPropagatesSourceFailure(t *testing.T) {
	location := time.UTC
	valid := ResetIdentityDiscoveryRequest{DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(1)}, MetricCode: MetricTotalTokens, PeriodType: PeriodDay}
	service := NewResetIdentityDiscoveryService(&resetScopeStub{}, &dirtyIdentitySourceStub{}, &aggregateIdentitySourceStub{}, location, 10)
	for _, request := range []ResetIdentityDiscoveryRequest{
		{},
		{DimensionValues: map[DimensionCode]DimensionValue{DimensionCode("bad"): StringValue("x")}, MetricCode: MetricTotalTokens, PeriodType: PeriodDay},
		{DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: WildcardValue()}, MetricCode: MetricTotalTokens, PeriodType: PeriodDay},
		{DimensionValues: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(1)}, MetricCode: MetricTotalTokens, PeriodType: PeriodType("X")},
	} {
		_, err := service.Discover(context.Background(), request)
		require.Error(t, err)
	}

	scopes := &resetScopeStub{projections: []ProjectionDefinition{{ID: 1, DimensionCodes: []DimensionCode{DimensionUserID}, MetricCodes: []MetricCode{MetricTotalTokens}}}, rules: []QuotaRule{{ProjectionID: 1, DimensionCodes: []DimensionCode{DimensionUserID}, DimensionValues: valid.DimensionValues, MetricCode: MetricTotalTokens, PeriodType: PeriodDay}}}
	service = NewResetIdentityDiscoveryService(scopes, &dirtyIdentitySourceStub{err: errors.New("redis unavailable")}, &aggregateIdentitySourceStub{}, location, 10)
	_, err := service.Discover(context.Background(), valid)
	require.ErrorContains(t, err, "redis unavailable")
}
