package logger

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type requestFailureCaptureSink struct {
	events []*LogEvent
}

func (s *requestFailureCaptureSink) WriteLogEvent(event *LogEvent) {
	s.events = append(s.events, event)
}

func TestWriteRequestFailureUsesDedicatedFileWithoutSink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "request-failure-logger-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	sink := &requestFailureCaptureSink{}
	SetSink(sink)
	defer SetSink(nil)
	defer resetRequestFailureLoggerForTest()

	logPath := filepath.Join(tmpDir, "info.log")
	if err := Init(InitOptions{
		Level:       "info",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output: OutputOptions{
			ToStdout: false,
			ToFile:   true,
			FilePath: logPath,
		},
		Rotation: RotationOptions{MaxSizeMB: 10, MaxBackups: 1, MaxAgeDays: 1},
	}); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	WriteRequestFailure(RequestFailureEvent{
		RequestID:       "req-1",
		Endpoint:        "/v1/messages",
		Method:          "POST",
		Status:          400,
		UserID:          7,
		GroupID:         9,
		ContentType:     "application/json",
		RequestBody:     `{"api_key":"***REDACTED***"}`,
		RequestBodySize: 27,
		BodyFormat:      "json",
	})

	requestFailureMu.RLock()
	current := requestFailureLogger
	requestFailureMu.RUnlock()
	if current == nil {
		t.Fatal("request failure logger is nil")
	}
	_ = current.Sync()
	if len(sink.events) != 0 {
		t.Fatalf("request failure leaked to global sink: %d events", len(sink.events))
	}

	file, err := os.Open(filepath.Join(tmpDir, RequestFailureLogFilename))
	if err != nil {
		t.Fatalf("open request failure log: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("request failure log is empty")
	}
	var entry map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
		t.Fatalf("decode request failure log: %v", err)
	}
	if entry["msg"] != "gateway request failed" || entry["request_id"] != "req-1" {
		t.Fatalf("unexpected request failure log: %#v", entry)
	}
	if entry["request_body"] != `{"api_key":"***REDACTED***"}` {
		t.Fatalf("request body not preserved as a JSON string: %#v", entry["request_body"])
	}
}
