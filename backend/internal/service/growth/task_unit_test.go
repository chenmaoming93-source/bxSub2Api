//go:build unit

package growth

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"

	"github.com/stretchr/testify/require"
)

func TestValidateTaskCreate(t *testing.T) {
	validRule := TaskRuleInput{
		Target:       1,
		RewardPoints: 10,
		PeriodType:   "DAILY",
		DailyLimit:   1,
		PerUserLimit: 1,
	}
	valid := TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "daily_checkin",
		Title:             "每日签到",
		Category:          "CHECKIN",
		TaskType:          "CHECKIN",
		InitialAllocation: 100,
		Rule:              validRule,
	}
	require.NoError(t, validateTaskCreate(valid))

	tests := []struct {
		name   string
		mutate func(*TaskCreateInput)
	}{
		{name: "missing code", mutate: func(input *TaskCreateInput) { input.TaskCode = "" }},
		{name: "invalid category", mutate: func(input *TaskCreateInput) { input.Category = "OTHER" }},
		{name: "zero allocation", mutate: func(input *TaskCreateInput) { input.InitialAllocation = 0 }},
		{name: "invalid rule", mutate: func(input *TaskCreateInput) { input.Rule.RewardPoints = 0 }},
		{name: "invalid task code characters", mutate: func(input *TaskCreateInput) { input.TaskCode = "任务 编码" }},
		{name: "title too long", mutate: func(input *TaskCreateInput) { input.Title = string(make([]rune, 256)) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			require.Error(t, validateTaskCreate(input))
		})
	}
}

func TestValidateTaskCreateAllowsMultipleRewards(t *testing.T) {
	input := TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "multiple_daily_rewards",
		Title:             "每日多次奖励",
		Category:          "DAILY",
		TaskType:          "DAILY",
		InitialAllocation: 100,
		Rule: TaskRuleInput{
			Target:       1,
			RewardPoints: 10,
			PeriodType:   "DAILY",
			DailyLimit:   3,
			PerUserLimit: 5,
		},
	}

	require.NoError(t, validateTaskCreate(input))
}

func TestBuildTaskViewExposesPublishedAndPendingVersions(t *testing.T) {
	task := &ent.GrowthTask{
		ID:                3,
		TaskCode:          "first_call",
		Title:             "首次调用",
		TaskStatus:        "ACTIVE",
		AllocatedPoints:   200,
		DistributedPoints: 40,
	}
	current := &ent.GrowthTaskRule{
		ID:           10,
		TaskID:       3,
		Version:      2,
		RewardPoints: 20,
		PeriodType:   "ONCE",
		PerUserLimit: 1,
	}
	pending := &ent.GrowthTaskRule{ID: 11, TaskID: 3, Version: 3}

	view := buildTaskView(task, current, pending)
	require.Equal(t, int64(160), view.RemainingPoints)
	require.Equal(t, 2, *view.RuleVersion)
	require.Equal(t, 3, *view.PendingRuleVersion)
	require.Equal(t, int64(20), view.RewardPoints)
}
