package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type scheduleAdminServiceStub struct {
	*stubAdminService
	schedules []service.ModelRouteConcurrencySchedule
	replaced  []service.ModelRouteConcurrencySchedule
}

func (s *scheduleAdminServiceStub) ListGroupModelRouteConcurrencySchedules(_ context.Context, _ int64, _ string, _ int64) ([]service.ModelRouteConcurrencySchedule, error) {
	return s.schedules, nil
}

func (s *scheduleAdminServiceStub) ReplaceGroupModelRouteConcurrencySchedules(_ context.Context, _ int64, _ string, _ int64, schedules []service.ModelRouteConcurrencySchedule) error {
	s.replaced = schedules
	return nil
}

func newScheduleHandlerRouter(svc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewGroupHandler(svc, nil, nil)
	router.GET("/groups/:id/model-route-references/concurrency-schedules", h.ListModelRouteConcurrencySchedules)
	router.PUT("/groups/:id/model-route-references/concurrency-schedules", h.ReplaceModelRouteConcurrencySchedules)
	return router
}

func TestGroupConcurrencyScheduleHandlerReplaceAndList(t *testing.T) {
	svc := &scheduleAdminServiceStub{stubAdminService: newStubAdminService()}
	router := newScheduleHandlerRouter(svc)

	replaceBody := `{"route_alias":"test-zlx","account_id":100,"schedules":[{"start":"09:30","end":"20:30","max_concurrency":50},{"start":"20:30","end":"24:00","max_concurrency":null}]}`
	replaceRequest := httptest.NewRequest(http.MethodPut, "/groups/1/model-route-references/concurrency-schedules", strings.NewReader(replaceBody))
	replaceRequest.Header.Set("Content-Type", "application/json")
	replaceRecorder := httptest.NewRecorder()
	router.ServeHTTP(replaceRecorder, replaceRequest)
	require.Equal(t, http.StatusOK, replaceRecorder.Code)
	require.Len(t, svc.replaced, 2)
	require.Equal(t, 570, svc.replaced[0].StartMinute)
	require.Equal(t, 1440, svc.replaced[1].EndMinute)
	require.Nil(t, svc.replaced[1].MaxConcurrency)

	svc.schedules = []service.ModelRouteConcurrencySchedule{{ID: 7, StartMinute: 0, EndMinute: 570}}
	listRequest := httptest.NewRequest(http.MethodGet, "/groups/1/model-route-references/concurrency-schedules?route_alias=test-zlx&account_id=100", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	require.Equal(t, http.StatusOK, listRecorder.Code)
	var envelope struct {
		Data []struct {
			ID    int64  `json:"id"`
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRecorder.Body.Bytes(), &envelope))
	require.Equal(t, []struct {
		ID    int64  `json:"id"`
		Start string `json:"start"`
		End   string `json:"end"`
	}{{ID: 7, Start: "00:00", End: "09:30"}}, envelope.Data)
}

func TestGroupConcurrencyScheduleHandlerRejectsOverlap(t *testing.T) {
	svc := &scheduleAdminServiceStub{stubAdminService: newStubAdminService()}
	router := newScheduleHandlerRouter(svc)
	body := `{"route_alias":"test-zlx","account_id":100,"schedules":[{"start":"09:00","end":"10:00","max_concurrency":10},{"start":"09:30","end":"11:00","max_concurrency":20}]}`
	request := httptest.NewRequest(http.MethodPut, "/groups/1/model-route-references/concurrency-schedules", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, svc.replaced)
}
