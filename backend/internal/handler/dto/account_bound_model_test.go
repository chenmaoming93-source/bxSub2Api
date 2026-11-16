package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountFromServiceDerivesBoundModelWithoutCredentialsMutation(t *testing.T) {
	account := &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"z-model": "z-upstream",
				"a-model": "a-upstream",
			},
		},
	}

	got := AccountFromServiceShallow(account)
	require.Equal(t, "a-upstream", got.BoundModel)
	require.True(t, got.BoundModelLegacyInferred)
	require.Len(t, account.Credentials["model_mapping"], 2)
}
