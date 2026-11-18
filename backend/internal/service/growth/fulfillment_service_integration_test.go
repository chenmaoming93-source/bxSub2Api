//go:build integration

package growth

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/growthfulfillmentjob"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"

	"github.com/stretchr/testify/require"
)

func TestOutboxWorkerRetryRefundManualAndAdminOperations(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	tasks := NewTaskService(client)
	products := NewProductService(client)
	transfers := NewTransferService(client)
	orders := NewOrderService(client, transfers)
	fulfillment := NewFulfillmentService(client, transfers)
	t.Cleanup(fulfillment.Stop)
	fulfillment.now = time.Now
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 200, Reason: "履约测试预算", RequestID: "fulfillment-topup",
	})
	require.NoError(t, err)
	task, err := tasks.Create(ctx, TaskCreateInput{
		OperatorID: 1, TaskCode: "fulfillment_funding", Title: "履约资金", Category: "ACHIEVEMENT",
		TaskType: "ACHIEVEMENT", InitialAllocation: 200,
		Rule: TaskRuleInput{Target: 1, RewardPoints: 200, PeriodType: "ONCE"}, RequestID: "fulfillment-funding-task",
	})
	require.NoError(t, err)
	_, err = tasks.Publish(ctx, task.TaskID, 1, nil, "fulfillment-funding-publish")
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.TaskID, Points: 200, BizType: "TASK",
		IdempotencyKey: "fulfillment-wallet", Title: "履约资金",
	})
	require.NoError(t, err)

	tokenProduct, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "fulfill_token", Title: "Token 履约", ProductType: "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC", StockMode: "LIMITED", InitialStock: 2,
		Version:   ProductVersionInput{PointsCost: 30, BenefitSummary: "Token", TokenAmount: 1000, PerUserLimit: 5},
		RequestID: "fulfill-token-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, tokenProduct.ProductID, 1, "fulfill-token-publish")
	require.NoError(t, err)
	tokenOrder, err := orders.Create(ctx, ExchangeCommand{UserID: 1, ProductID: tokenProduct.ProductID, IdempotencyKey: "fulfill-token-order"})
	require.NoError(t, err)

	var tokenCalls atomic.Int32
	fulfillment.SetAdapter("TOKEN_QUOTA", FulfillmentAdapterFunc(func(context.Context, FulfillmentRequest) (FulfillmentResponse, error) {
		tokenCalls.Add(1)
		return FulfillmentResponse{ProviderReference: "token-provider-1"}, nil
	}))
	processed, err := fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, int32(1), tokenCalls.Load())
	tokenOrderEntity, err := client.GrowthExchangeOrder.Get(ctx, tokenOrder.OrderID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", tokenOrderEntity.Status)
	tokenJob, err := client.GrowthFulfillmentJob.Query().Where(growthfulfillmentjob.OrderIDEQ(tokenOrder.OrderID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", tokenJob.Status)
	require.Equal(t, 1, tokenJob.AttemptCount)

	processed, err = fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	require.Zero(t, processed)
	require.Equal(t, int32(1), tokenCalls.Load())

	subscriptionProduct, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "fulfill_sub", Title: "订阅履约", ProductType: "SUBSCRIPTION",
		FulfillmentMode: "AUTOMATIC", StockMode: "LIMITED", InitialStock: 2,
		Version:   ProductVersionInput{PointsCost: 20, BenefitSummary: "订阅", PerUserLimit: 5},
		RequestID: "fulfill-sub-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, subscriptionProduct.ProductID, 1, "fulfill-sub-publish")
	require.NoError(t, err)
	subOrder, err := orders.Create(ctx, ExchangeCommand{UserID: 1, ProductID: subscriptionProduct.ProductID, IdempotencyKey: "fulfill-sub-order"})
	require.NoError(t, err)
	var subCalls atomic.Int32
	fulfillment.SetAdapter("SUBSCRIPTION", FulfillmentAdapterFunc(func(context.Context, FulfillmentRequest) (FulfillmentResponse, error) {
		if subCalls.Add(1) == 1 {
			return FulfillmentResponse{}, fmt.Errorf("%w: temporary provider timeout", ErrRetryableFulfillment)
		}
		return FulfillmentResponse{ProviderReference: "sub-provider-1"}, nil
	}))
	processed, err = fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	subJob, err := client.GrowthFulfillmentJob.Query().Where(growthfulfillmentjob.OrderIDEQ(subOrder.OrderID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "RETRY", subJob.Status)
	require.Equal(t, 1, subJob.AttemptCount)
	_, err = client.GrowthFulfillmentJob.UpdateOneID(subJob.ID).SetNextRetryAt(time.Now().Add(-time.Second)).Save(ctx)
	require.NoError(t, err)
	processed, err = fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, int32(2), subCalls.Load())
	subOrderEntity, err := client.GrowthExchangeOrder.Get(ctx, subOrder.OrderID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", subOrderEntity.Status)

	failingProduct, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "fulfill_fail", Title: "失败履约", ProductType: "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC", StockMode: "LIMITED", InitialStock: 1,
		Version:   ProductVersionInput{PointsCost: 10, BenefitSummary: "失败", TokenAmount: 100, PerUserLimit: 5},
		RequestID: "fulfill-fail-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, failingProduct.ProductID, 1, "fulfill-fail-publish")
	require.NoError(t, err)
	failOrder, err := orders.Create(ctx, ExchangeCommand{UserID: 1, ProductID: failingProduct.ProductID, IdempotencyKey: "fulfill-fail-order"})
	require.NoError(t, err)
	fulfillment.SetAdapter("TOKEN_QUOTA", FulfillmentAdapterFunc(func(context.Context, FulfillmentRequest) (FulfillmentResponse, error) {
		return FulfillmentResponse{}, errors.New("permanent provider rejection")
	}))
	_, err = fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	refundedOrder, err := client.GrowthExchangeOrder.Get(ctx, failOrder.OrderID)
	require.NoError(t, err)
	require.Equal(t, "REFUNDED", refundedOrder.Status)
	require.NotNil(t, refundedOrder.RefundTransferID)
	restoredInventory, err := client.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(failingProduct.ProductID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), restoredInventory.Available)
	require.Zero(t, restoredInventory.Reserved)
	require.Zero(t, restoredInventory.Sold)

	refundCount, err := client.GrowthTransfer.Query().
		Where(growthtransfer.BizTypeEQ("REFUND")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, refundCount)
	detail, err := fulfillment.AdminRefund(ctx, failOrder.OrderID, 1, "重复退款应幂等", "refund-repeat")
	require.NoError(t, err)
	require.Equal(t, "REFUNDED", detail.Status)
	refundCount, err = client.GrowthTransfer.Query().
		Where(growthtransfer.BizTypeEQ("REFUND")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, refundCount)

	physicalProduct, err := products.Create(ctx, ProductCreateInput{
		OperatorID: 1, ProductCode: "fulfill_physical", Title: "实物履约", ProductType: "PHYSICAL",
		FulfillmentMode: "MANUAL", StockMode: "UNLIMITED",
		Version:   ProductVersionInput{PointsCost: 10, BenefitSummary: "实物", PerUserLimit: 5},
		RequestID: "fulfill-physical-product",
	})
	require.NoError(t, err)
	_, err = products.Publish(ctx, physicalProduct.ProductID, 1, "fulfill-physical-publish")
	require.NoError(t, err)
	physicalOrder, err := orders.Create(ctx, ExchangeCommand{UserID: 1, ProductID: physicalProduct.ProductID, IdempotencyKey: "fulfill-physical-order"})
	require.NoError(t, err)
	processed, err = fulfillment.ProcessDue(ctx)
	require.NoError(t, err)
	require.Zero(t, processed)
	walletBefore, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	manual, err := fulfillment.ManualComplete(ctx, physicalOrder.OrderID, 1, "运营线下发货完成", "manual-complete")
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", manual.Status)
	walletAfter, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, walletBefore.AvailablePoints, walletAfter.AvailablePoints)

	retryOrder, err := orders.Create(ctx, ExchangeCommand{UserID: 1, ProductID: physicalProduct.ProductID, IdempotencyKey: "fulfill-retry-order"})
	require.NoError(t, err)
	_, err = client.GrowthExchangeOrder.UpdateOneID(retryOrder.OrderID).SetStatus("FAILED").Save(ctx)
	require.NoError(t, err)
	_, err = client.GrowthFulfillmentJob.Update().
		Where(growthfulfillmentjob.OrderIDEQ(retryOrder.OrderID)).
		SetStatus("FAILED").
		Save(ctx)
	require.NoError(t, err)
	beforeRetryWallet, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	retried, err := fulfillment.Retry(ctx, retryOrder.OrderID, 1, "admin-retry")
	require.NoError(t, err)
	require.Equal(t, "FULFILLING", retried.Status)
	afterRetryWallet, err := client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(1)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeRetryWallet.AvailablePoints, afterRetryWallet.AvailablePoints)

	adminItems, total, err := fulfillment.ListAdminOrders(ctx, 1, 50, "", "")
	require.NoError(t, err)
	require.Equal(t, int64(5), total)
	require.Len(t, adminItems, 5)
}
