//go:build integration

package growth

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/growthinventoryledger"
	"github.com/stretchr/testify/require"
)

func TestProductVersionInventoryLifecycle(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	service := NewProductService(client)
	ctx := context.Background()

	version := ProductVersionInput{
		PointsCost:     50,
		BenefitSummary: "100 万 Token 额度",
		TokenAmount:    0,
		PerUserLimit:   3,
	}
	product, err := service.Create(ctx, ProductCreateInput{
		OperatorID:      1,
		ProductCode:     "token_1m",
		Title:           "100 万 Token",
		ProductType:     "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC",
		StockMode:       "LIMITED",
		InitialStock:    100,
		Version:         version,
		RequestID:       "product-create",
	})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", product.Status)
	require.Zero(t, product.CurrentVersion)
	require.Equal(t, 1, *product.PendingVersion)
	require.Equal(t, int64(100), product.Stock)

	_, err = service.Publish(ctx, product.ProductID, 1, "publish-invalid")
	require.ErrorIs(t, err, ErrProductPublishInvalid)

	version.TokenAmount = 1_000_000
	updated, err := service.Update(ctx, product.ProductID, ProductUpdateInput{
		OperatorID:      1,
		Title:           "100 万 Token",
		FulfillmentMode: "AUTOMATIC",
		Version:         version,
		RequestID:       "product-update-v1",
	})
	require.NoError(t, err)
	require.Equal(t, 1, *updated.PendingVersion)

	published, err := service.Publish(ctx, product.ProductID, 1, "publish-v1")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", published.Status)
	require.Equal(t, 1, published.CurrentVersion)
	require.Nil(t, published.PendingVersion)
	require.Equal(t, int64(50), published.PointsCost)

	version.PointsCost = 80
	version.BenefitSummary = "100 万 Token 额度（新价格）"
	updated, err = service.Update(ctx, product.ProductID, ProductUpdateInput{
		OperatorID:      1,
		Title:           "100 万 Token",
		FulfillmentMode: "AUTOMATIC",
		Version:         version,
		RequestID:       "product-update-v2",
	})
	require.NoError(t, err)
	require.Equal(t, 1, updated.CurrentVersion)
	require.Equal(t, 2, *updated.PendingVersion)
	require.Equal(t, int64(50), updated.PointsCost)

	publishedV2, err := service.Publish(ctx, product.ProductID, 1, "publish-v2")
	require.NoError(t, err)
	require.Equal(t, 2, publishedV2.CurrentVersion)
	require.Equal(t, int64(80), publishedV2.PointsCost)

	versions, err := service.Versions(ctx, product.ProductID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 2, versions[0].Version)
	require.Equal(t, int64(50), versions[1].PointsCost)
	require.NotNil(t, versions[1].EffectiveTo)

	adjusted, err := service.AdjustStock(ctx, product.ProductID, StockAdjustment{
		OperatorID: 1, Delta: -30, Reason: "线下出库三十件", RequestID: "stock-out-30",
	})
	require.NoError(t, err)
	require.Equal(t, int64(70), adjusted.Stock)

	_, err = service.AdjustStock(ctx, product.ProductID, StockAdjustment{
		OperatorID: 1, Delta: -100, Reason: "库存不得为负数", RequestID: "stock-underflow",
	})
	require.ErrorIs(t, err, ErrInventoryUnderflow)

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
			_, err := service.AdjustStock(ctx, product.ProductID, StockAdjustment{
				OperatorID: 1,
				Delta:      -40,
				Reason:     fmt.Sprintf("并发库存核减 %d", index),
				RequestID:  fmt.Sprintf("stock-concurrent-%d", index),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.ErrorIs(t, err, ErrInventoryUnderflow)
				failed++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, success)
	require.Equal(t, 1, failed)

	got, err := service.Get(ctx, product.ProductID)
	require.NoError(t, err)
	require.Equal(t, int64(30), got.Stock)
	ledgerCount, err := client.GrowthInventoryLedger.Query().
		Where(growthinventoryledger.ProductIDEQ(product.ProductID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, ledgerCount)

	listed, total, err := service.List(ctx, BudgetListFilter{Page: 1, PageSize: 10}, "TOKEN_QUOTA")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, listed, 1)
	require.Equal(t, int64(80), listed[0].PointsCost)
	require.Equal(t, int64(30), listed[0].Stock)
	require.Equal(t, 3, listed[0].PerUserLimit)

	offline, err := service.Offline(ctx, product.ProductID, 1, "商品活动结束下架", "product-offline")
	require.NoError(t, err)
	require.Equal(t, "OFFLINE", offline.Status)
}
