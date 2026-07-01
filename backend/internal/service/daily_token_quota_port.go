package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrModelDailyTokenQuotaExhausted          = errors.New("model daily token quota exhausted")
	ErrUserModelDailyTokenQuotaExhausted      = errors.New("user model daily token quota exhausted")
	ErrGroupCandidateDailyTokenQuotaExhausted = errors.New("group candidate daily token quota exhausted")
)

type ModelDailyTokenQuotaKey struct {
	Model string
	At    time.Time
}

type UserModelDailyTokenQuotaKey struct {
	UserID int64
	Model  string
	At     time.Time
}

type GroupCandidateDailyTokenQuotaKey struct {
	GroupID       int64
	RouteAlias    string
	UpstreamModel string
	At            time.Time
}

type DailyTokenQuotaSnapshot struct {
	Exists           bool
	UsageDate        time.Time
	UsedTokens       int64
	DailyLimitTokens *int64
}

type DailyTokenQuotaRepository interface {
	GetModelDailyTokenQuota(context.Context, ModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error)
	GetUserModelDailyTokenQuota(context.Context, UserModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error)
	GetGroupCandidateDailyTokenQuota(context.Context, GroupCandidateDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error)
	IncrementDailyTokenQuotas(context.Context, DailyTokenQuotaIncrement) error
}

type DailyTokenQuotaIncrement struct {
	ModelKey          ModelDailyTokenQuotaKey
	UserModelKey      UserModelDailyTokenQuotaKey
	GroupCandidateKey GroupCandidateDailyTokenQuotaKey
	Tokens            int64
}

type ModelDailyTokenQuotaRecord struct {
	Model            string
	UsageDate        time.Time
	UsedTokens       int64
	DailyLimitTokens *int64
}

type UserModelDailyTokenQuotaRecord struct {
	UserID           int64
	Model            string
	UsageDate        time.Time
	UsedTokens       int64
	DailyLimitTokens *int64
}

type UserModelDailyTokenQuotaInput struct {
	Model            string
	DailyLimitTokens *int64
}

type ModelTokenQuotaAdminRepository interface {
	ListModelDailyTokenQuotas(context.Context, time.Time) ([]ModelDailyTokenQuotaRecord, error)
	SetModelDailyTokenQuota(context.Context, string, time.Time, *int64) (ModelDailyTokenQuotaRecord, error)
}

type ModelDailyTokenQuotaCacheInvalidator interface {
	InvalidateModelDailyTokenQuota(context.Context, ModelDailyTokenQuotaKey) error
}

type UserModelTokenQuotaAdminRepository interface {
	ListUserModelDailyTokenQuotas(context.Context, int64, time.Time) ([]UserModelDailyTokenQuotaRecord, error)
	UpsertUserModelDailyTokenQuotas(context.Context, int64, time.Time, []UserModelDailyTokenQuotaInput) ([]UserModelDailyTokenQuotaRecord, error)
}

type UserModelDailyTokenQuotaCacheInvalidator interface {
	InvalidateUserModelDailyTokenQuota(context.Context, UserModelDailyTokenQuotaKey) error
}

type DailyTokenQuotaExhaustedError struct {
	Scope       string
	UserID      int64
	GroupID     int64
	RouteAlias  string
	Model       string
	UsageDate   time.Time
	UsedTokens  int64
	LimitTokens int64
}

func (e *DailyTokenQuotaExhaustedError) Error() string {
	return fmt.Sprintf("%s daily token quota exhausted: user=%d group=%d route=%q model=%q date=%s used=%d limit=%d",
		e.Scope, e.UserID, e.GroupID, e.RouteAlias, e.Model, e.UsageDate.Format("2006-01-02"), e.UsedTokens, e.LimitTokens)
}

func (e *DailyTokenQuotaExhaustedError) Unwrap() error {
	switch e.Scope {
	case "user_model":
		return ErrUserModelDailyTokenQuotaExhausted
	case "group_candidate":
		return ErrGroupCandidateDailyTokenQuotaExhausted
	default:
		return ErrModelDailyTokenQuotaExhausted
	}
}

func CheckModelDailyTokenQuota(key ModelDailyTokenQuotaKey, snapshot DailyTokenQuotaSnapshot) error {
	return checkDailyTokenQuota("model", 0, 0, "", key.Model, snapshot)
}

func CheckUserModelDailyTokenQuota(key UserModelDailyTokenQuotaKey, snapshot DailyTokenQuotaSnapshot) error {
	return checkDailyTokenQuota("user_model", key.UserID, 0, "", key.Model, snapshot)
}

func CheckGroupCandidateDailyTokenQuota(key GroupCandidateDailyTokenQuotaKey, snapshot DailyTokenQuotaSnapshot) error {
	return checkDailyTokenQuota("group_candidate", 0, key.GroupID, key.RouteAlias, key.UpstreamModel, snapshot)
}

// CheckRouteCandidateDailyTokenQuotas applies all platform-independent quota
// gates for one concrete routing candidate.
func CheckRouteCandidateDailyTokenQuotas(ctx context.Context, repo DailyTokenQuotaRepository, groupID int64, routeAlias, upstreamModel string, userID int64) (bool, error) {
	if repo == nil || groupID <= 0 || routeAlias == "" || upstreamModel == "" {
		return false, nil
	}
	at := time.Now()
	groupKey := GroupCandidateDailyTokenQuotaKey{GroupID: groupID, RouteAlias: routeAlias, UpstreamModel: upstreamModel, At: at}
	groupSnapshot, err := repo.GetGroupCandidateDailyTokenQuota(ctx, groupKey)
	if err != nil {
		return false, err
	}
	if err := CheckGroupCandidateDailyTokenQuota(groupKey, groupSnapshot); err != nil {
		return dailyTokenQuotaCheckResult(err)
	}

	modelKey := ModelDailyTokenQuotaKey{Model: upstreamModel, At: at}
	modelSnapshot, err := repo.GetModelDailyTokenQuota(ctx, modelKey)
	if err != nil {
		return false, err
	}
	if err := CheckModelDailyTokenQuota(modelKey, modelSnapshot); err != nil {
		return dailyTokenQuotaCheckResult(err)
	}

	if userID <= 0 {
		return false, nil
	}
	userKey := UserModelDailyTokenQuotaKey{UserID: userID, Model: upstreamModel, At: at}
	userSnapshot, err := repo.GetUserModelDailyTokenQuota(ctx, userKey)
	if err != nil {
		return false, err
	}
	if err := CheckUserModelDailyTokenQuota(userKey, userSnapshot); err != nil {
		return dailyTokenQuotaCheckResult(err)
	}
	return false, nil
}

func dailyTokenQuotaCheckResult(err error) (bool, error) {
	if errors.Is(err, ErrGroupCandidateDailyTokenQuotaExhausted) ||
		errors.Is(err, ErrModelDailyTokenQuotaExhausted) ||
		errors.Is(err, ErrUserModelDailyTokenQuotaExhausted) {
		return true, nil
	}
	return false, err
}

func checkDailyTokenQuota(scope string, userID, groupID int64, routeAlias, model string, snapshot DailyTokenQuotaSnapshot) error {
	if !snapshot.Exists || snapshot.DailyLimitTokens == nil || *snapshot.DailyLimitTokens <= 0 {
		return nil
	}
	if snapshot.UsedTokens < *snapshot.DailyLimitTokens {
		return nil
	}
	return &DailyTokenQuotaExhaustedError{
		Scope: scope, UserID: userID, GroupID: groupID, RouteAlias: routeAlias, Model: model,
		UsageDate: snapshot.UsageDate, UsedTokens: snapshot.UsedTokens, LimitTokens: *snapshot.DailyLimitTokens,
	}
}
