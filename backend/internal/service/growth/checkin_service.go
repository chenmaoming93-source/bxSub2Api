package growth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthrewardclaim"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	"github.com/Wei-Shaw/sub2api/ent/growthtaskrule"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthusertask"
	"github.com/Wei-Shaw/sub2api/internal/config"
)

type CheckinResult struct {
	Result         string `json:"result"`
	ClaimID        int64  `json:"claim_id"`
	TransferID     *int64 `json:"transfer_id,omitempty"`
	BusinessDate   string `json:"business_date"`
	Timezone       string `json:"timezone"`
	AwardedPoints  int64  `json:"awarded_points"`
	CurrentBalance *int64 `json:"current_balance"`
	RuleVersion    int    `json:"rule_version"`
	ServerTime     string `json:"server_time"`
}

type CheckinService struct {
	client    *ent.Client
	transfers *TransferService
	location  *time.Location
	timezone  string
	now       func() time.Time
}

func NewCheckinService(client *ent.Client, transfers *TransferService, cfg *config.Config) *CheckinService {
	timezone := "Asia/Shanghai"
	if cfg != nil && strings.TrimSpace(cfg.Timezone) != "" {
		timezone = strings.TrimSpace(cfg.Timezone)
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return &CheckinService{client: client, transfers: transfers, location: location, timezone: timezone, now: time.Now}
}

func (s *CheckinService) Checkin(ctx context.Context, userID int64, idempotencyKey string) (CheckinResult, error) {
	now := s.now().In(s.location)
	result := CheckinResult{
		Result:       "NOT_ELIGIBLE",
		BusinessDate: now.Format("2006-01-02"),
		Timezone:     s.timezone,
		ServerTime:   now.Format(time.RFC3339Nano),
	}
	if userID <= 0 {
		return result, nil
	}
	task, err := s.client.GrowthTask.Query().
		Where(growthtask.TaskCodeEQ("daily_checkin"), growthtask.StatusEQ("ACTIVE")).
		Only(ctx)
	if ent.IsNotFound(err) {
		return result, nil
	}
	if err != nil {
		return CheckinResult{}, err
	}
	rule, err := s.client.GrowthTaskRule.Query().
		Where(
			growthtaskrule.TaskIDEQ(task.ID),
			growthtaskrule.PublishedAtNotNil(),
			growthtaskrule.EffectiveFromLTE(now),
			growthtaskrule.Or(growthtaskrule.EffectiveToIsNil(), growthtaskrule.EffectiveToGT(now)),
		).
		Order(ent.Desc(growthtaskrule.FieldVersion)).
		First(ctx)
	if ent.IsNotFound(err) {
		return result, nil
	}
	if err != nil {
		return CheckinResult{}, err
	}
	result.RuleVersion = rule.Version
	cycleKey, err := BusinessCycleKey(now, s.location, "DAILY")
	if err != nil {
		return CheckinResult{}, err
	}

	if existing, found, err := s.transfers.findExisting(ctx, userID, "CHECKIN", idempotencyKey); err != nil {
		return CheckinResult{}, err
	} else if found {
		result.Result = "AWARDED"
		result.TransferID = &existing.TransferID
		result.AwardedPoints = existing.Points
		result.CurrentBalance = &existing.CurrentBalance
		if claim, claimErr := s.client.GrowthRewardClaim.Query().
			Where(
				growthrewardclaim.UserIDEQ(userID),
				growthrewardclaim.TaskIDEQ(task.ID),
				growthrewardclaim.CycleKeyEQ(cycleKey),
			).
			Only(ctx); claimErr == nil {
			result.ClaimID = claim.ID
		}
		return result, nil
	}

	existingClaim, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(userID),
			growthrewardclaim.TaskIDEQ(task.ID),
			growthrewardclaim.CycleKeyEQ(cycleKey),
		).
		Only(ctx)
	if err == nil {
		result.Result = "ALREADY_CLAIMED"
		result.ClaimID = existingClaim.ID
		transferID := existingClaim.TransferID
		result.TransferID = &transferID
		if balance, balanceErr := s.transfers.CurrentBalance(ctx, userID); balanceErr == nil {
			result.CurrentBalance = &balance
		}
		return result, nil
	}
	if !ent.IsNotFound(err) {
		return CheckinResult{}, err
	}

	var claimedID int64
	award, err := s.transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID:         userID,
		TaskID:         task.ID,
		Points:         rule.RewardPoints,
		BizType:        "CHECKIN",
		IdempotencyKey: idempotencyKey,
		Title:          "每日签到",
		ReferenceID:    task.ID,
		AfterLedger: func(callbackCtx context.Context, tx *ent.Tx) error {
			transfer, err := tx.GrowthTransfer.Query().
				Where(
					growthtransfer.BizTypeEQ("CHECKIN"),
					growthtransfer.IdempotencyKeyEQ(idempotencyKey),
				).
				Only(callbackCtx)
			if err != nil {
				return err
			}
			userTask, err := tx.GrowthUserTask.Query().
				Where(
					growthusertask.UserIDEQ(userID),
					growthusertask.TaskIDEQ(task.ID),
					growthusertask.RuleIDEQ(rule.ID),
					growthusertask.CycleKeyEQ(cycleKey),
				).
				Only(callbackCtx)
			if ent.IsNotFound(err) {
				_, err = tx.GrowthUserTask.Create().
					SetUserID(userID).
					SetTaskID(task.ID).
					SetRuleID(rule.ID).
					SetCycleKey(cycleKey).
					SetProgress(rule.Target).
					SetTarget(rule.Target).
					SetCompletedCount(1).
					SetStatus("REWARDED").
					SetCompletedAt(now).
					SetRewardedAt(now).
					Save(callbackCtx)
			} else if err == nil {
				_, err = tx.GrowthUserTask.UpdateOneID(userTask.ID).
					SetProgress(rule.Target).
					SetTarget(rule.Target).
					SetCompletedCount(userTask.CompletedCount + 1).
					SetStatus("REWARDED").
					SetCompletedAt(now).
					SetRewardedAt(now).
					Save(callbackCtx)
			}
			if err != nil {
				return err
			}
			claim, err := tx.GrowthRewardClaim.Create().
				SetUserID(userID).
				SetTaskID(task.ID).
				SetRuleID(rule.ID).
				SetCycleKey(cycleKey).
				SetSequence(1).
				SetIdempotencyKey(idempotencyKey).
				SetTransferID(transfer.ID).
				SetPoints(rule.RewardPoints).
				SetStatus("SUCCEEDED").
				SetClaimedAt(now).
				Save(callbackCtx)
			if err != nil {
				return err
			}
			claimedID = claim.ID
			return nil
		},
	})
	if err != nil {
		if errors.Is(err, ErrPoolExhausted) {
			result.Result = "UNAVAILABLE"
			if balance, balanceErr := s.transfers.CurrentBalance(ctx, userID); balanceErr == nil {
				result.CurrentBalance = &balance
			}
			return result, nil
		}
		if isDuplicateKey(err) {
			if claim, claimErr := s.client.GrowthRewardClaim.Query().
				Where(
					growthrewardclaim.UserIDEQ(userID),
					growthrewardclaim.TaskIDEQ(task.ID),
					growthrewardclaim.CycleKeyEQ(cycleKey),
				).
				Only(ctx); claimErr == nil {
				result.Result = "ALREADY_CLAIMED"
				result.ClaimID = claim.ID
				transferID := claim.TransferID
				result.TransferID = &transferID
				if balance, balanceErr := s.transfers.CurrentBalance(ctx, userID); balanceErr == nil {
					result.CurrentBalance = &balance
				}
				return result, nil
			}
		}
		return CheckinResult{}, err
	}
	result.Result = "AWARDED"
	result.ClaimID = claimedID
	result.TransferID = &award.TransferID
	result.AwardedPoints = award.Points
	result.CurrentBalance = &award.CurrentBalance
	return result, nil
}
