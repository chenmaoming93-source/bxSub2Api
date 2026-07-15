package ldapauth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidCredentials = errors.New("ldap invalid credentials")

// LDAPAuthenticator authenticates a directory user without invoking the legacy Client flow.
type LDAPAuthenticator struct {
	directory *LDAPDirectory
}

func NewLDAPAuthenticator(directory *LDAPDirectory) *LDAPAuthenticator {
	return &LDAPAuthenticator{directory: directory}
}

func (a *LDAPAuthenticator) Authenticate(ctx context.Context, username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	if a == nil || a.directory == nil {
		return nil, fmt.Errorf("%w: authenticator is not configured", ErrDirectoryUnavailable)
	}

	user, err := a.directory.LookupUser(ctx, username)
	if err != nil {
		return nil, err
	}
	conn, err := a.directory.dial(ctx, a.directory.cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: connect for user bind: %v", ErrDirectoryUnavailable, err)
	}
	defer conn.Close()
	if !strings.HasPrefix(strings.ToLower(a.directory.cfg.ServerURL), "ldaps://") && a.directory.cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: a.directory.cfg.InsecureSkipVerify}); err != nil {
			return nil, fmt.Errorf("%w: starttls for user bind: %v", ErrDirectoryUnavailable, err)
		}
	}
	if err := conn.Bind(user.DN, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
