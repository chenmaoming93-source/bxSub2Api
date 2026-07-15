package ldapauth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-ldap/ldap/v3"
)

var (
	ErrDirectoryNotFound    = errors.New("ldap directory user not found")
	ErrDirectoryAmbiguous   = errors.New("ldap directory user is ambiguous")
	ErrDirectoryUnavailable = errors.New("ldap directory unavailable")
)

type DirectoryConn interface {
	Bind(username, password string) error
	StartTLS(*tls.Config) error
	Search(*ldap.SearchRequest) (*ldap.SearchResult, error)
	Close() error
}

type DirectoryDialer func(context.Context, config.LDAPConfig) (DirectoryConn, error)

type LDAPDirectory struct {
	cfg  config.LDAPConfig
	dial DirectoryDialer
}

func NewLDAPDirectory(cfg config.LDAPConfig, dial DirectoryDialer) *LDAPDirectory {
	return &LDAPDirectory{cfg: cfg, dial: dial}
}

func NewDefaultLDAPDirectory(cfg config.LDAPConfig) *LDAPDirectory {
	return NewLDAPDirectory(cfg, func(ctx context.Context, cfg config.LDAPConfig) (DirectoryConn, error) {
		timeout := time.Duration(cfg.ConnectTimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		dialer := &net.Dialer{Timeout: timeout}
		if deadline, ok := ctx.Deadline(); ok {
			dialer.Deadline = deadline
		}
		opts := []ldap.DialOpt{ldap.DialWithDialer(dialer)}
		if strings.HasPrefix(strings.ToLower(cfg.ServerURL), "ldaps://") {
			opts = append(opts, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}))
		}
		return ldap.DialURL(cfg.ServerURL, opts...)
	})
}

func (d *LDAPDirectory) LookupUser(ctx context.Context, username string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("%w: username is required", ErrDirectoryUnavailable)
	}
	if strings.TrimSpace(d.cfg.BindDN) == "" || d.cfg.BindPassword == "" {
		return nil, fmt.Errorf("%w: service bind credentials are required", ErrDirectoryUnavailable)
	}
	if d.dial == nil {
		return nil, fmt.Errorf("%w: directory dialer is not configured", ErrDirectoryUnavailable)
	}

	conn, err := d.dial(ctx, d.cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: connect: %v", ErrDirectoryUnavailable, err)
	}
	defer conn.Close()

	if !strings.HasPrefix(strings.ToLower(d.cfg.ServerURL), "ldaps://") && d.cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: d.cfg.InsecureSkipVerify}); err != nil {
			return nil, fmt.Errorf("%w: starttls: %v", ErrDirectoryUnavailable, err)
		}
	}
	if err := conn.Bind(d.cfg.BindDN, d.cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("%w: service bind: %v", ErrDirectoryUnavailable, err)
	}

	filter := fmt.Sprintf(d.cfg.UserFilter, ldap.EscapeFilter(username))
	request := ldap.NewSearchRequest(
		d.cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		2, 0, false, filter,
		[]string{d.cfg.UsernameAttribute, d.cfg.EmailAttribute, d.cfg.DisplayNameAttribute}, nil,
	)
	result, err := conn.Search(request)
	if err != nil {
		return nil, fmt.Errorf("%w: search: %v", ErrDirectoryUnavailable, err)
	}
	switch len(result.Entries) {
	case 0:
		return nil, ErrDirectoryNotFound
	case 1:
		entry := result.Entries[0]
		resolvedUsername := strings.TrimSpace(entry.GetAttributeValue(d.cfg.UsernameAttribute))
		if resolvedUsername == "" {
			resolvedUsername = username
		}
		return &User{
			Username:    resolvedUsername,
			Email:       strings.ToLower(strings.TrimSpace(entry.GetAttributeValue(d.cfg.EmailAttribute))),
			DisplayName: strings.TrimSpace(entry.GetAttributeValue(d.cfg.DisplayNameAttribute)),
			DN:          entry.DN,
		}, nil
	default:
		return nil, ErrDirectoryAmbiguous
	}
}
