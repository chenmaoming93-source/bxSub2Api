//go:build unit

package growth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type budgetServiceStub struct {
	entryMutation *growthservice.BudgetMutation
	allocTaskID   int64
	allocMutation *growthservice.BudgetMutation
}

func (s *budgetServiceStub) GetAccount(context.Context) (growthservice.BudgetAccount, error) {
	return growthservice.BudgetAccount{TotalAmount: 100}, nil
}

func (s *budgetServiceStub) TopUp(_ context.Context, mutation growthservice.BudgetMutation) (growthservice.BudgetAccount, error) {
	s.entryMutation = &mutation
	return growthservice.BudgetAccount{TotalAmount: mutation.Points}, nil
}

func (s *budgetServiceStub) Reduce(_ context.Context, mutation growthservice.BudgetMutation) (growthservice.BudgetAccount, error) {
	s.entryMutation = &mutation
	return growthservice.BudgetAccount{TotalAmount: 50}, nil
}

func (s *budgetServiceStub) Allocate(_ context.Context, taskID int64, mutation growthservice.BudgetMutation) (growthservice.BudgetAccount, error) {
	s.allocTaskID = taskID
	s.allocMutation = &mutation
	return growthservice.BudgetAccount{AllocatedAmount: mutation.Points}, nil
}

func (s *budgetServiceStub) Revoke(context.Context, int64, growthservice.BudgetMutation) (growthservice.BudgetAccount, error) {
	return growthservice.BudgetAccount{}, nil
}

func (s *budgetServiceStub) ListEntries(context.Context, growthservice.BudgetListFilter) ([]growthservice.BudgetEntry, int64, error) {
	return []growthservice.BudgetEntry{}, 0, nil
}

func (s *budgetServiceStub) ListAllocations(context.Context, growthservice.BudgetListFilter) ([]growthservice.TaskAllocation, int64, error) {
	return []growthservice.TaskAllocation{}, 0, nil
}

func TestCreateBudgetEntryUsesAuthenticatedOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &budgetServiceStub{}
	handler := NewHandler(stub, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/api/v1/admin/growth/budget/entries", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 9})
		handler.CreateBudgetEntry(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/growth/budget/entries",
		strings.NewReader(`{"direction":"INCREASE","points":100,"reason":"活动预算追加","request_id":"body-id"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-Id", "header-id")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, stub.entryMutation)
	require.Equal(t, int64(9), stub.entryMutation.OperatorID)
	require.Equal(t, int64(100), stub.entryMutation.Points)
	require.Equal(t, "header-id", stub.entryMutation.RequestID)
}

func TestAllocateRejectsMissingOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &budgetServiceStub{}
	handler := NewHandler(stub, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/api/v1/admin/growth/budget/allocations", handler.Allocate)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/growth/budget/allocations",
		strings.NewReader(`{"task_id":7,"points":100,"reason":"新任务初始额度"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, stub.allocTaskID)
}
