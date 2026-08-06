package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskUsageAPIKeyNeverReturnsConcreteKey(t *testing.T) {
	require.Equal(t, "sk-a****7890", maskUsageAPIKey("sk-api-secret-7890"))
	require.Equal(t, "****", maskUsageAPIKey("short"))
	require.NotContains(t, maskUsageAPIKey("sk-api-secret-7890"), "secret")
}
