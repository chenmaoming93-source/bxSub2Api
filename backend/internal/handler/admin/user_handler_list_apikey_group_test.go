package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// listUsersFilterStub 捕获传入 ListUsers 的 filters，其余 AdminService 方法走 baseline stub。
type listUsersFilterStub struct {
	service.AdminService
	captured service.UserListFilters
	users    []service.User
}

func (s *listUsersFilterStub) ListUsers(_ context.Context, _, _ int, filters service.UserListFilters, _, _ string) ([]service.User, int64, error) {
	s.captured = filters
	return s.users, int64(len(s.users)), nil
}

type superAdminUserProviderStub struct {
	ids []int64
}

func (s superAdminUserProviderStub) SuperAdminUserIDs(context.Context, []int64) ([]int64, error) {
	return s.ids, nil
}

func TestAdminUserList_ParsesAPIKeyGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name  string
		query string
		want  int64
	}{
		{"valid id", "?api_key_group_id=42", 42},
		{"missing", "", 0},
		{"zero ignored", "?api_key_group_id=0", 0},
		{"negative ignored", "?api_key_group_id=-3", 0},
		{"non-numeric ignored", "?api_key_group_id=abc", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &listUsersFilterStub{AdminService: newStubAdminService()}
			r := gin.New()
			h := NewUserHandler(stub, nil, nil, nil)
			r.GET("/admin/users", h.List)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/admin/users"+tc.query, nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, tc.want, stub.captured.APIKeyGroupID)
		})
	}
}

func TestAdminUserList_ParsesExcludeSuperAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &listUsersFilterStub{AdminService: newStubAdminService()}
	r := gin.New()
	h := NewUserHandler(stub, nil, nil, nil)
	r.GET("/admin/users", h.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?exclude_super_admins=true", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, stub.captured.ExcludeSuperAdmins)
}

func TestAdminUserList_MarksSuperAdministrators(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &listUsersFilterStub{
		AdminService: newStubAdminService(),
		users: []service.User{
			{ID: 1, Role: service.RoleAdmin, Email: "admin@example.com"},
			{ID: 2, Role: service.RoleUser, Email: "user@example.com"},
		},
	}
	r := gin.New()
	h := NewUserHandler(stub, nil, nil, nil)
	h.SetSuperAdminUserProvider(superAdminUserProviderStub{ids: []int64{1}})
	r.GET("/admin/users", h.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"is_super_admin":true`)
	require.Contains(t, w.Body.String(), `"is_super_admin":false`)
}
