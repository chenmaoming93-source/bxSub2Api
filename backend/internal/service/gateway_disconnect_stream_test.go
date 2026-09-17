package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const disconnectAnthropicContentBlockSSE = "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"hello\"}}\n\n"

func newDisconnectTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Writer = &failingGinWriter{ResponseWriter: c.Writer, failAfter: 0}
	return c, rec
}

func newDisconnectTestResponse() *http.Response {
	return &http.Response{
		Header: http.Header{"x-request-id": []string{"disconnect-test"}},
		Body:   io.NopCloser(strings.NewReader(disconnectAnthropicContentBlockSSE)),
	}
}

func TestHandleCCStreamingFromAnthropic_CancelsUpstreamOnClientWriteFailure(t *testing.T) {
	parent := context.Background()
	control := newStreamUpstreamContextControl(parent, true)
	t.Cleanup(control.Release)
	c, _ := newDisconnectTestContext()

	svc := &GatewayService{}
	result, err := svc.handleCCStreamingFromAnthropic(control.ctx, newDisconnectTestResponse(), c, "gpt-5", "claude-sonnet", nil, time.Now(), true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.ClientDisconnect)
	select {
	case <-control.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("upstream context was not canceled after client write failure")
	}
}

func TestHandleResponsesStreamingResponse_CancelsUpstreamOnClientWriteFailure(t *testing.T) {
	parent := context.Background()
	control := newStreamUpstreamContextControl(parent, true)
	t.Cleanup(control.Release)
	c, _ := newDisconnectTestContext()

	svc := &GatewayService{}
	result, err := svc.handleResponsesStreamingResponse(control.ctx, newDisconnectTestResponse(), c, "gpt-5", "claude-sonnet", nil, time.Now())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.ClientDisconnect)
	select {
	case <-control.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("upstream context was not canceled after client write failure")
	}
}
