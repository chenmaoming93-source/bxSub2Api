package service

import (
	"errors"
	"testing"
	"time"
)

func TestDailyTokenQuotaBoundaryAndUnlimitedSemantics(t *testing.T) {
	limit := int64(100)
	zero := int64(0)
	key := ModelDailyTokenQuotaKey{Model: "gpt-5"}
	for _, snapshot := range []DailyTokenQuotaSnapshot{
		{},
		{Exists: true, UsedTokens: 100, DailyLimitTokens: nil},
		{Exists: true, UsedTokens: 100, DailyLimitTokens: &zero},
		{Exists: true, UsedTokens: 99, DailyLimitTokens: &limit},
	} {
		if err := CheckModelDailyTokenQuota(key, snapshot); err != nil {
			t.Fatalf("unlimited/below-limit snapshot %+v returned %v", snapshot, err)
		}
	}
	for _, used := range []int64{100, 101} {
		err := CheckModelDailyTokenQuota(key, DailyTokenQuotaSnapshot{Exists: true, UsedTokens: used, DailyLimitTokens: &limit})
		if !errors.Is(err, ErrModelDailyTokenQuotaExhausted) {
			t.Fatalf("used=%d error=%v, want model exhausted", used, err)
		}
	}
}

func TestUserModelDailyTokenQuotaErrorCarriesUserContext(t *testing.T) {
	limit := int64(50)
	date := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local)
	key := UserModelDailyTokenQuotaKey{UserID: 42, Model: "claude-sonnet", At: date}
	err := CheckUserModelDailyTokenQuota(key, DailyTokenQuotaSnapshot{Exists: true, UsageDate: date, UsedTokens: 50, DailyLimitTokens: &limit})
	if !errors.Is(err, ErrUserModelDailyTokenQuotaExhausted) {
		t.Fatalf("error=%v, want user exhausted", err)
	}
	var detail *DailyTokenQuotaExhaustedError
	if !errors.As(err, &detail) || detail.UserID != 42 || detail.Model != "claude-sonnet" || detail.UsedTokens != 50 || detail.LimitTokens != 50 {
		t.Fatalf("error context = %+v", detail)
	}
	other := UserModelDailyTokenQuotaKey{UserID: 43, Model: key.Model}
	if err := CheckUserModelDailyTokenQuota(other, DailyTokenQuotaSnapshot{}); err != nil {
		t.Fatalf("missing quota for other user returned %v", err)
	}
}

func TestGroupCandidateDailyTokenQuotaError(t *testing.T) {
	limit := int64(10)
	key := GroupCandidateDailyTokenQuotaKey{GroupID: 7, RouteAlias: "fast-code", UpstreamModel: "gpt-5"}
	err := CheckGroupCandidateDailyTokenQuota(key, DailyTokenQuotaSnapshot{Exists: true, UsedTokens: 11, DailyLimitTokens: &limit})
	if !errors.Is(err, ErrGroupCandidateDailyTokenQuotaExhausted) {
		t.Fatalf("error=%v, want group candidate exhausted", err)
	}
}
