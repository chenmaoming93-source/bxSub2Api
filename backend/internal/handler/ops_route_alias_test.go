package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetOpsRequestContextCapturesAndClearsRouteAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{
		ID:                  7,
		ModelRoutingEnabled: true,
		ModelRouting: map[string][]domain.ModelRouteCandidate{
			"coding-fast": {{AccountIDs: []int64{11}}},
		},
	}
	ctx := contextWithTestGroup(group)
	for _, endpoint := range []string{
		"/v1/messages",
		"/v1/responses",
		"/v1/chat/completions",
		"/v1beta/models/gemini-2.5-pro:generateContent",
	} {
		t.Run(endpoint, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, endpoint, nil).WithContext(ctx)

			setOpsRequestContext(c, "coding-fast", false)
			require.Equal(t, "coding-fast", getOpsRouteAlias(c))

			// A later failover into a non-routed group must clear stale context.
			setOpsRouteAlias(c, "")
			require.Empty(t, getOpsRouteAlias(c))
		})
	}
}

func contextWithTestGroup(group *service.Group) context.Context {
	return context.WithValue(context.Background(), ctxkey.Group, group)
}
