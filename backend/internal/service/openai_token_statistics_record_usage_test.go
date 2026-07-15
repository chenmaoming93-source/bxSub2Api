package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type tokenStatisticsAccumulatorStub struct {
	calls int
	last  TokenStatisticsIncrement
	err   error
}

func (s *tokenStatisticsAccumulatorStub) Accumulate(_ context.Context, increment TokenStatisticsIncrement) error {
	s.calls++
	s.last = increment
	return s.err
}

func TestOpenAIRecordUsageTokenStatisticsSwitch(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	accumulator := &tokenStatisticsAccumulatorStub{}
	svc.SetTokenStatisticsAccumulator(accumulator)
	groupID := int64(9)
	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "token-stats", Usage: OpenAIUsage{InputTokens: 10, OutputTokens: 5}, Model: "gpt-5", UpstreamModel: "gpt-5-upstream", Duration: time.Second}, APIKey: &APIKey{ID: 1, GroupID: &groupID, Group: &Group{ID: groupID}}, User: &User{ID: 2}, Account: &Account{ID: 3, Type: AccountTypeAPIKey}, RouteAlias: "chat"})
	require.NoError(t, err)
	require.Equal(t, 1, usageRepo.calls)
	require.Zero(t, billingRepo.calls)
	require.Equal(t, 1, accumulator.calls)
	require.Equal(t, int64(15), accumulator.last.TotalTokens)
	require.Equal(t, "gpt-5-upstream", accumulator.last.Model)
	require.Zero(t, usageRepo.lastLog.TotalCost)
	require.Zero(t, usageRepo.lastLog.ActualCost)
}
