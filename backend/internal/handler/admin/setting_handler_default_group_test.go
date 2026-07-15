package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type defaultGroupSettingRepo struct{ values map[string]string }

func (r *defaultGroupSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r *defaultGroupSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}
func (r *defaultGroupSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = make(map[string]string)
	}
	r.values[key] = value
	return nil
}
func (r *defaultGroupSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *defaultGroupSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *defaultGroupSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *defaultGroupSettingRepo) Delete(context.Context, string) error { return nil }

type defaultGroupHandlerLookup struct {
	groups map[string]*service.Group
}

func (l defaultGroupHandlerLookup) GetByNameExact(_ context.Context, name string) (*service.Group, error) {
	group, ok := l.groups[name]
	if !ok {
		return nil, service.ErrGroupNotFound
	}
	return group, nil
}

func TestSettingHandler_DefaultGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &defaultGroupSettingRepo{values: map[string]string{}}
	svc := service.NewSettingService(repo, &config.Config{})
	svc.SetDefaultGroupNameLookup(defaultGroupHandlerLookup{groups: map[string]*service.Group{}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	t.Run("save missing group", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"  future-group  "}`)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/default-group", body)
		ctx.Request.Header.Set("Content-Type", "application/json")
		handler.UpdateDefaultGroup(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if repo.values[service.SettingKeyDefaultGroupName] != "future-group" {
			t.Fatalf("saved=%q", repo.values[service.SettingKeyDefaultGroupName])
		}
		var envelope struct {
			Data defaultGroupResponse `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if !envelope.Data.Configured || envelope.Data.Exists || envelope.Data.Name != "future-group" {
			t.Fatalf("data=%+v", envelope.Data)
		}
	})

	t.Run("reject blank", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/default-group", bytes.NewBufferString(`{"name":" "}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		handler.UpdateDefaultGroup(ctx)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

}
