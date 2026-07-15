package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type defaultAPIKeyRepoStub struct {
	APIKeyRepository
	defaultKey     *APIKey
	createErr      error
	conflictWinner *APIKey
	creates        []*APIKey
	reads          int
}

func (s *defaultAPIKeyRepoStub) GetDefaultByUserID(context.Context, int64) (*APIKey, error) {
	s.reads++
	if s.defaultKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *s.defaultKey
	return &clone, nil
}

func (s *defaultAPIKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	clone := *key
	s.creates = append(s.creates, &clone)
	if s.createErr != nil {
		if s.conflictWinner != nil {
			s.defaultKey = s.conflictWinner
		}
		return s.createErr
	}
	key.ID = 101
	s.defaultKey = key
	return nil
}

type defaultGroupResolverStub struct {
	result DefaultGroupResult
	err    error
}

func (s defaultGroupResolverStub) ResolveDefaultGroup(context.Context) (DefaultGroupResult, error) {
	return s.result, s.err
}

func newDefaultAPIKeyTestService(repo APIKeyRepository, resolver defaultGroupResolverStub) *APIKeyService {
	svc := &APIKeyService{
		apiKeyRepo: repo,
		cfg:        &config.Config{Default: config.DefaultConfig{APIKeyPrefix: "sk-test-"}},
	}
	svc.SetDefaultGroupResolver(resolver)
	return svc
}

func TestEnsureDefaultAPIKey_FoundGroupAndIdempotent(t *testing.T) {
	repo := &defaultAPIKeyRepoStub{}
	svc := newDefaultAPIKeyTestService(repo, defaultGroupResolverStub{
		result: DefaultGroupResult{State: DefaultGroupFound, Group: &Group{ID: 77}},
	})

	first, err := svc.EnsureDefaultAPIKey(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, int64(101), first.ID)
	require.Equal(t, "Default API Key", first.Name)
	require.Equal(t, "default", first.Purpose)
	require.Nil(t, first.Platform)
	require.Equal(t, int64(77), *first.GroupID)

	second, err := svc.EnsureDefaultAPIKey(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, first.Key, second.Key)
	require.Len(t, repo.creates, 1)
}

func TestEnsureDefaultAPIKey_MissingGroupCreatesUnbound(t *testing.T) {
	repo := &defaultAPIKeyRepoStub{}
	svc := newDefaultAPIKeyTestService(repo, defaultGroupResolverStub{
		result: DefaultGroupResult{State: DefaultGroupMissing, Name: "missing"},
	})

	key, err := svc.EnsureDefaultAPIKey(context.Background(), 13)
	require.NoError(t, err)
	require.Nil(t, key.GroupID)
	require.Len(t, repo.creates, 1)
}

func TestEnsureDefaultAPIKey_ConflictReadsExisting(t *testing.T) {
	winner := &APIKey{ID: 202, UserID: 14, Key: "sk-winner", Purpose: "default"}
	repo := &defaultAPIKeyRepoStub{createErr: ErrAPIKeyExists, conflictWinner: winner}
	svc := newDefaultAPIKeyTestService(repo, defaultGroupResolverStub{})

	key, err := svc.EnsureDefaultAPIKey(context.Background(), 14)
	require.NoError(t, err)
	require.Equal(t, winner.Key, key.Key)
	require.Len(t, repo.creates, 1)
	require.Equal(t, 2, repo.reads)
}
