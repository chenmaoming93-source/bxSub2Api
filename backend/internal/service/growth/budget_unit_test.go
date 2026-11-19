//go:build unit

package growth

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"

	"github.com/stretchr/testify/require"
)

func TestBudgetAccountDerivedFields(t *testing.T) {
	updated := time.Now()
	account := accountFromValues(120000, 78000, 42150, 4, updated)
	require.Equal(t, int64(42000), account.UnallocatedAmount)
	require.InDelta(t, 0.35125, account.DistributedRate, 0.000001)
	require.Equal(t, int64(4), account.Version)
	require.Equal(t, updated, account.UpdatedAt)
}

func TestValidateBudgetMutation(t *testing.T) {
	valid := BudgetMutation{OperatorID: 1, Points: 10, Reason: "活动预算追加", RequestID: "req-1"}
	require.NoError(t, validateBudgetMutation(valid))

	tests := []struct {
		name     string
		mutation BudgetMutation
	}{
		{name: "missing operator", mutation: BudgetMutation{Points: 1, Reason: "活动预算追加"}},
		{name: "zero points", mutation: BudgetMutation{OperatorID: 1, Reason: "活动预算追加"}},
		{name: "short reason", mutation: BudgetMutation{OperatorID: 1, Points: 1, Reason: "短"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, validateBudgetMutation(test.mutation))
		})
	}
}

func TestTaskAllocationUsage(t *testing.T) {
	task := &ent.GrowthTask{
		ID:                7,
		TaskCode:          "daily_checkin",
		Title:             "每日签到",
		TaskStatus:        "ACTIVE",
		AllocatedPoints:   200,
		DistributedPoints: 50,
		UpdatedAt:         time.Now(),
	}
	allocation := taskAllocationFromEntity(task)
	require.Equal(t, int64(150), allocation.RemainingPoints)
	require.InDelta(t, 0.25, allocation.UsageRate, 0.000001)
}
