//go:build integration

package growthmysqltest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestGrowthCompatibilityUpgradeScripts(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("GROWTH_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		t.Skip("set GROWTH_MYSQL_TEST_DSN to run Growth MySQL integration tests")
	}
	cfg, err := mysqlDriver.ParseDSN(baseDSN)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.MultiStatements = false
	adminDB, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = adminDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	require.NoError(t, adminDB.PingContext(ctx))
	databaseName := fmt.Sprintf("sub2api_growth_upgrade_it_%d", time.Now().UnixNano())
	_, err = adminDB.ExecContext(ctx, "CREATE DATABASE `"+databaseName+"` CHARACTER SET utf8mb4")
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.ExecContext(cleanupCtx, "DROP DATABASE IF EXISTS `"+databaseName+"`")
	})

	cfg.DBName = databaseName
	db, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))

	legacyStatements := []string{
		`CREATE TABLE growth_budgets (
			id BIGINT PRIMARY KEY,
			total_amount BIGINT NOT NULL,
			allocated_amount BIGINT NOT NULL,
			distributed_amount BIGINT NOT NULL,
			CONSTRAINT chk_growth_budgets_nonnegative CHECK (
				total_amount >= 0 AND allocated_amount >= 0 AND distributed_amount >= 0
				AND allocated_amount <= total_amount AND distributed_amount <= allocated_amount
			)
		)`,
		`CREATE TABLE growth_transfers (
			id BIGINT PRIMARY KEY,
			from_account_type VARCHAR(32) NOT NULL,
			to_account_type VARCHAR(32) NOT NULL,
			points BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL,
			CONSTRAINT chk_growth_transfers_accounts CHECK (
				from_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
				AND to_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
				AND points > 0
			),
			CONSTRAINT chk_growth_transfers_status CHECK (
				status IN ('PENDING', 'SUCCEEDED', 'FAILED', 'REVERSED')
			)
		)`,
		`CREATE TABLE growth_account_ledger (
			id BIGINT PRIMARY KEY,
			account_type VARCHAR(32) NOT NULL,
			direction VARCHAR(8) NOT NULL,
			points BIGINT NOT NULL,
			balance_after BIGINT NULL,
			CONSTRAINT chk_growth_account_ledger_values CHECK (
				account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
				AND direction IN ('IN', 'OUT')
			)
		)`,
		`CREATE TABLE growth_reward_claims (
			id BIGINT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			task_id BIGINT NOT NULL,
			cycle_key VARCHAR(64) NOT NULL,
			status VARCHAR(20) NOT NULL,
			UNIQUE KEY uq_growth_reward_claims_cycle (user_id, task_id, cycle_key)
		)`,
		`CREATE TABLE growth_tasks (
			id BIGINT PRIMARY KEY,
			description VARCHAR(2048) NOT NULL,
			status VARCHAR(20) NOT NULL,
			CONSTRAINT chk_growth_tasks_status CHECK (status IN ('DRAFT', 'ACTIVE', 'OFFLINE'))
		)`,
		`CREATE TABLE growth_user_tasks (id BIGINT PRIMARY KEY, status VARCHAR(32) NOT NULL)`,
		`CREATE TABLE growth_products (
			id BIGINT PRIMARY KEY,
			product_type VARCHAR(32) NOT NULL,
			fulfillment_mode VARCHAR(32) NOT NULL,
			status VARCHAR(20) NOT NULL,
			current_version INT NOT NULL,
			CONSTRAINT chk_growth_products_values CHECK (
				product_type IN ('TOKEN_QUOTA', 'SUBSCRIPTION', 'PHYSICAL')
				AND fulfillment_mode IN ('AUTOMATIC', 'MANUAL')
				AND status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'OFFLINE')
				AND current_version >= 0
			)
		)`,
		`CREATE TABLE growth_task_rules (id BIGINT PRIMARY KEY, conditions JSON NULL)`,
		`CREATE TABLE growth_product_versions (id BIGINT PRIMARY KEY, benefit_detail JSON NULL)`,
		`CREATE TABLE growth_exchange_orders (
			id BIGINT PRIMARY KEY,
			product_version INT NOT NULL,
			points_cost BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL,
			product_snapshot JSON NOT NULL,
			CONSTRAINT chk_growth_exchange_orders_values CHECK (
				product_version > 0
				AND points_cost > 0
				AND status IN ('CREATED', 'FULFILLING', 'SUCCEEDED', 'FAILED', 'REFUNDED')
			)
		)`,
		`CREATE TABLE growth_fulfillment_jobs (
			id BIGINT PRIMARY KEY,
			status VARCHAR(20) NOT NULL,
			attempt_count INT NOT NULL,
			max_attempts INT NOT NULL,
			CONSTRAINT chk_growth_fulfillment_jobs_values CHECK (
				status IN ('PENDING', 'PROCESSING', 'RETRY', 'SUCCEEDED', 'FAILED', 'CANCELLED')
				AND attempt_count >= 0
				AND max_attempts > 0
				AND attempt_count <= max_attempts
			)
		)`,
		`CREATE TABLE growth_admin_audit (
			id BIGINT PRIMARY KEY,
			action VARCHAR(64) NOT NULL,
			before_json JSON NULL,
			after_json JSON NULL
		)`,
	}
	for _, statement := range legacyStatements {
		_, err := db.ExecContext(ctx, statement)
		require.NoErrorf(t, err, "create legacy table: %s", statement)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO growth_exchange_orders
		(id, product_version, points_cost, status, product_snapshot)
		VALUES (1, 1, 10, 'CREATED', '{"title":"legacy"}')`)
	require.NoError(t, err)

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	for _, filename := range []string{
		"184_growth_adjustment_account.sql",
		"185_growth_multi_reward_claims.sql",
		"186_growth_goldendb_schema_compat.sql",
	} {
		path := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "sqlArchiving", filename)
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, statement := range growthSQLStatements(string(raw)) {
			_, err := db.ExecContext(ctx, statement)
			require.NoErrorf(t, err, "%s failed: %s", filename, statement)
		}
	}

	for _, column := range []struct {
		table string
		name  string
		typ   string
	}{
		{"growth_transfers", "transfer_status", "varchar"},
		{"growth_tasks", "remark", "varchar"},
		{"growth_tasks", "task_status", "varchar"},
		{"growth_user_tasks", "progress_status", "varchar"},
		{"growth_reward_claims", "claim_status", "varchar"},
		{"growth_reward_claims", "sequence", "int"},
		{"growth_products", "product_status", "varchar"},
		{"growth_task_rules", "conditions", "longtext"},
		{"growth_product_versions", "benefit_detail", "longtext"},
		{"growth_exchange_orders", "order_status", "varchar"},
		{"growth_exchange_orders", "product_snapshot", "longtext"},
		{"growth_fulfillment_jobs", "job_status", "varchar"},
		{"growth_admin_audit", "action_type", "varchar"},
		{"growth_admin_audit", "before_json", "longtext"},
		{"growth_admin_audit", "after_json", "longtext"},
	} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ? AND data_type = ?`,
			column.table, column.name, column.typ).Scan(&count))
		require.Equalf(t, 1, count, "expected %s.%s as %s", column.table, column.name, column.typ)
	}

	var snapshot string
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT product_snapshot FROM growth_exchange_orders WHERE id = 1`).Scan(&snapshot))
	require.JSONEq(t, `{"title":"legacy"}`, snapshot)
}
