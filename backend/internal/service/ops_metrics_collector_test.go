package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWriteOpenAIFastPolicyBlockedResponseMarksBusinessLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeOpenAIFastPolicyBlockedResponse(c, &OpenAIFastBlockedError{Message: "custom fast policy block"})

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.True(t, HasOpsClientBusinessLimited(c))
	reason, ok := c.Get(OpsClientBusinessLimitedReasonKey)
	require.True(t, ok)
	require.Equal(t, OpsClientBusinessLimitedReasonLocalPolicyDenied, reason)
}

func TestOpsMetricsCollectorQueryErrorCountsExcludesCountTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`(?s)SUM\(CASE WHEN.*FROM ops_error_logs\s+WHERE created_at >= \? AND created_at < \?\s+AND is_count_tokens = FALSE`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total",
			"business_limited",
			"error_sla",
			"upstream_excl",
			"upstream_429",
			"upstream_529",
		}).AddRow(int64(5), int64(2), int64(3), int64(1), int64(1), int64(1)))

	errorTotal, businessLimited, errorSLA, upstreamExcl429529, upstream429, upstream529, err := collector.queryErrorCounts(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(5), errorTotal)
	require.Equal(t, int64(2), businessLimited)
	require.Equal(t, int64(3), errorSLA)
	require.Equal(t, int64(1), upstreamExcl429529)
	require.Equal(t, int64(1), upstream429)
	require.Equal(t, int64(1), upstream529)
	require.NoError(t, mock.ExpectationsWereMet())
	mock.ExpectClose()
	require.NoError(t, db.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsMetricsCollectorQueryUsageLatencyUsesMySQLWindowFunctions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)

	latencyRows := sqlmock.NewRows([]string{"p50", "p90", "p95", "p99", "avg_ms", "max_ms"}).
		AddRow(float64(100), float64(200), float64(300), float64(400), float64(123.45), int64(500))
	ttftRows := sqlmock.NewRows([]string{"p50", "p90", "p95", "p99", "avg_ms", "max_ms"}).
		AddRow(float64(10), float64(20), float64(30), float64(40), float64(12.34), int64(50))

	mock.ExpectQuery(regexp.QuoteMeta("WITH ordered AS")).
		WithArgs(start, end).
		WillReturnRows(latencyRows)
	mock.ExpectQuery(regexp.QuoteMeta("WITH ordered AS")).
		WithArgs(start, end).
		WillReturnRows(ttftRows)

	duration, ttft, err := collector.queryUsageLatency(context.Background(), start, end)
	require.NoError(t, err)
	require.NotNil(t, duration.p50)
	require.Equal(t, 100, *duration.p50)
	require.NotNil(t, duration.avg)
	require.Equal(t, 123.5, *duration.avg)
	require.NotNil(t, ttft.p99)
	require.Equal(t, 40, *ttft.p99)
	require.NoError(t, mock.ExpectationsWereMet())
	mock.ExpectClose()
	require.NoError(t, db.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}
