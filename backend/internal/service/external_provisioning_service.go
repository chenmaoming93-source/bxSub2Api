package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
)

var (
	ErrProvisioningGroupInactive     = infraerrors.Conflict("GROUP_INACTIVE", "group is inactive")
	ErrProvisioningSubscriptionGroup = infraerrors.BadRequest("SUBSCRIPTION_GROUP_NOT_SUPPORTED", "subscription groups are not supported")
	ErrProvisioningGroupNotAllowed   = infraerrors.Forbidden("GROUP_NOT_ALLOWED", "user is not allowed to use this group")
)

// ExternalProvisioningService orchestrates user lookup, LDAP fallback,
// provisioning, and platform-key retrieval for external API callers.
type ExternalProvisioningService struct {
	users        ExternalProvisioningUserLookup
	ldap         ExternalProvisioningLDAPDirectory
	provisioner  ExternalUserProvisioner
	platformKeys *PlatformAPIKeyService
	groups       ExternalProvisioningGroupLookup
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

type ExternalProvisioningGroupLookup interface {
	GetByNameExact(ctx context.Context, name string) (*Group, error)
}

// NewExternalProvisioningService constructs the orchestration service.
func NewExternalProvisioningService(
	users ExternalProvisioningUserLookup,
	ldap ExternalProvisioningLDAPDirectory,
	provisioner ExternalUserProvisioner,
	platformKeys *PlatformAPIKeyService,
	groups ExternalProvisioningGroupLookup,
) *ExternalProvisioningService {
	return &ExternalProvisioningService{
		users:        users,
		ldap:         ldap,
		provisioner:  provisioner,
		platformKeys: platformKeys,
		groups:       groups,
	}
}

// EnsurePlatformKeyInput carries the external request payload.
type EnsurePlatformKeyInput struct {
	User      string
	Platform  string
	GroupName string
}

// EnsurePlatformKeyResult carries the response.
type EnsurePlatformKeyResult struct {
	User        *User
	APIKey      *APIKey
	UserCreated bool
	KeyCreated  bool
	Group       *Group
}

type ListGroupModelRoutesInput struct {
	GroupName string
}

type GroupModelRouteProjection struct {
	RouteAlias     string
	UpstreamModels []string
}

func (s *ExternalProvisioningService) ListGroupModelRoutes(ctx context.Context, input ListGroupModelRoutesInput) ([]GroupModelRouteProjection, error) {
	groupName := strings.TrimSpace(input.GroupName)
	if groupName == "" {
		return nil, fmt.Errorf("group_name is required")
	}
	if s.groups == nil {
		return nil, fmt.Errorf("group lookup is not configured")
	}
	group, err := s.groups.GetByNameExact(ctx, groupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("lookup group routes: %w", err)
	}
	if group.ModelRouting == nil {
		return []GroupModelRouteProjection{}, nil
	}

	data, err := json.Marshal(group.ModelRouting)
	if err != nil {
		return nil, fmt.Errorf("encode group model routing: %w", err)
	}
	config, err := domain.ParseModelRoutingConfig(data)
	if err != nil {
		return nil, fmt.Errorf("parse group model routing: %w", err)
	}

	aliases := make([]string, 0, len(config))
	for alias := range config {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	result := make([]GroupModelRouteProjection, 0, len(aliases))
	for _, alias := range aliases {
		modelSet := make(map[string]struct{})
		models := make([]string, 0, len(config[alias]))
		for _, candidate := range config[alias] {
			model := strings.TrimSpace(candidate.Model)
			if model == "" {
				continue
			}
			if _, exists := modelSet[model]; exists {
				continue
			}
			modelSet[model] = struct{}{}
			models = append(models, model)
		}
		result = append(result, GroupModelRouteProjection{RouteAlias: alias, UpstreamModels: models})
	}
	return result, nil
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
	groupName := strings.TrimSpace(input.GroupName)
	if groupName == "" {
		return nil, fmt.Errorf("group_name is required")
	}
	group, err := s.resolveAllowedGroup(ctx, groupName)
	if err != nil {
		return nil, err
	}

	// 1. Try local user lookup.
	user, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return s.ensureKeyForUser(ctx, user, input.Platform, group, false)
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

	return s.ensureKeyForUser(ctx, result.User, input.Platform, group, result.Created)
}

func (s *ExternalProvisioningService) resolveAllowedGroup(ctx context.Context, groupName string) (*Group, error) {
	if s.groups == nil {
		return nil, fmt.Errorf("group lookup is not configured")
	}
	group, err := s.groups.GetByNameExact(ctx, groupName)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("lookup group: %w", err)
	}
	if !group.IsActive() {
		return nil, ErrProvisioningGroupInactive
	}
	if group.IsSubscriptionType() {
		return nil, ErrProvisioningSubscriptionGroup
	}
	return group, nil
}

func (s *ExternalProvisioningService) ensureKeyForUser(ctx context.Context, user *User, platform string, group *Group, userCreated bool) (*EnsurePlatformKeyResult, error) {
	if !user.IsActive() {
		return nil, fmt.Errorf("user %d is not active", user.ID)
	}
	if !user.CanBindGroup(group.ID, group.IsExclusive) {
		return nil, ErrProvisioningGroupNotAllowed
	}

	if s.platformKeys == nil {
		return nil, fmt.Errorf("platform key service is not configured")
	}
	key, err := s.platformKeys.GetOrCreatePlatformKey(ctx, user.ID, platform, group.ID)
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
		Group:       group,
	}, nil
}
