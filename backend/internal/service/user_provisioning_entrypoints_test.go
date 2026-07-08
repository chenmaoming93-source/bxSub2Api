//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister_UserProvisioningCreatesExactlyOneDefaultKey(t *testing.T) {
	users := &userRepoStub{nextID: 301}
	keys := &provisioningKeyStub{}
	svc := newAuthService(users, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	svc.SetUserProvisioningService(NewUserProvisioningService(provisioningTxStub{}, users, keys))

	_, user, err := svc.Register(context.Background(), "provision-register@test.com", "password")
	require.NoError(t, err)
	require.Equal(t, int64(301), user.ID)
	require.NotNil(t, keys.key)
	require.Equal(t, user.ID, keys.key.UserID)
	require.Len(t, users.created, 1)
}

func TestAdminCreateUser_UserProvisioningCreatesExactlyOneDefaultKey(t *testing.T) {
	users := &userRepoStub{nextID: 302}
	keys := &provisioningKeyStub{}
	svc := &adminServiceImpl{userRepo: users}
	svc.SetUserProvisioningService(NewUserProvisioningService(provisioningTxStub{}, users, keys))

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: "provision-admin@test.com", Password: "password"})
	require.NoError(t, err)
	require.Equal(t, int64(302), user.ID)
	require.NotNil(t, keys.key)
	require.Equal(t, user.ID, keys.key.UserID)
	require.Len(t, users.created, 1)
}

func TestOAuthFirstLogin_UserProvisioningCreatesDefaultKeyOnlyForNewUser(t *testing.T) {
	users := &userRepoStub{nextID: 303}
	keys := &provisioningKeyStub{}
	svc := newAuthService(users, map[string]string{SettingKeyRegistrationEnabled: "true"}, nil, nil)
	svc.SetUserProvisioningService(NewUserProvisioningService(provisioningTxStub{}, users, keys))

	_, first, err := svc.LoginOrRegisterOAuth(context.Background(), "oauth@test.com", "oauth-user")
	require.NoError(t, err)
	require.Equal(t, int64(303), first.ID)
	require.NotNil(t, keys.key)
	require.Len(t, users.created, 1)

	_, second, err := svc.LoginOrRegisterOAuth(context.Background(), "oauth@test.com", "oauth-user")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, users.created, 1)
}
