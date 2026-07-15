package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type handlerModelReportRepo struct{}

func (handlerModelReportRepo) ListModelTokenUsage(context.Context, service.TokenUsageReportQuery) ([]service.ModelTokenUsageRow, int64, int64, error) {
	return []service.ModelTokenUsageRow{}, 0, 0, nil
}

func TestTokenUsageReportAPIContractShapes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTokenUsageReportHandler(service.NewTokenUsageReportService(handlerModelReportRepo{}))
	r := gin.New()
	r.GET("/models", h.GetModels)
	r.GET("/routes", h.GetRoutes)
	r.GET("/users", h.GetUsers)
	for _, path := range []string{"/models?start_date=2026-07-01&end_date=2026-07-01", "/routes?start_date=2026-07-01&end_date=2026-07-01", "/users?start_date=2026-07-01&end_date=2026-07-01"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		var envelope map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		data, ok := envelope["data"].(map[string]any)
		if !ok {
			t.Fatalf("%s data=%T", path, envelope["data"])
		}
		for _, key := range []string{"items", "summary", "pagination"} {
			if _, ok := data[key]; !ok {
				t.Fatalf("%s missing %s: %v", path, key, data)
			}
		}
		summary := data["summary"].(map[string]any)
		if _, ok := summary["used_tokens"]; !ok {
			t.Fatalf("%s summary=%v", path, summary)
		}
		pagination := data["pagination"].(map[string]any)
		for _, key := range []string{"page", "page_size", "total"} {
			if _, ok := pagination[key]; !ok {
				t.Fatalf("%s pagination missing %s", path, key)
			}
		}
	}
}

func TestTokenUsageReportHandlerRouteAndUserValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTokenUsageReportHandler(service.NewTokenUsageReportService(handlerModelReportRepo{}))
	r := gin.New()
	r.GET("/routes", h.GetRoutes)
	r.GET("/users", h.GetUsers)
	for _, path := range []string{"/routes?group_id=1", "/routes?group_id=x&route_alias=r&start_date=2026-07-01&end_date=2026-07-01", "/users", "/users?user_id=x&start_date=2026-07-01&end_date=2026-07-01", "/users?start_date=2026-07-01&end_date=2026-07-01&include_deleted=maybe"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"/routes?start_date=2026-07-01&end_date=2026-07-01", "/routes?group_id=1&route_alias=r&start_date=2026-07-01&end_date=2026-07-01", "/users?start_date=2026-07-01&end_date=2026-07-01", "/users?user_id=2&start_date=2026-07-01&end_date=2026-07-01"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}
func (handlerModelReportRepo) ListRouteTokenUsage(context.Context, service.RouteTokenUsageReportQuery) ([]service.RouteTokenUsageRow, int64, int64, error) {
	return nil, 0, 0, nil
}
func (handlerModelReportRepo) ListUserTokenUsage(context.Context, service.UserTokenUsageReportQuery) ([]service.UserTokenUsageRow, int64, int64, error) {
	return nil, 0, 0, nil
}
func (handlerModelReportRepo) SearchTokenUsageOptions(context.Context, string, int64, string, int) ([]service.TokenUsageOption, error) {
	return nil, nil
}
func (handlerModelReportRepo) FindDefaultTokenUsageTarget(context.Context, string, time.Time) (*service.TokenUsageOption, error) {
	return nil, nil
}

func TestTokenUsageReportHandlerValidationAndSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTokenUsageReportHandler(service.NewTokenUsageReportService(handlerModelReportRepo{}))
	r := gin.New()
	r.GET("/api/v1/admin/token-usage/models", h.GetModels)
	for _, path := range []string{"/api/v1/admin/token-usage/models", "/api/v1/admin/token-usage/models?model=gpt&start_date=bad", "/api/v1/admin/token-usage/models?model=gpt&page_size=101"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: got %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/token-usage/models?start_date=2026-07-01&end_date=2026-07-01", nil))
	if w.Code != 200 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
}
