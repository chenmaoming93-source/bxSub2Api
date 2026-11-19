package handler

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayEntrypointsUseUnifiedSuperAdminExemption(t *testing.T) {
	read := func(path string) string {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		return string(content)
	}

	messages := read("gateway_handler.go")
	chat := read("gateway_handler_chat_completions.go")
	responses := read("gateway_handler_responses.go")
	gemini := read("gemini_v1beta_handler.go")
	openai := read("openai_gateway_handler.go")

	require.Contains(t, messages, "AcquireUserSlotWithWait(c, subject.UserID")
	require.Contains(t, chat, "AcquireUserSlotWithWait(c, subject.UserID")
	require.Contains(t, responses, "AcquireUserSlotWithWait(c, subject.UserID")
	require.Contains(t, gemini, "AcquireUserSlotWithWait(c, authSubject.UserID")
	require.Contains(t, openai, "acquireResponsesUserSlot")
	require.Contains(t, openai, "TryAcquireUserSlotForSubject")
	require.False(t, strings.Contains(openai, "TryAcquireUserSlot(ctx, subject.UserID"),
		"WebSocket paths must use the shared super-admin-aware user-slot helper")
}

func TestAPIKeyAuthenticationCarriesSuperAdminFlag(t *testing.T) {
	auth := readFileForSuperAdminTest(t, "../server/middleware/api_key_auth.go")
	google := readFileForSuperAdminTest(t, "../server/middleware/api_key_auth_google.go")
	subject := readFileForSuperAdminTest(t, "../server/middleware/auth_subject.go")

	require.Contains(t, subject, "SuperAdmin  bool")
	require.Contains(t, auth, "SuperAdmin:  apiKey.User.IsSuperAdmin()")
	require.Contains(t, auth, "service.WithSuperAdminExemption")
	require.Contains(t, google, "SuperAdmin:  apiKey.User.IsSuperAdmin()")
	require.Contains(t, google, "service.WithSuperAdminExemption")
}

func readFileForSuperAdminTest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
