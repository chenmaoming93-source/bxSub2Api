package growth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthadminaudit"
	"github.com/Wei-Shaw/sub2api/ent/growthbudget"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type AdminUserPoints struct {
	UserID          int64      `json:"user_id"`
	Email           string     `json:"email"`
	Username        string     `json:"username"`
	AvailablePoints int64      `json:"available_points"`
	TotalEarned     int64      `json:"total_earned"`
	TotalSpent      int64      `json:"total_spent"`
	LatestChangeAt  *time.Time `json:"latest_change_at,omitempty"`
}

type AdjustmentCommand struct {
	OperatorID int64
	UserID     int64
	Direction  string
	Points     int64
	Reason     string
	RequestID  string
}

type AdjustmentResult struct {
	UserID        int64  `json:"user_id"`
	Direction     string `json:"direction"`
	Points        int64  `json:"points"`
	BeforeBalance int64  `json:"before_balance"`
	AfterBalance  int64  `json:"after_balance"`
	TransferID    int64  `json:"transfer_id"`
}

type ReconciliationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ReconciliationReport struct {
	OK                bool                  `json:"ok"`
	CheckedAt         time.Time             `json:"checked_at"`
	TotalAmount       int64                 `json:"total_amount"`
	AllocatedAmount   int64                 `json:"allocated_amount"`
	DistributedAmount int64                 `json:"distributed_amount"`
	TaskAllocated     int64                 `json:"task_allocated"`
	TaskDistributed   int64                 `json:"task_distributed"`
	AdjustmentIn      int64                 `json:"adjustment_in"`
	AdjustmentOut     int64                 `json:"adjustment_out"`
	Issues            []ReconciliationIssue `json:"issues"`
}

type AdminGrowthService struct {
	client    *ent.Client
	transfers *TransferService
	now       func() time.Time
}

func NewAdminGrowthService(client *ent.Client, transfers *TransferService) *AdminGrowthService {
	return &AdminGrowthService{client: client, transfers: transfers, now: time.Now}
}

func (s *AdminGrowthService) ListUsers(ctx context.Context, page, pageSize int, search string, minPoints, maxPoints *int64) ([]AdminUserPoints, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	where := []string{"u.deleted_at IS NULL"}
	args := make([]any, 0, 5)
	if strings.TrimSpace(search) != "" {
		where = append(where, "(u.email LIKE ? OR u.username LIKE ?)")
		value := "%" + strings.TrimSpace(search) + "%"
		args = append(args, value, value)
	}
	if minPoints != nil {
		where = append(where, "COALESCE(w.available_points, 0) >= ?")
		args = append(args, *minPoints)
	}
	if maxPoints != nil {
		where = append(where, "COALESCE(w.available_points, 0) <= ?")
		args = append(args, *maxPoints)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	rows, err := s.client.QueryContext(ctx, `SELECT COUNT(*)
FROM users u LEFT JOIN growth_wallets w ON w.user_id = u.id
WHERE `+whereSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, 0, err
	}
	_ = rows.Close()
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err = s.client.QueryContext(ctx, `SELECT
  u.id, u.email, COALESCE(u.username, ''),
  COALESCE(w.available_points, 0), COALESCE(w.earned_total, 0), COALESCE(w.spent_total, 0),
  (SELECT MAX(created_at) FROM growth_account_ledger l
   WHERE l.user_id = u.id AND l.account_type = 'WALLET') AS latest_change_at
FROM users u LEFT JOIN growth_wallets w ON w.user_id = u.id
WHERE `+whereSQL+`
ORDER BY COALESCE(w.available_points, 0) DESC, u.id ASC
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]AdminUserPoints, 0)
	for rows.Next() {
		var item AdminUserPoints
		var username sql.NullString
		var latest sql.NullTime
		if err := rows.Scan(&item.UserID, &item.Email, &username, &item.AvailablePoints, &item.TotalEarned, &item.TotalSpent, &latest); err != nil {
			return nil, 0, err
		}
		item.Username = username.String
		if latest.Valid {
			value := latest.Time
			item.LatestChangeAt = &value
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *AdminGrowthService) Adjust(ctx context.Context, command AdjustmentCommand) (AdjustmentResult, error) {
	if command.OperatorID <= 0 || command.UserID <= 0 {
		return AdjustmentResult{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator and user identity are required")
	}
	if command.Points <= 0 || (command.Direction != "INCREASE" && command.Direction != "DECREASE") {
		return AdjustmentResult{}, infraerrors.BadRequest("INVALID_ADJUSTMENT", "direction and positive points are required")
	}
	if strings.TrimSpace(command.RequestID) == "" {
		return AdjustmentResult{}, infraerrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "request_id is required")
	}
	if err := validateReason(command.Reason); err != nil {
		return AdjustmentResult{}, err
	}
	if command.Direction == "DECREASE" {
		return s.decrease(ctx, command)
	}
	return s.increase(ctx, command)
}

func (s *AdminGrowthService) increase(ctx context.Context, command AdjustmentCommand) (AdjustmentResult, error) {
	key := adjustmentKey(command)
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return AdjustmentResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	budget, err := tx.GrowthBudget.Query().Where(growthbudget.ID(1)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return AdjustmentResult{}, ErrBudgetInsufficient
	}
	if err != nil {
		return AdjustmentResult{}, err
	}
	if budget.TotalAmount-budget.AllocatedAmount < command.Points {
		return AdjustmentResult{}, ErrBudgetInsufficient
	}
	wallet, err := lockOrCreateWallet(ctx, tx, command.UserID)
	if err != nil {
		return AdjustmentResult{}, err
	}
	before := wallet.AvailablePoints
	transfer, err := createPendingTransfer(ctx, tx, "ADJUST", key, "SYSTEM", "WALLET", nil, &command.UserID, nil, command.Points)
	if err != nil {
		return AdjustmentResult{}, err
	}
	wallet, err = tx.GrowthWallet.UpdateOneID(wallet.ID).
		SetAvailablePoints(wallet.AvailablePoints + command.Points).
		SetEarnedTotal(wallet.EarnedTotal + command.Points).
		SetVersion(wallet.Version + 1).
		SetBalanceUpdatedAt(s.now()).
		Save(ctx)
	if err != nil {
		return AdjustmentResult{}, err
	}
	if _, err := tx.GrowthBudget.UpdateOneID(1).
		SetDistributedAmount(budget.DistributedAmount + command.Points).
		SetVersion(budget.Version + 1).
		Save(ctx); err != nil {
		return AdjustmentResult{}, err
	}
	if err := createLedgerPair(ctx, tx, transfer.ID, "ADJUST", "人工调账增加",
		0, "SYSTEM", "OUT", nil, command.Points, nil,
		"WALLET", "IN", &command.UserID, command.Points, &wallet.AvailablePoints); err != nil {
		return AdjustmentResult{}, err
	}
	transfer, err = finishTransfer(ctx, tx, transfer.ID)
	if err != nil {
		return AdjustmentResult{}, err
	}
	if err := appendAdjustmentAudit(ctx, tx, command, before, wallet.AvailablePoints); err != nil {
		return AdjustmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return AdjustmentResult{}, err
	}
	committed = true
	return AdjustmentResult{
		UserID: command.UserID, Direction: command.Direction, Points: command.Points,
		BeforeBalance: before, AfterBalance: wallet.AvailablePoints, TransferID: transfer.ID,
	}, nil
}

func (s *AdminGrowthService) decrease(ctx context.Context, command AdjustmentCommand) (AdjustmentResult, error) {
	key := adjustmentKey(command)
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return AdjustmentResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	wallet, err := lockOrCreateWallet(ctx, tx, command.UserID)
	if err != nil {
		return AdjustmentResult{}, err
	}
	before := wallet.AvailablePoints
	if before < command.Points {
		return AdjustmentResult{}, ErrInsufficientPoints
	}
	var after int64
	_, err = s.transfers.debitWalletTx(ctx, tx, WalletDebitCommand{
		UserID: command.UserID, Points: command.Points, BizType: "ADJUST",
		IdempotencyKey: key, Title: "人工调账扣减",
		ToAccountType: "ADJUST", IsAdjustment: true,
		AfterLedger: func(callbackCtx context.Context, callbackTx *ent.Tx) error {
			current, err := callbackTx.GrowthWallet.Query().Where(growthwallet.UserIDEQ(command.UserID)).Only(callbackCtx)
			if err != nil {
				return err
			}
			after = current.AvailablePoints
			return appendAdjustmentAudit(callbackCtx, callbackTx, command, before, after)
		},
	})
	if err != nil {
		return AdjustmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return AdjustmentResult{}, err
	}
	committed = true
	transfer, err := s.client.GrowthTransfer.Query().
		Where(
			growthtransfer.BizTypeEQ("ADJUST"),
			growthtransfer.IdempotencyKeyEQ(key),
		).
		Only(ctx)
	if err != nil {
		return AdjustmentResult{}, err
	}
	return AdjustmentResult{
		UserID: command.UserID, Direction: command.Direction, Points: command.Points,
		BeforeBalance: before, AfterBalance: after, TransferID: transfer.ID,
	}, nil
}

func appendAdjustmentAudit(ctx context.Context, tx *ent.Tx, command AdjustmentCommand, before, after int64) error {
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(command.OperatorID).
		SetAction("points.adjust." + strings.ToLower(command.Direction)).
		SetResourceType("growth_wallet").
		SetResourceID(command.UserID).
		SetBeforeJSON(map[string]any{"available_points": before}).
		SetAfterJSON(map[string]any{"available_points": after}).
		SetReason(strings.TrimSpace(command.Reason)).
		SetRequestID(strings.TrimSpace(command.RequestID)).
		Save(ctx)
	return err
}

func adjustmentKey(command AdjustmentCommand) string {
	if strings.TrimSpace(command.RequestID) != "" {
		return "adjust:" + command.RequestID
	}
	return fmt.Sprintf("adjust:%d:%d:%d", command.UserID, command.OperatorID, time.Now().UnixNano())
}

func (s *AdminGrowthService) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	report := ReconciliationReport{OK: true, CheckedAt: s.now(), Issues: []ReconciliationIssue{}}
	budget, err := s.client.GrowthBudget.Query().Where(growthbudget.ID(1)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return report, err
	}
	if budget != nil {
		report.TotalAmount = budget.TotalAmount
		report.AllocatedAmount = budget.AllocatedAmount
		report.DistributedAmount = budget.DistributedAmount
	}
	var ledgerTotal int64
	totalRows, err := s.client.QueryContext(ctx, `SELECT COALESCE(SUM(CASE
WHEN entry_type='TOPUP' THEN points WHEN entry_type='REDUCE' THEN -points ELSE 0 END),0)
FROM growth_budget_entries`)
	if err != nil {
		return report, err
	}
	if totalRows.Next() {
		if err := totalRows.Scan(&ledgerTotal); err != nil {
			_ = totalRows.Close()
			return report, err
		}
	}
	_ = totalRows.Close()
	if ledgerTotal != report.TotalAmount {
		report.Issues = append(report.Issues, ReconciliationIssue{Code: "BUDGET_TOTAL_MISMATCH", Message: "budget total differs from TOPUP-REDUCE entries"})
	}
	taskRows, err := s.client.QueryContext(ctx, `SELECT COALESCE(SUM(allocated_points),0), COALESCE(SUM(distributed_points),0)
FROM growth_tasks`)
	if err != nil {
		return report, err
	}
	if taskRows.Next() {
		if err := taskRows.Scan(&report.TaskAllocated, &report.TaskDistributed); err != nil {
			_ = taskRows.Close()
			return report, err
		}
	}
	_ = taskRows.Close()
	if report.TaskAllocated != report.AllocatedAmount {
		report.Issues = append(report.Issues, ReconciliationIssue{Code: "TASK_ALLOCATION_MISMATCH", Message: "task allocated points differ from budget allocated amount"})
	}
	adjustRows, err := s.client.QueryContext(ctx, `SELECT COALESCE(SUM(points),0)
FROM growth_transfers
WHERE biz_type='ADJUST' AND from_account_type='SYSTEM' AND to_account_type='WALLET' AND status='SUCCEEDED'`)
	if err != nil {
		return report, err
	}
	if adjustRows.Next() {
		if err := adjustRows.Scan(&report.AdjustmentIn); err != nil {
			_ = adjustRows.Close()
			return report, err
		}
	}
	_ = adjustRows.Close()
	if report.TaskDistributed+report.AdjustmentIn != report.DistributedAmount {
		report.Issues = append(report.Issues, ReconciliationIssue{Code: "TASK_DISTRIBUTION_MISMATCH", Message: "task distributed points differ from budget distributed amount"})
	}
	adjustOutRows, err := s.client.QueryContext(ctx, `SELECT COALESCE(SUM(points),0)
FROM growth_transfers
WHERE biz_type='ADJUST' AND from_account_type='WALLET' AND to_account_type='ADJUST' AND status='SUCCEEDED'`)
	if err != nil {
		return report, err
	}
	if adjustOutRows.Next() {
		if err := adjustOutRows.Scan(&report.AdjustmentOut); err != nil {
			_ = adjustOutRows.Close()
			return report, err
		}
	}
	_ = adjustOutRows.Close()
	rows, err := s.client.QueryContext(ctx, `SELECT w.user_id, w.available_points,
COALESCE(SUM(CASE WHEN l.direction='IN' THEN l.points ELSE -l.points END),0)
FROM growth_wallets w
LEFT JOIN growth_account_ledger l
  ON l.user_id=w.user_id AND l.account_type='WALLET'
GROUP BY w.user_id,w.available_points`)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var userID, balance, ledger int64
		if err := rows.Scan(&userID, &balance, &ledger); err != nil {
			_ = rows.Close()
			return report, err
		}
		if balance != ledger {
			report.Issues = append(report.Issues, ReconciliationIssue{Code: "WALLET_LEDGER_MISMATCH", Message: fmt.Sprintf("user %d wallet=%d ledger=%d", userID, balance, ledger)})
		}
	}
	_ = rows.Close()
	badTransfers, err := s.client.QueryContext(ctx, `SELECT transfer_id
FROM growth_account_ledger
GROUP BY transfer_id
HAVING COUNT(*) <> 2 OR SUM(CASE WHEN direction='IN' THEN points ELSE -points END) <> 0`)
	if err != nil {
		return report, err
	}
	for badTransfers.Next() {
		var transferID int64
		if err := badTransfers.Scan(&transferID); err != nil {
			_ = badTransfers.Close()
			return report, err
		}
		report.Issues = append(report.Issues, ReconciliationIssue{Code: "TRANSFER_ENTRY_MISMATCH", Message: fmt.Sprintf("transfer %d is not a balanced pair", transferID)})
	}
	_ = badTransfers.Close()
	report.OK = len(report.Issues) == 0
	return report, nil
}

func (s *AdminGrowthService) AuditLogs(ctx context.Context, page, pageSize int, action string) ([]*ent.GrowthAdminAudit, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.GrowthAdminAudit.Query()
	if strings.TrimSpace(action) != "" {
		query = query.Where(growthadminaudit.ActionEQ(strings.TrimSpace(action)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.Order(ent.Desc(growthadminaudit.FieldCreatedAt)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	return items, int64(total), err
}
