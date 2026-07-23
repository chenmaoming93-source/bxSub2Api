package service

import (
	"context"
	"testing"
	"time"
)

type realtimeUserQuotaRepo struct {
	records []UserModelDailyTokenQuotaRecord
}

func (r *realtimeUserQuotaRepo) ListUserModelDailyTokenQuotas(context.Context, int64, time.Time) ([]UserModelDailyTokenQuotaRecord, error) {
	return append([]UserModelDailyTokenQuotaRecord(nil), r.records...), nil
}
func (r *realtimeUserQuotaRepo) UpsertUserModelDailyTokenQuotas(context.Context, int64, time.Time, []UserModelDailyTokenQuotaInput) ([]UserModelDailyTokenQuotaRecord, error) {
	return append([]UserModelDailyTokenQuotaRecord(nil), r.records...), nil
}
func (*realtimeUserQuotaRepo) DeleteUserModelTokenQuotaByModel(context.Context, int64, string) error {
	return nil
}

type realtimeUserInvalidator struct{}

func (realtimeUserInvalidator) InvalidateUserModelDailyTokenQuota(context.Context, UserModelDailyTokenQuotaKey) error {
	return nil
}
func TestUserModelTokenQuotaRealtimeListAndUpsert(t *testing.T) {
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	limit := int64(100)
	repo := &realtimeUserQuotaRepo{records: []UserModelDailyTokenQuotaRecord{{UserID: 7, Model: "both", UsageDate: day, UsedTokens: 10, DailyLimitTokens: &limit}, {UserID: 7, Model: "mysql", UsageDate: day, UsedTokens: 20}, {UserID: 7, Model: "zero", UsageDate: day}}}
	reader := &hybridReaderStub{users: []UserTokenUsageRow{{UsageDate: day, UserID: 7, Model: "both", UsedTokens: 30}, {UsageDate: day, UserID: 7, Model: "redis-only", UsedTokens: 99}}}
	repair := &hybridRepairStub{}
	svc := NewUserModelTokenQuotaAdminService(repo, realtimeUserInvalidator{}).ConfigureCurrentTokenUsage(reader, repair)
	svc.now = func() time.Time { return day.Add(12 * time.Hour) }
	records, err := svc.List(context.Background(), 7)
	if err != nil || len(records) != 3 || records[0].UsedTokens != 30 || records[1].UsedTokens != 20 || records[2].UsedTokens != 0 || len(repair.userRows) != 1 || repair.userRows[0].Model != "mysql" {
		t.Fatalf("records=%+v repair=%+v err=%v", records, repair.userRows, err)
	}
	updated, err := svc.Upsert(context.Background(), 7, []UserModelDailyTokenQuotaInput{{Model: "both", DailyLimitTokens: &limit}})
	if err != nil || updated[0].UsedTokens != 30 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
}
