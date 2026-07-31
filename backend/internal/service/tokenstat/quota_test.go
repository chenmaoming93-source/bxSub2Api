package tokenstat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
