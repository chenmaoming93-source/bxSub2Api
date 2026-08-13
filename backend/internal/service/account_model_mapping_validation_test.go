package service

import (
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateAccountModelMapping(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		credentials map[string]any
		wantCode    string
	}{
		{name: "missing mapping", accountType: AccountTypeAPIKey, credentials: map[string]any{}},
		{name: "empty mapping", accountType: AccountTypeAPIKey, credentials: map[string]any{"model_mapping": map[string]any{}}},
		{name: "blank key ignored", accountType: AccountTypeAPIKey, credentials: map[string]any{"model_mapping": map[string]any{" ": "ignored"}}},
		{name: "single model", accountType: AccountTypeAPIKey, credentials: map[string]any{"model_mapping": map[string]any{"claude-3": "alias"}}},
		{name: "multiple models", accountType: AccountTypeAPIKey, credentials: map[string]any{"model_mapping": map[string]any{"claude-3": "a", "claude-4": "b"}}, wantCode: "ACCOUNT_MULTIPLE_UPSTREAM_MODELS"},
		{name: "invalid mapping type", accountType: AccountTypeAPIKey, credentials: map[string]any{"model_mapping": "claude-3"}, wantCode: "ACCOUNT_INVALID_MODEL_MAPPING"},
		{name: "oauth bypasses multiple models", accountType: AccountTypeOAuth, credentials: map[string]any{"model_mapping": map[string]any{"claude-3": "a", "claude-4": "b"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAccountModelMapping(tt.accountType, tt.credentials)
			if tt.wantCode == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var appErr *infraerrors.ApplicationError
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, tt.wantCode, appErr.Reason)
			require.NotContains(t, appErr.Message, "claude-3")
		})
	}
}
