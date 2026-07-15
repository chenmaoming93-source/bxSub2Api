package ldapauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-ldap/ldap/v3"
)

// User is the minimal directory identity needed by the local account system.
type User struct {
	Username    string
	Email       string
	DisplayName string
	DN          string
}

// Authenticator allows the login handler to remain independent of LDAP details.
type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (*User, error)
}

type Client struct {
	cfg config.LDAPConfig
}

func New(cfg config.LDAPConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Authenticate(_ context.Context, username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, fmt.Errorf("ldap: username and password are required")
	}

	conn, err := c.dial()
	if err != nil {
		return nil, fmt.Errorf("ldap: connect failed: %w", err)
	}
	defer conn.Close()

	if c.cfg.Domain != "" {
		if err := conn.NTLMBind(c.cfg.Domain, username, password); err != nil {
			return nil, fmt.Errorf("ldap: NTLM bind failed: %w", err)
		}
		return c.searchUser(conn, username)
	}

	if !strings.HasPrefix(strings.ToLower(c.cfg.ServerURL), "ldaps://") && c.cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: c.cfg.InsecureSkipVerify}); err != nil {
			return nil, fmt.Errorf("ldap: StartTLS failed: %w", err)
		}
	}

	if c.cfg.UserDN != "" {
		dn := fmt.Sprintf(c.cfg.UserDN, ldap.EscapeFilter(username))
		if err := conn.Bind(dn, password); err != nil {
			return nil, fmt.Errorf("ldap: user bind failed: %w", err)
		}
		user, err := c.lookupDN(conn, dn)
		if err != nil {
			return &User{Username: username, DN: dn}, nil
		}
		return user, nil
	}

	if err := conn.Bind(c.cfg.BindDN, c.cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("ldap: service bind failed: %w", err)
	}
	user, err := c.searchUser(conn, username)
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(user.DN, password); err != nil {
		return nil, fmt.Errorf("ldap: user bind failed: %w", err)
	}
	return user, nil
}

func (c *Client) dial() (*ldap.Conn, error) {
	timeout := time.Duration(c.cfg.ConnectTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	opts := []ldap.DialOpt{ldap.DialWithDialer(dialer)}
	if strings.HasPrefix(strings.ToLower(c.cfg.ServerURL), "ldaps://") {
		opts = append(opts, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: c.cfg.InsecureSkipVerify}))
	}
	return ldap.DialURL(c.cfg.ServerURL, opts...)
}

func (c *Client) searchUser(conn *ldap.Conn, username string) (*User, error) {
	filter := fmt.Sprintf(c.cfg.UserFilter, ldap.EscapeFilter(username))
	req := ldap.NewSearchRequest(
		c.cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		2, 0, false, filter,
		[]string{c.cfg.UsernameAttribute, c.cfg.EmailAttribute, c.cfg.DisplayNameAttribute}, nil,
	)
	result, err := conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("ldap: user search failed: %w", err)
	}
	if len(result.Entries) != 1 {
		return nil, fmt.Errorf("ldap: expected one user, found %d", len(result.Entries))
	}
	return c.userFromEntry(result.Entries[0], username), nil
}

func (c *Client) lookupDN(conn *ldap.Conn, dn string) (*User, error) {
	req := ldap.NewSearchRequest(
		dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases,
		1, 0, false, "(objectClass=*)",
		[]string{c.cfg.UsernameAttribute, c.cfg.EmailAttribute, c.cfg.DisplayNameAttribute}, nil,
	)
	result, err := conn.Search(req)
	if err != nil || len(result.Entries) != 1 {
		return nil, fmt.Errorf("ldap: user attribute lookup failed")
	}
	return c.userFromEntry(result.Entries[0], ""), nil
}

func (c *Client) userFromEntry(entry *ldap.Entry, fallbackUsername string) *User {
	username := entry.GetAttributeValue(c.cfg.UsernameAttribute)
	if username == "" {
		username = fallbackUsername
	}
	return &User{
		Username:    strings.TrimSpace(username),
		Email:       strings.TrimSpace(strings.ToLower(entry.GetAttributeValue(c.cfg.EmailAttribute))),
		DisplayName: strings.TrimSpace(entry.GetAttributeValue(c.cfg.DisplayNameAttribute)),
		DN:          entry.DN,
	}
}
