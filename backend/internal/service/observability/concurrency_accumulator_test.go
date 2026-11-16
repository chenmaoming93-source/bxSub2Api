package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrencyAccumulatorWritesOnlyAtMinuteBoundary(t *testing.T) {
	acc := NewConcurrencyAccumulator()
	minute := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	sample := ConcurrencySample{
		Kind: ConcurrencySampleKindAccount, AccountID: 1,
		Current: 2, Limit: 10, Waiting: 1, Available: true,
	}

	flushed, accumulated := acc.Add(minute.Add(10*time.Second), []ConcurrencySample{sample})
	require.Empty(t, flushed)
	require.Equal(t, 1, accumulated)

	sample.Current, sample.Waiting = 5, 3
	flushed, accumulated = acc.Add(minute.Add(20*time.Second), []ConcurrencySample{sample})
	require.Empty(t, flushed)
	require.Equal(t, 1, accumulated)

	sample.Current = 1
	flushed, accumulated = acc.Add(minute.Add(1*time.Minute), []ConcurrencySample{sample})
	require.Len(t, flushed, 1)
	require.Equal(t, 1, accumulated)
	require.Equal(t, 5, flushed[0].CurrentPeak)
	require.Equal(t, int64(7), flushed[0].CurrentSum)
	require.Equal(t, int64(2), flushed[0].SampleCount)
	require.Equal(t, 3, flushed[0].WaitingPeak)
	require.Equal(t, 10, flushed[0].EffectiveLimit)
	require.Equal(t, 5, flushed[0].PeakRatioActual)
	require.Equal(t, 10, flushed[0].PeakRatioLimit)
	require.Equal(t, minute.Add(20*time.Second), flushed[0].PeakRatioSampleAt)
}

func TestConcurrencyAccumulatorSeparatesAccountAndRoute(t *testing.T) {
	acc := NewConcurrencyAccumulator()
	minute := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	_, accumulated := acc.Add(minute, []ConcurrencySample{
		{Kind: ConcurrencySampleKindAccount, AccountID: 1, Current: 3, Limit: 10, Available: true},
		{Kind: ConcurrencySampleKindRoute, AccountID: 1, GroupID: 7, RouteAlias: "coding", Current: 2, Limit: 4, Available: true},
	})
	require.Equal(t, 2, accumulated)
	rows := acc.Flush()
	require.Len(t, rows, 2)
	require.NotEqual(t, rows[0].DimensionHash, rows[1].DimensionHash)
}

func TestConcurrencySampleQualityBecomesStaleAfterSamplerStops(t *testing.T) {
	last := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	require.Equal(t, "unavailable", ConcurrencySampleQuality("", nil, last, 90*time.Second))
	require.Equal(t, "healthy", ConcurrencySampleQuality("healthy", &last, last.Add(20*time.Second), 90*time.Second))
	require.Equal(t, "stale", ConcurrencySampleQuality("healthy", &last, last.Add(2*time.Minute), 90*time.Second))
}

func TestConcurrencyAccumulatorRecordsAccountWaitWithoutChangingConcurrencySamples(t *testing.T) {
	minute := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	accumulator := NewConcurrencyAccumulator()
	accumulator.Add(minute.Add(10*time.Second), []ConcurrencySample{{
		Kind: ConcurrencySampleKindAccount, AccountID: 7,
		Current: 4, Limit: 10, Waiting: 2, Available: true,
	}})
	_, _ = accumulator.AddWait(minute.Add(20*time.Second), 7, 450*time.Millisecond)
	_, _ = accumulator.AddWait(minute.Add(30*time.Second), 7, 150*time.Millisecond)
	flushed := accumulator.Flush()

	require.Len(t, flushed, 1)
	require.Equal(t, int64(1), flushed[0].SampleCount)
	require.Equal(t, 4, flushed[0].CurrentPeak)
	require.Equal(t, int64(600), flushed[0].WaitSumMS)
	require.Equal(t, int64(2), flushed[0].WaitSampleCount)
}
