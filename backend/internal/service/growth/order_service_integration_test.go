//go:build integration

package growth

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/growthinventoryledger"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"

	"github.com/stretchr/testify/require"
)

func TestExchangeOrderAtomicityReplayErrorsConcurrencyAndIsolation(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	products := NewProductService(client)
	transfers := NewTransferService(client)
	orders := NewOrderService(client, transfers)
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "兑换测试预算", RequestID: "exchange-topup",
	})
	require.NoError(t, err)
	rule := TaskRuleInput{Target: 1, RewardPoints: 100, PeriodType: "ONCE"}
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "exchange_funding", Title: "兑换资金", Category: "ACHIEVEMENT",
		TaskType: "ACHIEVEMENT", InitialAllocation: 100, Rule: rule, RequestID: "exchange-funding-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task.TaskID, 1, nil, "exchange-funding-publish")
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.TaskID, Points: 100, BizType: "TASK",
		IdempotencyKey: "exchange-wallet-fund", Title: "兑换资金",
	})
	require.NoError(t, err)

	_, err = client.ExecContext(ctx, `INSERT INTO users (id, email, password_hash) VALUES
		(2, 'second@example.com', 'x'), (3, 'third@example.com', 'x')`)
	require.NoError(t, err)
	_, err = client.GrowthWallet.Create().SetUserID(2).SetAvailablePoints(100).SetEarnedTotal(100).Save(ctx)
	require.NoError(t, err)
	_, err = client.GrowthWallet.Create().SetUserID(3).SetAvailablePoints(100).SetEarnedTotal(100).Save(ctx)
	require.NoError(t, err)

	limited, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "limited_exchange", Title: "限量商品", ProductType: "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC", StockMode: "LIMITED", InitialStock: 2,
		Version:   ProductVersionInput{PointsCost: 30, BenefitSummary: "30 分", TokenAmount: 1000, PerUserLimit: 3},
		RequestID: "limited-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, limited.ProductID, 1, "limited-publish")
	require.NoError(t, err)

	created, err := orders.Create(ctx, ExchangeCommand{
		UserID: 1, ProductID: limited.ProductID, IdempotencyKey: "exchange-key-1",
	})
	require.NoError(t, err)
	require.Equal(t, "FULFILLING", created.Status)
	require.Equal(t, "PENDING", created.FulfillmentStatus)
	require.Equal(t, int64(70), created.CurrentBalance)
	require.NotNil(t, created.TransferID)

	replayed, err := orders.Create(ctx, ExchangeCommand{
		UserID: 1, ProductID: limited.ProductID, IdempotencyKey: "exchange-key-1",
	})
	require.NoError(t, err)
	require.Equal(t, created.OrderID, replayed.OrderID)
	require.Equal(t, int64(70), replayed.CurrentBalance)

	wrongVersion := 99
	_, err = orders.Create(ctx, ExchangeCommand{
		UserID: 1, ProductID: limited.ProductID, ExpectedVersion: &wrongVersion,
		IdempotencyKey: "exchange-version-error",
	})
	require.ErrorIs(t, err, ErrProductVersionChanged)

	secondOrder, err := orders.Create(ctx, ExchangeCommand{
		UserID: 1, ProductID: limited.ProductID, IdempotencyKey: "exchange-key-2",
	})
	require.NoError(t, err)
	require.NotEqual(t, created.OrderID, secondOrder.OrderID)
	require.Equal(t, int64(40), secondOrder.CurrentBalance)

	_, err = orders.Create(ctx, ExchangeCommand{
		UserID: 1, ProductID: limited.ProductID, IdempotencyKey: "exchange-limit",
	})
	require.ErrorIs(t, err, ErrProductSoldOut)

	_, err = orders.Create(ctx, ExchangeCommand{
		UserID: 2, ProductID: limited.ProductID, IdempotencyKey: "exchange-sold-out",
	})
	require.ErrorIs(t, err, ErrProductSoldOut)

	offline, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "offline_exchange", Title: "下线商品", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "UNLIMITED",
		Version: ProductVersionInput{PointsCost: 10, BenefitSummary: "下线"}, RequestID: "offline-exchange-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, offline.ProductID, 1, "offline-exchange-publish")
	require.NoError(t, err)
	_, err = products.Offline(ctx, offline.ProductID, 1, "兑换测试下架商品", "offline-exchange-off")
	require.NoError(t, err)
	_, err = orders.Create(ctx, ExchangeCommand{
		UserID: 3, ProductID: offline.ProductID, IdempotencyKey: "exchange-offline",
	})
	require.ErrorIs(t, err, ErrProductOffline)

	stockRace, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "stock_race", Title: "库存竞争", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "LIMITED", InitialStock: 1,
		Version: ProductVersionInput{PointsCost: 50, BenefitSummary: "库存竞争"}, RequestID: "stock-race-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, stockRace.ProductID, 1, "stock-race-publish")
	require.NoError(t, err)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		success  int
		conflict int
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := orders.Create(ctx, ExchangeCommand{
				UserID: 2, ProductID: stockRace.ProductID,
				IdempotencyKey: "stock-race-" + string(rune('a'+index)),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.True(t, errors.Is(err, ErrProductSoldOut))
				conflict++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflict)

	unlimited, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "wallet_race", Title: "余额竞争", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "UNLIMITED",
		Version: ProductVersionInput{PointsCost: 80, BenefitSummary: "余额竞争"}, RequestID: "wallet-race-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, unlimited.ProductID, 1, "wallet-race-publish")
	require.NoError(t, err)
	success = 0
	conflict = 0
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := orders.Create(ctx, ExchangeCommand{
				UserID: 3, ProductID: unlimited.ProductID,
				IdempotencyKey: "wallet-race-" + string(rune('a'+index)),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.True(t, errors.Is(err, ErrInsufficientPoints))
				conflict++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflict)

	_, err = orders.UDetail(ctx, 2, created.OrderID)
	require.ErrorIs(t, err, ErrOrderNotFound)
	detail, err := orders.WDetail(ctx, 1, created.OrderID)
	require.NoError(t, err)
	require.Equal(t, created.OrderID, detail.OrderID)

	orderCount, err := client.GrowthExchangeOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, orderCount)
	jobCount, err := client.GrowthFulfillmentJob.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, jobCount)
	reserveCount, err := client.GrowthInventoryLedger.Query().
		Where(growthinventoryledger.OperationTypeEQ("RESERVE")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, reserveCount)
	limitedInventory, err := client.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(limited.ProductID)).
		Only(ctx)
	require.NoError(t, err)
	require.Zero(t, limitedInventory.Available)
	require.Equal(t, int64(2), limitedInventory.Reserved)
	wallet2, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(2)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(50), wallet2.AvailablePoints)
}
