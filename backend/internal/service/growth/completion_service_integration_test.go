//go:build integration

package growth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/growthcompletionrequest"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/stretchr/testify/require"
)

func TestCompletionAwardDedupeLimitsConcurrencyAndQuota(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	transfers := NewTransferService(client)
	completions := NewCompletionService(client, transfers, &config.Config{Timezone: "Asia/Shanghai"})
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "任务完成测试预算", RequestID: "completion-topup",
	})
	require.NoError(t, err)
	rule := TaskRuleInput{
		Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 1, PerUserLimit: 1,
	}
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "complete_once", Title: "完成一次", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 20, Rule: rule, RequestID: "completion-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task.TaskID, 1, nil, "completion-publish")
	require.NoError(t, err)

	dayOne := time.Now().In(time.FixedZone("CST", 8*60*60))
	completions.now = func() time.Time { return dayOne }

	first, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task.TaskID, CompletionRef: "ref-1",
		RequestID: "cr-1", IdempotencyKey: "completion-key-1",
	})
	require.NoError(t, err)
	require.Equal(t, "AWARDED", first.Result)
	require.NotNil(t, first.TransferID)
	require.Equal(t, int64(10), first.AwardedPoints)
	require.Equal(t, int64(10), *first.CurrentBalance)

	sameKey, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task.TaskID, CompletionRef: "ref-1",
		RequestID: "cr-1", IdempotencyKey: "completion-key-1",
	})
	require.NoError(t, err)
	require.Equal(t, "AWARDED", sameKey.Result)
	require.Equal(t, *first.TransferID, *sameKey.TransferID)

	differentKey, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task.TaskID, CompletionRef: "ref-1",
		RequestID: "cr-2", IdempotencyKey: "completion-key-2",
	})
	require.NoError(t, err)
	require.Equal(t, "ALREADY_COMPLETED", differentKey.Result)
	require.Equal(t, int64(10), differentKey.AwardedPoints)

	differentRef, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task.TaskID, CompletionRef: "ref-2",
		RequestID: "cr-3", IdempotencyKey: "completion-key-3",
	})
	require.NoError(t, err)
	require.Equal(t, "ALREADY_COMPLETED", differentRef.Result)
	require.Zero(t, differentRef.AwardedPoints)

	completions.now = func() time.Time { return dayOne.Add(24 * time.Hour) }
	notEligible, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task.TaskID, CompletionRef: "ref-day2",
		RequestID: "cr-day2", IdempotencyKey: "completion-day2",
	})
	require.NoError(t, err)
	require.Equal(t, "NOT_ELIGIBLE", notEligible.Result)

	task2, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "complete_concurrent", Title: "并发任务", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 20,
		Rule:              TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY"},
		RequestID:         "completion-task2",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task2.TaskID, 1, nil, "completion-publish2")
	require.NoError(t, err)
	dayTwo := dayOne.Add(24 * time.Hour)
	completions.now = func() time.Time { return dayTwo }

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		awarded int
		already int
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := completions.Complete(ctx, CompletionCommand{
				UserID: 1, TaskID: task2.TaskID, CompletionRef: "concurrent-ref-" + string(rune('a'+index)),
				RequestID: "concurrent-cr", IdempotencyKey: "concurrent-key-" + string(rune('a'+index)),
			})
			require.NoError(t, err)
			mu.Lock()
			defer mu.Unlock()
			switch result.Result {
			case "AWARDED":
				awarded++
			case "ALREADY_COMPLETED":
				already++
			default:
				t.Fatalf("unexpected completion result %s", result.Result)
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, awarded)
	require.Equal(t, 1, already)

	task3, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "complete_exhausted", Title: "额度耗尽", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 5,
		Rule:              TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "ONCE"},
		RequestID:         "completion-task3",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task3.TaskID, 1, nil, "completion-publish3")
	require.NoError(t, err)
	_, err = completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task3.TaskID, CompletionRef: "ref-no-quota",
		RequestID: "cr-no-quota", IdempotencyKey: "completion-no-quota",
	})
	require.True(t, errors.Is(err, ErrPoolExhausted))

	transferCount, err := client.GrowthTransfer.Query().Where(growthtransfer.BizTypeEQ("TASK")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, transferCount)
	ledgerCount, err := client.GrowthAccountLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, ledgerCount)
	requestCount, err := client.GrowthCompletionRequest.Query().
		Where(growthcompletionrequest.TaskIDEQ(task.TaskID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, requestCount)
}
