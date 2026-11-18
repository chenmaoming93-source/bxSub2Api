//go:build integration

package growth

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/growthadminaudit"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"

	"github.com/stretchr/testify/require"
)

func TestAdjustmentAuditAndReconciliation(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	transfers := NewTransferService(client)
	admin := NewAdminGrowthService(client, transfers)
	reads := NewReadService(client, nil)
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "调账测试预算", RequestID: "adjustment-topup",
	})
	require.NoError(t, err)
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "adjustment_fund", Title: "调账资金", Category: "ACHIEVEMENT",
		TaskType: "ACHIEVEMENT", InitialAllocation: 50,
		Rule: TaskRuleInput{Target: 1, RewardPoints: 20, PeriodType: "ONCE"}, RequestID: "adjustment-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task.TaskID, 1, nil, "adjustment-publish")
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.TaskID, Points: 20, BizType: "TASK",
		IdempotencyKey: "adjustment-award", Title: "调账资金",
	})
	require.NoError(t, err)

	increased, err := admin.Adjust(ctx, AdjustmentCommand{
		OperatorID: 1, UserID: 1, Direction: "INCREASE", Points: 30,
		Reason: "线下活动补发积分", RequestID: "adjust-in",
	})
	require.NoError(t, err)
	require.Equal(t, int64(20), increased.BeforeBalance)
	require.Equal(t, int64(50), increased.AfterBalance)

	decreased, err := admin.Adjust(ctx, AdjustmentCommand{
		OperatorID: 1, UserID: 1, Direction: "DECREASE", Points: 10,
		Reason: "纠正重复补发积分", RequestID: "adjust-out",
	})
	require.NoError(t, err)
	require.Equal(t, int64(40), decreased.AfterBalance)

	_, err = admin.Adjust(ctx, AdjustmentCommand{
		OperatorID: 1, UserID: 1, Direction: "DECREASE", Points: 50,
		Reason: "扣减超过当前余额", RequestID: "adjust-overdraw",
	})
	require.True(t, errors.Is(err, ErrInsufficientPoints))
	_, err = admin.Adjust(ctx, AdjustmentCommand{
		OperatorID: 1, UserID: 1, Direction: "INCREASE", Points: 1,
		Reason: "短", RequestID: "adjust-short",
	})
	require.Error(t, err)

	_, err = client.ExecContext(ctx,
		"INSERT INTO users (id, email, password_hash, username) VALUES (2, 'no-wallet@example.com', 'x', 'no-wallet')",
	)
	require.NoError(t, err)
	users, total, err := admin.ListUsers(ctx, 1, 20, "", 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, users, 2)
	require.Equal(t, int64(40), users[0].AvailablePoints)
	require.Equal(t, int64(50), users[0].TotalEarned)
	require.Equal(t, int64(10), users[0].TotalSpent)
	require.Equal(t, int64(2), users[1].UserID)
	require.Zero(t, users[1].AvailablePoints)
	require.Zero(t, users[1].TotalEarned)
	require.Zero(t, users[1].TotalSpent)

	ledger, err := reads.Ledger(ctx, 1, 0, 20, "")
	require.NoError(t, err)
	require.Len(t, ledger.Items, 3)
	require.NotNil(t, ledger.Items[0].BalanceAfter)
	require.Equal(t, int64(40), *ledger.Items[0].BalanceAfter)

	adjustTransfers, err := client.GrowthTransfer.Query().
		Where(growthtransfer.BizTypeEQ("ADJUST")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, adjustTransfers)
	auditCount, err := client.GrowthAdminAudit.Query().
		Where(growthadminaudit.ActionContains("adjust")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, auditCount)

	report, err := admin.Reconcile(ctx)
	require.NoError(t, err)
	require.True(t, report.OK, "issues: %+v", report.Issues)
	require.Equal(t, int64(30), report.AdjustmentIn)

	wallet, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	_, err = client.GrowthWallet.UpdateOneID(wallet.ID).SetAvailablePoints(41).Save(ctx)
	require.NoError(t, err)
	broken, err := admin.Reconcile(ctx)
	require.NoError(t, err)
	require.False(t, broken.OK)
	require.NotEmpty(t, broken.Issues)
	walletAfter, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(41), walletAfter.AvailablePoints)

	logs, logTotal, err := admin.AuditLogs(ctx, 1, 20, "")
	require.NoError(t, err)
	require.GreaterOrEqual(t, logTotal, int64(2))
	require.NotEmpty(t, logs)
}
