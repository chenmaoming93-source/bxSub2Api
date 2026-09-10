package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type securityCheckLogRepoStub struct {
	records []SecurityCheckLogRecord
	err     error
}

func (r *securityCheckLogRepoStub) InsertSecurityCheckLogs(_ context.Context, records []SecurityCheckLogRecord) error {
	r.records = append(r.records, records...)
	return r.err
}

func TestSecurityCheckCollectorPrioritizesUnsafeRecords(t *testing.T) {
	repo := &securityCheckLogRepoStub{}
	collector := NewSecurityCheckCollector(repo)
	collector.Start()
	defer collector.Stop()
	config := domain.DefaultSecurityCheckConfig()
	config.CollectEnabled = true
	config.SampleRate = 0
	metadata := SecurityCheckLogMetadata{EventID: "unsafe-event"}
	unsafe := SecurityCheckResult{Status: domain.SecurityCheckStatusSuccess, Decision: domain.SecurityCheckDecisionWarn, IsUnsafe: true}
	if !collector.Enqueue(config, unsafe, metadata, []byte(`{"messages":[]}`)) {
		t.Fatal("unsafe record should bypass sampling")
	}
	safe := SecurityCheckResult{Status: domain.SecurityCheckStatusSuccess, Decision: domain.SecurityCheckDecisionAllow}
	if collector.Enqueue(config, safe, SecurityCheckLogMetadata{EventID: "safe-event"}, nil) {
		t.Fatal("safe record should be rejected at zero sample rate")
	}
}

func TestPrepareSecurityCheckRequestBodyCompressesAndMarksTruncation(t *testing.T) {
	body := bytes.Repeat([]byte("a"), securityCollectionMaxRequestBodyBytes+1)
	stored, original, storedBytes, truncated, err := PrepareSecurityCheckRequestBody(body)
	if err != nil {
		t.Fatalf("prepare body: %v", err)
	}
	if original != int64(len(body)) || !truncated || storedBytes != int64(len(stored)) {
		t.Fatalf("unexpected metadata original=%d stored=%d truncated=%v", original, storedBytes, truncated)
	}
	reader, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("decode compressed body: %v", err)
	}
	if len(decoded) != securityCollectionMaxRequestBodyBytes {
		t.Fatalf("decoded body size=%d", len(decoded))
	}
}

func TestSecurityCheckCollectorCircuitOpensAndReopens(t *testing.T) {
	collector := NewSecurityCheckCollector(&securityCheckLogRepoStub{err: errors.New("db down")})
	collector.recordFailure(errors.New("one"))
	collector.recordFailure(errors.New("two"))
	collector.recordFailure(errors.New("three"))
	if !collector.isCircuitOpen() {
		t.Fatal("circuit should be open after threshold failures")
	}
	collector.Reopen()
	if collector.isCircuitOpen() {
		t.Fatal("circuit should be closed after reopen")
	}
}

func TestStableSecuritySampleBoundsRates(t *testing.T) {
	if !stableSecuritySample("event", 100) {
		t.Fatal("100 percent sample must accept")
	}
	if stableSecuritySample("event", 0) {
		t.Fatal("zero percent sample must reject")
	}
	if stableSecuritySample("", 10) {
		t.Fatal("empty event ID must reject")
	}
}
