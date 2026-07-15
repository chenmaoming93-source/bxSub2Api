package ldapauth

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-ldap/ldap/v3"
)

type directoryConnStub struct {
	bindUser string
	bindErr  error
	request  *ldap.SearchRequest
	result   *ldap.SearchResult
	err      error
	tls      bool
}

func (c *directoryConnStub) Bind(username, _ string) error { c.bindUser = username; return c.bindErr }
func (c *directoryConnStub) StartTLS(*tls.Config) error    { c.tls = true; return nil }
func (c *directoryConnStub) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	c.request = req
	return c.result, c.err
}
func (c *directoryConnStub) Close() error { return nil }

func directoryConfig() config.LDAPConfig {
	return config.LDAPConfig{
		ServerURL: "ldap://directory.example", BaseDN: "dc=example,dc=com",
		BindDN: "cn=service,dc=example,dc=com", BindPassword: "service-secret",
		UserFilter: "(uid=%s)", UsernameAttribute: "uid", EmailAttribute: "mail",
		DisplayNameAttribute: "displayName", StartTLS: true,
	}
}

func TestLDAPDirectoryLookupUser(t *testing.T) {
	t.Run("found without user password bind", func(t *testing.T) {
		conn := &directoryConnStub{result: &ldap.SearchResult{Entries: []*ldap.Entry{
			ldap.NewEntry("uid=alice,dc=example,dc=com", map[string][]string{"uid": {"alice"}, "mail": {" ALICE@EXAMPLE.COM "}, "displayName": {" Alice "}}),
		}}}
		directory := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) { return conn, nil })
		user, err := directory.LookupUser(context.Background(), " alice ")
		if err != nil {
			t.Fatal(err)
		}
		if conn.bindUser != "cn=service,dc=example,dc=com" || user.Email != "alice@example.com" || user.DisplayName != "Alice" {
			t.Fatalf("bind=%q user=%+v", conn.bindUser, user)
		}
		if !conn.tls || conn.request.SizeLimit != 2 || conn.request.Filter != "(uid=alice)" {
			t.Fatalf("tls=%v request=%+v", conn.tls, conn.request)
		}
	})

	t.Run("not found", func(t *testing.T) {
		conn := &directoryConnStub{result: &ldap.SearchResult{}}
		_, err := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) { return conn, nil }).LookupUser(context.Background(), "nobody")
		if !errors.Is(err, ErrDirectoryNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
		conn := &directoryConnStub{result: &ldap.SearchResult{Entries: []*ldap.Entry{{DN: "one"}, {DN: "two"}}}}
		_, err := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) { return conn, nil }).LookupUser(context.Background(), "duplicate")
		if !errors.Is(err, ErrDirectoryAmbiguous) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("unavailable and input escaped", func(t *testing.T) {
		conn := &directoryConnStub{result: &ldap.SearchResult{}}
		directory := NewLDAPDirectory(directoryConfig(), func(context.Context, config.LDAPConfig) (DirectoryConn, error) { return conn, nil })
		_, _ = directory.LookupUser(context.Background(), "*)(uid=*)")
		if conn.request == nil || strings.Contains(conn.request.Filter, "*)(uid=*)") {
			t.Fatalf("unsafe filter=%q", conn.request.Filter)
		}

		cfg := directoryConfig()
		cfg.BindPassword = ""
		_, err := NewLDAPDirectory(cfg, nil).LookupUser(context.Background(), "alice")
		if !errors.Is(err, ErrDirectoryUnavailable) {
			t.Fatalf("err=%v", err)
		}
	})
}
