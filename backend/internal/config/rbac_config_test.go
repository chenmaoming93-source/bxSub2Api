package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRBACConfigValidation(t *testing.T) {
	require.NoError(t, ValidateRBACConfig(RBACConfig{Mode: "shadow", CacheTTLMinutes: 20}))
	require.NoError(t, ValidateRBACConfig(RBACConfig{Mode: "enforce", CacheTTLMinutes: 20}))
	require.ErrorContains(t, ValidateRBACConfig(RBACConfig{Mode: "disabled", CacheTTLMinutes: 20}), "rbac.mode")
	require.ErrorContains(t, ValidateRBACConfig(RBACConfig{Mode: "enforce", CacheTTLMinutes: -1}), "cache_ttl")
}
