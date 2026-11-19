package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const RequestFailureLogFilename = "request-failures.log"

// RequestFailureEvent is intentionally typed so callers cannot accidentally
// route request bodies through the global logger and its database sink.
type RequestFailureEvent struct {
	RequestID       string
	Endpoint        string
	Method          string
	Status          int
	UserID          int64
	GroupID         int64
	ContentType     string
	RequestBody     string
	RequestBodySize int64
	Truncated       bool
	BodyFormat      string
	BodyOmitted     string
}

var (
	requestFailureMu     sync.RWMutex
	requestFailureLogger *zap.Logger
	requestFailureFile   io.Closer
)

func requestFailureLogPath(mainLogPath string) string {
	return filepath.Join(filepath.Dir(resolveLogFilePath(mainLogPath)), RequestFailureLogFilename)
}

func configureRequestFailureLogger(options InitOptions) error {
	path := requestFailureLogPath(options.Output.FilePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	writer := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    options.Rotation.MaxSizeMB,
		MaxBackups: options.Rotation.MaxBackups,
		MaxAge:     options.Rotation.MaxAgeDays,
		Compress:   options.Rotation.Compress,
		LocalTime:  options.Rotation.LocalTime,
	}
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	})
	next := zap.New(
		zapcore.NewCore(
			encoder,
			zapcore.AddSync(writer),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level >= zapcore.InfoLevel
			}),
		),
		zap.AddCaller(),
	).With(
		zap.String("service", options.ServiceName),
		zap.String("env", options.Environment),
		zap.String("component", "gateway.request_failure"),
	)

	requestFailureMu.Lock()
	previous := requestFailureLogger
	previousFile := requestFailureFile
	requestFailureLogger = next
	requestFailureFile = writer
	requestFailureMu.Unlock()

	if previous != nil {
		_ = previous.Sync()
	}
	if previousFile != nil {
		_ = previousFile.Close()
	}
	return nil
}

// WriteRequestFailure writes only to the dedicated rotating file. It bypasses
// stdout and the global sink, so request bodies can never reach ops_system_logs.
func WriteRequestFailure(event RequestFailureEvent) {
	requestFailureMu.RLock()
	fileLogger := requestFailureLogger
	requestFailureMu.RUnlock()
	if fileLogger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("event", "gateway.request_failure"),
		zap.String("request_id", strings.TrimSpace(event.RequestID)),
		zap.String("endpoint", strings.TrimSpace(event.Endpoint)),
		zap.String("method", strings.TrimSpace(event.Method)),
		zap.Int("status", event.Status),
		zap.Int64("user_id", event.UserID),
		zap.Int64("group_id", event.GroupID),
		zap.String("content_type", strings.TrimSpace(event.ContentType)),
		zap.Int64("request_body_bytes", event.RequestBodySize),
		zap.Bool("request_body_truncated", event.Truncated),
		zap.String("request_body_format", strings.TrimSpace(event.BodyFormat)),
		zap.String("request_body_omitted_reason", strings.TrimSpace(event.BodyOmitted)),
	}
	if event.RequestBody != "" {
		fields = append(fields, zap.String("request_body", event.RequestBody))
	}
	fileLogger.Info("gateway request failed", fields...)
}

func resetRequestFailureLoggerForTest() {
	requestFailureMu.Lock()
	previous := requestFailureLogger
	previousFile := requestFailureFile
	requestFailureLogger = nil
	requestFailureFile = nil
	requestFailureMu.Unlock()

	if previous != nil {
		_ = previous.Sync()
	}
	if previousFile != nil {
		_ = previousFile.Close()
	}
}
