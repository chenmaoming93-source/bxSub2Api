package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildBucketPlanAsiaShanghaiAndDayBoundary(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	start := time.Date(2026, 11, 10, 15, 0, 0, 0, time.UTC) // 23:00 Shanghai
	plan, err := BuildBucketPlan(start, start.Add(time.Hour), Window1h, Bucket5m, loc)
	require.NoError(t, err)
	require.Len(t, plan, 12)
	require.Equal(t, "23:00", plan[0].Label)
	require.Equal(t, "23:55", plan[len(plan)-1].Label)
}

func TestBuildBucketPlanHandlesDSTChange(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	start := time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC)
	plan, err := BuildBucketPlan(start, start.Add(24*time.Hour), Window24h, Bucket1h, loc)
	require.NoError(t, err)
	require.Len(t, plan, 24)
	require.Equal(t, "00:00", plan[0].Label)
	require.Equal(t, "03:00", plan[2].Label)
}

func TestBuildRangeBucketPlanUsesExactAbsoluteRange(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	start := time.Date(2026, 11, 12, 1, 2, 3, 0, time.UTC)
	end := start.Add(95 * time.Minute)

	plan, err := BuildRangeBucketPlan(start, end, Bucket15m, loc)
	require.NoError(t, err)
	require.Len(t, plan, 7)
	require.Equal(t, start.Truncate(15*time.Minute), plan[0].Start)
	require.False(t, plan[len(plan)-1].End.Before(end))

	repeated, err := BuildRangeBucketPlan(start, end, Bucket15m, loc)
	require.NoError(t, err)
	require.Equal(t, plan, repeated)
}
