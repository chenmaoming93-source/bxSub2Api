package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAlignObservabilityRollupRangeExpandsToFullBuckets(t *testing.T) {
	start := time.Date(2026, 11, 12, 3, 7, 30, 0, time.UTC)
	end := time.Date(2026, 11, 12, 3, 8, 0, 0, time.UTC)

	alignedStart, alignedEnd := alignObservabilityRollupRange(start, end, 300)
	require.Equal(t, time.Date(2026, 11, 12, 3, 5, 0, 0, time.UTC), alignedStart)
	require.Equal(t, time.Date(2026, 11, 12, 3, 10, 0, 0, time.UTC), alignedEnd)

	hourStart, hourEnd := alignObservabilityRollupRange(start, end, 3600)
	require.Equal(t, time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC), hourStart)
	require.Equal(t, time.Date(2026, 11, 12, 4, 0, 0, 0, time.UTC), hourEnd)
}

func TestObservabilityRollupSQLUsesWholeSecondBuckets(t *testing.T) {
	start := time.Date(2026, 11, 12, 3, 5, 0, 0, time.UTC)
	end := time.Date(2026, 11, 12, 3, 10, 0, 0, time.UTC)
	query, args := buildObservabilityRollupInsertSQL(300, start, end)

	require.Contains(t, query, "DIV 300) * 300")
	require.Contains(t, query, "'1970-01-01 00:00:00'")
	require.Equal(t, 6, strings.Count(query, "?"))
	require.Len(t, args, 6)
	require.Equal(t, 300, args[0])
}
