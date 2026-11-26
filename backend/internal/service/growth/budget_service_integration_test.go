//go:build integration

package growth

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestBudgetTopUpReduceAllocateRevoke(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	service := NewBudgetService(client)
	ctx := context.Background()

	account, err := service.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 1000, Reason: "初始季度预算", RequestID: "budget-topup-1",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1000), account.TotalAmount)
	require.Equal(t, int64(1000), account.UnallocatedAmount)

	task1 := createIntegrationTask(t, client, "task_one", "DRAFT")
	task2 := createIntegrationTask(t, client, "task_two", "OFFLINE")

	account, err = service.Allocate(ctx, task1.ID, BudgetMutation{
		OperatorID: 1, Points: 700, Reason: "任务一初始额度", RequestID: "allocate-1",
	})
	require.NoError(t, err)
	require.Equal(t, int64(700), account.AllocatedAmount)

	account, err = service.Allocate(ctx, task2.ID, BudgetMutation{
		OperatorID: 1, Points: 200, Reason: "任务二初始额度", RequestID: "allocate-2",
	})
	require.NoError(t, err)
	require.Equal(t, int64(900), account.AllocatedAmount)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		success  int
		failures int
	)
	for index := 0; index < 2; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := service.Allocate(ctx, task1.ID, BudgetMutation{
				OperatorID: 1,
				Points:     100,
				Reason:     fmt.Sprintf("并发追加额度 %d", index),
				RequestID:  fmt.Sprintf("allocate-concurrent-%d", index),
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				require.ErrorIs(t, err, ErrBudgetInsufficient)
				failures++
			}
		}(index)
	}
	wg.Wait()
	require.Equal(t, 1, success)
	require.Equal(t, 1, failures)

	account, err = service.GetAccount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1000), account.AllocatedAmount)
	require.Zero(t, account.UnallocatedAmount)

	_, err = service.Reduce(ctx, BudgetMutation{
		OperatorID: 1, Points: 201, Reason: "核减超过未分配额度", RequestID: "reduce-locked",
	})
	require.ErrorIs(t, err, ErrBudgetBelowLocked)

	account, err = service.Revoke(ctx, task2.ID, BudgetMutation{
		OperatorID: 1, Points: 200, Reason: "下线任务回收额度", RequestID: "revoke-2",
	})
	require.NoError(t, err)
	require.Equal(t, int64(800), account.AllocatedAmount)
	require.Equal(t, int64(200), account.UnallocatedAmount)

	_, err = service.Revoke(ctx, task1.ID, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "草稿任务不可回收", RequestID: "revoke-draft",
	})
	require.ErrorIs(t, err, ErrTaskStateInvalid)

	account, err = service.Reduce(ctx, BudgetMutation{
		OperatorID: 1, Points: 200, Reason: "年度预算核减", RequestID: "reduce-ok",
	})
	require.NoError(t, err)
	require.Equal(t, int64(800), account.TotalAmount)
	require.Equal(t, int64(800), account.AllocatedAmount)
	require.Zero(t, account.UnallocatedAmount)

	entries, total, err := service.ListEntries(ctx, BudgetListFilter{Page: 1, PageSize: 100})
	require.NoError(t, err)
	require.Equal(t, int64(6), total)
	require.Len(t, entries, 6)
	require.Equal(t, "REDUCE", entries[0].EntryType)
	require.Equal(t, int64(1), entries[0].OperatorID)
	require.NotEmpty(t, entries[0].Reason)
	require.False(t, entries[0].CreatedAt.IsZero())

	auditCount, err := client.GrowthAdminAudit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, auditCount)

	allocations, allocationTotal, err := service.ListAllocations(ctx, BudgetListFilter{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), allocationTotal)
	require.Len(t, allocations, 2)
}

func TestBudgetReduceHonorsDistributedAndAllocationLockOrderIsStable(t *testing.T) {
	client := openGrowthIntegrationClient(t)
	budget := NewBudgetService(client)
	transfers := NewTransferService(client)
	admin := NewAdminGrowthService(client, transfers)
	ctx := context.Background()

	_, err := budget.TopUp(ctx, BudgetMutation{
		OperatorID: 1, Points: 100, Reason: "账务不变量测试预算", RequestID: "invariant-topup",
	})
	require.NoError(t, err)
	task := createIntegrationTask(t, client, "invariant_task", "DRAFT")
	_, err = budget.Allocate(ctx, task.ID, BudgetMutation{
		OperatorID: 1, Points: 60, Reason: "任务初始额度", RequestID: "invariant-allocate",
	})
	require.NoError(t, err)
	_, err = transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID: 1, TaskID: task.ID, Points: 20, BizType: "TASK",
		IdempotencyKey: "invariant-award", Title: "不变量奖励",
	})
	require.NoError(t, err)
	_, err = client.GrowthTask.UpdateOneID(task.ID).SetTaskStatus("OFFLINE").Save(ctx)
	require.NoError(t, err)
	_, err = budget.Revoke(ctx, task.ID, BudgetMutation{
		OperatorID: 1, Points: 40, Reason: "回收未发放额度", RequestID: "invariant-revoke",
	})
	require.NoError(t, err)
	_, err = admin.Adjust(ctx, AdjustmentCommand{
		OperatorID: 1, UserID: 1, Direction: "INCREASE", Points: 30,
		Reason: "制造已发放高于已分配场景", RequestID: "invariant-adjust",
	})
	require.NoError(t, err)

	account, err := budget.GetAccount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(20), account.AllocatedAmount)
	require.Equal(t, int64(50), account.DistributedAmount)
	require.Equal(t, int64(50), account.LockedAmount)

	_, err = budget.Reduce(ctx, BudgetMutation{
		OperatorID: 1, Points: 60, Reason: "核减后低于已发放余额", RequestID: "invariant-reduce",
	})
	require.ErrorIs(t, err, ErrBudgetBelowLocked)

	lockTask := createIntegrationTask(t, client, "lock_order_task", "DRAFT")
	_, err = budget.Allocate(ctx, lockTask.ID, BudgetMutation{
		OperatorID: 1, Points: 40, Reason: "并发锁顺序额度", RequestID: "lock-order-initial",
	})
	require.NoError(t, err)
	start := make(chan struct{})
	errs := make(chan error, 12)
	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := transfers.AwardFromTask(ctx, TaskAwardCommand{
				UserID: 1, TaskID: lockTask.ID, Points: 5, BizType: "TASK",
				IdempotencyKey: fmt.Sprintf("lock-order-award-%d", index), Title: "并发发分",
			})
			errs <- err
		}(index)
	}
	for index := 0; index < 4; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := budget.Allocate(ctx, lockTask.ID, BudgetMutation{
				OperatorID: 1, Points: 5, Reason: "并发追加额度",
				RequestID: fmt.Sprintf("lock-order-allocate-%d", index),
			})
			errs <- err
		}(index)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func createIntegrationTask(t *testing.T, client *ent.Client, code, status string) *ent.GrowthTask {
	t.Helper()
	task, err := client.GrowthTask.Create().
		SetTaskCode(code).
		SetTitle(code).
		SetCategory("DAILY").
		SetTaskType("DAILY").
		SetTaskStatus(status).
		Save(context.Background())
	require.NoError(t, err)
	return task
}

func openGrowthIntegrationClient(t *testing.T) *ent.Client {
	t.Helper()
	baseDSN := strings.TrimSpace(os.Getenv("GROWTH_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		t.Skip("set GROWTH_MYSQL_TEST_DSN to run growth MySQL integration tests")
	}
	cfg, err := mysqlDriver.ParseDSN(baseDSN)
	require.NoError(t, err)
	cfg.DBName = ""
	adminDB, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = adminDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	require.NoError(t, adminDB.PingContext(ctx))
	databaseName := fmt.Sprintf("sub2api_budget_it_%d", time.Now().UnixNano())
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
	_, err = db.ExecContext(ctx, `CREATE TABLE users (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		username VARCHAR(100) NOT NULL DEFAULT '',
		deleted_at DATETIME(6) NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash) VALUES (1, 'operator@example.com', 'x')`)
	require.NoError(t, err)

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	archivePath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "sqlArchiving", "growth", "ddl", "182_create_growth_tables.sql")
	raw, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	for _, statement := range budgetSQLStatements(string(raw)) {
		_, err := db.ExecContext(ctx, statement)
		require.NoErrorf(t, err, "execute statement: %s", statement)
	}
	client, err := ent.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func budgetSQLStatements(raw string) []string {
	var result []string
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
		result = append(result, statement)
	}
	return result
}
