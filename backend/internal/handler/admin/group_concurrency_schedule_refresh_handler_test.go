package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type blockingScheduleRefreshRepo struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingScheduleRefreshRepo) ListModelRouteConcurrencyScheduleCandidates(context.Context) ([]service.ModelRouteConcurrencyScheduleCandidate, error) {
	select {
	case <-r.started:
	default:
		close(r.started)
	}
	<-r.release
	return nil, nil
}

func TestGroupConcurrencyScheduleRefreshHandlerStartsImmediatelyAndRejectsOverlap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	cache := repository.NewConcurrencyCache(rdb, 15, 60)
	concurrency := service.NewConcurrencyService(cache)
	repo := &blockingScheduleRefreshRepo{started: make(chan struct{}), release: make(chan struct{})}
	refresher := service.NewModelRouteConcurrencyScheduleRefresher(repo, concurrency, &config.Config{Timezone: "Asia/Shanghai"})

	h := NewGroupHandler(newStubAdminService(), nil, nil)
	h.SetModelRouteConcurrencyScheduleRefresher(refresher)
	router := gin.New()
	router.POST("/groups/:id/model-route-references/concurrency-schedules/refresh", h.RefreshModelRouteConcurrencySchedules)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/groups/1/model-route-references/concurrency-schedules/refresh", nil))
	require.Equal(t, http.StatusAccepted, first.Code)
	require.Contains(t, first.Body.String(), "task_id")
	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("immediate refresh did not start promptly")
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/groups/1/model-route-references/concurrency-schedules/refresh", nil))
	require.Equal(t, http.StatusConflict, second.Code)
	require.Contains(t, second.Body.String(), "refresh task is already running")

	close(repo.release)
	require.Eventually(t, func() bool {
		exists, err := rdb.Exists(context.Background(), "concurrency:route-schedule:refresh-lock").Result()
		return err == nil && exists == 0
	}, time.Second, 10*time.Millisecond)
}
