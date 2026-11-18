//go:build unit

package growth

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestFulfillmentSnapshotHelpers(t *testing.T) {
	order := &ent.GrowthExchangeOrder{ProductSnapshot: map[string]any{
		"product_type": "TOKEN_QUOTA",
		"token_amount": float64(1000),
	}}
	require.Equal(t, "TOKEN_QUOTA", productType(order))
	require.Equal(t, int64(1000), int64FromSnapshot(order.ProductSnapshot, "token_amount"))
	require.Equal(t, 1<<6, 1<<min(10, 6))
}
