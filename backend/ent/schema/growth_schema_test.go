//go:build unit

package schema

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrowthSchemaAndArchiveStayAligned(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	schemaDir := filepath.Dir(sourceFile)
	archivePath := filepath.Join(schemaDir, "..", "..", "sqlArchiving", "182_create_growth_tables.sql")
	raw, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	sql := string(raw)

	expectedTables := map[string]string{
		"growth_budget.go":             "growth_budgets",
		"growth_budget_entry.go":       "growth_budget_entries",
		"growth_wallet.go":             "growth_wallets",
		"growth_transfer.go":           "growth_transfers",
		"growth_account_ledger.go":     "growth_account_ledger",
		"growth_task.go":               "growth_tasks",
		"growth_task_rule.go":          "growth_task_rules",
		"growth_user_task.go":          "growth_user_tasks",
		"growth_completion_request.go": "growth_completion_requests",
		"growth_reward_claim.go":       "growth_reward_claims",
		"growth_product.go":            "growth_products",
		"growth_product_version.go":    "growth_product_versions",
		"growth_product_inventory.go":  "growth_product_inventory",
		"growth_exchange_order.go":     "growth_exchange_orders",
		"growth_fulfillment_job.go":    "growth_fulfillment_jobs",
		"growth_inventory_ledger.go":   "growth_inventory_ledger",
		"growth_admin_audit.go":        "growth_admin_audit",
	}
	require.Len(t, expectedTables, 17)

	createPattern := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS\s+(growth_[a-z_]+)`)
	created := createPattern.FindAllStringSubmatch(sql, -1)
	require.Len(t, created, 17)
	createdNames := make(map[string]bool, len(created))
	for _, match := range created {
		createdNames[match[1]] = true
	}

	for file, table := range expectedTables {
		source, err := os.ReadFile(filepath.Join(schemaDir, file))
		require.NoError(t, err)
		require.Contains(t, string(source), `entsql.Annotation{Table: "`+table+`"}`)
		require.True(t, createdNames[table], "missing CREATE TABLE for %s", table)
	}

	require.NotContains(t, tableBlock(t, sql, "growth_tasks"), "deleted_at")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_tasks_code (task_code)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_task_rules_version (task_id, version)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_user_tasks_cycle (user_id, task_id, rule_id, cycle_key)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_completion_business (user_id, task_id, completion_ref)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_reward_claims_cycle (user_id, task_id, cycle_key, sequence)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_exchange_orders_key (user_id, idempotency_key)")
	require.Contains(t, sql, "UNIQUE KEY uq_growth_fulfillment_jobs_order (order_id)")
	require.NotContains(t, sql, " description ")
	require.NotContains(t, sql, "\n    status ")
	require.NotContains(t, sql, "\n    action ")
	require.NotContains(t, strings.ToUpper(sql), " JSON ")
	require.Contains(t, sql, "conditions LONGTEXT NULL")
	require.Contains(t, sql, "benefit_detail LONGTEXT NULL")
	require.Contains(t, sql, "product_snapshot LONGTEXT NULL")
	require.Contains(t, sql, "before_json LONGTEXT NULL")
	require.Contains(t, sql, "after_json LONGTEXT NULL")
	require.Contains(t, sql, "CHECK")
	require.NotContains(t, strings.ToUpper(sql), "ON DELETE CASCADE")
}

func tableBlock(t *testing.T, sql, table string) string {
	t.Helper()
	start := strings.Index(sql, "CREATE TABLE IF NOT EXISTS "+table+" (")
	require.NotEqual(t, -1, start, "missing table %s", table)
	end := strings.Index(sql[start:], "ENGINE=InnoDB")
	require.NotEqual(t, -1, end, "missing engine terminator for %s", table)
	return sql[start : start+end]
}
