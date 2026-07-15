package service

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

func TestLDAPIdentity(t *testing.T) {
	t.Run("normalizes identity", func(t *testing.T) {
		identity, err := NewLDAPIdentityService().Normalize(&ldapauth.User{Username: " Alice ", Email: " ALICE@EXAMPLE.COM ", DisplayName: " "})
		if err != nil {
			t.Fatal(err)
		}
		if identity.Username != "alice" || identity.Email != "alice@example.com" || identity.DisplayName != "alice" || identity.LocalAccountKey != "ldap:alice" || identity.Source != "ldap" {
			t.Fatalf("identity=%+v", identity)
		}
	})

	t.Run("rejects missing and long username", func(t *testing.T) {
		for _, username := range []string{" ", strings.Repeat("界", 101)} {
			if _, err := NewLDAPIdentityService().Normalize(&ldapauth.User{Username: username}); err == nil {
				t.Fatalf("username %q should fail", username)
			}
		}
	})

	t.Run("truncates display name by rune", func(t *testing.T) {
		identity, err := NewLDAPIdentityService().Normalize(&ldapauth.User{Username: "alice", DisplayName: strings.Repeat("界", 101)})
		if err != nil {
			t.Fatal(err)
		}
		if len([]rune(identity.DisplayName)) != 100 {
			t.Fatalf("display rune count=%d", len([]rune(identity.DisplayName)))
		}
	})
}
