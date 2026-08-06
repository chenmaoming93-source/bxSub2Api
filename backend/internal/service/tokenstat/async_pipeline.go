package tokenstat

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var defaultPipeline atomic.Pointer[AsyncPipeline]

func SetDefaultPipeline(pipeline *AsyncPipeline) {
	defaultPipeline.Store(pipeline)
}

func TryEnqueueDefault(event UsageEvent) bool {
	if !RuntimeEnabled() {
		return false
	}
	pipeline := defaultPipeline.Load()
	return pipeline != nil && pipeline.TryEnqueue(event)
}

type AccountingOperation struct {
	Period          Period
	ProjectionID    int64
	DimensionHash   [16]byte
	DimensionValues map[string]any
	MetricCode      MetricCode
	Delta           int64
}

type AccountingWriter interface {
	Add(ctx context.Context, operations []AccountingOperation) error
}

type ActiveProjectionSource interface {
	ActiveProjections() []ProjectionDefinition
}

type AsyncPipelineStats struct {
	Enqueued         uint64
	DroppedQueueFull uint64
	InvalidEvents    uint64
	WriteFailures    uint64
	ProcessedEvents  uint64
}

type AsyncPipeline struct {
	queue     chan UsageEvent
	writer    AccountingWriter
	source    ActiveProjectionSource
	location  *time.Location
	timeout   time.Duration
	retries   int
	workers   sync.WaitGroup
	stop      chan struct{}
	stopOnce  sync.Once
	enqueued  atomic.Uint64
	dropped   atomic.Uint64
	invalid   atomic.Uint64
	failures  atomic.Uint64
	processed atomic.Uint64
}

func NewAsyncPipeline(capacity, workers int, timeout time.Duration, retries int, timezone string, source ActiveProjectionSource, writer AccountingWriter) (*AsyncPipeline, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	pipeline := &AsyncPipeline{
		queue: make(chan UsageEvent, capacity), writer: writer, source: source,
		location: location, timeout: timeout, retries: retries, stop: make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		pipeline.workers.Add(1)
		go pipeline.run()
	}
	return pipeline, nil
}

// TryEnqueue never waits for Redis, MySQL, or queue capacity.
func (p *AsyncPipeline) TryEnqueue(event UsageEvent) bool {
	if p == nil {
		return false
	}
	select {
	case <-p.stop:
		return false
	default:
	}
	select {
	case p.queue <- event:
		p.enqueued.Add(1)
		observability.enqueued.Add(1)
		return true
	default:
		p.dropped.Add(1)
		observability.dropped.Add(1)
		return false
	}
}

func (p *AsyncPipeline) Close() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() { close(p.stop) })
	p.workers.Wait()
}

func (p *AsyncPipeline) Stats() AsyncPipelineStats {
	return AsyncPipelineStats{
		Enqueued: p.enqueued.Load(), DroppedQueueFull: p.dropped.Load(),
		InvalidEvents: p.invalid.Load(), WriteFailures: p.failures.Load(),
		ProcessedEvents: p.processed.Load(),
	}
}

// HasPendingBefore is conservative: queued events may belong to the closing
// period, so any queued item blocks deletion until workers drain the queue.
func (p *AsyncPipeline) HasPendingBefore(_ time.Time) bool {
	return p != nil && len(p.queue) > 0
}

func (p *AsyncPipeline) run() {
	defer p.workers.Done()
	for {
		select {
		case event := <-p.queue:
			p.process(event)
		case <-p.stop:
			for {
				select {
				case event := <-p.queue:
					p.process(event)
				default:
					return
				}
			}
		}
	}
}

func (p *AsyncPipeline) process(event UsageEvent) {
	if err := event.Validate(); err != nil {
		p.invalid.Add(1)
		observability.invalid.Add(1)
		return
	}
	operations, err := p.operations(event)
	if err != nil {
		p.invalid.Add(1)
		observability.invalid.Add(1)
		return
	}
	if len(operations) == 0 {
		p.processed.Add(1)
		return
	}
	for attempt := 0; attempt <= p.retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		err = p.writer.Add(ctx, operations)
		cancel()
		if err == nil {
			p.processed.Add(1)
			return
		}
	}
	p.failures.Add(1)
	observability.redisWriteFailure.Add(1)
}

func (p *AsyncPipeline) operations(event UsageEvent) ([]AccountingOperation, error) {
	periods := NaturalPeriods(event.OccurredAt, p.location)
	var operations []AccountingOperation
	for _, projection := range p.source.ActiveProjections() {
		values := make(map[DimensionCode]DimensionValue, len(projection.DimensionCodes))
		raw := make(map[string]any, len(projection.DimensionCodes))
		complete := true
		for _, code := range projection.DimensionCodes {
			value, ok := event.Dimensions[code]
			if !ok {
				complete = false
				break
			}
			values[code] = value
			if value.Type == ValueTypeInt64 {
				raw[string(code)] = value.Int64
			} else {
				raw[string(code)] = value.String
			}
		}
		// A protocol may legitimately omit dimensions that are irrelevant to it.
		// Skip only the incompatible projection so other projections (for example,
		// user-only accounting) continue to receive the event.
		if !complete {
			continue
		}
		identity, err := BuildDimensionIdentity(projection.DimensionCodes, values)
		if err != nil {
			return nil, err
		}
		for _, metric := range projection.MetricCodes {
			delta, ok := event.Metrics[metric]
			if !ok {
				return nil, fmt.Errorf("missing metric %s", metric)
			}
			for _, period := range periods {
				operations = append(operations, AccountingOperation{
					Period: period, ProjectionID: int64(projection.ID), DimensionHash: identity.Hash,
					DimensionValues: raw, MetricCode: metric, Delta: delta,
				})
			}
		}
	}
	return operations, nil
}
