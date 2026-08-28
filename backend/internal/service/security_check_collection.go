package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

const (
	securityCollectionHighQueueSize       = 256
	securityCollectionNormalQueueSize     = 1024
	securityCollectionBatchSize           = 32
	securityCollectionFlushInterval       = 100 * time.Millisecond
	securityCollectionFailureThreshold    = 3
	securityCollectionCooldown            = 60 * time.Second
	securityCollectionMaxRequestBodyBytes = 16 * 1024 * 1024
)

// SecurityCheckLogRepository persists collected security events in batches.
type SecurityCheckLogRepository interface {
	InsertSecurityCheckLogs(ctx context.Context, records []SecurityCheckLogRecord) error
}

// SecurityCheckLogMetadata contains request metadata captured before upstream selection.
type SecurityCheckLogMetadata struct {
	EventID         string
	RequestID       string
	ClientRequestID string
	UserID          int64
	APIKeyID        int64
	APIKeyName      string
	GroupID         int64
	GroupName       string
	Model           string
	Provider        string
	Protocol        string
	Endpoint        string
}

// SecurityCheckLogRecord is the in-process representation of one collected event.
type SecurityCheckLogRecord struct {
	Metadata          SecurityCheckLogMetadata
	ConfigVersion     int64
	RulesSnapshot     []domain.SecurityCheckRule
	RequestBody       []byte
	SingGuardResponse []byte
	CheckStatus       domain.SecurityCheckStatus
	Decision          domain.SecurityCheckDecision
	IsUnsafe          bool
	TriggeredRules    []SecurityCheckTriggeredRule
	Latency           time.Duration
	SingGuardLatency  time.Duration
	QueueDelay        time.Duration
	ExceptionType     string
	ExceptionMessage  string
	EnqueuedAt        time.Time
}

// SecurityCheckCollector asynchronously persists records without blocking model calls.
type SecurityCheckCollector struct {
	repo   SecurityCheckLogRepository
	high   chan SecurityCheckLogRecord
	normal chan SecurityCheckLogRecord
	stop   chan struct{}
	done   chan struct{}
	onDrop func(SecurityCheckLogRecord)

	mu              sync.Mutex
	failureCount    int
	circuitOpenedAt time.Time
}

func NewSecurityCheckCollector(repo SecurityCheckLogRepository) *SecurityCheckCollector {
	return &SecurityCheckCollector{
		repo:   repo,
		high:   make(chan SecurityCheckLogRecord, securityCollectionHighQueueSize),
		normal: make(chan SecurityCheckLogRecord, securityCollectionNormalQueueSize),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start launches the bounded batch writer. A nil repository still drains and drops safely.
func (c *SecurityCheckCollector) Start() {
	if c == nil {
		return
	}
	go c.run()
}

func (c *SecurityCheckCollector) Stop() {
	if c == nil {
		return
	}
	select {
	case <-c.stop:
		return
	default:
		close(c.stop)
	}
	<-c.done
}

// Enqueue applies collection enablement, sampling, priority, and circuit protection.
func (c *SecurityCheckCollector) Enqueue(config domain.SecurityCheckConfig, result SecurityCheckResult, metadata SecurityCheckLogMetadata, body []byte) bool {
	if c == nil || !config.CollectEnabled || c.isCircuitOpen() {
		return false
	}
	if metadata.EventID == "" {
		metadata.EventID = newSecurityEventID()
	}
	highPriority := result.IsUnsafe || result.Decision == domain.SecurityCheckDecisionBlock
	if !highPriority && !stableSecuritySample(metadata.EventID, config.SampleRate) {
		return false
	}
	record := SecurityCheckLogRecord{
		Metadata:         metadata,
		ConfigVersion:    config.Version,
		RulesSnapshot:    append([]domain.SecurityCheckRule(nil), config.Rules...),
		RequestBody:      append([]byte(nil), body...),
		CheckStatus:      result.Status,
		Decision:         result.Decision,
		IsUnsafe:         result.IsUnsafe,
		TriggeredRules:   append([]SecurityCheckTriggeredRule(nil), result.TriggeredRules...),
		Latency:          result.Latency,
		SingGuardLatency: result.SingGuardLatency,
		ExceptionMessage: errorMessage(result.Err),
		EnqueuedAt:       time.Now(),
	}
	record.SingGuardResponse = append([]byte(nil), result.RawResponse...)
	queue := c.normal
	if highPriority {
		queue = c.high
	}
	select {
	case queue <- record:
		return true
	default:
		if c.onDrop != nil {
			c.onDrop(record)
		}
		return false
	}
}

func (c *SecurityCheckCollector) run() {
	defer close(c.done)
	ticker := time.NewTicker(securityCollectionFlushInterval)
	defer ticker.Stop()
	batch := make([]SecurityCheckLogRecord, 0, securityCollectionBatchSize)
	lastCleanup := time.Now().Add(-time.Hour)
	cleanup := func() {
		store, ok := c.repo.(SecurityCheckLogStore)
		if !ok || time.Since(lastCleanup) < time.Hour {
			return
		}
		before := time.Now().AddDate(0, 0, -3)
		for {
			deleted, err := store.DeleteSecurityCheckLogsBefore(context.Background(), before, 1000)
			if err != nil {
				c.recordFailure(err)
				break
			}
			if deleted < 1000 {
				break
			}
		}
		lastCleanup = time.Now()
	}
	flush := func() {
		if len(batch) == 0 || c.repo == nil || c.isCircuitOpen() {
			batch = batch[:0]
			return
		}
		for i := range batch {
			batch[i].QueueDelay = time.Since(batch[i].EnqueuedAt)
		}
		if err := c.repo.InsertSecurityCheckLogs(context.Background(), batch); err != nil {
			c.recordFailure(err)
		} else {
			c.recordSuccess()
		}
		batch = batch[:0]
	}
	for {
		select {
		case <-c.stop:
			flush()
			return
		case record := <-c.high:
			batch = append(batch, record)
			if len(batch) >= securityCollectionBatchSize {
				flush()
			}
		default:
			select {
			case <-c.stop:
				flush()
				return
			case record := <-c.high:
				batch = append(batch, record)
			case record := <-c.normal:
				batch = append(batch, record)
			case <-ticker.C:
				flush()
				cleanup()
			}
		}
	}
}

func (c *SecurityCheckCollector) isCircuitOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.circuitOpenedAt.IsZero() {
		return false
	}
	if time.Since(c.circuitOpenedAt) >= securityCollectionCooldown {
		return false
	}
	return true
}

func (c *SecurityCheckCollector) recordFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCount++
	if c.failureCount >= securityCollectionFailureThreshold && c.circuitOpenedAt.IsZero() {
		c.circuitOpenedAt = time.Now()
		slog.Warn("security_check.collection_circuit_open", "err", err, "failures", c.failureCount)
	}
}

func (c *SecurityCheckCollector) recordSuccess() {
	c.mu.Lock()
	c.failureCount = 0
	c.circuitOpenedAt = time.Time{}
	c.mu.Unlock()
}

// Reopen clears the local circuit after an operator or health check decides collection is safe.
func (c *SecurityCheckCollector) Reopen() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.failureCount = 0
	c.circuitOpenedAt = time.Time{}
	c.mu.Unlock()
}

func stableSecuritySample(eventID string, sampleRate int) bool {
	if sampleRate >= 100 {
		return true
	}
	if sampleRate <= 0 || eventID == "" {
		return false
	}
	var hash uint32
	for i := 0; i < len(eventID); i++ {
		hash = hash*16777619 ^ uint32(eventID[i])
	}
	return int(hash%100) < sampleRate
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func newSecurityEventID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(raw[:])
}

// PrepareSecurityCheckRequestBody compresses and bounds the stored logical request body.
// SecurityCheckLogFilter controls administrator log queries.
type SecurityCheckLogFilter struct {
	Page     int
	PageSize int
	From     time.Time
	To       time.Time
	GroupID  int64
	Decision string
	Status   string
}

type SecurityCheckLogSummary struct {
	ID                       int64                               `json:"id"`
	EventID                  string                              `json:"event_id"`
	RequestID                *string                             `json:"request_id,omitempty"`
	GroupID                  *int64                              `json:"group_id,omitempty"`
	GroupName                *string                             `json:"group_name,omitempty"`
	Model                    *string                             `json:"model,omitempty"`
	Protocol                 *string                             `json:"protocol,omitempty"`
	CheckStatus              string                              `json:"check_status"`
	Decision                 string                              `json:"decision"`
	IsUnsafe                 bool                                `json:"is_unsafe"`
	TriggeredRules           []domain.SecurityCheckTriggeredRule `json:"triggered_rules"`
	LatencyMs                *int                                `json:"latency_ms,omitempty"`
	CreatedAt                time.Time                           `json:"created_at"`
	RequestBodyOriginalBytes int64                               `json:"request_body_original_bytes"`
	RequestBodyStoredBytes   int64                               `json:"request_body_stored_bytes"`
	RequestBodyTruncated     bool                                `json:"request_body_truncated"`
}

type SecurityCheckLogDetail struct {
	SecurityCheckLogSummary
	ConfigVersion      int64                      `json:"config_version"`
	RulesSnapshot      []domain.SecurityCheckRule `json:"rules_snapshot"`
	RequestBody        string                     `json:"request_body,omitempty"`
	SingGuardResponse  string                     `json:"singguard_response,omitempty"`
	ExceptionType      *string                    `json:"exception_type,omitempty"`
	ExceptionMessage   *string                    `json:"exception_message,omitempty"`
	SingGuardLatencyMs *int                       `json:"singguard_latency_ms,omitempty"`
	QueueDelayMs       *int                       `json:"queue_delay_ms,omitempty"`
}

type SecurityCheckLogPage struct {
	Items    []SecurityCheckLogSummary `json:"items"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// SecurityCheckLogStore provides administrator query and lifecycle operations.
type SecurityCheckLogStore interface {
	ListSecurityCheckLogs(context.Context, SecurityCheckLogFilter) (SecurityCheckLogPage, error)
	GetSecurityCheckLog(context.Context, int64) (SecurityCheckLogDetail, error)
	DeleteSecurityCheckLogsBefore(context.Context, time.Time, int) (int, error)
}

func (c *SecurityCheckCollector) CircuitOpen() bool { return c != nil && c.isCircuitOpen() }

func (c *SecurityCheckCollector) CircuitFailureCount() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failureCount
}

func (c *SecurityCheckCollector) SecurityCheckCollectionStatus() map[string]any {
	return map[string]any{"circuit_open": c.CircuitOpen(), "failure_count": c.CircuitFailureCount()}
}

func PrepareSecurityCheckRequestBody(body []byte) (stored []byte, originalBytes, storedBytes int64, truncated bool, err error) {
	originalBytes = int64(len(body))
	if len(body) > securityCollectionMaxRequestBodyBytes {
		body = body[:securityCollectionMaxRequestBodyBytes]
		truncated = true
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(body); err != nil {
		return nil, originalBytes, 0, truncated, err
	}
	if err := writer.Close(); err != nil {
		return nil, originalBytes, 0, truncated, err
	}
	stored = buffer.Bytes()
	if len(stored) > securityCollectionMaxRequestBodyBytes {
		stored = body
		truncated = true
	}
	return stored, originalBytes, int64(len(stored)), truncated, nil
}
