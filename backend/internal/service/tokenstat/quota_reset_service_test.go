package tokenstat

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type runtimeStub bool

func (r runtimeStub) Enabled() bool { return bool(r) }

type discoveryStub struct {
	result IdentityDiscoveryResult
	err    error
	calls  int
}

func (s *discoveryStub) Discover(context.Context, ResetIdentityDiscoveryRequest) (IdentityDiscoveryResult, error) {
	s.calls++
	return s.result, s.err
}

type snapshotWriterStub struct {
	results []QuotaResetEntryResult
	batches [][]StatisticIdentity
	index   int
}

func (s *snapshotWriterStub) SnapshotBatch(_ context.Context, identities []StatisticIdentity) ([]QuotaResetEntryResult, error) {
	s.batches = append(s.batches, append([]StatisticIdentity(nil), identities...))
	result := append([]QuotaResetEntryResult(nil), s.results[s.index:s.index+len(identities)]...)
	s.index += len(identities)
	return result, nil
}

func TestQuotaResetServiceBatchesAndPreservesPartialSuccess(t *testing.T) {
	identities := make([]StatisticIdentity, 5)
	for i := range identities {
		identities[i].ProjectionID = int64(i + 1)
	}
	discovery := &discoveryStub{result: IdentityDiscoveryResult{Status: IdentityDiscoveryFound, MatchedQuotaCount: 2, MatchedQuotas: []QuotaRule{{ID: 10, Name: "quota-a"}, {ID: 20, Name: "quota-b"}}, Identities: identities}}
	writer := &snapshotWriterStub{results: []QuotaResetEntryResult{
		{Status: QuotaResetEntryReset}, {Status: QuotaResetEntryFailed, Err: errors.New("one failed")},
		{Status: QuotaResetEntryNoUsage}, {Status: QuotaResetEntryReset}, {Status: QuotaResetEntryReset},
	}}
	before := MetricsSnapshot()
	service := NewQuotaResetService(discovery, writer, runtimeStub(true), 2)
	result, err := service.ResetQuotaUsage(context.Background(), ResetIdentityDiscoveryRequest{MetricCode: MetricTotalTokens, PeriodType: PeriodDay, IncludeDetails: true})
	require.NoError(t, err)
	require.Equal(t, QuotaResetStatusPartialReset, result.Status)
	require.Equal(t, 2, result.MatchedQuotaCount)
	require.Equal(t, 5, result.MatchedUsageCount)
	require.Equal(t, 3, result.ResetCount)
	require.Equal(t, 1, result.NoUsageCount)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, []string{"quota-a", "quota-b"}, []string{result.MatchedQuotas[0].Name, result.MatchedQuotas[1].Name})
	require.Equal(t, []QuotaResetEntryStatus{QuotaResetEntryReset, QuotaResetEntryFailed, QuotaResetEntryNoUsage, QuotaResetEntryReset, QuotaResetEntryReset}, []QuotaResetEntryStatus{result.MatchedEntries[0].Status, result.MatchedEntries[1].Status, result.MatchedEntries[2].Status, result.MatchedEntries[3].Status, result.MatchedEntries[4].Status})
	require.Len(t, writer.batches, 3)
	require.Equal(t, []int{2, 2, 1}, []int{len(writer.batches[0]), len(writer.batches[1]), len(writer.batches[2])})
	require.Equal(t, int64(5), writer.batches[2][0].ProjectionID, "batching must preserve candidate order")
	after := MetricsSnapshot()
	require.Equal(t, before.QuotaResetRequests+1, after.QuotaResetRequests)
	require.Equal(t, before.QuotaResetEntries+3, after.QuotaResetEntries)
	require.Equal(t, before.QuotaResetPartial+1, after.QuotaResetPartial)
	require.Equal(t, before.QuotaResetFailedEntries+1, after.QuotaResetFailedEntries)
}

func TestQuotaResetServiceTerminalStatuses(t *testing.T) {
	request := ResetIdentityDiscoveryRequest{MetricCode: MetricTotalTokens, PeriodType: PeriodDay}
	for _, testCase := range []struct {
		name      string
		discovery IdentityDiscoveryResult
		entries   []QuotaResetEntryResult
		status    QuotaResetStatus
		wantErr   bool
	}{
		{name: "no quota", discovery: IdentityDiscoveryResult{Status: IdentityDiscoveryNoQuota}, status: QuotaResetStatusNoQuota},
		{name: "no discovered usage", discovery: IdentityDiscoveryResult{Status: IdentityDiscoveryNoUsage, MatchedQuotaCount: 1}, status: QuotaResetStatusNoUsage},
		{name: "all reset", discovery: IdentityDiscoveryResult{Status: IdentityDiscoveryFound, Identities: make([]StatisticIdentity, 2)}, entries: []QuotaResetEntryResult{{Status: QuotaResetEntryReset}, {Status: QuotaResetEntryReset}}, status: QuotaResetStatusReset},
		{name: "raw disappeared", discovery: IdentityDiscoveryResult{Status: IdentityDiscoveryFound, Identities: make([]StatisticIdentity, 2)}, entries: []QuotaResetEntryResult{{Status: QuotaResetEntryNoUsage}, {Status: QuotaResetEntryNoUsage}}, status: QuotaResetStatusNoUsage},
		{name: "all failed", discovery: IdentityDiscoveryResult{Status: IdentityDiscoveryFound, Identities: make([]StatisticIdentity, 2)}, entries: []QuotaResetEntryResult{{Status: QuotaResetEntryFailed}, {Status: QuotaResetEntryFailed}}, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			discovery := &discoveryStub{result: testCase.discovery}
			writer := &snapshotWriterStub{results: testCase.entries}
			result, err := NewQuotaResetService(discovery, writer, runtimeStub(true), 100).ResetQuotaUsage(context.Background(), request)
			if testCase.wantErr {
				require.ErrorIs(t, err, ErrTokenQuotaResetUnavailable)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.status, result.Status)
			}
			if testCase.discovery.Status != IdentityDiscoveryFound {
				require.Empty(t, writer.batches)
			}
		})
	}
}

func TestQuotaResetServiceDisabledAndValidationErrors(t *testing.T) {
	discovery := &discoveryStub{}
	writer := &snapshotWriterStub{}
	_, err := NewQuotaResetService(discovery, writer, runtimeStub(false), 10).ResetQuotaUsage(context.Background(), ResetIdentityDiscoveryRequest{})
	require.ErrorIs(t, err, ErrTokenQuotaResetUnavailable)
	require.Zero(t, discovery.calls)
	require.Empty(t, writer.batches)

	discovery.err = ErrInvalidQuotaResetRequest
	_, err = NewQuotaResetService(discovery, writer, runtimeStub(true), 10).ResetQuotaUsage(context.Background(), ResetIdentityDiscoveryRequest{})
	require.ErrorIs(t, err, ErrInvalidQuotaResetRequest)
}
