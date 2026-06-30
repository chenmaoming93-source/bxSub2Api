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
	ModelKey                       ModelDailyTokenQuotaKey
	UserModelKey                   UserModelDailyTokenQuotaKey
	GroupCandidateKey              GroupCandidateDailyTokenQuotaKey
	Tokens                         int64
	ModelDailyLimitTokens          *int64
	UserModelDailyLimitTokens      *int64
	GroupCandidateDailyLimitTokens *int64
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
