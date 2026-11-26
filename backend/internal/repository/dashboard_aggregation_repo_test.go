package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestDashboardAggregationUsesFixedSQLOffset(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { _ = timezone.Init("UTC") })

	exec := &recordingSQLExecutor{result: rowsAffectedResult(1)}
	repo := newDashboardAggregationRepositoryWithSQL(exec)
	start := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	require.NoError(t, repo.aggregateRangeInTx(context.Background(), start, end, start, end))
	require.Len(t, exec.execArgs, 4)
	for index, args := range exec.execArgs {
		require.NotEmptyf(t, args, "query %d has no arguments", index)
		require.Equalf(t, "+08:00", args[0], "query %d must use a fixed SQL offset", index)
		require.NotContainsf(t, args, "Asia/Shanghai", "query %d must not pass a named timezone", index)
	}
}
