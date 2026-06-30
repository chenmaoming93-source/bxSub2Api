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

type modelTokenQuotaRepoStub struct {
	records []service.ModelDailyTokenQuotaRecord
	set     service.ModelDailyTokenQuotaRecord
}

func (s *modelTokenQuotaRepoStub) ListModelDailyTokenQuotas(context.Context, time.Time) ([]service.ModelDailyTokenQuotaRecord, error) {
	return s.records, nil
}

func (s *modelTokenQuotaRepoStub) SetModelDailyTokenQuota(_ context.Context, model string, at time.Time, limit *int64) (service.ModelDailyTokenQuotaRecord, error) {
	s.set = service.ModelDailyTokenQuotaRecord{
		Model:            model,
		UsageDate:        at,
		UsedTokens:       12,
		DailyLimitTokens: limit,
	}
	s.records = []service.ModelDailyTokenQuotaRecord{s.set}
	return s.set, nil
}

type modelTokenQuotaInvalidatorStub struct {
	keys []service.ModelDailyTokenQuotaKey
}

func (s *modelTokenQuotaInvalidatorStub) InvalidateModelDailyTokenQuota(_ context.Context, key service.ModelDailyTokenQuotaKey) error {
	s.keys = append(s.keys, key)
	return nil
}

func newModelTokenQuotaTestHandler(repo *modelTokenQuotaRepoStub, invalidator *modelTokenQuotaInvalidatorStub) *ModelTokenQuotaHandler {
	svc := service.NewModelTokenQuotaAdminService(repo, invalidator)
	return NewModelTokenQuotaHandler(svc)
}

func TestModelTokenQuotaHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limit := int64(100)
	repo := &modelTokenQuotaRepoStub{records: []service.ModelDailyTokenQuotaRecord{{
		Model:            "gpt-5",
		UsageDate:        time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		UsedTokens:       42,
		DailyLimitTokens: &limit,
	}}}
	handler := newModelTokenQuotaTestHandler(repo, &modelTokenQuotaInvalidatorStub{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/model-token-quotas", nil)

	handler.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	quotas := data["quotas"].([]any)
	require.Len(t, quotas, 1)
	first := quotas[0].(map[string]any)
	require.Equal(t, "gpt-5", first["model"])
	require.EqualValues(t, 42, first["used_tokens"])
	require.EqualValues(t, 100, first["daily_limit_tokens"])
}

func TestModelTokenQuotaHandlerUpdateInvalidatesCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &modelTokenQuotaRepoStub{}
	invalidator := &modelTokenQuotaInvalidatorStub{}
	handler := newModelTokenQuotaTestHandler(repo, invalidator)
	body := []byte(`{"model":" gpt-5 ","daily_limit_tokens":0}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/model-token-quotas", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "gpt-5", repo.set.Model)
	require.NotNil(t, repo.set.DailyLimitTokens)
	require.EqualValues(t, 0, *repo.set.DailyLimitTokens)
	require.Len(t, invalidator.keys, 1)
	require.Equal(t, "gpt-5", invalidator.keys[0].Model)
	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	quota := data["quota"].(map[string]any)
	require.EqualValues(t, 12, quota["used_tokens"])
	require.EqualValues(t, 0, quota["daily_limit_tokens"])
}

func TestModelTokenQuotaHandlerRejectsNegativeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newModelTokenQuotaTestHandler(&modelTokenQuotaRepoStub{}, &modelTokenQuotaInvalidatorStub{})
	body := []byte(`{"model":"gpt-5","daily_limit_tokens":-1}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/model-token-quotas", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
