//go:build integration

package growth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthcompletionrequest"
	"github.com/Wei-Shaw/sub2api/ent/growthrewardclaim"
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
	reads := NewReadService(client, &config.Config{Timezone: "Asia/Shanghai"})
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 200, Reason: "任务完成测试预算", RequestID: "completion-topup",
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
	require.Equal(t, "NOT_ELIGIBLE", differentRef.Result)
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
		InitialAllocation: 30,
		Rule:              TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 2, PerUserLimit: 2},
		RequestID:         "completion-task2",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task2.TaskID, 1, nil, "completion-publish2")
	require.NoError(t, err)
	dayTwo := dayOne.Add(24 * time.Hour)
	completions.now = func() time.Time { return dayTwo }

	var (
		wg      sync.WaitGroup
		results = make(chan CompletionResult, 2)
		errs    = make(chan error, 2)
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := completions.Complete(ctx, CompletionCommand{
				UserID: 1, TaskID: task2.TaskID, CompletionRef: "concurrent-ref-" + string(rune('a'+index)),
				RequestID: "concurrent-cr", IdempotencyKey: "concurrent-key-" + string(rune('a'+index)),
			})
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}(index)
	}
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		require.NoError(t, err)
	}
	for result := range results {
		require.Equal(t, "AWARDED", result.Result)
	}

	dailyExhausted, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task2.TaskID, CompletionRef: "concurrent-ref-c",
		RequestID: "concurrent-cr-c", IdempotencyKey: "concurrent-key-c",
	})
	require.NoError(t, err)
	require.Equal(t, "NOT_ELIGIBLE", dailyExhausted.Result)

	task4, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "complete_multiple", Title: "多次任务", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 60,
		Rule:              TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 3, PerUserLimit: 5},
		RequestID:         "completion-task4",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task4.TaskID, 1, nil, "completion-publish4")
	require.NoError(t, err)
	completions.now = func() time.Time { return dayOne.Add(48 * time.Hour) }
	for index := 1; index <= 3; index++ {
		result, err := completions.Complete(ctx, CompletionCommand{
			UserID: 1, TaskID: task4.TaskID, CompletionRef: fmt.Sprintf("multi-ref-%d", index),
			RequestID: fmt.Sprintf("multi-cr-%d", index), IdempotencyKey: fmt.Sprintf("multi-key-%d", index),
		})
		require.NoError(t, err)
		require.Equal(t, "AWARDED", result.Result)
	}
	reads.now = completions.now
	multiTaskItems, _, err := reads.UTasks(ctx, 1, 1, 20, "")
	require.NoError(t, err)
	var multiTaskItem *UTaskItemView
	for index := range multiTaskItems {
		if multiTaskItems[index].TaskID == task4.TaskID {
			multiTaskItem = &multiTaskItems[index]
			break
		}
	}
	require.NotNil(t, multiTaskItem)
	require.Equal(t, "REWARDED", multiTaskItem.Status)
	require.Equal(t, int64(3), multiTaskItem.MyCompletedCount)
	dailyExhausted, err = completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task4.TaskID, CompletionRef: "multi-ref-4",
		RequestID: "multi-cr-4", IdempotencyKey: "multi-key-4",
	})
	require.NoError(t, err)
	require.Equal(t, "NOT_ELIGIBLE", dailyExhausted.Result)
	for day := 3; day <= 4; day++ {
		completions.now = func() time.Time { return dayOne.Add(time.Duration(day) * 24 * time.Hour) }
		result, err := completions.Complete(ctx, CompletionCommand{
			UserID: 1, TaskID: task4.TaskID, CompletionRef: fmt.Sprintf("multi-ref-day-%d", day),
			RequestID: fmt.Sprintf("multi-cr-day-%d", day), IdempotencyKey: fmt.Sprintf("multi-key-day-%d", day),
		})
		require.NoError(t, err)
		require.Equal(t, "AWARDED", result.Result)
	}
	completions.now = func() time.Time { return dayOne.Add(5 * 24 * time.Hour) }
	perUserExhausted, err := completions.Complete(ctx, CompletionCommand{
		UserID: 1, TaskID: task4.TaskID, CompletionRef: "multi-ref-6",
		RequestID: "multi-cr-6", IdempotencyKey: "multi-key-6",
	})
	require.NoError(t, err)
	require.Equal(t, "NOT_ELIGIBLE", perUserExhausted.Result)

	multiClaims, err := client.GrowthRewardClaim.Query().
		Where(growthrewardclaim.TaskIDEQ(task4.TaskID)).
		Order(ent.Asc(growthrewardclaim.FieldSequence)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, multiClaims, 5)
	sequencesByCycle := make(map[string][]int)
	for _, claim := range multiClaims {
		sequencesByCycle[claim.CycleKey] = append(sequencesByCycle[claim.CycleKey], claim.Sequence)
	}
	require.Len(t, sequencesByCycle, 3)
	require.ElementsMatch(t, []int{1, 2, 3}, sequencesByCycle[dayOne.Add(48*time.Hour).Format("2006-01-02")])
	require.Equal(t, []int{1}, sequencesByCycle[dayOne.Add(72*time.Hour).Format("2006-01-02")])
	require.Equal(t, []int{1}, sequencesByCycle[dayOne.Add(96*time.Hour).Format("2006-01-02")])

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
	require.Equal(t, 8, transferCount)
	ledgerCount, err := client.GrowthAccountLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 16, ledgerCount)
	requestCount, err := client.GrowthCompletionRequest.Query().
		Where(growthcompletionrequest.TaskIDEQ(task.TaskID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, requestCount)
}
