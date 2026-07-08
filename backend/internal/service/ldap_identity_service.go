package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

const LDAPIdentitySource = "ldap"

type LDAPIdentity struct {
	Username        string
	Email           string
	DisplayName     string
	LocalAccountKey string
	Source          string
}

type LDAPIdentityService struct{}

func NewLDAPIdentityService() *LDAPIdentityService { return &LDAPIdentityService{} }

func (s *LDAPIdentityService) Normalize(user *ldapauth.User) (LDAPIdentity, error) {
	if user == nil {
		return LDAPIdentity{}, fmt.Errorf("ldap identity is required")
	}
	username := strings.ToLower(strings.TrimSpace(user.Username))
	if username == "" {
		return LDAPIdentity{}, fmt.Errorf("ldap username is required")
	}
	if utf8.RuneCountInString(username) > 100 {
		return LDAPIdentity{}, fmt.Errorf("ldap username must not exceed 100 characters")
	}
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = username
	}
	if utf8.RuneCountInString(displayName) > 100 {
		displayName = string([]rune(displayName)[:100])
	}
	return LDAPIdentity{
		Username: username, Email: strings.ToLower(strings.TrimSpace(user.Email)),
		DisplayName: displayName, LocalAccountKey: LDAPIdentitySource + ":" + username,
		Source: LDAPIdentitySource,
	}, nil
}
