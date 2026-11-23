//go:build unit

package growth

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestFulfillmentSnapshotHelpers(t *testing.T) {
	order := &ent.GrowthExchangeOrder{ProductSnapshot: normalizeJSONMap(map[string]any{
		"product_type":     "TOKEN_QUOTA",
		"fulfillment_mode": "MANUAL",
		"token_amount":     float64(1000),
	})}
	require.Equal(t, "TOKEN_QUOTA", productType(order))
	require.Equal(t, "MANUAL", fulfillmentMode(order))
	require.True(t, requiresManualFulfillment(order))
	require.Equal(t, int64(1000), int64FromSnapshot(normalizeJSONMap(order.ProductSnapshot), "token_amount"))
	require.Equal(t, 1<<6, 1<<min(10, 6))

	legacy := &ent.GrowthExchangeOrder{ProductSnapshot: normalizeJSONMap(map[string]any{
		"product_type": "TOKEN_QUOTA",
	})}
	require.Equal(t, "AUTOMATIC", fulfillmentMode(legacy))
	require.False(t, requiresManualFulfillment(legacy))

	legacyPhysical := &ent.GrowthExchangeOrder{ProductSnapshot: normalizeJSONMap(map[string]any{
		"product_type": "PHYSICAL",
	})}
	require.True(t, requiresManualFulfillment(legacyPhysical))
}
