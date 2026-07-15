package ldapauth

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-ldap/ldap/v3"
)

func TestLDAPAuthenticator(t *testing.T) {
	t.Run("authenticates resolved DN", func(t *testing.T) {
		lookupConn := &directoryConnStub{result: &ldap.SearchResult{Entries: []*ldap.Entry{
			ldap.NewEntry("uid=alice,dc=example,dc=com", map[string][]string{"uid": {"alice"}, "mail": {"alice@example.com"}}),
		}}}
		bindConn := &directoryConnStub{result: &ldap.SearchResult{}}
		calls := 0
		directory := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) {
			calls++
			if calls == 1 {
				return lookupConn, nil
			}
			return bindConn, nil
		})
		user, err := NewLDAPAuthenticator(directory).Authenticate(context.Background(), "alice", "correct-password")
		if err != nil {
			t.Fatal(err)
		}
		if user.Username != "alice" || bindConn.bindUser != user.DN {
			t.Fatalf("user=%+v bind=%q", user, bindConn.bindUser)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		lookupConn := &directoryConnStub{result: &ldap.SearchResult{Entries: []*ldap.Entry{{DN: "uid=alice"}}}}
		bindConn := &directoryConnStub{bindErr: errors.New("ldap result invalid credentials")}
		calls := 0
		directory := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) {
			calls++
			if calls == 1 {
				return lookupConn, nil
			}
			return bindConn, nil
		})
		_, err := NewLDAPAuthenticator(directory).Authenticate(context.Background(), "alice", "wrong")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err=%v", err)
		}
	})
}
