package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryGetModelLatencyPercentiles_WithFilters(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	groupID := int64(7)
	filter := &service.OpsDashboardFilter{
		StartTime: start,
		EndTime:   end,
		Platform:  " OpenAI ",
		GroupID:   &groupID,
		Model:     "gpt-5",
	}

	rows := sqlmock.NewRows([]string{"model", "request_count", "avg_ms", "p50_ms", "p90_ms", "p95_ms", "p99_ms"}).
		AddRow("gpt-5", int64(10), int64(125), int64(100), int64(200), int64(230), int64(250))
	mock.ExpectQuery(`WITH filtered AS \(`).
		WithArgs(start, end, groupID, "openai", "gpt-5").
		WillReturnRows(rows)

	response, err := repo.GetModelLatencyPercentiles(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, "OpenAI", response.Platform)
	require.Equal(t, "gpt-5", response.Model)
	require.Len(t, response.Models, 1)
	require.Equal(t, int64(10), response.Models[0].RequestCount)
	require.Equal(t, 125, *response.Models[0].AvgMS)
	require.Equal(t, 250, *response.Models[0].P99MS)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetModelLatencyPercentiles_EmptyResult(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`WITH filtered AS \(`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"model", "request_count", "avg_ms", "p50_ms", "p90_ms", "p95_ms", "p99_ms"}))

	response, err := repo.GetModelLatencyPercentiles(context.Background(), &service.OpsDashboardFilter{StartTime: start, EndTime: end})
	require.NoError(t, err)
	require.NotNil(t, response.Models)
	require.Empty(t, response.Models)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetModelLatencyTrend(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(6 * time.Hour)
	bucket := start.Add(5 * time.Minute)

	rows := sqlmock.NewRows([]string{"bucket_start", "model", "request_count", "avg_ms", "p50_ms", "p90_ms", "p95_ms", "p99_ms"}).
		AddRow(bucket, "gpt-5", int64(2), int64(150), int64(100), int64(200), int64(200), int64(200))
	mock.ExpectQuery(`PARTITION BY bucket_start, model`).
		WithArgs(start, end).
		WillReturnRows(rows)

	response, err := repo.GetModelLatencyTrend(context.Background(), &service.OpsDashboardFilter{StartTime: start, EndTime: end}, 300)
	require.NoError(t, err)
	require.Equal(t, "5m", response.Bucket)
	require.Len(t, response.Points, 1)
	require.Equal(t, bucket, response.Points[0].BucketStart)
	require.Equal(t, int64(2), response.Points[0].RequestCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetModelLatencyTrend_RejectsUnsupportedBucket(t *testing.T) {
	db, _ := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	_, err := repo.GetModelLatencyTrend(context.Background(), &service.OpsDashboardFilter{StartTime: start, EndTime: start.Add(time.Hour)}, 30)
	require.ErrorContains(t, err, "unsupported bucket")
}
