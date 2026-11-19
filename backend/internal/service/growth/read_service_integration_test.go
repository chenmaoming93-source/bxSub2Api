//go:build integration

package growth

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/growthproductversion"
	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/stretchr/testify/require"
)

func TestGrowthUAndWReadModelsIsolationAndCursor(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	products := NewProductService(client)
	transfers := NewTransferService(client)
	reads := NewReadService(client, &config.Config{Timezone: "Asia/Shanghai"})
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 1000, Reason: "读取模型测试预算", RequestID: "read-topup",
	})
	require.NoError(t, err)

	rule := TaskRuleInput{Target: 1, RewardPoints: 10, PeriodType: "DAILY", DailyLimit: 1, PerUserLimit: 1}
	checkin, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "daily_checkin", Title: "每日签到", Category: "CHECKIN", TaskType: "CHECKIN",
		InitialAllocation: 100, Rule: rule, RequestID: "read-checkin",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, checkin.TaskID, 1, nil, "read-publish-checkin")
	require.NoError(t, err)

	draft, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "draft_task", Title: "草稿任务", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 100, Rule: rule, RequestID: "read-draft-task",
	})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", draft.Status)

	offlineTask, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "offline_task", Title: "下线任务", Category: "DAILY", TaskType: "DAILY",
		InitialAllocation: 100, Rule: rule, RequestID: "read-offline-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, offlineTask.TaskID, 1, nil, "read-offline-publish")
	require.NoError(t, err)
	_, err = tasks.Offline(ctx, offlineTask.TaskID, 1, "测试下线任务", true, "read-offline")
	require.NoError(t, err)

	productVersion := ProductVersionInput{
		PointsCost: 5, BenefitSummary: "100 万 Token", TokenAmount: 1_000_000, PerUserLimit: 3,
	}
	product, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "token_read", Title: "Token 商品", ProductType: "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC", StockMode: "LIMITED", InitialStock: 5,
		Version: productVersion, RequestID: "read-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, product.ProductID, 1, "read-product-publish")
	require.NoError(t, err)

	_, err = products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "draft_product", Title: "草稿商品", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "LIMITED", InitialStock: 1,
		Version: ProductVersionInput{PointsCost: 10, BenefitSummary: "草稿"}, RequestID: "read-draft-product",
	})
	require.NoError(t, err)

	offlineProduct, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "offline_product", Title: "下线商品", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "UNLIMITED",
		Version: ProductVersionInput{PointsCost: 10, BenefitSummary: "下线"}, RequestID: "read-offline-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, offlineProduct.ProductID, 1, "read-offline-product-publish")
	require.NoError(t, err)
	_, err = products.Offline(ctx, offlineProduct.ProductID, 1, "测试商品下线", "read-offline-product-off")
	require.NoError(t, err)

	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: checkin.TaskID, Points: 10, BizType: "CHECKIN",
		IdempotencyKey: "read-award-1", Title: "每日签到", ReferenceID: 1,
	})
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: checkin.TaskID, Points: 5, BizType: "TASK",
		IdempotencyKey: "read-award-2", Title: "额外奖励", ReferenceID: 2,
	})
	require.NoError(t, err)

	_, err = client.ExecContext(ctx, `INSERT INTO users (id, email, password_hash) VALUES (2, 'second@example.com', 'x')`)
	require.NoError(t, err)
	_, err = client.GrowthWallet.Create().SetUserID(2).SetAvailablePoints(99).SetEarnedTotal(99).Save(ctx)
	require.NoError(t, err)

	currentVersion, err := client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(product.ProductID)).
		Only(ctx)
	require.NoError(t, err)
	_, err = client.GrowthExchangeOrder.Create().
		SetOrderNo("EX-READ-1").
		SetUserID(1).
		SetProductID(product.ProductID).
		SetProductVersion(currentVersion.Version).
		SetProductSnapshot(normalizeJSONMap(map[string]any{"title": product.Title, "version": currentVersion.Version, "points_cost": 5})).
		SetPointsCost(5).
		SetOrderStatus("SUCCEEDED").
		SetIdempotencyKey("read-order-1").
		Save(ctx)
	require.NoError(t, err)

	overview, err := reads.UOverview(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(15), overview.AvailablePoints)
	require.Equal(t, "AVAILABLE", overview.Checkin.Status)
	require.Equal(t, "Asia/Shanghai", overview.Checkin.Timezone)
	require.NotEmpty(t, overview.ServerTime)

	otherOverview, err := reads.WOverview(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, int64(99), otherOverview.AvailablePoints)

	uTasks, total, err := reads.UTasks(ctx, 1, 1, 20, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, uTasks, 1)
	require.Equal(t, "daily_checkin", uTasks[0].TaskCode)
	require.NotEmpty(t, uTasks[0].Action)

	wTasks, err := reads.WTasks(ctx, 1)
	require.NoError(t, err)
	require.Len(t, wTasks, 1)

	uProducts, total, err := reads.UProducts(ctx, 1, 1, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, uProducts, 1)
	require.Equal(t, "token_read", uProducts[0].ProductCode)
	require.True(t, uProducts[0].MyRedeemedCount >= 0)

	wProducts, err := reads.WProducts(ctx, 1)
	require.NoError(t, err)
	require.Len(t, wProducts, 1)
	require.True(t, wProducts[0].CanExchange)

	productDetail, err := reads.UProduct(ctx, 1, product.ProductID)
	require.NoError(t, err)
	require.True(t, productDetail.Eligible)
	require.Equal(t, int64(5), productDetail.RemainingQuota)

	uOrders, orderTotal, err := reads.UExchanges(ctx, 1, 1, 20, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), orderTotal)
	require.Len(t, uOrders, 1)
	require.Equal(t, "Token 商品", uOrders[0].ProductTitle)

	firstLedger, err := reads.Ledger(ctx, 1, 0, 1, "")
	require.NoError(t, err)
	require.Len(t, firstLedger.Items, 1)
	require.True(t, firstLedger.HasMore)
	require.NotEmpty(t, firstLedger.NextCursor)

	beforeID, err := parseCursor(firstLedger.NextCursor)
	require.NoError(t, err)
	secondLedger, err := reads.Ledger(ctx, 1, beforeID, 1, "")
	require.NoError(t, err)
	require.Len(t, secondLedger.Items, 1)
	require.NotEqual(t, firstLedger.Items[0].EntryID, secondLedger.Items[0].EntryID)
}

func parseCursor(value string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}
