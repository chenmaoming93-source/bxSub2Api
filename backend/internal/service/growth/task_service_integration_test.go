//go:build integration

package growth

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/growthadminaudit"
	"github.com/stretchr/testify/require"
)

func TestTaskLifecycleWithQuota(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 1000, Reason: "任务活动总预算", RequestID: "task-topup",
	})
	require.NoError(t, err)

	rule := TaskRuleInput{
		Target:       1,
		RewardPoints: 10,
		PeriodType:   "DAILY",
		DailyLimit:   1,
		PerUserLimit: 1,
	}
	_, err = tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "zero_quota",
		Title:             "零额度任务",
		Category:          "DAILY",
		TaskType:          "DAILY",
		InitialAllocation: 0,
		Rule:              rule,
	})
	require.Error(t, err)
	_, err = tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "insufficient_quota",
		Title:             "额度不足任务",
		Category:          "DAILY",
		TaskType:          "DAILY",
		InitialAllocation: 2000,
		Rule:              rule,
	})
	require.ErrorIs(t, err, ErrBudgetInsufficient)
	taskCount, err := client.GrowthTask.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, taskCount)

	created, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "daily_checkin_test",
		Title:             "每日签到",
		Category:          "CHECKIN",
		TaskType:          "CHECKIN",
		InitialAllocation: 200,
		Rule:              rule,
		RequestID:         "task-create",
	})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", created.Status)
	require.Equal(t, int64(200), created.AllocatedPoints)
	require.Equal(t, 1, *created.PendingRuleVersion)
	require.Equal(t, int64(200), mustBudget(t, budget).AllocatedAmount)

	_, err = tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "daily_checkin_test",
		Title:             "重复任务",
		Category:          "CHECKIN",
		TaskType:          "CHECKIN",
		InitialAllocation: 100,
		Rule:              rule,
		RequestID:         "task-duplicate",
	})
	require.ErrorIs(t, err, ErrTaskCodeDuplicated)
	require.Equal(t, int64(200), mustBudget(t, budget).AllocatedAmount)

	published, err := tasks.Publish(ctx, created.TaskID, 1, nil, "publish-v1")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", published.Status)
	require.Equal(t, 1, *published.RuleVersion)
	require.Nil(t, published.PendingRuleVersion)

	rule.RewardPoints = 20
	updated, err := tasks.Update(ctx, created.TaskID, TaskUpdateInput{
		OperatorID: 1,
		Title:      "每日签到新版",
		Category:   "CHECKIN",
		TaskType:   "CHECKIN",
		Rule:       rule,
		RequestID:  "task-update-v2",
	})
	require.NoError(t, err)
	require.Equal(t, 1, *updated.RuleVersion)
	require.Equal(t, 2, *updated.PendingRuleVersion)
	require.Equal(t, int64(10), updated.RewardPoints)

	publishedV2, err := tasks.Publish(ctx, created.TaskID, 1, nil, "publish-v2")
	require.NoError(t, err)
	require.Equal(t, 2, *publishedV2.RuleVersion)
	require.Equal(t, int64(20), publishedV2.RewardPoints)
	require.Nil(t, publishedV2.PendingRuleVersion)
	require.Equal(t, "daily_checkin_test", publishedV2.TaskCode)

	versions, err := tasks.RuleVersions(ctx, created.TaskID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 2, versions[0].Version)
	require.NotNil(t, versions[1].EffectiveTo)
	require.Equal(t, int64(10), versions[1].RewardPoints)

	offline, err := tasks.Offline(ctx, created.TaskID, 1, "活动结束后下线", true, "offline-1")
	require.NoError(t, err)
	require.Equal(t, "OFFLINE", offline.Status)
	require.Zero(t, offline.AllocatedPoints)
	require.Zero(t, mustBudget(t, budget).AllocatedAmount)
	offlineAudit, err := client.GrowthAdminAudit.Query().
		Where(growthadminaudit.ActionEQ("task.offline")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "活动结束后下线", offlineAudit.Reason)

	rule.RewardPoints = 30
	updatedV3, err := tasks.Update(ctx, created.TaskID, TaskUpdateInput{
		OperatorID: 1,
		Title:      "重新上线版本",
		Category:   "CHECKIN",
		TaskType:   "CHECKIN",
		Rule:       rule,
		RequestID:  "task-update-v3",
	})
	require.NoError(t, err)
	require.Equal(t, 3, *updatedV3.PendingRuleVersion)
	publishedV3, err := tasks.Publish(ctx, created.TaskID, 1, nil, "publish-v3")
	require.NoError(t, err)
	require.Equal(t, "OFFLINE", publishedV3.Status)
	require.Equal(t, 3, *publishedV3.RuleVersion)

	online, err := tasks.Online(ctx, created.TaskID, 1, 100, "online-1")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", online.Status)
	require.Equal(t, int64(100), online.AllocatedPoints)
	require.Equal(t, int64(100), mustBudget(t, budget).AllocatedAmount)

	listed, total, err := tasks.List(ctx, BudgetListFilter{Page: 1, PageSize: 20}, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, listed, 1)
	require.Equal(t, "ACTIVE", listed[0].Status)
	require.Equal(t, 3, *listed[0].RuleVersion)
}

func mustBudget(t *testing.T, service *BudgetService) BudgetAccount {
	t.Helper()
	account, err := service.GetAccount(context.Background())
	require.NoError(t, err)
	return account
}
