package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type provisioningTxStub struct{ rollback func() }

func (s provisioningTxStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		if s.rollback != nil {
			s.rollback()
		}
		return err
	}
	return nil
}

type provisioningUserStub struct {
	user           *User
	createErr      error
	conflictWinner *User
	nextID         int64
}

func (s *provisioningUserStub) GetByEmail(context.Context, string) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	clone := *s.user
	return &clone, nil
}
func (s *provisioningUserStub) Create(_ context.Context, user *User) error {
	if s.createErr != nil {
		if s.conflictWinner != nil {
			s.user = s.conflictWinner
		}
		return s.createErr
	}
	user.ID = s.nextID
	clone := *user
	s.user = &clone
	return nil
}

type provisioningKeyStub struct {
	key *APIKey
	err error
}

func (s *provisioningKeyStub) EnsureDefaultAPIKey(_ context.Context, userID int64) (*APIKey, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.key == nil {
		s.key = &APIKey{ID: 9, UserID: userID, Purpose: "default"}
	}
	return s.key, nil
}

func TestUserProvisioning_CoreSuccessAndPostCommit(t *testing.T) {
	users := &provisioningUserStub{nextID: 7}
	keys := &provisioningKeyStub{}
	postCalled := false
	svc := NewUserProvisioningService(provisioningTxStub{}, users, keys)
	svc.makeHash = func() (string, error) { return "hash", nil }
	result, err := svc.Provision(context.Background(), UserProvisioningInput{Email: " New@Example.com ", SignupSource: "admin", PostCommit: []UserProvisioningPostCommitAction{{Name: "subscriptions-and-quotas", Run: func(context.Context, *User) error { postCalled = true; return nil }}}})
	require.NoError(t, err)
	require.True(t, result.Created)
	require.Equal(t, "new@example.com", result.User.Email)
	require.Equal(t, int64(7), result.DefaultAPIKey.UserID)
	require.True(t, postCalled)
}

func TestUserProvisioning_DefaultKeyFailureRollsBackUser(t *testing.T) {
	users := &provisioningUserStub{nextID: 8}
	tx := provisioningTxStub{rollback: func() { users.user = nil }}
	svc := NewUserProvisioningService(tx, users, &provisioningKeyStub{err: errors.New("key failed")})
	svc.makeHash = func() (string, error) { return "hash", nil }
	_, err := svc.Provision(context.Background(), UserProvisioningInput{Email: "rollback@example.com"})
	require.Error(t, err)
	require.Nil(t, users.user)
}

func TestUserProvisioning_ExistingAndConflictReadback(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		users := &provisioningUserStub{user: &User{ID: 10, Email: "exists@example.com"}}
		result, err := NewUserProvisioningService(provisioningTxStub{}, users, &provisioningKeyStub{}).Provision(context.Background(), UserProvisioningInput{Email: "exists@example.com"})
		require.NoError(t, err)
		require.False(t, result.Created)
		require.Equal(t, int64(10), result.User.ID)
	})
	t.Run("conflict", func(t *testing.T) {
		winner := &User{ID: 11, Email: "race@example.com"}
		users := &provisioningUserStub{nextID: 12, createErr: ErrEmailExists, conflictWinner: winner}
		// Simulate the concurrent winner becoming visible when Create reports conflict.
		result, err := NewUserProvisioningService(provisioningTxStub{}, users, &provisioningKeyStub{}).Provision(context.Background(), UserProvisioningInput{Email: "race@example.com"})
		require.NoError(t, err)
		require.False(t, result.Created)
		require.Equal(t, int64(11), result.User.ID)
	})
}
