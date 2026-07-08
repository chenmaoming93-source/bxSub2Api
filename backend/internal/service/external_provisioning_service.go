package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

// ExternalProvisioningService orchestrates user lookup, LDAP fallback,
// provisioning, and platform-key retrieval for external API callers.
type ExternalProvisioningService struct {
	users        ExternalProvisioningUserLookup
	ldap         ExternalProvisioningLDAPDirectory
	provisioner  ExternalUserProvisioner
	platformKeys *PlatformAPIKeyService
}

// ExternalProvisioningUserLookup is the narrow user lookup needed by the
// external provisioning flow.
type ExternalProvisioningUserLookup interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// ExternalProvisioningLDAPDirectory exposes only the directory lookup used by
// the external provisioning flow.
type ExternalProvisioningLDAPDirectory interface {
	LookupUser(ctx context.Context, username string) (*ldapauth.User, error)
}

// ExternalUserProvisioner creates or retrieves a local user.
type ExternalUserProvisioner interface {
	Provision(ctx context.Context, input UserProvisioningInput) (*UserProvisioningResult, error)
}

// NewExternalProvisioningService constructs the orchestration service.
func NewExternalProvisioningService(
	users ExternalProvisioningUserLookup,
	ldap ExternalProvisioningLDAPDirectory,
	provisioner ExternalUserProvisioner,
	platformKeys *PlatformAPIKeyService,
) *ExternalProvisioningService {
	return &ExternalProvisioningService{
		users:        users,
		ldap:         ldap,
		provisioner:  provisioner,
		platformKeys: platformKeys,
	}
}

// EnsurePlatformKeyInput carries the external request payload.
type EnsurePlatformKeyInput struct {
	User    string
	Platform  string
}

// EnsurePlatformKeyResult carries the response.
type EnsurePlatformKeyResult struct {
	User        *User
	APIKey      *APIKey
	UserCreated bool
	KeyCreated  bool
}

// EnsurePlatformKey resolves a user (local or LDAP) and returns a
// platform-scoped API key. It returns ErrUserNotFound when neither local nor
// LDAP lookup succeeds, and the usual service errors for database or
// provisioning failures.
func (s *ExternalProvisioningService) EnsurePlatformKey(ctx context.Context, input EnsurePlatformKeyInput) (*EnsurePlatformKeyResult, error) {
	if err := ValidatePlatform(input.Platform); err != nil {
		return nil, fmt.Errorf("validate platform: %w", err)
	}

	email := strings.TrimSpace(strings.ToLower(input.User))
	if email == "" {
		return nil, fmt.Errorf("user_email is required")
	}

	// 1. Try local user lookup.
	user, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return s.ensureKeyForUser(ctx, user, input.Platform, false)
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("lookup local user: %w", err)
	}

	// 2. LDAP fallback.
	if s.ldap == nil {
		return nil, ErrUserNotFound
	}
	ldapUser, ldapErr := s.ldap.LookupUser(ctx, email)
	if ldapErr != nil {
		if errors.Is(ldapErr, ldapauth.ErrDirectoryNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("ldap lookup: %w", ldapErr)
	}

	// 3. Provision local user from LDAP identity.
	if s.provisioner == nil {
		return nil, fmt.Errorf("provisioner is not configured")
	}
	account := strings.TrimSpace(strings.ToLower(ldapUser.Username))
	if account == "" {
		account = email
	}
	displayName := strings.TrimSpace(ldapUser.DisplayName)
	if displayName == "" {
		displayName = account
	}
	if len([]rune(displayName)) > 100 {
		displayName = string([]rune(displayName)[:100])
	}

	result, err := s.provisioner.Provision(ctx, UserProvisioningInput{
		Email:        email,
		Username:     displayName,
		SignupSource: "ldap",
		Role:         RoleUser,
		Status:       StatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("provision ldap user: %w", err)
	}

	return s.ensureKeyForUser(ctx, result.User, input.Platform, result.Created)
}

func (s *ExternalProvisioningService) ensureKeyForUser(ctx context.Context, user *User, platform string, userCreated bool) (*EnsurePlatformKeyResult, error) {
	if !user.IsActive() {
		return nil, fmt.Errorf("user %d is not active", user.ID)
	}

	if s.platformKeys == nil {
		return nil, fmt.Errorf("platform key service is not configured")
	}
	key, err := s.platformKeys.GetOrCreatePlatformKey(ctx, user.ID, platform)
	if err != nil {
		return nil, fmt.Errorf("get or create platform key: %w", err)
	}

	// A key is newly created when either the user was just created or the
	// key's creation timestamp is later than the user's.
	keyCreated := userCreated || key.CreatedAt.After(user.CreatedAt)

	return &EnsurePlatformKeyResult{
		User:        user,
		APIKey:      key,
		UserCreated: userCreated,
		KeyCreated:  keyCreated,
	}, nil
}
