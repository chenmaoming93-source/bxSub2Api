package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ldapSyncHandlerRepoStub struct {
	service.UserRepository
	users []service.User
}

func (r *ldapSyncHandlerRepoStub) List(context.Context, pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return append([]service.User(nil), r.users...), &pagination.PaginationResult{Total: int64(len(r.users))}, nil
}

func (r *ldapSyncHandlerRepoStub) Update(_ context.Context, user *service.User) error {
	for i := range r.users {
		if r.users[i].ID == user.ID {
			r.users[i] = *user
		}
	}
	return nil
}

type ldapSyncHandlerLookupStub struct{}

func (ldapSyncHandlerLookupStub) LookupUser(context.Context, string) (*ldapauth.User, error) {
	return &ldapauth.User{Username: "alice", DisplayName: "Alice", Department: "研发部"}, nil
}

func TestUserHandlerSyncLDAPReturnsSummaryWithoutUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &ldapSyncHandlerRepoStub{users: []service.User{{ID: 1, Email: "alice", Username: "Old"}}}
	syncService := service.NewLDAPUserSyncService(repo, ldapSyncHandlerLookupStub{}, nil, nil)
	handler := NewUserHandlerWithLDAPSync(newStubAdminService(), nil, nil, nil, syncService)
	router := gin.New()
	router.POST("/api/v1/admin/users/sync-ldap", handler.SyncLDAP)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/sync-ldap", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			Total          int `json:"total"`
			Synced         int `json:"synced"`
			Failed         int `json:"failed"`
			LDAPCandidates int `json:"ldap_candidates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, 1, body.Data.Total)
	require.Equal(t, 1, body.Data.LDAPCandidates)
	require.Equal(t, 1, body.Data.Synced)
	require.Zero(t, body.Data.Failed)
	require.Equal(t, "Alice", repo.users[0].Username)
	require.Equal(t, "研发部", repo.users[0].Department)
}
