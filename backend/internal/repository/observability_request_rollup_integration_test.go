//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestObservabilityRequestRollupIdempotentAndDeduplicated(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { require.NoError(t, timezone.Init("UTC")) })

	ddl, err := os.ReadFile("../../sqlArchiving/177_create_observability_request_rollup.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(ddl))
	require.NoError(t, err)
	capacityDDL, err := os.ReadFile("../../sqlArchiving/179_add_observability_capacity_metrics.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(capacityDDL))
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(capacityDDL))
	require.NoError(t, err, "capacity migration must be idempotent")

	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{Email: "observability-rollup@example.com"})
	group := mustCreateGroup(t, client, &service.Group{Name: "observability-rollup"})
	account := mustCreateAccount(t, client, &service.Account{Name: "observability-rollup"})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, GroupID: &group.ID})

	base := time.Date(2026, 11, 10, 15, 59, 20, 0, time.UTC)
	usageRepo := newUsageLogRepositoryWithSQL(client, integrationDB)
	for _, item := range []struct {
		requestID string
		route     string
		input     int
		output    int
		cache     int
		duration  int
		ttft      *int
		request   service.RequestType
	}{
		{"success-nonstream", "coding-fast", 100, 40, 25, 800, nil, service.RequestTypeSync},
		{"success-stream", "coding-fast", 120, 60, 30, 1500, intPtrForRollup(200), service.RequestTypeStream},
		{"failed-retry-final-error", "coding-fast", 50, 0, 0, 1200, nil, service.RequestTypeSync},
	} {
		_, err := usageRepo.createSingle(context.Background(), integrationDB, &service.UsageLog{
			UserID:          user.ID,
			APIKeyID:        apiKey.ID,
			AccountID:       account.ID,
			GroupID:         &group.ID,
			RequestID:       item.requestID,
			Model:           "claude-sonnet-4",
			RequestedModel:  item.route,
			RouteAlias:      item.route,
			InputTokens:     item.input,
			OutputTokens:    item.output,
			CacheReadTokens: item.cache,
			DurationMs:      &item.duration,
			FirstTokenMs:    item.ttft,
			RequestType:     item.request,
			CreatedAt:       base,
		})
		require.NoError(t, err)
	}

	opsRepo := NewOpsRepository(integrationDB)
	_, err = opsRepo.InsertErrorLog(context.Background(), &service.OpsInsertErrorLogInput{
		RequestID:         "failed-retry-final-error",
		UserID:            &user.ID,
		APIKeyID:          &apiKey.ID,
		AccountID:         &account.ID,
		GroupID:           &group.ID,
		Platform:          service.PlatformAnthropic,
		Model:             "claude-sonnet-4",
		RequestedModel:    "coding-fast",
		RouteAlias:        "coding-fast",
		RequestType:       int16PtrForRollup(int16(service.RequestTypeSync)),
		ErrorPhase:        "upstream",
		ErrorType:         "upstream_error",
		StatusCode:        502,
		ResponseLatencyMs: int64PtrForRollup(1200),
		IsBusinessLimited: false,
		CreatedAt:         base,
	})
	require.NoError(t, err)
	_, err = opsRepo.InsertErrorLog(context.Background(), &service.OpsInsertErrorLogInput{
		RequestID:         "rate-limited-final",
		UserID:            &user.ID,
		APIKeyID:          &apiKey.ID,
		AccountID:         &account.ID,
		GroupID:           &group.ID,
		Platform:          service.PlatformAnthropic,
		Model:             "claude-sonnet-4",
		RequestedModel:    "coding-fast",
		RouteAlias:        "coding-fast",
		RequestType:       int16PtrForRollup(int16(service.RequestTypeSync)),
		ErrorPhase:        "upstream",
		ErrorType:         "rate_limit_error",
		StatusCode:        429,
		ResponseLatencyMs: int64PtrForRollup(300),
		IsBusinessLimited: false,
		CreatedAt:         base.Add(time.Second),
	})
	require.NoError(t, err)

	repo := NewObservabilityRequestRollupRepository(integrationDB)
	start := base.Add(-time.Minute)
	end := base.Add(2 * time.Minute)
	require.NoError(t, repo.AggregateRequestRange(context.Background(), start, end))
	require.NoError(t, repo.AggregateRequestRange(context.Background(), start, end))

	minute, err := repo.QueryRequestBucketTotals(context.Background(), start, end, service.ObservabilityWindowMinute)
	require.NoError(t, err)
	require.Len(t, minute, 1)
	require.Equal(t, int64(4), minute[0].RequestCount)
	require.Equal(t, int64(2), minute[0].SuccessCount)
	require.Equal(t, int64(2), minute[0].ErrorCount)
	require.Equal(t, int64(1), minute[0].RateLimitedCount)
	require.Equal(t, int64(100+40+25+120+60+30+50), minute[0].TotalTokens)
	require.Equal(t, "2026-11-10T23:59:00+08:00", minute[0].BucketStart.In(timezone.Location()).Format(time.RFC3339))

	fiveMinute, err := repo.QueryRequestBucketTotals(context.Background(), start, end.Add(10*time.Minute), service.ObservabilityWindowFiveMinute)
	require.NoError(t, err)
	require.Len(t, fiveMinute, 1)
	require.Equal(t, int64(4), fiveMinute[0].RequestCount)
	nextLocalDay := time.Date(2026, 11, 11, 0, 0, 0, 0, timezone.Location())
	nextDay, err := repo.QueryRequestBucketTotals(context.Background(), nextLocalDay, nextLocalDay.Add(24*time.Hour), service.ObservabilityWindowHour)
	require.NoError(t, err)
	require.Empty(t, nextDay, "today's rate-limit count must reset at the configured local day boundary")

	require.NoError(t, repo.RecordRequestRollupFailure(context.Background(), true))
	state, err := repo.GetRequestRollupState(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, state.DelayedCount, int64(1))
	require.Equal(t, "failed", state.QualityStatus)
}

func intPtrForRollup(value int) *int { return &value }

func int16PtrForRollup(value int16) *int16 { return &value }

func int64PtrForRollup(value int64) *int64 { return &value }
