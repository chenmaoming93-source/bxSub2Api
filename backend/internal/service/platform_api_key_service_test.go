package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{name: "valid simple", input: "github", wantErr: false},
		{name: "valid with underscore", input: "git_lab", wantErr: false},
		{name: "valid single char", input: "a", wantErr: false},
		{name: "valid long", input: strings.Repeat("a", 50), wantErr: false},
		{name: "empty", input: "", wantErr: true, errMsg: "required"},
		{name: "whitespace only", input: "   ", wantErr: true, errMsg: "required"},
		{name: "too long", input: strings.Repeat("a", 51), wantErr: true, errMsg: "50 characters"},
		{name: "starts with number", input: "123abc", wantErr: true, errMsg: "only lowercase letters"},
		{name: "uppercase", input: "GitHub", wantErr: true, errMsg: "only lowercase letters"},
		{name: "hyphen", input: "git-hub", wantErr: true, errMsg: "only lowercase letters"},
		{name: "space inside", input: "git hub", wantErr: true, errMsg: "only lowercase letters"},
		{name: "special chars", input: "github!", wantErr: true, errMsg: "only lowercase letters"},
		{name: "underscore only not allowed", input: "_", wantErr: true, errMsg: "only lowercase letters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlatform(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for input %q: %v", tt.input, err)
				}
			}
		})
	}
}

// ---- stub types for GetOrCreatePlatformKey tests ----

type platformKeyRepoStub struct {
	store map[string]*APIKey // composite key "userID:platform"
}

func newPlatformKeyRepoStub() *platformKeyRepoStub {
	return &platformKeyRepoStub{store: make(map[string]*APIKey)}
}

func (r *platformKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	keyStr := key.Platform
	if keyStr == nil {
		return nil
	}
	composite := fmt.Sprintf("%d:%s", key.UserID, *keyStr)
	if _, exists := r.store[composite]; exists {
		return ErrAPIKeyExists
	}
	key.ID = int64(len(r.store) + 1)
	r.store[composite] = key
	return nil
}

func (r *platformKeyRepoStub) GetByUserIDAndPlatform(_ context.Context, userID int64, platform string) (*APIKey, error) {
	composite := fmt.Sprintf("%d:%s", userID, platform)
	key, exists := r.store[composite]
	if !exists {
		return nil, ErrAPIKeyNotFound
	}
	return key, nil
}

// Remaining APIKeyRepository methods are no-ops for this test.
func (r *platformKeyRepoStub) GetByID(_ context.Context, _ int64) (*APIKey, error)     { return nil, nil }
func (r *platformKeyRepoStub) GetKeyAndOwnerID(_ context.Context, _ int64) (string, int64, error) {
	return "", 0, nil
}
func (r *platformKeyRepoStub) GetByKey(_ context.Context, _ string) (*APIKey, error)     { return nil, nil }
func (r *platformKeyRepoStub) GetByKeyForAuth(_ context.Context, _ string) (*APIKey, error) {
	return nil, nil
}
func (r *platformKeyRepoStub) Update(_ context.Context, _ *APIKey) error                   { return nil }
func (r *platformKeyRepoStub) Delete(_ context.Context, _ int64) error                     { return nil }
func (r *platformKeyRepoStub) DeleteWithAudit(_ context.Context, _ int64) error            { return nil }
func (r *platformKeyRepoStub) ListByUserID(_ context.Context, _ int64, _ pagination.PaginationParams, _ APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *platformKeyRepoStub) VerifyOwnership(_ context.Context, _ int64, _ []int64) ([]int64, error) {
	return nil, nil
}
func (r *platformKeyRepoStub) CountByUserID(_ context.Context, _ int64) (int64, error)          { return 0, nil }
func (r *platformKeyRepoStub) ExistsByKey(_ context.Context, _ string) (bool, error)            { return false, nil }
func (r *platformKeyRepoStub) ListByGroupID(_ context.Context, _ int64, _ pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *platformKeyRepoStub) SearchAPIKeys(_ context.Context, _ int64, _ string, _ int) ([]APIKey, error) {
	return nil, nil
}
func (r *platformKeyRepoStub) ClearGroupIDByGroupID(_ context.Context, _ int64) (int64, error) { return 0, nil }
func (r *platformKeyRepoStub) UpdateGroupIDByUserAndGroup(_ context.Context, _, _, _ int64) (int64, error) {
	return 0, nil
}
func (r *platformKeyRepoStub) CountByGroupID(_ context.Context, _ int64) (int64, error)       { return 0, nil }
func (r *platformKeyRepoStub) ListKeysByUserID(_ context.Context, _ int64) ([]string, error)   { return nil, nil }
func (r *platformKeyRepoStub) ListKeysByGroupID(_ context.Context, _ int64) ([]string, error)  { return nil, nil }
func (r *platformKeyRepoStub) IncrementQuotaUsed(_ context.Context, _ int64, _ float64) (float64, error) {
	return 0, nil
}
func (r *platformKeyRepoStub) UpdateLastUsed(_ context.Context, _ int64, _ time.Time) error  { return nil }
func (r *platformKeyRepoStub) IncrementRateLimitUsage(_ context.Context, _ int64, _ float64) error {
	return nil
}
func (r *platformKeyRepoStub) ResetRateLimitWindows(_ context.Context, _ int64) error { return nil }
func (r *platformKeyRepoStub) GetRateLimitData(_ context.Context, _ int64) (*APIKeyRateLimitData, error) {
	return nil, nil
}

type platformKeyGenStub struct{ next int }

func (g *platformKeyGenStub) GenerateKey() (string, error) {
	g.next++
	return fmt.Sprintf("sk-plat-%d", g.next), nil
}

type platformGroupResolverStub struct{}

func (platformGroupResolverStub) ResolveDefaultGroup(_ context.Context) (DefaultGroupResult, error) {
	return DefaultGroupResult{State: DefaultGroupUnconfigured}, nil
}

func TestGetOrCreatePlatformKey_Create(t *testing.T) {
	repo := newPlatformKeyRepoStub()
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	key, err := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected key, got nil")
	}
	if key.UserID != 1 {
		t.Fatalf("expected userID 1, got %d", key.UserID)
	}
	if key.Platform == nil || *key.Platform != "github" {
		t.Fatalf("expected platform 'github', got %v", key.Platform)
	}
	if key.Purpose != "platform" {
		t.Fatalf("expected purpose 'platform', got %s", key.Purpose)
	}
	if !strings.Contains(key.Name, "github") {
		t.Fatalf("expected name containing 'github', got %s", key.Name)
	}
}

func TestGetOrCreatePlatformKey_Idempotent(t *testing.T) {
	repo := newPlatformKeyRepoStub()
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	key1, err := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	key2, err := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	if key1.ID != key2.ID {
		t.Fatalf("expected same key (id %d vs %d)", key1.ID, key2.ID)
	}
}

func TestGetOrCreatePlatformKey_DifferentUsers(t *testing.T) {
	repo := newPlatformKeyRepoStub()
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	key1, _ := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	key2, _ := svc.GetOrCreatePlatformKey(context.Background(), 2, "github")

	if key1.ID == key2.ID {
		t.Fatal("different users should have different keys")
	}
}

func TestGetOrCreatePlatformKey_DifferentPlatforms(t *testing.T) {
	repo := newPlatformKeyRepoStub()
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	key1, _ := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	key2, _ := svc.GetOrCreatePlatformKey(context.Background(), 1, "gitlab")

	if key1.ID == key2.ID {
		t.Fatal("different platforms should have different keys")
	}
}

func TestGetOrCreatePlatformKey_InvalidPlatform(t *testing.T) {
	repo := newPlatformKeyRepoStub()
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	_, err := svc.GetOrCreatePlatformKey(context.Background(), 1, "Git Hub")
	if err == nil {
		t.Fatal("expected error for invalid platform name, got nil")
	}
}

func TestGetOrCreatePlatformKey_NoPlatformRepo(t *testing.T) {
	// A repository that implements APIKeyRepository but NOT PlatformAPIKeyRepository.
	type noPlatformRepo struct {
		APIKeyRepository
	}

	repo := &noPlatformRepo{newPlatformKeyRepoStub()}
	svc := NewPlatformAPIKeyService(repo, &platformKeyGenStub{}, platformGroupResolverStub{})

	_, err := svc.GetOrCreatePlatformKey(context.Background(), 1, "github")
	if err == nil {
		t.Fatal("expected error when repository does not implement PlatformAPIKeyRepository")
	}
}
