package service

import (
	"context"
	"errors"
	"reflect"
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
	return NewPlatformAPIKeyService(repo, &platformKeyGenStub{})
}

type epGroupRepoStub struct {
	groups map[string]*Group
	err    error
}

func (s *epGroupRepoStub) GetByNameExact(_ context.Context, name string) (*Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	group, ok := s.groups[name]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

func publicEPGroup() *Group {
	return &Group{ID: 10, Name: "public", Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}
}

func epGroups(groups ...*Group) *epGroupRepoStub {
	byName := make(map[string]*Group, len(groups))
	for _, group := range groups {
		byName[group.Name] = group
	}
	return &epGroupRepoStub{groups: byName}
}

// ----- tests -----

func TestEnsurePlatformKey_LocalUser(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com", Username: "test", Status: StatusActive, CreatedAt: dummyTime}
	users := &epUserRepoStub{users: map[string]*User{"test@example.com": user}}
	svc := NewExternalProvisioningService(users, nil, nil, newEPPlatformKeyService(), epGroups(publicEPGroup()))

	result, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "test@example.com",
		Platform:  "anthropic",
		GroupName: " public ",
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

	svc := NewExternalProvisioningService(users, ldap, provisioner, newEPPlatformKeyService(), epGroups(publicEPGroup()))
	result, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "ldapuser@example.com",
		Platform:  "openai",
		GroupName: "public",
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
	svc := NewExternalProvisioningService(users, ldap, nil, nil, epGroups(publicEPGroup()))

	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "noone@example.com",
		Platform:  "openai",
		GroupName: "public",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestEnsurePlatformKey_InvalidPlatform(t *testing.T) {
	svc := NewExternalProvisioningService(nil, nil, nil, nil, nil)
	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "test@example.com",
		Platform:  "INVALID!",
		GroupName: "public",
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestEnsurePlatformKey_IdempotentKey(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com", Username: "test", Status: StatusActive, CreatedAt: dummyTime}
	users := &epUserRepoStub{users: map[string]*User{"test@example.com": user}}
	svc := NewExternalProvisioningService(users, nil, nil, newEPPlatformKeyService(), epGroups(publicEPGroup()))

	first, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "test@example.com",
		Platform:  "gemini",
		GroupName: "public",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{
		User:      "test@example.com",
		Platform:  "gemini",
		GroupName: "public",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.APIKey.Key != second.APIKey.Key {
		t.Fatal("expected same key for repeated calls")
	}
}

func TestEnsurePlatformKey_GroupAccess(t *testing.T) {
	tests := []struct {
		name       string
		group      *Group
		allowed    []int64
		wantErr    error
		wantCreate bool
	}{
		{name: "public standard", group: &Group{ID: 10, Name: "target", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}, wantCreate: true},
		{name: "authorized exclusive", group: &Group{ID: 11, Name: "target", Status: StatusActive, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard}, allowed: []int64{11}, wantCreate: true},
		{name: "unauthorized exclusive", group: &Group{ID: 12, Name: "target", Status: StatusActive, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard}, wantErr: ErrProvisioningGroupNotAllowed},
		{name: "inactive", group: &Group{ID: 13, Name: "target", Status: StatusDisabled, SubscriptionType: SubscriptionTypeStandard}, wantErr: ErrProvisioningGroupInactive},
		{name: "subscription", group: &Group{ID: 14, Name: "target", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}, wantErr: ErrProvisioningSubscriptionGroup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{ID: 1, Email: "test@example.com", Status: StatusActive, AllowedGroups: tt.allowed, CreatedAt: dummyTime}
			users := &epUserRepoStub{users: map[string]*User{user.Email: user}}
			keyRepo := newPlatformKeyRepoStub()
			svc := NewExternalProvisioningService(users, nil, nil, NewPlatformAPIKeyService(keyRepo, &platformKeyGenStub{}), epGroups(tt.group))
			result, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{User: user.Email, Platform: "anthropic", GroupName: "target"})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				if len(keyRepo.store) != 0 {
					t.Fatal("rejected group must not create a key")
				}
				return
			}
			if err != nil || result == nil || result.APIKey == nil {
				t.Fatalf("expected successful key creation, result=%v err=%v", result, err)
			}
			if !tt.wantCreate || result.APIKey.GroupID == nil || *result.APIKey.GroupID != tt.group.ID {
				t.Fatalf("unexpected group on key: %v", result.APIKey.GroupID)
			}
		})
	}
}

func TestEnsurePlatformKey_GroupNotFoundDoesNotCreateKey(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com", Status: StatusActive}
	keyRepo := newPlatformKeyRepoStub()
	svc := NewExternalProvisioningService(&epUserRepoStub{users: map[string]*User{user.Email: user}}, nil, nil, NewPlatformAPIKeyService(keyRepo, &platformKeyGenStub{}), epGroups())
	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{User: user.Email, Platform: "openai", GroupName: "missing"})
	if !errors.Is(err, ErrGroupNotFound) || len(keyRepo.store) != 0 {
		t.Fatalf("expected not found without key creation, err=%v keys=%d", err, len(keyRepo.store))
	}
}

func TestEnsurePlatformKey_LDAPNewUserCannotUseExclusiveGroup(t *testing.T) {
	users := &epUserRepoStub{users: map[string]*User{}}
	ldap := &epLDAPStub{users: map[string]*ldapauth.User{"new@example.com": {Username: "new", DisplayName: "New User"}}}
	created := &User{ID: 2, Email: "new@example.com", Status: StatusActive}
	provisioner := &epProvisionerStub{results: map[string]*UserProvisioningResult{"new@example.com": {User: created, Created: true}}}
	exclusive := &Group{ID: 20, Name: "exclusive", Status: StatusActive, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard}
	keyRepo := newPlatformKeyRepoStub()
	svc := NewExternalProvisioningService(users, ldap, provisioner, NewPlatformAPIKeyService(keyRepo, &platformKeyGenStub{}), epGroups(exclusive))
	_, err := svc.EnsurePlatformKey(context.Background(), EnsurePlatformKeyInput{User: "new@example.com", Platform: "openai", GroupName: "exclusive"})
	if !errors.Is(err, ErrProvisioningGroupNotAllowed) || len(keyRepo.store) != 0 {
		t.Fatalf("expected exclusive group rejection without key creation, err=%v", err)
	}
}

func TestExternalProvisioningService_ListGroupModelRoutes(t *testing.T) {
	group := publicEPGroup()
	group.ModelRoutingEnabled = false
	group.ModelRouting = map[string]any{
		"z-route": []map[string]any{
			{"model": "model-a-low-priority", "account_ids": []int64{1}, "priority": 20},
			{"model": "model-z-high-priority", "account_ids": []int64{2}, "priority": 0},
			{"model": "model-m-middle-priority", "account_ids": []int64{3}, "priority": 10},
			{"model": "model-z-high-priority", "account_ids": []int64{4}, "priority": 30},
		},
		"a-route": []map[string]any{
			{"model": "model-c", "account_ids": []int64{1}, "priority": 0},
		},
	}
	svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(group))

	routes, err := svc.ListGroupModelRoutes(context.Background(), ListGroupModelRoutesInput{GroupName: " public "})
	if err != nil {
		t.Fatalf("ListGroupModelRoutes() error = %v", err)
	}
	want := []GroupModelRouteProjection{
		{RouteAlias: "a-route", UpstreamModels: []string{"model-c"}},
		{RouteAlias: "z-route", UpstreamModels: []string{"model-z-high-priority", "model-m-middle-priority", "model-a-low-priority"}},
	}
	if !reflect.DeepEqual(routes, want) {
		t.Fatalf("ListGroupModelRoutes() = %#v, want %#v", routes, want)
	}
}

func TestExternalProvisioningService_ListGroupModelRoutesEmpty(t *testing.T) {
	svc := NewExternalProvisioningService(nil, nil, nil, nil, epGroups(publicEPGroup()))
	routes, err := svc.ListGroupModelRoutes(context.Background(), ListGroupModelRoutesInput{GroupName: "public"})
	if err != nil || routes == nil || len(routes) != 0 {
		t.Fatalf("expected non-nil empty routes, routes=%#v err=%v", routes, err)
	}
}

func TestExternalProvisioningService_ListGroupModelRoutesErrors(t *testing.T) {
	repoErr := errors.New("database unavailable")
	tests := []struct {
		name string
		svc  *ExternalProvisioningService
		want error
	}{
		{name: "not found", svc: NewExternalProvisioningService(nil, nil, nil, nil, epGroups()), want: ErrGroupNotFound},
		{name: "repository", svc: NewExternalProvisioningService(nil, nil, nil, nil, &epGroupRepoStub{err: repoErr}), want: repoErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.ListGroupModelRoutes(context.Background(), ListGroupModelRoutesInput{GroupName: "missing"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want wrapped %v", err, tt.want)
			}
		})
	}
}
