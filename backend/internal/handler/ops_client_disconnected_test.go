package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type clientDisconnectedOpsStub struct {
	calls  int
	entry  *service.OpsInsertErrorLogInput
	ctxErr error
}

func (s *clientDisconnectedOpsStub) RecordError(ctx context.Context, entry *service.OpsInsertErrorLogInput) error {
	s.calls++
	s.entry = entry
	s.ctxErr = ctx.Err()
	return nil
}

func newTestOpsContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	return c
}

func TestRecordClientDisconnectedOpsError_IsAtMostOncePerRequest(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	t.Cleanup(func() { resetOpsErrorLoggerStateForTest(t) })
	opsErrorLogOnce.Do(func() {})
	stub := &clientDisconnectedOpsStub{}
	c := newTestOpsContext(t)

	require.True(t, recordClientDisconnectedOpsError(c, stub, &service.OpsInsertErrorLogInput{}))
	require.False(t, recordClientDisconnectedOpsError(c, stub, &service.OpsInsertErrorLogInput{}))
	require.Equal(t, 1, stub.calls)
	require.Equal(t, service.OpsErrorTypeClientDisconnected, stub.entry.ErrorType)
	require.Equal(t, 499, stub.entry.StatusCode)
}

func TestEnqueueCriticalOpsErrorLog_FallsBackSynchronouslyWhenQueueFull(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	t.Cleanup(func() { resetOpsErrorLoggerStateForTest(t) })

	// Prevent worker startup and use a one-slot queue to force the second event
	// through the synchronous fallback path.
	opsErrorLogOnce.Do(func() {})
	opsErrorLogMu.Lock()
	opsErrorLogQueue = make(chan opsErrorLogJob, 1)
	opsErrorLogMu.Unlock()

	stub := &clientDisconnectedOpsStub{}
	enqueueCriticalOpsErrorLog(stub, &service.OpsInsertErrorLogInput{ErrorType: service.OpsErrorTypeClientDisconnected})
	enqueueCriticalOpsErrorLog(stub, &service.OpsInsertErrorLogInput{ErrorType: service.OpsErrorTypeClientDisconnected})

	require.Equal(t, 1, stub.calls)
	require.NoError(t, stub.ctxErr)
	require.Equal(t, int64(1), OpsErrorLogDroppedTotal())
	require.Equal(t, int64(1), OpsErrorLogFallbackTotal())
}

func newTestRequestContext(t *testing.T, canceled bool) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	if canceled {
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
	}
	c.Request = req
	return c
}

// A canceled request context alone must never mark a request as a caller disconnect:
// an upstream 429/5xx would otherwise be reported as a disconnect and lose its record.
func TestIsOpenAIClientDisconnected_RequiresAnExplicitDisconnectSignal(t *testing.T) {
	cases := []struct {
		name        string
		ctxCanceled bool
		result      *service.OpenAIForwardResult
		err         error
		want        bool
	}{
		{
			name:   "service marked the disconnect",
			result: &service.OpenAIForwardResult{ClientDisconnect: true},
			want:   true,
		},
		{
			name:        "canceled context without error",
			ctxCanceled: true,
		},
		{
			name:        "upstream error while the context is canceled",
			ctxCanceled: true,
			err:         errors.New("upstream returned 429"),
		},
		{
			name:        "successful forward while the context is canceled",
			ctxCanceled: true,
			result:      &service.OpenAIForwardResult{Model: "gpt-5"},
		},
		{
			name:        "wrapped context cancellation",
			ctxCanceled: true,
			err:         fmt.Errorf("stream read error: %w", context.Canceled),
			want:        true,
		},
		{
			name: "context cancellation while the request context is alive",
			err:  context.Canceled,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isOpenAIClientDisconnected(newTestRequestContext(t, tc.ctxCanceled), tc.result, tc.err))
		})
	}
}

func TestOpsRequestCarriesRealUpstreamError(t *testing.T) {
	cases := []struct {
		name  string
		setup func(c *gin.Context)
		want  bool
	}{
		{
			name:  "clean successful request",
			setup: func(c *gin.Context) { c.Status(http.StatusOK) },
		},
		{
			name:  "upstream failure status",
			setup: func(c *gin.Context) { c.Status(http.StatusTooManyRequests) },
			want:  true,
		},
		{
			name: "recorded upstream error events",
			setup: func(c *gin.Context) {
				c.Status(http.StatusOK)
				c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{
					Platform:           "openai",
					AccountID:          7,
					UpstreamStatusCode: 429,
					Kind:               "failover",
					Message:            "rate limited",
				}})
			},
			want: true,
		},
		{
			name: "single upstream status field",
			setup: func(c *gin.Context) {
				c.Status(http.StatusOK)
				service.SetOpsUpstreamError(c, http.StatusServiceUnavailable, "overloaded", "")
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestRequestContext(t, false)
			tc.setup(c)
			require.Equal(t, tc.want, opsRequestCarriesRealUpstreamError(c))
		})
	}
}

func installOpsErrorLogTestQueue(t *testing.T, capacity int) chan opsErrorLogJob {
	t.Helper()
	resetOpsErrorLoggerStateForTest(t)
	t.Cleanup(func() { resetOpsErrorLoggerStateForTest(t) })
	// Keep the worker pool from starting so the test queue stays the only sink.
	opsErrorLogOnce.Do(func() {})
	queue := make(chan opsErrorLogJob, capacity)
	opsErrorLogMu.Lock()
	opsErrorLogQueue = queue
	opsErrorLogMu.Unlock()
	return queue
}

func drainOpsErrorLogQueue(queue chan opsErrorLogJob) []*service.OpsInsertErrorLogInput {
	var entries []*service.OpsInsertErrorLogInput
	for {
		select {
		case job := <-queue:
			entries = append(entries, job.entry)
		default:
			return entries
		}
	}
}

func serveOpsErrorLoggerRequest(t *testing.T, ops *service.OpsService, handler gin.HandlerFunc) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OpsErrorLoggerMiddleware(ops))
	r.POST("/v1/responses", handler)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
}

// A caller disconnect that coincides with a genuine upstream failure must not hide that
// failure: the regular path records the upstream error instead of a disconnect row.
func TestOpsErrorLoggerMiddleware_RealUpstreamErrorBeatsDisconnectRecord(t *testing.T) {
	queue := installOpsErrorLogTestQueue(t, 4)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	serveOpsErrorLoggerRequest(t, ops, func(c *gin.Context) {
		queueClientDisconnectedOpsError(c, buildGatewayClientDisconnectedOpsEntry(c, &service.Account{ID: 7}, "gpt-5", "gpt-5", true))
		c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{
			Platform:           "openai",
			AccountID:          7,
			UpstreamStatusCode: 429,
			Kind:               "failover",
			Message:            "rate limited",
		}})
		c.Status(http.StatusOK)
	})

	entries := drainOpsErrorLogQueue(queue)
	require.Len(t, entries, 1)
	require.NotEqual(t, service.OpsErrorTypeClientDisconnected, entries[0].ErrorType)
	require.NotNil(t, entries[0].UpstreamStatusCode)
	require.Equal(t, 429, *entries[0].UpstreamStatusCode)
}

func TestOpsErrorLoggerMiddleware_RecordsDisconnectWithoutUpstreamError(t *testing.T) {
	queue := installOpsErrorLogTestQueue(t, 4)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	serveOpsErrorLoggerRequest(t, ops, func(c *gin.Context) {
		queueClientDisconnectedOpsError(c, buildGatewayClientDisconnectedOpsEntry(c, &service.Account{ID: 7}, "gpt-5", "gpt-5", true))
		c.Status(http.StatusOK)
	})

	entries := drainOpsErrorLogQueue(queue)
	require.Len(t, entries, 1)
	require.Equal(t, service.OpsErrorTypeClientDisconnected, entries[0].ErrorType)
	require.Equal(t, 499, entries[0].StatusCode)
	require.Equal(t, "client", entries[0].ErrorOwner)
}
