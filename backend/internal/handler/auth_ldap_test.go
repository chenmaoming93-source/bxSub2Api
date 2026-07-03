package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
	"github.com/stretchr/testify/require"
)

type ldapAuthenticatorStub struct{}

func (ldapAuthenticatorStub) Authenticate(string, string) (*ldapauth.User, error) {
	return nil, nil
}

func TestAuthHandlerShouldUseLDAP(t *testing.T) {
	h := &AuthHandler{
		cfg: &config.Config{LDAP: config.LDAPConfig{
			Enabled:            true,
			LocalLoginAccounts: []string{"admin@example.com", " emergency-admin "},
		}},
		ldapAuthenticator: ldapAuthenticatorStub{},
	}

	require.True(t, h.shouldUseLDAP("zhangsan"))
	require.False(t, h.shouldUseLDAP("ADMIN@example.com"))
	require.False(t, h.shouldUseLDAP("emergency-admin"))

	h.cfg.LDAP.Enabled = false
	require.False(t, h.shouldUseLDAP("zhangsan"))
}
