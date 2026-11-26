//go:build unit

package growth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdjustmentValidation(t *testing.T) {
	service := NewAdminGrowthService(nil, nil)
	_, err := service.Adjust(t.Context(), AdjustmentCommand{
		OperatorID: 1, UserID: 2, Direction: "INCREASE", Points: 10, Reason: "活动补发",
	})
	require.Error(t, err)
	_, err = service.Adjust(t.Context(), AdjustmentCommand{
		OperatorID: 1, UserID: 2, Direction: "INCREASE", Points: 10, Reason: "短",
	})
	require.Error(t, err)
}

func TestReconciliationReportFields(t *testing.T) {
	raw, err := json.Marshal(ReconciliationReport{})
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	for _, field := range []string{
		"ok", "checked_at", "total_amount", "allocated_amount", "distributed_amount",
		"task_allocated", "task_distributed", "adjustment_in", "issues",
	} {
		require.Contains(t, fields, field)
	}
}
