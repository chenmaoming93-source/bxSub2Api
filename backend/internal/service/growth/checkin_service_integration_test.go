//go:build integration

package growth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/growthrewardclaim"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/stretchr/testify/require"
)

func TestCheckinAwardReplayConcurrencyAndQuotaExhaustion(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	transfers := NewTransferService(client)
	checkins := NewCheckinService(client, transfers, &config.Config{Timezone: "Asia/Shanghai"})
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "签到测试预算", RequestID: "checkin-topup",
	})
	require.NoError(t, err)
	rule := TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 1, PerUserLimit: 1}
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "daily_checkin", Title: "每日签到", Category: "CHECKIN", TaskType: "CHECKIN",
		InitialAllocation: 20, Rule: rule, RequestID: "checkin-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task.TaskID, 1, nil, "checkin-publish")
	require.NoError(t, err)

	location := time.FixedZone("CST", 8*60*60)
	dayOne := time.Now().In(location)
	checkins.now = func() time.Time { return dayOne }
	first, err := checkins.Checkin(ctx, 1, "checkin-key-1")
	require.NoError(t, err)
	require.Equal(t, "AWARDED", first.Result)
	require.NotZero(t, first.ClaimID)
	require.NotNil(t, first.TransferID)
	require.Equal(t, int64(10), first.AwardedPoints)
	require.NotNil(t, first.CurrentBalance)
	require.Equal(t, int64(10), *first.CurrentBalance)

	replay, err := checkins.Checkin(ctx, 1, "checkin-key-1")
	require.NoError(t, err)
	require.Equal(t, "AWARDED", replay.Result)
	require.Equal(t, *first.TransferID, *replay.TransferID)
	repeated, err := checkins.Checkin(ctx, 1, "checkin-key-different")
	require.NoError(t, err)
	require.Equal(t, "ALREADY_CLAIMED", repeated.Result)
	require.Zero(t, repeated.AwardedPoints)

	checkins.now = func() time.Time { return dayOne.Add(24 * time.Hour) }
	var (
		wg      sync.WaitGroup
		results = make(chan CheckinResult, 2)
		errs    = make(chan error, 2)
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := checkins.Checkin(ctx, 1, "checkin-day2-"+string(rune('a'+index)))
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
	awarded := 0
	notAwarded := 0
	var balances []int64
	for result := range results {
		switch result.Result {
		case "AWARDED":
			awarded++
			if result.CurrentBalance != nil {
				balances = append(balances, *result.CurrentBalance)
			}
		case "ALREADY_CLAIMED", "UNAVAILABLE":
			notAwarded++
		default:
			t.Fatalf("unexpected concurrent result %s", result.Result)
		}
	}
	require.Equal(t, 1, awarded)
	require.Equal(t, 1, notAwarded)
	require.Contains(t, balances, int64(20))

	claimCount, err := client.GrowthRewardClaim.Query().Where(growthrewardclaim.TaskIDEQ(task.TaskID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, claimCount)
	transferCount, err := client.GrowthTransfer.Query().Where(growthtransfer.BizTypeEQ("CHECKIN")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, transferCount)
	ledgerCount, err := client.GrowthAccountLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, ledgerCount)

	taskEntity, err := client.GrowthTask.Get(ctx, task.TaskID)
	require.NoError(t, err)
	require.Equal(t, int64(20), taskEntity.DistributedPoints)
	require.Equal(t, int64(20), taskEntity.AllocatedPoints)

	checkins.now = func() time.Time { return dayOne.Add(48 * time.Hour) }
	unavailable, err := checkins.Checkin(ctx, 1, "checkin-day3")
	require.NoError(t, err)
	require.Equal(t, "UNAVAILABLE", unavailable.Result)
	require.Zero(t, unavailable.AwardedPoints)
	require.NotNil(t, unavailable.CurrentBalance)
	require.Equal(t, int64(20), *unavailable.CurrentBalance)

	claimCountAfter, err := client.GrowthRewardClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, claimCount, claimCountAfter)
	ledgerCountAfter, err := client.GrowthAccountLedger.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, ledgerCount, ledgerCountAfter)
}
