//go:build integration

package growth

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthaccountledger"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"
	"github.com/stretchr/testify/require"
)

func TestDoubleEntryTransferConcurrencyRollbackAndReplay(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	transfers := NewTransferService(client)
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 1000, Reason: "发分测试预算", RequestID: "transfer-topup",
	})
	require.NoError(t, err)

	rule := TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 1, PerUserLimit: 1}
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "transfer_task",
		Title:             "发分任务",
		Category:          "DAILY",
		TaskType:          "DAILY",
		InitialAllocation: 100,
		Rule:              rule,
		RequestID:         "transfer-task-create",
	})
	require.NoError(t, err)

	award, err := transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.TaskID, Points: 30,
		BizType: "CHECKIN", IdempotencyKey: "award-1", Title: "每日签到", ReferenceID: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(30), award.CurrentBalance)
	require.Equal(t, int64(30), mustBudget(t, budget).DistributedAmount)

	replay, err := transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.TaskID, Points: 30,
		BizType: "CHECKIN", IdempotencyKey: "award-1", Title: "每日签到", ReferenceID: 1,
	})
	require.NoError(t, err)
	require.Equal(t, award.TransferID, replay.TransferID)
	require.True(t, replay.Replayed)
	require.Equal(t, int64(30), replay.CurrentBalance)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
		failed  int
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := transfers.AwardFromTask(ctx, TaskAwardCommand{
				UserID: 1, TaskID: task.TaskID, Points: 70,
				BizType: "TASK", IdempotencyKey: "concurrent-award-" + string(rune('a'+index)),
				Title: "并发任务奖励", ReferenceID: int64(index + 1),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.ErrorIs(t, err, ErrPoolExhausted)
				failed++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, success)
	require.Equal(t, 1, failed)

	debit, err := transfers.DebitWallet(ctx, WalletDebitCommand{
		UserID: 1, Points: 40, BizType: "EXCHANGE", IdempotencyKey: "exchange-1",
		Title: "兑换商品", ReferenceID: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(60), debit.CurrentBalance)

	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := transfers.DebitWallet(ctx, WalletDebitCommand{
				UserID: 1, Points: 50, BizType: "EXCHANGE",
				IdempotencyKey: "concurrent-debit-" + string(rune('a'+index)),
				Title:          "并发兑换", ReferenceID: int64(index + 10),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.ErrorIs(t, err, ErrInsufficientPoints)
				failed++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 2, success)
	require.Equal(t, 2, failed)

	refund, err := transfers.ReverseRedeem(ctx, RedeemReverseCommand{
		UserID: 1, Points: 40, BizType: "REFUND", IdempotencyKey: "refund-1",
		Title: "兑换失败退款", ReferenceID: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(50), refund.CurrentBalance)

	task2, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID:        1,
		TaskCode:          "rollback_task",
		Title:             "回滚任务",
		Category:          "DAILY",
		TaskType:          "DAILY",
		InitialAllocation: 50,
		Rule:              rule,
		RequestID:         "rollback-task-create",
	})
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task2.TaskID, Points: 20,
		BizType: "TASK", IdempotencyKey: "rollback-award", Title: "应回滚奖励",
		AfterLedger: func(context.Context, *ent.Tx) error {
			return errors.New("injected post-ledger failure")
		},
	})
	require.Error(t, err)

	task2Entity, err := client.GrowthTask.Get(ctx, task2.TaskID)
	require.NoError(t, err)
	require.Zero(t, task2Entity.DistributedPoints)
	require.Equal(t, int64(100), mustBudget(t, budget).DistributedAmount)

	wallet, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	ledgerEntries, err := client.GrowthAccountLedger.Query().
		Where(growthaccountledger.UserIDEQ(1)).
		All(ctx)
	require.NoError(t, err)
	require.Equal(t, wallet.AvailablePoints, LedgerSignedSum(ledgerEntries, "WALLET"))

	transferEntities, err := client.GrowthTransfer.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, transferEntities, 5)
	for _, transfer := range transferEntities {
		entries, err := client.GrowthAccountLedger.Query().
			Where(growthaccountledger.TransferIDEQ(transfer.ID)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, entries, 2)
		var net int64
		for _, entry := range entries {
			if entry.Direction == "IN" {
				net += entry.Points
			} else {
				net -= entry.Points
			}
		}
		require.Zero(t, net)
	}
}
