package tokenstat

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type staticProjectionSource []ProjectionDefinition

func (s staticProjectionSource) ActiveProjections() []ProjectionDefinition { return s }

type recordingWriter struct {
	mu         sync.Mutex
	operations []AccountingOperation
	err        error
}

func (w *recordingWriter) Add(_ context.Context, operations []AccountingOperation) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.operations = append(w.operations, operations...)
	return w.err
}

func TestAsyncPipelineRepresentativeRequestTypes(t *testing.T) {
	for _, requestType := range []string{"openai_non_stream", "openai_stream", "anthropic_stream", "gemini_non_stream"} {
		t.Run(requestType, func(t *testing.T) {
			writer := &recordingWriter{}
			pipeline, err := NewAsyncPipeline(4, 1, time.Second, 0, "Asia/Shanghai",
				staticProjectionSource{{ID: 1, Name: "user-model", DimensionCodes: []DimensionCode{DimensionUserID, DimensionUpstreamModel}, MetricCodes: []MetricCode{MetricTotalTokens}}},
				writer,
			)
			require.NoError(t, err)
			event := UsageEvent{
				OccurredAt: time.Now(), RequestType: requestType,
				Dimensions: map[DimensionCode]DimensionValue{
					DimensionUserID: Int64Value(7), DimensionUpstreamModel: StringValue("model"),
				},
				Metrics: map[MetricCode]int64{MetricTotalTokens: 12},
			}
			require.True(t, pipeline.TryEnqueue(event))
			pipeline.Close()
			require.Equal(t, uint64(1), pipeline.Stats().ProcessedEvents)
			require.Len(t, writer.operations, 3)
		})
	}
}

func TestAsyncPipelineQueueFullAndWriterFailureAreFailOpen(t *testing.T) {
	full, err := NewAsyncPipeline(1, 0, time.Second, 0, "Asia/Shanghai", staticProjectionSource{}, &recordingWriter{})
	require.NoError(t, err)
	require.True(t, full.TryEnqueue(UsageEvent{}))
	require.False(t, full.TryEnqueue(UsageEvent{}))
	require.Equal(t, uint64(1), full.Stats().DroppedQueueFull)

	writer := &recordingWriter{err: errors.New("redis unavailable")}
	pipeline, err := NewAsyncPipeline(2, 1, 10*time.Millisecond, 1, "Asia/Shanghai",
		staticProjectionSource{{ID: 1, Name: "user", DimensionCodes: []DimensionCode{DimensionUserID}, MetricCodes: []MetricCode{MetricTotalTokens}}},
		writer,
	)
	require.NoError(t, err)
	require.True(t, pipeline.TryEnqueue(UsageEvent{
		OccurredAt: time.Now(),
		Dimensions: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(1)},
		Metrics:    map[MetricCode]int64{MetricTotalTokens: 1},
	}))
	pipeline.Close()
	require.Equal(t, uint64(1), pipeline.Stats().WriteFailures)
}

func TestAsyncPipelineMissingDimensionSkipsOnlyIncompatibleProjection(t *testing.T) {
	writer := &recordingWriter{}
	pipeline, err := NewAsyncPipeline(2, 1, time.Second, 0, "Asia/Shanghai",
		staticProjectionSource{
			{ID: 1, Name: "user-account", DimensionCodes: []DimensionCode{DimensionUserID, DimensionAccountID}, MetricCodes: []MetricCode{MetricTotalTokens}},
			{ID: 2, Name: "user", DimensionCodes: []DimensionCode{DimensionUserID}, MetricCodes: []MetricCode{MetricTotalTokens}},
		},
		writer,
	)
	require.NoError(t, err)
	require.True(t, pipeline.TryEnqueue(UsageEvent{
		OccurredAt: time.Now(), Dimensions: map[DimensionCode]DimensionValue{DimensionUserID: Int64Value(1)},
		Metrics: map[MetricCode]int64{MetricTotalTokens: 1},
	}))
	pipeline.Close()
	require.Equal(t, uint64(0), pipeline.Stats().InvalidEvents)
	require.Len(t, writer.operations, 3)
	require.Equal(t, int64(2), writer.operations[0].ProjectionID)
}
