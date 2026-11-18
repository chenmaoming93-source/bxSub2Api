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

func TestGrowthMySQLArchiveAndConstraints(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("GROWTH_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		t.Skip("set GROWTH_MYSQL_TEST_DSN to a MySQL 8/GoldenDB root or database-admin DSN")
	}
	cfg, err := mysqlDriver.ParseDSN(baseDSN)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.MultiStatements = false
	adminDB, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	defer adminDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	require.NoError(t, adminDB.PingContext(ctx))

	databaseName := fmt.Sprintf("sub2api_growth_it_%d", time.Now().UnixNano())
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
	defer db.Close()
	require.NoError(t, db.PingContext(ctx))

	_, err = db.ExecContext(ctx, `CREATE TABLE users (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO users (email, password_hash) VALUES
		('growth-a@example.com', 'x'), ('growth-b@example.com', 'x')`)
	require.NoError(t, err)

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	archivePath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "sqlArchiving", "182_create_growth_tables.sql")
	raw, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	statements := growthSQLStatements(string(raw))

	for pass := 1; pass <= 2; pass++ {
		for _, statement := range statements {
			_, err := db.ExecContext(ctx, statement)
			require.NoErrorf(t, err, "growth archive pass %d failed: %s", pass, statement)
		}
	}

	tables := []string{
		"growth_budgets", "growth_budget_entries", "growth_wallets", "growth_transfers",
		"growth_account_ledger", "growth_tasks", "growth_task_rules", "growth_user_tasks",
		"growth_completion_requests", "growth_reward_claims", "growth_products",
		"growth_product_versions", "growth_product_inventory", "growth_exchange_orders",
		"growth_fulfillment_jobs", "growth_inventory_ledger", "growth_admin_audit",
	}
	for _, table := range tables {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&count))
		require.Equalf(t, 1, count, "expected table %s", table)
	}
	var taskDeleteColumn int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'growth_tasks' AND column_name = 'deleted_at'`).Scan(&taskDeleteColumn))
	require.Zero(t, taskDeleteColumn)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_tasks
		(task_code, title, category, task_type, status, allocated_points)
		VALUES ('dup_task', 'Dup', 'DAILY', 'DAILY', 'ACTIVE', 10)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_tasks
		(task_code, title, category, task_type, status, allocated_points)
		VALUES ('dup_task', 'Dup 2', 'DAILY', 'DAILY', 'ACTIVE', 10)`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_task_rules
		(task_id, version, reward_points, period_type, created_by, effective_from)
		VALUES (1, 1, 10, 'DAILY', 1, CURRENT_TIMESTAMP(6))`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_task_rules
		(task_id, version, reward_points, period_type, created_by, effective_from)
		VALUES (1, 1, 20, 'DAILY', 1, CURRENT_TIMESTAMP(6))`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_user_tasks
		(user_id, task_id, rule_id, cycle_key) VALUES (1, 1, 1, '2026-11-17')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_user_tasks
		(user_id, task_id, rule_id, cycle_key) VALUES (1, 1, 1, '2026-11-17')`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_completion_requests
		(user_id, task_id, completion_ref, result, rule_version, cycle_key)
		VALUES (1, 1, 'completion-1', 'NOT_ELIGIBLE', 1, '2026-11-17')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_completion_requests
		(user_id, task_id, completion_ref, result, rule_version, cycle_key)
		VALUES (1, 1, 'completion-1', 'NOT_ELIGIBLE', 1, '2026-11-17')`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_exchange_orders
		(order_no, user_id, product_id, product_version, product_snapshot, points_cost, idempotency_key)
		VALUES ('ORDER-1', 1, 1, 1, JSON_OBJECT('title', 'Product'), 10, 'order-key-1')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_exchange_orders
		(order_no, user_id, product_id, product_version, product_snapshot, points_cost, idempotency_key)
		VALUES ('ORDER-2', 1, 1, 1, JSON_OBJECT('title', 'Product'), 10, 'order-key-1')`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_fulfillment_jobs (order_id) VALUES (1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO growth_fulfillment_jobs (order_id) VALUES (1)`)
	requireMySQLError(t, err, 1062)

	_, err = db.ExecContext(ctx, `INSERT INTO growth_wallets (user_id) VALUES (1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE growth_wallets SET available_points = -1 WHERE user_id = 1`)
	requireMySQLError(t, err, 3819)
	_, err = db.ExecContext(ctx, `UPDATE growth_tasks SET distributed_points = -1 WHERE id = 1`)
	requireMySQLError(t, err, 3819)
}

func growthSQLStatements(raw string) []string {
	var statements []string
	for _, part := range strings.Split(raw, ";") {
		var lines []string
		for _, line := range strings.Split(part, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			lines = append(lines, line)
		}
		statement := strings.TrimSpace(strings.Join(lines, "\n"))
		if statement == "" || strings.HasPrefix(strings.ToUpper(statement), "SELECT") {
			continue
		}
		statements = append(statements, statement)
	}
	return statements
}

func requireMySQLError(t *testing.T, err error, number uint16) {
	t.Helper()
	require.Error(t, err)
	var mysqlErr *mysqlDriver.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.Equal(t, number, mysqlErr.Number)
}
