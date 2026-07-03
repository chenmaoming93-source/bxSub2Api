package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsTransportErrorIsFailoverableBeforeResponseWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"hello"}],"stream":false}`)

	tests := []struct {
		name    string
		account *Account
		forward func(*OpenAIGatewayService, context.Context, *gin.Context, *Account, []byte) (*OpenAIForwardResult, error)
	}{
		{
			name: "responses compatibility path",
			account: &Account{
				ID: 1, Name: "oauth", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Concurrency: 1,
				Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "account"},
			},
			forward: func(s *OpenAIGatewayService, ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return s.ForwardAsChatCompletions(ctx, c, account, body, "", "")
			},
		},
		{
			name: "raw chat completions path",
			account: &Account{
				ID: 2, Name: "apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": "http://upstream.invalid"},
			},
			forward: func(s *OpenAIGatewayService, ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
				return s.forwardAsRawChatCompletions(ctx, c, account, body, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			upstream := &httpUpstreamRecorder{err: errors.New("dial tcp: connection refused")}
			svc := &OpenAIGatewayService{
				cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
					Enabled: false, AllowInsecureHTTP: true,
				}}},
				httpUpstream: upstream,
			}

			result, err := tt.forward(svc, context.Background(), c, tt.account, body)
			require.Nil(t, result)
			var failoverErr *UpstreamFailoverError
			require.ErrorAs(t, err, &failoverErr)
			require.Zero(t, failoverErr.StatusCode)
			require.False(t, c.Writer.Written(), "transport failure must not commit a client response before failover")
			require.Empty(t, rec.Body.String())
		})
	}
}
