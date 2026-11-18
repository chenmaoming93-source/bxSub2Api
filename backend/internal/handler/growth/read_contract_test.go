//go:build unit || integration

package growth

import (
	"encoding/json"
	"testing"

	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"

	"github.com/stretchr/testify/require"
)

func TestUReadWReadContractFields(t *testing.T) {
	assertJSONFields(t, growthservice.UOverviewView{}, []string{
		"user_id", "available_points", "earned_total", "spent_total", "checkin", "task_summary", "server_time",
	})
	assertJSONFields(t, growthservice.UTaskItemView{}, []string{
		"task_id", "task_code", "title", "category", "task_type", "status", "progress", "target",
		"cycle_key", "reward_hint", "reward_pool", "rule_version", "action", "icon", "sort_order",
	})
	assertJSONFields(t, growthservice.UProductItemView{}, []string{
		"product_id", "product_code", "title", "product_type", "version", "points_cost",
		"benefit_summary", "stock_hint", "per_user_limit", "my_redeemed_count", "status",
	})
	assertJSONFields(t, growthservice.UProductDetailView{}, []string{
		"benefit_detail", "redeem_window", "remaining_quota", "eligible", "ineligible_reason",
	})
	assertJSONFields(t, growthservice.UExchangeItemView{}, []string{
		"order_id", "product_title", "points_cost", "status", "created_at",
	})
	assertJSONFields(t, growthservice.LedgerPage{}, []string{"items", "next_cursor", "has_more"})
}

func TestCheckinContractFields(t *testing.T) {
	assertJSONFields(t, growthservice.CheckinResult{}, []string{
		"result", "claim_id", "business_date", "timezone", "awarded_points",
		"current_balance", "rule_version", "server_time",
	})
}

func TestCompletionContractFields(t *testing.T) {
	assertJSONFields(t, growthservice.CompletionResult{}, []string{
		"request_id", "result", "task_id", "completion_ref", "awarded_points",
		"current_balance", "rule_version", "cycle_key", "server_time",
	})
}

func TestExchangeContractFields(t *testing.T) {
	assertJSONFields(t, growthservice.ExchangeCreateResult{}, []string{
		"order_id", "order_no", "status", "product_version", "points_cost",
		"current_balance", "fulfillment_status", "server_time",
	})
	assertJSONFields(t, growthservice.ExchangeOrderDetail{}, []string{
		"order_id", "order_no", "status", "product_snapshot", "points_cost",
		"fulfillment_status", "failure_reason", "created_at",
	})
}

func assertJSONFields(t *testing.T, value any, fields []string) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var object map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &object))
	for _, field := range fields {
		_, exists := object[field]
		require.Truef(t, exists, "missing contract field %s in %T: %s", field, value, raw)
	}
}
