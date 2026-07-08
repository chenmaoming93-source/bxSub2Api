package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// platformNamePattern allows lowercase letters and underscores only.
var platformNamePattern = regexp.MustCompile(`^[a-z][a-z_]*$`)

// ValidatePlatform returns nil when name matches the platform naming rules.
func ValidatePlatform(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("platform name is required")
	}
	if len(name) > 50 {
		return fmt.Errorf("platform name must not exceed 50 characters")
	}
	if !platformNamePattern.MatchString(name) {
		return fmt.Errorf("platform name must contain only lowercase letters and underscores, and start with a letter")
	}
	return nil
}

// PlatformAPIKeyService provides an idempotent get-or-create for
// platform-specific API keys. Concurrent callers always observe the same key.
type PlatformAPIKeyService struct {
	keys     APIKeyRepository
	keyGen   interface{ GenerateKey() (string, error) }
	resolver interface {
		ResolveDefaultGroup(ctx context.Context) (DefaultGroupResult, error)
	}
}

// NewPlatformAPIKeyService creates a PlatformAPIKeyService.
// keyGen is typically *APIKeyService (for GenerateKey).
func NewPlatformAPIKeyService(
	keys APIKeyRepository,
	keyGen interface{ GenerateKey() (string, error) },
	resolver interface {
		ResolveDefaultGroup(ctx context.Context) (DefaultGroupResult, error)
	},
) *PlatformAPIKeyService {
	return &PlatformAPIKeyService{keys: keys, keyGen: keyGen, resolver: resolver}
}

// GetOrCreatePlatformKey returns the platform-scoped key for (userID, platform).
// It returns an existing key regardless of status (active, disabled, expired,
// quota_exhausted, soft-deleted keys are excluded at the repository layer).
func (s *PlatformAPIKeyService) GetOrCreatePlatformKey(ctx context.Context, userID int64, platform string) (*APIKey, error) {
	if err := ValidatePlatform(platform); err != nil {
		return nil, fmt.Errorf("validate platform: %w", err)
	}

	platformRepo, ok := s.keys.(PlatformAPIKeyRepository)
	if !ok {
		return nil, fmt.Errorf("api key repository does not support platform-key lookup")
	}

	existing, err := platformRepo.GetByUserIDAndPlatform(ctx, userID, platform)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrAPIKeyNotFound) {
		return nil, fmt.Errorf("get platform api key: %w", err)
	}

	var groupID *int64
	if s.resolver != nil {
		resolved, err := s.resolver.ResolveDefaultGroup(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve default group: %w", err)
		}
		if resolved.State == DefaultGroupFound && resolved.Group != nil {
			id := resolved.Group.ID
			groupID = &id
		}
	}

	rawKey, err := s.keyGen.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate platform api key: %w", err)
	}

	created := &APIKey{
		UserID:   userID,
		Key:      rawKey,
		Name:     platform + " API Key",
		Platform: &platform,
		Purpose:  "platform",
		GroupID:  groupID,
		Status:   StatusActive,
	}
	if err := s.keys.Create(ctx, created); err != nil {
		if errors.Is(err, ErrAPIKeyExists) {
			existing, readErr := platformRepo.GetByUserIDAndPlatform(ctx, userID, platform)
			if readErr == nil {
				return existing, nil
			}
			return nil, fmt.Errorf("read platform api key after conflict: %w", readErr)
		}
		return nil, fmt.Errorf("create platform api key: %w", err)
	}
	return created, nil
}
