package tokenstat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDynamicTokenStatisticsObservabilitySnapshot(t *testing.T) {
	before := MetricsSnapshot()
	RecordSync(3, false)
	RecordSync(0, true)
	RecordFinalization(false)
	RecordFinalization(true)
	after := MetricsSnapshot()
	require.Equal(t, before.SyncedRows+3, after.SyncedRows)
	require.Equal(t, before.SyncFailures+1, after.SyncFailures)
	require.Equal(t, before.FinalizedPeriods+1, after.FinalizedPeriods)
	require.Equal(t, before.FinalizeFailures+1, after.FinalizeFailures)
}

func BenchmarkDynamicTokenStatisticsQueueFullFailOpen(b *testing.B) {
	pipeline, err := NewAsyncPipeline(1, 0, time.Millisecond, 0, "Asia/Shanghai", nil, nil)
	if err != nil {
		b.Fatal(err)
	}
	pipeline.TryEnqueue(UsageEvent{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.TryEnqueue(UsageEvent{})
	}
}
