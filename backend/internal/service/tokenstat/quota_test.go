package tokenstat

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type trackingQuotaReader struct {
	values map[string]int64
	fields []string
}

func (r *trackingQuotaReader) Read(_ context.Context, _ string, field string) (int64, error) {
	r.fields = append(r.fields, field)
	return r.values[field], nil
}

type quotaReaderStub struct {
	value int64
	err   error
}

func (r quotaReaderStub) Read(context.Context, string, string) (int64, error) {
	return r.value, r.err
}

func TestDynamicTokenQuotaObserveEnforceAndFailOpen(t *testing.T) {
	values := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(7)}
	base := QuotaRule{
		ID: 1, ProjectionID: 2, DimensionCodes: []DimensionCode{DimensionUserID},
		DimensionValues: values, MetricCode: MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 100,
	}
	for _, tc := range []struct {
		name     string
		mode     QuotaMode
		reader   quotaReaderStub
		enforced bool
		count    int
	}{
		{"observe", QuotaModeObserve, quotaReaderStub{value: 100}, false, 1},
		{"enforce", QuotaModeEnforce, quotaReaderStub{value: 101}, true, 1},
		{"below", QuotaModeEnforce, quotaReaderStub{value: 99}, false, 1},
		{"redis failure", QuotaModeEnforce, quotaReaderStub{err: errors.New("timeout")}, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := NewQuotaChecker(tc.reader, 16)
			rule := base
			rule.Mode = tc.mode
			checker.ReplaceRules([]QuotaRule{rule})
			decisions := checker.Check(context.Background(), time.Now(), values)
			require.Len(t, decisions, tc.count)
			if tc.count > 0 {
				require.Equal(t, tc.enforced, decisions[0].Enforced)
			}
		})
	}
}

func TestDynamicTokenQuotaTwoStageMatching(t *testing.T) {
	user := map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(7)}
	withAccount := map[DimensionCode]DimensionValue{
		DimensionUserID: Int64Value(7), DimensionAccountID: Int64Value(9),
	}
	checker := NewQuotaChecker(quotaReaderStub{value: 10}, 16)
	checker.ReplaceRules([]QuotaRule{
		{ID: 1, ProjectionID: 1, DimensionCodes: []DimensionCode{DimensionUserID}, DimensionValues: user, MetricCode: MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 10, Mode: QuotaModeEnforce},
		{ID: 2, ProjectionID: 2, DimensionCodes: []DimensionCode{DimensionUserID, DimensionAccountID}, DimensionValues: withAccount, MetricCode: MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 10, Mode: QuotaModeEnforce},
	})
	require.Len(t, checker.Check(context.Background(), time.Now(), user), 1, "pre-scheduling only matches known dimensions")
	require.Len(t, checker.Check(context.Background(), time.Now(), withAccount), 2, "post-selection matches account dimensions")
}

func TestDynamicTokenQuotaWildcardUsesConcreteRequestIdentityWithoutMutatingRule(t *testing.T) {
	codes := []DimensionCode{DimensionGroupID, DimensionRouteAlias, DimensionAccountID}
	ruleValues := map[DimensionCode]DimensionValue{
		DimensionGroupID: Int64Value(3), DimensionRouteAlias: WildcardValue(), DimensionAccountID: Int64Value(18),
	}
	ruleSnapshot := map[DimensionCode]DimensionValue{
		DimensionGroupID: Int64Value(3), DimensionRouteAlias: WildcardValue(), DimensionAccountID: Int64Value(18),
	}
	claude := map[DimensionCode]DimensionValue{
		DimensionGroupID: Int64Value(3), DimensionRouteAlias: StringValue("claude-code"), DimensionAccountID: Int64Value(18),
	}
	gpt := map[DimensionCode]DimensionValue{
		DimensionGroupID: Int64Value(3), DimensionRouteAlias: StringValue("gpt-code"), DimensionAccountID: Int64Value(18),
	}
	claudeID, _ := BuildDimensionIdentity(codes, claude)
	gptID, _ := BuildDimensionIdentity(codes, gpt)
	reader := &trackingQuotaReader{values: map[string]int64{
		claudeID.HashHex() + ":total_tokens": 600_000,
		gptID.HashHex() + ":total_tokens":    450_000,
	}}
	checker := NewQuotaChecker(reader, 16)
	checker.ReplaceRules([]QuotaRule{{
		ID: 1, ProjectionID: 9, DimensionCodes: codes, DimensionValues: ruleValues,
		MetricCode: MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 1_000_000, Mode: QuotaModeEnforce,
	}})

	claudeDecision := checker.Check(context.Background(), time.Now(), claude)
	gptDecision := checker.Check(context.Background(), time.Now(), gpt)
	require.Len(t, claudeDecision, 1)
	require.Len(t, gptDecision, 1)
	require.Equal(t, int64(600_000), claudeDecision[0].Used)
	require.Equal(t, int64(450_000), gptDecision[0].Used)
	require.False(t, claudeDecision[0].Enforced)
	require.False(t, gptDecision[0].Enforced)
	require.NotEqual(t, reader.fields[0], reader.fields[1], "different concrete route aliases must use different counters")
	require.True(t, reflect.DeepEqual(ruleSnapshot, ruleValues), "wildcard rule values must not be rewritten")
}

func TestDynamicTokenQuotaWildcardRequiresActualValueAndDeduplicatesReads(t *testing.T) {
	codes := []DimensionCode{DimensionGroupID, DimensionRouteAlias}
	rule := QuotaRule{
		ProjectionID: 9, DimensionCodes: codes,
		DimensionValues: map[DimensionCode]DimensionValue{DimensionGroupID: Int64Value(3), DimensionRouteAlias: WildcardValue()},
		MetricCode:      MetricTotalTokens, PeriodType: PeriodDay, LimitValue: 100, Mode: QuotaModeEnforce,
	}
	reader := &trackingQuotaReader{values: map[string]int64{}}
	checker := NewQuotaChecker(reader, 16)
	second := rule
	second.ID = 2
	rule.ID = 1
	checker.ReplaceRules([]QuotaRule{rule, second})
	require.Empty(t, checker.Check(context.Background(), time.Now(), map[DimensionCode]DimensionValue{DimensionGroupID: Int64Value(3)}))
	require.Empty(t, reader.fields)

	decisions := checker.Check(context.Background(), time.Now(), map[DimensionCode]DimensionValue{
		DimensionGroupID: Int64Value(3), DimensionRouteAlias: StringValue("deepseek"),
	})
	require.Len(t, decisions, 2)
	require.Len(t, reader.fields, 1, "same concrete lookup identity is read once per request")
}

func TestUsageEventRejectsWildcardDimension(t *testing.T) {
	event := UsageEvent{OccurredAt: time.Now(), Dimensions: map[DimensionCode]DimensionValue{DimensionRouteAlias: WildcardValue()}}
	require.Error(t, event.Validate())
	require.NoError(t, validateQuotaDimensionValue(DimensionDefinition{Code: DimensionRouteAlias, ValueType: ValueTypeString}, WildcardValue()))
}
