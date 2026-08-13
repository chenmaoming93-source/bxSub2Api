package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// modelAttributesStubService 记录 UpdateAccount 入参，并把 model_attributes 回带到响应。
type modelAttributesStubService struct {
	*stubAdminService
	lastUpdateInput *service.UpdateAccountInput
}

func (s *modelAttributesStubService) UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.lastUpdateInput = input
	return &service.Account{
		ID:              id,
		Name:            input.Name,
		Status:          service.StatusActive,
		ModelAttributes: input.ModelAttributes,
	}, nil
}

func newAccountHandlerForModelAttributesTest(svc service.AdminService) *AccountHandler {
	return NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestAccountHandler_Update_ModelAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &modelAttributesStubService{stubAdminService: newStubAdminService()}
	handler := newAccountHandlerForModelAttributesTest(stub)
	router := gin.New()
	router.PUT("/api/v1/admin/accounts/:id", handler.Update)

	body := map[string]any{
		"name": "renamed",
		"model_attributes": map[string]any{
			"context_window":  map[string]any{"description": "上下文窗口总大小（token）", "value": 200000},
			"supports_vision": map[string]any{"description": "支持图片输入", "value": true},
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/1", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, stub.lastUpdateInput)
	require.Equal(t, "renamed", stub.lastUpdateInput.Name)
	// JSON 数字反序列化为 float64（Go 对 any 值的默认行为，符合“信任前端”设计）
	require.Equal(t, float64(200000), stub.lastUpdateInput.ModelAttributes["context_window"].Value)
	require.Equal(t, true, stub.lastUpdateInput.ModelAttributes["supports_vision"].Value)

	// 响应（{code, message, data}）的 data 应回带 model_attributes
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs, ok := resp.Data["model_attributes"].(map[string]any)
	require.True(t, ok, "response data should include model_attributes")
	require.Contains(t, attrs, "context_window")
	require.Contains(t, attrs, "supports_vision")
}

func TestAccountHandler_Create_ModelAttributesForwarded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	handler := newAccountHandlerForModelAttributesTest(adminSvc)
	router := gin.New()
	router.POST("/api/v1/admin/accounts", handler.Create)

	body := map[string]any{
		"name":        "acc-1",
		"platform":    "anthropic",
		"type":        "apikey",
		"credentials": map[string]any{"api_key": "sk-ant-xxx"},
		"model_attributes": map[string]any{
			"context_window": map[string]any{"description": "上下文窗口总大小（token）", "value": 200000},
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Equal(t, float64(200000), adminSvc.createdAccounts[0].ModelAttributes["context_window"].Value)
	require.Equal(t, "上下文窗口总大小（token）", adminSvc.createdAccounts[0].ModelAttributes["context_window"].Description)
}
