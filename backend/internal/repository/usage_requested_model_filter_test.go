package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsageModelFilterUsesRequestedModelWithTrimmedHistoricalFallback(t *testing.T) {
	conditions, args := appendUsageLogRequestedModelWhereCondition(nil, nil, "coding-alias")
	require.Equal(t, []any{"coding-alias"}, args)
	require.Len(t, conditions, 1)
	require.Contains(t, conditions[0], usageLogRequestedModelExpr)
	require.Contains(t, conditions[0], "= ?")
	require.NotContains(t, conditions[0], "upstream_model")

	query, queryArgs := appendUsageLogRequestedModelQueryFilter("SELECT * FROM usage_logs WHERE 1=1", nil, "coding-alias")
	require.Contains(t, query, "COALESCE(NULLIF(TRIM(requested_model), ''), model) = ?")
	require.Equal(t, []any{"coding-alias"}, queryArgs)
}

func TestUsageModelFilterIgnoresBlankInput(t *testing.T) {
	query, args := appendUsageLogRequestedModelQueryFilter("base", []any{1}, "  ")
	require.Equal(t, "base", query)
	require.Equal(t, []any{1}, args)
	require.False(t, strings.Contains(query, usageLogRequestedModelExpr))
}
