package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type ldapSyncUserRepoStub struct {
	UserRepository
	mu      sync.Mutex
	users   []User
	updates []User
}

func (r *ldapSyncUserRepoStub) List(_ context.Context, _ pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	users := append([]User(nil), r.users...)
	return users, &pagination.PaginationResult{Total: int64(len(users))}, nil
}

func (r *ldapSyncUserRepoStub) Update(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.users {
		if r.users[i].ID == user.ID {
			r.users[i] = *user
		}
	}
	r.updates = append(r.updates, *user)
	return nil
}

type ldapSyncLookupStub struct {
	users map[string]*ldapauth.User
	errs  map[string]error
}

func (s ldapSyncLookupStub) LookupUser(_ context.Context, account string) (*ldapauth.User, error) {
	if err := s.errs[account]; err != nil {
		return nil, err
	}
	return s.users[account], nil
}

type ldapSyncCacheInvalidatorStub struct {
	mu  sync.Mutex
	ids []int64
}

func (s *ldapSyncCacheInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {}

func (s *ldapSyncCacheInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, userID)
}

func (s *ldapSyncCacheInvalidatorStub) InvalidateAuthCacheByGroupID(context.Context, int64) {}

func TestLDAPUserSyncServiceClassifiesAndContinuesAfterFailures(t *testing.T) {
	repo := &ldapSyncUserRepoStub{users: []User{
		{ID: 1, Email: "Admin@Example.com", Username: "Admin", Department: "Operations"},
		{ID: 2, Email: "alice", Username: "Old Name", Department: "Old Department"},
		{ID: 3, Email: "missing", Username: "Missing", Department: "Keep Me"},
		{ID: 4, Email: "broken", Username: "Broken", Department: "Keep Me"},
	}}
	cache := &ldapSyncCacheInvalidatorStub{}
	svc := NewLDAPUserSyncService(repo, ldapSyncLookupStub{
		users: map[string]*ldapauth.User{
			"alice": {Username: "alice", DisplayName: "Alice Example", Department: "研发部"},
		},
		errs: map[string]error{
			"missing": ldapauth.ErrDirectoryNotFound,
			"broken":  errors.New("directory timeout"),
		},
	}, cache, []string{" admin@example.com "})

	result, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 4, result.Total)
	require.Equal(t, 3, result.LDAPCandidates)
	require.Equal(t, 1, result.Synced)
	require.Equal(t, 1, result.LocalCleared)
	require.Equal(t, 1, result.NotFound)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Errors, 1)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	byID := make(map[int64]User, len(repo.users))
	for _, user := range repo.users {
		byID[user.ID] = user
	}
	require.Empty(t, byID[1].Department)
	require.Equal(t, "Alice Example", byID[2].Username)
	require.Equal(t, "研发部", byID[2].Department)
	require.Equal(t, "Keep Me", byID[3].Department)
	require.Equal(t, "Keep Me", byID[4].Department)

	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.ElementsMatch(t, []int64{1, 2}, cache.ids)
}

func TestLDAPUserSyncServiceClearsSuccessfulEmptyDepartment(t *testing.T) {
	repo := &ldapSyncUserRepoStub{users: []User{{ID: 7, Email: "empty", Username: "Old", Department: "Existing"}}}
	cache := &ldapSyncCacheInvalidatorStub{}
	svc := NewLDAPUserSyncService(repo, ldapSyncLookupStub{
		users: map[string]*ldapauth.User{"empty": {Username: "empty", DisplayName: "New"}},
		errs:  map[string]error{},
	}, cache, nil)

	result, err := svc.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.Synced)
	require.Empty(t, repo.users[0].Department)
	require.Equal(t, []int64{7}, cache.ids)
}
