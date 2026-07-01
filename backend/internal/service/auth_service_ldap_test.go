//go:build unit

package service_test

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceLoginLDAPAcceptsNonEmailUsername(t *testing.T) {
	svc, repo, _ := newAuthServiceWithEnt(t, nil, nil)
	hash, err := svc.HashPassword("unused-random-local-password")
	require.NoError(t, err)

	user := &service.User{
		Email:        "zhangsan",
		Username:     "张三",
		PasswordHash: hash,
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(context.Background(), user))

	token, loggedIn, err := svc.LoginLDAP(context.Background(), " ZhangSan ", "张三")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, user.ID, loggedIn.ID)
	require.Equal(t, "zhangsan", loggedIn.Email)
}
