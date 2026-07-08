package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

var dummyTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// ----- stubs -----

type epUserRepoStub struct {
	users map[string]*User
	err   error
}

func (s *epUserRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	u, ok := s.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

type epLDAPStub struct {
	users map[string]*ldapauth.User
	err   error
}

func (s *epLDAPStub) LookupUser(_ context.Context, username string) (*ldapauth.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	u, ok := s.users[username]
	if !ok {
		return nil, ldapauth.ErrDirectoryNotFound
	}
	return u, nil
}

type epProvisionerStub struct {
	results map[string]*UserProvisioningResult
	err     error
}

func (s *epProvisionerStub) Provision(_ context.Context, input UserProvisioningInput) (*UserProvisioningResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	r, ok := s.results[input.Email]
	if !ok {
		return nil, errors.New("unexpected provision call")
	}
	return r, nil
}

func newEPPlatformKeyService() *PlatformAPIKeyService {
	repo := newPlatformKeyRepoStub()
	return NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})
}

// ----- tests -----

func TestEnsurePlatformKey_LocalUser(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com", Username: "test", Status: StatusActive, CreatedAt: dummyTime}
	users := &epUserRepoStub{users: map[string]*User{"test@example.com": user}}
	svc := NewExternalProvisioningService(users, nil, nil, newEPPlatformKeyService())

	result, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "test@example.com",
		Platform:  "anthropic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.ID != 1 {
		t.Fatalf("expected user ID 1, got %d", result.User.ID)
	}
	if result.APIKey == nil {
		t.Fatal("expected api key")
	}
	if result.UserCreated {
		t.Fatal("expected user not created")
	}
}

func TestEnsurePlatformKey_LDAPFallback(t *testing.T) {
	users := &epUserRepoStub{users: map[string]*User{}}
	ldap := &epLDAPStub{users: map[string]*ldapauth.User{
		"ldapuser@example.com": {Username: "ldapuser", DisplayName: "LDAP User", Email: "ldap@corp.com"},
	}}
	provisioned := &User{ID: 2, Email: "ldapuser@example.com", Username: "LDAP User", Status: StatusActive, CreatedAt: dummyTime}
	provisioner := &epProvisionerStub{results: map[string]*UserProvisioningResult{
		"ldapuser@example.com": {User: provisioned, Created: true},
	}}

	svc := NewExternalProvisioningService(users, ldap, provisioner, newEPPlatformKeyService())
	result, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "ldapuser@example.com",
		Platform:  "openai",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.UserCreated {
		t.Fatal("expected user created")
	}
	if result.APIKey == nil {
		t.Fatal("expected api key")
	}
}

func TestEnsurePlatformKey_UserNotFound(t *testing.T) {
	users := &epUserRepoStub{users: map[string]*User{}}
	ldap := &epLDAPStub{users: map[string]*ldapauth.User{}}
	svc := NewExternalProvisioningService(users, ldap, nil, nil)

	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "noone@example.com",
		Platform:  "openai",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestEnsurePlatformKey_InvalidPlatform(t *testing.T) {
	svc := NewExternalProvisioningService(nil, nil, nil, nil)
	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "test@example.com",
		Platform:  "INVALID!",
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestEnsurePlatformKey_IdempotentKey(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com", Username: "test", Status: StatusActive, CreatedAt: dummyTime}
	users := &epUserRepoStub{users: map[string]*User{"test@example.com": user}}
	svc := NewExternalProvisioningService(users, nil, nil, newEPPlatformKeyService())

	first, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "test@example.com",
		Platform:  "gemini",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:     "test@example.com",
		Platform:  "gemini",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.APIKey.Key != second.APIKey.Key {
		t.Fatal("expected same key for repeated calls")
	}
}
