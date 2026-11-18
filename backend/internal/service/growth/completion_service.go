package growth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthcompletionrequest"
	"github.com/Wei-Shaw/sub2api/ent/growthrewardclaim"
	"github.com/Wei-Shaw/sub2api/ent/growthtask"
	"github.com/Wei-Shaw/sub2api/ent/growthtaskrule"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthusertask"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/google/uuid"
)

type CompletionResult struct {
	RequestID      string `json:"request_id"`
	Result         string `json:"result"`
	TaskID         int64  `json:"task_id"`
	CompletionRef  string `json:"completion_ref"`
	TransferID     *int64 `json:"transfer_id,omitempty"`
	AwardedPoints  int64  `json:"awarded_points"`
	CurrentBalance *int64 `json:"current_balance"`
	RuleVersion    int    `json:"rule_version"`
	CycleKey       string `json:"cycle_key"`
	ServerTime     string `json:"server_time"`
}

type CompletionCommand struct {
	UserID         int64
	TaskID         int64
	CompletionRef  string
	RequestID      string
	IdempotencyKey string
}

type CompletionService struct {
	client    *ent.Client
	transfers *TransferService
	location  *time.Location
	now       func() time.Time
}

func NewCompletionService(client *ent.Client, transfers *TransferService, cfg *config.Config) *CompletionService {
	timezone := "Asia/Shanghai"
	if cfg != nil && strings.TrimSpace(cfg.Timezone) != "" {
		timezone = strings.TrimSpace(cfg.Timezone)
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return &CompletionService{client: client, transfers: transfers, location: location, now: time.Now}
}

func (s *CompletionService) Complete(ctx context.Context, command CompletionCommand) (CompletionResult, error) {
	now := s.now().In(s.location)
	result := CompletionResult{
		RequestID:     command.RequestID,
		Result:        "NOT_ELIGIBLE",
		TaskID:        command.TaskID,
		CompletionRef: strings.TrimSpace(command.CompletionRef),
		ServerTime:    now.Format(time.RFC3339Nano),
	}
	if result.RequestID == "" {
		result.RequestID = "cr-" + uuid.NewString()
	}
	if command.UserID <= 0 || command.TaskID <= 0 || result.CompletionRef == "" ||
		strings.TrimSpace(command.IdempotencyKey) == "" {
		return result, infraerrors.BadRequest("INVALID_REQUEST", "user, task, completion_ref and idempotency_key are required")
	}
	if existing, err := s.existingRequest(ctx, command.UserID, command.TaskID, result.CompletionRef); err == nil {
		_, sameIdempotencyKey, transferErr := s.transfers.findExisting(ctx, "TASK", command.IdempotencyKey)
		if transferErr != nil {
			return CompletionResult{}, transferErr
		}
		if sameIdempotencyKey {
			result.Result = existing.Result
		} else {
			result.Result = "ALREADY_COMPLETED"
		}
		result.RuleVersion = existing.RuleVersion
		result.CycleKey = existing.CycleKey
		result.AwardedPoints = existing.AwardedPoints
		if existing.TransferID != nil {
			value := *existing.TransferID
			result.TransferID = &value
		}
		if balance, balanceErr := s.transfers.CurrentBalance(ctx, command.UserID); balanceErr == nil {
			result.CurrentBalance = &balance
		}
		if existing.RequestID != "" {
			result.RequestID = existing.RequestID
		}
		return result, nil
	} else if !ent.IsNotFound(err) {
		return CompletionResult{}, err
	}

	task, err := s.client.GrowthTask.Query().
		Where(growthtask.ID(command.TaskID), growthtask.StatusEQ("ACTIVE")).
		Only(ctx)
	if ent.IsNotFound(err) {
		return s.persistNonAward(ctx, command, result, now)
	}
	if err != nil {
		return CompletionResult{}, err
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
		return s.persistNonAward(ctx, command, result, now)
	}
	if err != nil {
		return CompletionResult{}, err
	}
	result.RuleVersion = rule.Version
	cycleKey, err := BusinessCycleKey(now, s.location, rule.PeriodType)
	if err != nil {
		return CompletionResult{}, err
	}
	result.CycleKey = cycleKey

	if claim, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(command.UserID),
			growthrewardclaim.TaskIDEQ(task.ID),
			growthrewardclaim.CycleKeyEQ(cycleKey),
		).
		Only(ctx); err == nil {
		transferID := claim.TransferID
		result.Result = "ALREADY_COMPLETED"
		result.TransferID = &transferID
		result.AwardedPoints = 0
		if balance, balanceErr := s.transfers.CurrentBalance(ctx, command.UserID); balanceErr == nil {
			result.CurrentBalance = &balance
		}
		return s.persistNonAward(ctx, command, result, now)
	} else if !ent.IsNotFound(err) {
		return CompletionResult{}, err
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	dailyClaims, err := s.client.GrowthRewardClaim.Query().
		Where(
			growthrewardclaim.UserIDEQ(command.UserID),
			growthrewardclaim.TaskIDEQ(task.ID),
			growthrewardclaim.ClaimedAtGTE(dayStart),
		).
		Count(ctx)
	if err != nil {
		return CompletionResult{}, err
	}
	perUserClaims, err := s.client.GrowthRewardClaim.Query().
		Where(growthrewardclaim.UserIDEQ(command.UserID), growthrewardclaim.TaskIDEQ(task.ID)).
		Count(ctx)
	if err != nil {
		return CompletionResult{}, err
	}
	if (rule.DailyLimit > 0 && dailyClaims >= rule.DailyLimit) ||
		(rule.PerUserLimit > 0 && perUserClaims >= rule.PerUserLimit) {
		result.Result = "NOT_ELIGIBLE"
		return s.persistNonAward(ctx, command, result, now)
	}

	var claimedID int64
	award, err := s.transfers.AwardFromTask(ctx, TaskAwardCommand{
		UserID:         command.UserID,
		TaskID:         task.ID,
		Points:         rule.RewardPoints,
		BizType:        "TASK",
		IdempotencyKey: command.IdempotencyKey,
		Title:          task.Title,
		ReferenceID:    task.ID,
		AfterLedger: func(callbackCtx context.Context, tx *ent.Tx) error {
			transfer, err := tx.GrowthTransfer.Query().
				Where(
					growthtransfer.BizTypeEQ("TASK"),
					growthtransfer.IdempotencyKeyEQ(command.IdempotencyKey),
				).
				Only(callbackCtx)
			if err != nil {
				return err
			}
			userTask, err := tx.GrowthUserTask.Query().
				Where(
					growthusertask.UserIDEQ(command.UserID),
					growthusertask.TaskIDEQ(task.ID),
					growthusertask.RuleIDEQ(rule.ID),
					growthusertask.CycleKeyEQ(cycleKey),
				).
				Only(callbackCtx)
			nextProgress := rule.Target
			completed := int64(1)
			if ent.IsNotFound(err) {
				_, err = tx.GrowthUserTask.Create().
					SetUserID(command.UserID).
					SetTaskID(task.ID).
					SetRuleID(rule.ID).
					SetCycleKey(cycleKey).
					SetProgress(nextProgress).
					SetTarget(rule.Target).
					SetCompletedCount(completed).
					SetStatus("REWARDED").
					SetCompletedAt(now).
					SetRewardedAt(now).
					Save(callbackCtx)
			} else if err == nil {
				completed = userTask.CompletedCount + 1
				nextProgress = userTask.Progress + 1
				if nextProgress < rule.Target {
					nextProgress = rule.Target
				}
				_, err = tx.GrowthUserTask.UpdateOneID(userTask.ID).
					SetProgress(nextProgress).
					SetCompletedCount(completed).
					SetStatus("REWARDED").
					SetCompletedAt(now).
					SetRewardedAt(now).
					Save(callbackCtx)
			}
			if err != nil {
				return err
			}
			claim, err := tx.GrowthRewardClaim.Create().
				SetUserID(command.UserID).
				SetTaskID(task.ID).
				SetRuleID(rule.ID).
				SetCycleKey(cycleKey).
				SetIdempotencyKey(command.IdempotencyKey).
				SetTransferID(transfer.ID).
				SetPoints(rule.RewardPoints).
				SetStatus("SUCCEEDED").
				SetClaimedAt(now).
				Save(callbackCtx)
			if err != nil {
				return err
			}
			claimedID = claim.ID
			_, err = tx.GrowthCompletionRequest.Create().
				SetUserID(command.UserID).
				SetTaskID(task.ID).
				SetCompletionRef(result.CompletionRef).
				SetRequestID(result.RequestID).
				SetResult("AWARDED").
				SetTransferID(transfer.ID).
				SetAwardedPoints(rule.RewardPoints).
				SetRuleVersion(rule.Version).
				SetCycleKey(cycleKey).
				Save(callbackCtx)
			return err
		},
	})
	if err != nil {
		if errors.Is(err, ErrPoolExhausted) {
			return CompletionResult{}, err
		}
		if isDuplicateKey(err) {
			if existing, existingErr := s.existingRequest(ctx, command.UserID, task.ID, result.CompletionRef); existingErr == nil {
				result.Result = existing.Result
				result.AwardedPoints = existing.AwardedPoints
				result.RuleVersion = existing.RuleVersion
				result.CycleKey = existing.CycleKey
				if existing.TransferID != nil {
					value := *existing.TransferID
					result.TransferID = &value
				}
				if balance, balanceErr := s.transfers.CurrentBalance(ctx, command.UserID); balanceErr == nil {
					result.CurrentBalance = &balance
				}
				return result, nil
			}
			result.Result = "ALREADY_COMPLETED"
			result.AwardedPoints = 0
			if balance, balanceErr := s.transfers.CurrentBalance(ctx, command.UserID); balanceErr == nil {
				result.CurrentBalance = &balance
			}
			return s.persistNonAward(ctx, command, result, now)
		}
		return CompletionResult{}, err
	}
	result.Result = "AWARDED"
	result.TransferID = &award.TransferID
	result.AwardedPoints = award.Points
	result.CurrentBalance = &award.CurrentBalance
	_ = claimedID
	return result, nil
}

func (s *CompletionService) existingRequest(ctx context.Context, userID, taskID int64, completionRef string) (*ent.GrowthCompletionRequest, error) {
	return s.client.GrowthCompletionRequest.Query().
		Where(
			growthcompletionrequest.UserIDEQ(userID),
			growthcompletionrequest.TaskIDEQ(taskID),
			growthcompletionrequest.CompletionRefEQ(completionRef),
		).
		Only(ctx)
}

func (s *CompletionService) persistNonAward(ctx context.Context, command CompletionCommand, result CompletionResult, now time.Time) (CompletionResult, error) {
	create := s.client.GrowthCompletionRequest.Create().
		SetUserID(command.UserID).
		SetTaskID(command.TaskID).
		SetCompletionRef(result.CompletionRef).
		SetRequestID(result.RequestID).
		SetResult(result.Result).
		SetAwardedPoints(result.AwardedPoints).
		SetRuleVersion(result.RuleVersion).
		SetCycleKey(result.CycleKey)
	if result.TransferID != nil {
		create.SetTransferID(*result.TransferID)
	}
	if _, err := create.Save(ctx); err != nil {
		if isDuplicateKey(err) {
			if existing, existingErr := s.existingRequest(ctx, command.UserID, command.TaskID, result.CompletionRef); existingErr == nil {
				result.Result = existing.Result
				result.AwardedPoints = existing.AwardedPoints
				result.RuleVersion = existing.RuleVersion
				result.CycleKey = existing.CycleKey
				if existing.TransferID != nil {
					value := *existing.TransferID
					result.TransferID = &value
				}
				return result, nil
			}
		}
		return CompletionResult{}, err
	}
	return result, nil
}
