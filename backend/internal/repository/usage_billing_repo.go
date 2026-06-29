package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	res, err := tx.ExecContext(ctx, `
		INSERT IGNORE INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES (?, ?, ?)
	`, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = ? AND api_key_id = ?
		`, cmd.RequestID, cmd.APIKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = ? AND api_key_id = ?
	`, cmd.RequestID, cmd.APIKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if err := incrementUsageBillingSubscription(ctx, tx, *cmd.SubscriptionID, cmd.SubscriptionCost); err != nil {
			return err
		}
	}

	if cmd.BalanceCost > 0 {
		newBalance, err := deductUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
		if err != nil {
			return err
		}
		result.NewBalance = &newBalance
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		quotaState, err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost)
		if err != nil {
			return err
		}
		result.QuotaState = quotaState
	}

	return nil
}

func incrementUsageBillingSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_usd = us.daily_usage_usd + ?,
			weekly_usage_usd = us.weekly_usage_usd + ?,
			monthly_usage_usd = us.monthly_usage_usd + ?,
			updated_at = NOW()
		FROM groups g
		WHERE us.id = ?
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, costUSD, subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func deductUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (float64, error) {
	res, err := tx.ExecContext(ctx, `
		UPDATE users
		SET balance = balance - ?,
			updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
	`, amount, userID)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return 0, service.ErrUserNotFound
	}
	var newBalance float64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = ? AND deleted_at IS NULL`, userID).Scan(&newBalance); err != nil {
		return 0, err
	}
	return newBalance, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var prevUsed, quota float64
	if err := tx.QueryRowContext(ctx, `
		SELECT quota_used, quota
		FROM api_keys
		WHERE id = ? AND deleted_at IS NULL
		FOR UPDATE
	`, apiKeyID).Scan(&prevUsed, &quota); errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	} else if err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + ?,
			status = CASE
				WHEN quota > 0
					AND status = ?
					AND quota_used < quota
					AND quota_used + ? >= quota
				THEN ?
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
	`, amount, amount, service.StatusAPIKeyActive, amount, service.StatusAPIKeyQuotaExhausted, apiKeyID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return false, service.ErrAPIKeyNotFound
	}
	return quota > 0 && prevUsed < quota && prevUsed+amount >= quota, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND DATE_ADD(window_5h_start, INTERVAL 5 HOUR) <= NOW() THEN ? ELSE usage_5h + ? END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND DATE_ADD(window_1d_start, INTERVAL 24 HOUR) <= NOW() THEN ? ELSE usage_1d + ? END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND DATE_ADD(window_7d_start, INTERVAL 7 DAY) <= NOW() THEN ? ELSE usage_7d + ? END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR DATE_ADD(window_5h_start, INTERVAL 5 HOUR) <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR DATE_ADD(window_1d_start, INTERVAL 24 HOUR) <= NOW() THEN DATE(NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR DATE_ADD(window_7d_start, INTERVAL 7 DAY) <= NOW() THEN DATE(NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) (*service.AccountQuotaState, error) {
	res, err := tx.ExecContext(ctx,
		`UPDATE accounts SET extra = JSON_SET(
			COALESCE(extra, JSON_OBJECT()),
			'$.quota_used', CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_used')), ''), '0') AS DECIMAL(20,10)) + ?,
			'$.quota_daily_used',
				CASE WHEN CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_limit')), ''), '0') AS DECIMAL(20,10)) > 0
					THEN CASE WHEN `+dailyExpiredExpr+` THEN ? ELSE CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_used')), ''), '0') AS DECIMAL(20,10)) + ? END
					ELSE CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_used')), ''), '0') AS DECIMAL(20,10)) END,
			'$.quota_daily_start',
				CASE WHEN CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_limit')), ''), '0') AS DECIMAL(20,10)) > 0 AND `+dailyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_start')), `+nowUTC+`) END,
			'$.quota_weekly_used',
				CASE WHEN CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_limit')), ''), '0') AS DECIMAL(20,10)) > 0
					THEN CASE WHEN `+weeklyExpiredExpr+` THEN ? ELSE CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_used')), ''), '0') AS DECIMAL(20,10)) + ? END
					ELSE CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_used')), ''), '0') AS DECIMAL(20,10)) END,
			'$.quota_weekly_start',
				CASE WHEN CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_limit')), ''), '0') AS DECIMAL(20,10)) > 0 AND `+weeklyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_start')), `+nowUTC+`) END
		), updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL`,
		amount, amount, amount, amount, amount, accountID)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, service.ErrAccountNotFound
	}

	var state service.AccountQuotaState
	if err := tx.QueryRowContext(ctx, `
SELECT
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_used')), ''), '0') AS DECIMAL(20,10)),
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_limit')), ''), '0') AS DECIMAL(20,10)),
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_used')), ''), '0') AS DECIMAL(20,10)),
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_daily_limit')), ''), '0') AS DECIMAL(20,10)),
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_used')), ''), '0') AS DECIMAL(20,10)),
	CAST(COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(extra, '$.quota_weekly_limit')), ''), '0') AS DECIMAL(20,10))
FROM accounts
WHERE id = ? AND deleted_at IS NULL`, accountID).Scan(
		&state.TotalUsed, &state.TotalLimit,
		&state.DailyUsed, &state.DailyLimit,
		&state.WeeklyUsed, &state.WeeklyLimit,
	); err != nil {
		return nil, err
	}

	crossedTotal := state.TotalLimit > 0 && state.TotalUsed >= state.TotalLimit && (state.TotalUsed-amount) < state.TotalLimit
	crossedDaily := state.DailyLimit > 0 && state.DailyUsed >= state.DailyLimit && (state.DailyUsed-amount) < state.DailyLimit
	crossedWeekly := state.WeeklyLimit > 0 && state.WeeklyUsed >= state.WeeklyLimit && (state.WeeklyUsed-amount) < state.WeeklyLimit
	if crossedTotal || crossedDaily || crossedWeekly {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return nil, err
		}
	}
	return &state, nil
}