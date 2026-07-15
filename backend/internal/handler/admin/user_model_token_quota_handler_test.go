package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userModelTokenQuotaRepoStub struct {
	recordsByUser map[int64][]service.UserModelDailyTokenQuotaRecord
	upserts       []struct {
		userID int64
		inputs []service.UserModelDailyTokenQuotaInput
	}
}

func (s *userModelTokenQuotaRepoStub) DeleteUserModelTokenQuotaByModel(context.Context, int64, string) error {
	return nil
}

func (s *userModelTokenQuotaRepoStub) ListUserModelDailyTokenQuotas(_ context.Context, userID int64, _ time.Time) ([]service.UserModelDailyTokenQuotaRecord, error) {
	return s.recordsByUser[userID], nil
}

func (s *userModelTokenQuotaRepoStub) UpsertUserModelDailyTokenQuotas(_ context.Context, userID int64, at time.Time, inputs []service.UserModelDailyTokenQuotaInput) ([]service.UserModelDailyTokenQuotaRecord, error) {
	copied := append([]service.UserModelDailyTokenQuotaInput(nil), inputs...)
	s.upserts = append(s.upserts, struct {
		userID int64
		inputs []service.UserModelDailyTokenQuotaInput
	}{userID: userID, inputs: copied})
	records := make([]service.UserModelDailyTokenQuotaRecord, 0, len(inputs))
	for _, input := range inputs {
		records = append(records, service.UserModelDailyTokenQuotaRecord{
			UserID:           userID,
			Model:            input.Model,
			UsageDate:        at,
			UsedTokens:       3,
			DailyLimitTokens: input.DailyLimitTokens,
		})
	}
	s.recordsByUser[userID] = records
	return records, nil
}

type userModelTokenQuotaInvalidatorStub struct {
	keys []service.UserModelDailyTokenQuotaKey
}

func (s *userModelTokenQuotaInvalidatorStub) InvalidateUserModelDailyTokenQuota(_ context.Context, key service.UserModelDailyTokenQuotaKey) error {
	s.keys = append(s.keys, key)
	return nil
}

func newUserModelTokenQuotaTestHandler(adminSvc *stubAdminService, repo *userModelTokenQuotaRepoStub, invalidator *userModelTokenQuotaInvalidatorStub) *UserModelTokenQuotaHandler {
	svc := service.NewUserModelTokenQuotaAdminService(repo, invalidator)
	return NewUserModelTokenQuotaHandler(adminSvc, svc)
}

func TestUserModelTokenQuotaHandlerListIsUserScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limit := int64(50)
	repo := &userModelTokenQuotaRepoStub{recordsByUser: map[int64][]service.UserModelDailyTokenQuotaRecord{
		1: {{UserID: 1, Model: "gpt-5", UsageDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), UsedTokens: 8, DailyLimitTokens: &limit}},
		2: {{UserID: 2, Model: "gpt-5", UsageDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), UsedTokens: 99, DailyLimitTokens: &limit}},
	}}
	handler := newUserModelTokenQuotaTestHandler(newStubAdminService(), repo, &userModelTokenQuotaInvalidatorStub{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/users/1/model-token-quotas", nil)

	handler.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	quotas := data["quotas"].([]any)
	require.Len(t, quotas, 1)
	first := quotas[0].(map[string]any)
	require.EqualValues(t, 1, first["user_id"])
	require.EqualValues(t, 8, first["used_tokens"])
}

func TestUserModelTokenQuotaHandlerUpdateDoesNotTouchOtherUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	otherLimit := int64(70)
	repo := &userModelTokenQuotaRepoStub{recordsByUser: map[int64][]service.UserModelDailyTokenQuotaRecord{
		2: {{UserID: 2, Model: "gpt-5", UsageDate: time.Now(), UsedTokens: 99, DailyLimitTokens: &otherLimit}},
	}}
	invalidator := &userModelTokenQuotaInvalidatorStub{}
	handler := newUserModelTokenQuotaTestHandler(newStubAdminService(), repo, invalidator)
	body := []byte(`{"quotas":[{"model":" gpt-5 ","daily_limit_tokens":0}]}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/users/1/model-token-quotas", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.upserts, 1)
	require.EqualValues(t, 1, repo.upserts[0].userID)
	require.Equal(t, "gpt-5", repo.upserts[0].inputs[0].Model)
	require.Len(t, repo.recordsByUser[2], 1)
	require.EqualValues(t, 99, repo.recordsByUser[2][0].UsedTokens)
	require.Len(t, invalidator.keys, 1)
	require.EqualValues(t, 1, invalidator.keys[0].UserID)
	require.Equal(t, "gpt-5", invalidator.keys[0].Model)
}

func TestUserModelTokenQuotaHandlerRejectsInvalidInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newUserModelTokenQuotaTestHandler(newStubAdminService(), &userModelTokenQuotaRepoStub{recordsByUser: map[int64][]service.UserModelDailyTokenQuotaRecord{}}, &userModelTokenQuotaInvalidatorStub{})

	t.Run("invalid user id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "bad"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/admin/users/bad/model-token-quotas", nil)
		handler.List(c)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing user", func(t *testing.T) {
		adminSvc := newStubAdminService()
		adminSvc.getUserErr = service.ErrUserNotFound
		missingHandler := newUserModelTokenQuotaTestHandler(adminSvc, &userModelTokenQuotaRepoStub{recordsByUser: map[int64][]service.UserModelDailyTokenQuotaRecord{}}, &userModelTokenQuotaInvalidatorStub{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/admin/users/1/model-token-quotas", nil)
		missingHandler.List(c)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("negative limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodPut, "/admin/users/1/model-token-quotas", bytes.NewReader([]byte(`{"quotas":[{"model":"gpt-5","daily_limit_tokens":-1}]}`)))
		c.Request.Header.Set("Content-Type", "application/json")
		handler.Update(c)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
