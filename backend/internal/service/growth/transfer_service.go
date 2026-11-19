package growth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthbudget"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrPoolExhausted      = infraerrors.ServiceUnavailable("POOL_EXHAUSTED", "task quota is exhausted")
	ErrInsufficientPoints = infraerrors.Conflict("INSUFFICIENT_POINTS", "growth point balance is insufficient")
	ErrTransferReplay     = infraerrors.Conflict("IDEMPOTENCY_CONFLICT", "transfer business key has an incompatible result")
)

type TaskAwardCommand struct {
	UserID         int64
	TaskID         int64
	Points         int64
	BizType        string
	IdempotencyKey string
	Title          string
	ReferenceID    int64
	AfterLedger    func(context.Context, *ent.Tx) error
}

type WalletDebitCommand struct {
	UserID         int64
	Points         int64
	BizType        string
	IdempotencyKey string
	Title          string
	ReferenceID    int64
	ToAccountType  string
	IsAdjustment   bool
	AfterLedger    func(context.Context, *ent.Tx) error
}

type RedeemReverseCommand struct {
	UserID         int64
	Points         int64
	BizType        string
	IdempotencyKey string
	Title          string
	ReferenceID    int64
	AfterLedger    func(context.Context, *ent.Tx) error
}

type TransferResult struct {
	TransferID     int64
	UserID         int64
	Points         int64
	CurrentBalance int64
	Replayed       bool
}

type TransferService struct {
	client *ent.Client
}

func NewTransferService(client *ent.Client) *TransferService {
	return &TransferService{client: client}
}

func (s *TransferService) CurrentBalance(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, infraerrors.BadRequest("INVALID_USER_ID", "user_id must be positive")
	}
	wallet, err := s.client.GrowthWallet.Query().Where(growthwallet.UserIDEQ(userID)).Only(ctx)
	if ent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return wallet.AvailablePoints, nil
}

func (s *TransferService) AwardFromTask(ctx context.Context, command TaskAwardCommand) (TransferResult, error) {
	if err := validateTransferIdentity(command.UserID, command.Points, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	}
	if command.TaskID <= 0 {
		return TransferResult{}, infraerrors.BadRequest("INVALID_TASK_ID", "task_id must be positive")
	}
	if existing, ok, err := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	} else if ok {
		return existing, nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	result, err := s.awardFromTaskTx(ctx, tx, command)
	if err != nil {
		_ = tx.Rollback()
		if isDuplicateKey(err) {
			if existing, ok, readErr := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); readErr == nil && ok {
				existing.Replayed = true
				return existing, nil
			}
		}
		return TransferResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return TransferResult{}, err
	}
	return result, nil
}

func (s *TransferService) awardFromTaskTx(ctx context.Context, tx *ent.Tx, command TaskAwardCommand) (TransferResult, error) {
	transfer, err := createPendingTransfer(ctx, tx, command.BizType, command.IdempotencyKey, "TASK", "WALLET", nil, &command.UserID, &command.TaskID, command.Points)
	if err != nil {
		return TransferResult{}, err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE growth_tasks
SET distributed_points = distributed_points + ?, updated_at = CURRENT_TIMESTAMP(6)
WHERE id = ? AND allocated_points - distributed_points >= ?`,
		command.Points, command.TaskID, command.Points)
	if err != nil {
		return TransferResult{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return TransferResult{}, err
	}
	if affected == 0 {
		exists, existErr := tx.GrowthTask.Query().Where(growthtask.ID(command.TaskID)).Exist(ctx)
		if existErr != nil {
			return TransferResult{}, existErr
		}
		if !exists {
			return TransferResult{}, ErrGrowthTaskNotFound
		}
		return TransferResult{}, ErrPoolExhausted
	}
	task, err := tx.GrowthTask.Get(ctx, command.TaskID)
	if err != nil {
		return TransferResult{}, err
	}
	budget, err := tx.GrowthBudget.Query().Where(growthbudget.ID(1)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return TransferResult{}, ErrPoolExhausted
	}
	if err != nil {
		return TransferResult{}, err
	}
	if budget.DistributedAmount > budget.TotalAmount-command.Points {
		return TransferResult{}, ErrPoolExhausted
	}
	wallet, err := lockOrCreateWallet(ctx, tx, command.UserID)
	if err != nil {
		return TransferResult{}, err
	}

	if _, err := tx.GrowthBudget.UpdateOneID(1).
		SetDistributedAmount(budget.DistributedAmount + command.Points).
		SetVersion(budget.Version + 1).
		Save(ctx); err != nil {
		return TransferResult{}, err
	}
	wallet, err = tx.GrowthWallet.UpdateOneID(wallet.ID).
		SetAvailablePoints(wallet.AvailablePoints + command.Points).
		SetEarnedTotal(wallet.EarnedTotal + command.Points).
		SetVersion(wallet.Version + 1).
		SetBalanceUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	remainingTaskQuota := task.AllocatedPoints - task.DistributedPoints
	if err := createLedgerPair(
		ctx,
		tx,
		transfer.ID,
		command.BizType,
		command.Title,
		command.ReferenceID,
		"TASK",
		"OUT",
		nil,
		command.Points,
		int64Pointer(remainingTaskQuota),
		"WALLET",
		"IN",
		&command.UserID,
		command.Points,
		int64Pointer(wallet.AvailablePoints),
	); err != nil {
		return TransferResult{}, err
	}
	if command.AfterLedger != nil {
		if err := command.AfterLedger(ctx, tx); err != nil {
			return TransferResult{}, err
		}
	}
	transfer, err = finishTransfer(ctx, tx, transfer.ID)
	if err != nil {
		return TransferResult{}, err
	}
	return TransferResult{
		TransferID:     transfer.ID,
		UserID:         command.UserID,
		Points:         command.Points,
		CurrentBalance: wallet.AvailablePoints,
	}, nil
}

func (s *TransferService) DebitWallet(ctx context.Context, command WalletDebitCommand) (TransferResult, error) {
	if err := validateTransferIdentity(command.UserID, command.Points, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	}
	if existing, ok, err := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	} else if ok {
		return existing, nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	result, err := s.debitWalletTx(ctx, tx, command)
	if err != nil {
		_ = tx.Rollback()
		if isDuplicateKey(err) {
			if existing, ok, readErr := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); readErr == nil && ok {
				existing.Replayed = true
				return existing, nil
			}
		}
		return TransferResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return TransferResult{}, err
	}
	return result, nil
}

func (s *TransferService) debitWalletTx(ctx context.Context, tx *ent.Tx, command WalletDebitCommand) (TransferResult, error) {
	toAccount := strings.TrimSpace(command.ToAccountType)
	if toAccount == "" {
		toAccount = "REDEEMED"
	}
	transfer, err := createPendingTransfer(ctx, tx, command.BizType, command.IdempotencyKey, "WALLET", toAccount, &command.UserID, nil, nil, command.Points)
	if err != nil {
		return TransferResult{}, err
	}
	wallet, err := lockOrCreateWallet(ctx, tx, command.UserID)
	if err != nil {
		return TransferResult{}, err
	}
	if wallet.AvailablePoints < command.Points {
		return TransferResult{}, ErrInsufficientPoints
	}
	update := tx.GrowthWallet.UpdateOneID(wallet.ID).
		SetAvailablePoints(wallet.AvailablePoints - command.Points).
		SetVersion(wallet.Version + 1).
		SetBalanceUpdatedAt(time.Now())
	if !command.IsAdjustment {
		update.SetSpentTotal(wallet.SpentTotal + command.Points)
	}
	wallet, err = update.Save(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	if err := createLedgerPair(
		ctx,
		tx,
		transfer.ID,
		command.BizType,
		command.Title,
		command.ReferenceID,
		"WALLET",
		"OUT",
		&command.UserID,
		command.Points,
		int64Pointer(wallet.AvailablePoints),
		toAccount,
		"IN",
		&command.UserID,
		command.Points,
		nil,
	); err != nil {
		return TransferResult{}, err
	}
	if command.AfterLedger != nil {
		if err := command.AfterLedger(ctx, tx); err != nil {
			return TransferResult{}, err
		}
	}
	transfer, err = finishTransfer(ctx, tx, transfer.ID)
	if err != nil {
		return TransferResult{}, err
	}
	return TransferResult{
		TransferID:     transfer.ID,
		UserID:         command.UserID,
		Points:         command.Points,
		CurrentBalance: wallet.AvailablePoints,
	}, nil
}

func (s *TransferService) ReverseRedeem(ctx context.Context, command RedeemReverseCommand) (TransferResult, error) {
	if err := validateTransferIdentity(command.UserID, command.Points, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	}
	if existing, ok, err := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); err != nil {
		return TransferResult{}, err
	} else if ok {
		return existing, nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	result, err := s.reverseRedeemTx(ctx, tx, command)
	if err != nil {
		_ = tx.Rollback()
		if isDuplicateKey(err) {
			if existing, ok, readErr := s.findExisting(ctx, command.UserID, command.BizType, command.IdempotencyKey); readErr == nil && ok {
				existing.Replayed = true
				return existing, nil
			}
		}
		return TransferResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return TransferResult{}, err
	}
	return result, nil
}

func (s *TransferService) reverseRedeemTx(ctx context.Context, tx *ent.Tx, command RedeemReverseCommand) (TransferResult, error) {
	transfer, err := createPendingTransfer(ctx, tx, command.BizType, command.IdempotencyKey, "REDEEMED", "WALLET", nil, &command.UserID, nil, command.Points)
	if err != nil {
		return TransferResult{}, err
	}
	wallet, err := lockOrCreateWallet(ctx, tx, command.UserID)
	if err != nil {
		return TransferResult{}, err
	}
	nextSpent := wallet.SpentTotal - command.Points
	if nextSpent < 0 {
		nextSpent = 0
	}
	wallet, err = tx.GrowthWallet.UpdateOneID(wallet.ID).
		SetAvailablePoints(wallet.AvailablePoints + command.Points).
		SetSpentTotal(nextSpent).
		SetVersion(wallet.Version + 1).
		SetBalanceUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return TransferResult{}, err
	}
	if err := createLedgerPair(
		ctx,
		tx,
		transfer.ID,
		command.BizType,
		command.Title,
		command.ReferenceID,
		"REDEEMED",
		"OUT",
		&command.UserID,
		command.Points,
		nil,
		"WALLET",
		"IN",
		&command.UserID,
		command.Points,
		int64Pointer(wallet.AvailablePoints),
	); err != nil {
		return TransferResult{}, err
	}
	if command.AfterLedger != nil {
		if err := command.AfterLedger(ctx, tx); err != nil {
			return TransferResult{}, err
		}
	}
	transfer, err = finishTransfer(ctx, tx, transfer.ID)
	if err != nil {
		return TransferResult{}, err
	}
	return TransferResult{
		TransferID:     transfer.ID,
		UserID:         command.UserID,
		Points:         command.Points,
		CurrentBalance: wallet.AvailablePoints,
	}, nil
}

func createPendingTransfer(
	ctx context.Context,
	tx *ent.Tx,
	bizType, idempotencyKey, fromAccount, toAccount string,
	fromUserID, toUserID, taskID *int64,
	points int64,
) (*ent.GrowthTransfer, error) {
	create := tx.GrowthTransfer.Create().
		SetBizType(bizType).
		SetIdempotencyKey(idempotencyKey).
		SetFromAccountType(fromAccount).
		SetToAccountType(toAccount).
		SetPoints(points).
		SetStatus("PENDING")
	if fromUserID != nil {
		create.SetFromUserID(*fromUserID)
	}
	if toUserID != nil {
		create.SetToUserID(*toUserID)
	}
	if taskID != nil {
		create.SetTaskID(*taskID)
	}
	return create.Save(ctx)
}

func finishTransfer(ctx context.Context, tx *ent.Tx, transferID int64) (*ent.GrowthTransfer, error) {
	return tx.GrowthTransfer.UpdateOneID(transferID).
		SetStatus("SUCCEEDED").
		SetCompletedAt(time.Now()).
		Save(ctx)
}

func createLedgerPair(
	ctx context.Context,
	tx *ent.Tx,
	transferID int64,
	bizType, title string,
	referenceID int64,
	fromAccount, fromDirection string,
	fromUserID *int64,
	points int64,
	fromBalance *int64,
	toAccount, toDirection string,
	toUserID *int64,
	toPoints int64,
	toBalance *int64,
) error {
	create := func(accountType, direction string, userID *int64, amount int64, balance *int64) error {
		entry := tx.GrowthAccountLedger.Create().
			SetTransferID(transferID).
			SetAccountType(accountType).
			SetDirection(direction).
			SetPoints(amount).
			SetBizType(bizType).
			SetTitle(title)
		if userID != nil {
			entry.SetUserID(*userID)
		}
		if balance != nil {
			entry.SetBalanceAfter(*balance)
		}
		if referenceID > 0 {
			entry.SetReferenceID(referenceID)
		}
		_, err := entry.Save(ctx)
		return err
	}
	if err := create(fromAccount, fromDirection, fromUserID, points, fromBalance); err != nil {
		return err
	}
	if toPoints != points {
		return fmt.Errorf("growth transfer is unbalanced: %d != %d", points, toPoints)
	}
	return create(toAccount, toDirection, toUserID, toPoints, toBalance)
}

func lockOrCreateWallet(ctx context.Context, tx *ent.Tx, userID int64) (*ent.GrowthWallet, error) {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO growth_wallets (user_id, available_points, earned_total, spent_total, version, balance_updated_at, created_at, updated_at)
VALUES (?, 0, 0, 0, 1, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE user_id = user_id`, userID); err != nil {
		return nil, err
	}
	return tx.GrowthWallet.Query().Where(growthwallet.UserIDEQ(userID)).ForUpdate().Only(ctx)
}

func validateTransferIdentity(userID, points int64, bizType, key string) error {
	if userID <= 0 {
		return infraerrors.BadRequest("INVALID_USER_ID", "user_id must be positive")
	}
	if points <= 0 {
		return infraerrors.BadRequest("INVALID_POINTS", "points must be positive")
	}
	if strings.TrimSpace(bizType) == "" || strings.TrimSpace(key) == "" {
		return infraerrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "biz_type and idempotency_key are required")
	}
	return nil
}

func (s *TransferService) findExisting(ctx context.Context, userID int64, bizType, key string) (TransferResult, bool, error) {
	transfer, err := s.client.GrowthTransfer.Query().
		Where(
			growthtransfer.BizTypeEQ(bizType),
			growthtransfer.IdempotencyKeyEQ(key),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return TransferResult{}, false, nil
	}
	if err != nil {
		return TransferResult{}, false, err
	}
	if transfer.Status != "SUCCEEDED" {
		return TransferResult{}, false, ErrTransferReplay
	}
	transferUserID := int64(0)
	if transfer.ToUserID != nil {
		transferUserID = *transfer.ToUserID
	} else if transfer.FromUserID != nil {
		transferUserID = *transfer.FromUserID
	}
	if transferUserID != userID {
		return TransferResult{}, false, ErrTransferReplay
	}
	balance, err := s.CurrentBalance(ctx, userID)
	if err != nil {
		return TransferResult{}, false, err
	}
	return TransferResult{
		TransferID:     transfer.ID,
		UserID:         userID,
		Points:         transfer.Points,
		CurrentBalance: balance,
		Replayed:       true,
	}, true, nil
}

func BusinessCycleKey(now time.Time, location *time.Location, periodType string) (string, error) {
	if location == nil {
		location = time.UTC
	}
	local := now.In(location)
	switch periodType {
	case "DAILY", "CHECKIN":
		return local.Format("2006-01-02"), nil
	case "WEEKLY":
		year, week := local.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week), nil
	case "ONCE":
		return "once", nil
	default:
		return "", fmt.Errorf("unsupported growth period type %q", periodType)
	}
}

func LedgerSignedSum(entries []*ent.GrowthAccountLedger, accountType string) int64 {
	var total int64
	for _, entry := range entries {
		if entry.AccountType != accountType {
			continue
		}
		if entry.Direction == "IN" {
			total += entry.Points
		} else {
			total -= entry.Points
		}
	}
	return total
}

func int64Pointer(value int64) *int64 { return &value }
