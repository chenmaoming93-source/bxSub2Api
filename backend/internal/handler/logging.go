package handler

import (
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func requestLogger(c *gin.Context, component string, fields ...zap.Field) *zap.Logger {
	base := logger.L()
	if c != nil && c.Request != nil {
		base = logger.FromContext(c.Request.Context())
	}

	if component != "" {
		fields = append([]zap.Field{zap.String("component", component)}, fields...)
	}
	return base.With(fields...)
}

func modelForwardAccountFields(account *service.Account) []zap.Field {
	if account == nil {
		return nil
	}
	fields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("account_platform", account.Platform),
		zap.String("account_type", account.Type),
	}
	if account.Proxy != nil {
		fields = append(fields,
			zap.Int64("proxy_id", account.Proxy.ID),
			zap.String("proxy_name", account.Proxy.Name),
			zap.String("proxy_host", account.Proxy.Host),
			zap.Int("proxy_port", account.Proxy.Port),
		)
	} else if account.ProxyID != nil {
		fields = append(fields, zap.Int64p("proxy_id", account.ProxyID))
	}
	return fields
}

func logModelForwardStarted(reqLog *zap.Logger, event string, account *service.Account, requestedModel, forwardModel string, stream bool, switchCount int, bodyBytes int, extra ...zap.Field) {
	if reqLog == nil {
		return
	}
	fields := modelForwardAccountFields(account)
	fields = append(fields,
		zap.String("requested_model", requestedModel),
		zap.String("forward_model", forwardModel),
		zap.Bool("stream", stream),
		zap.Int("switch_count", switchCount),
		zap.Int("body_bytes", bodyBytes),
	)
	fields = append(fields, extra...)
	reqLog.Info(event, fields...)
}

func logModelForwardFinished(reqLog *zap.Logger, event string, c *gin.Context, startedAt time.Time, account *service.Account, requestedModel, forwardModel string, stream bool, switchCount int, writerSizeBefore int, forwardErr error, extra ...zap.Field) {
	if reqLog == nil {
		return
	}
	durationMs := time.Since(startedAt).Milliseconds()
	if durationMs < 0 {
		durationMs = 0
	}
	statusCode := 0
	bytesWritten := 0
	if c != nil && c.Writer != nil {
		statusCode = c.Writer.Status()
		bytesWritten = c.Writer.Size()
	}
	fields := modelForwardAccountFields(account)
	fields = append(fields,
		zap.String("requested_model", requestedModel),
		zap.String("forward_model", forwardModel),
		zap.Bool("stream", stream),
		zap.Int("switch_count", switchCount),
		zap.Int64("duration_ms", durationMs),
		zap.Bool("success", forwardErr == nil),
		zap.Int("status_code", statusCode),
		zap.Int("bytes_written", bytesWritten),
		zap.Int("bytes_written_delta", bytesWritten-writerSizeBefore),
	)
	var failoverErr *service.UpstreamFailoverError
	if errors.As(forwardErr, &failoverErr) {
		fields = append(fields, zap.Int("upstream_status", failoverErr.StatusCode))
	}
	if forwardErr != nil {
		fields = append(fields, zap.Error(forwardErr))
	}
	fields = append(fields, extra...)
	if forwardErr != nil {
		reqLog.Warn(event, fields...)
		return
	}
	reqLog.Info(event, fields...)
}
