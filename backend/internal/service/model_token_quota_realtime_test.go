package service

import (
	"context"
	"testing"
	"time"
)

type realtimeModelQuotaRepo struct {
	records []ModelDailyTokenQuotaRecord
	set     ModelDailyTokenQuotaRecord
}

func (r *realtimeModelQuotaRepo) ListModelDailyTokenQuotas(context.Context, time.Time) ([]ModelDailyTokenQuotaRecord, error) {
	return append([]ModelDailyTokenQuotaRecord(nil), r.records...), nil
}
func (r *realtimeModelQuotaRepo) SetModelDailyTokenQuota(context.Context, string, time.Time, *int64) (ModelDailyTokenQuotaRecord, error) {
	return r.set, nil
}

type realtimeInvalidator struct{}

func (realtimeInvalidator) InvalidateModelDailyTokenQuota(context.Context, ModelDailyTokenQuotaKey) error {
	return nil
}
func TestModelTokenQuotaRealtimeListAndSet(t *testing.T) {
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	limit := int64(100)
	repo := &realtimeModelQuotaRepo{records: []ModelDailyTokenQuotaRecord{{Model: "both", UsageDate: day, UsedTokens: 10, DailyLimitTokens: &limit}, {Model: "mysql", UsageDate: day, UsedTokens: 20}, {Model: "zero", UsageDate: day}}, set: ModelDailyTokenQuotaRecord{Model: "both", UsageDate: day, UsedTokens: 10, DailyLimitTokens: &limit}}
	reader := &hybridReaderStub{models: []ModelTokenUsageRow{{UsageDate: day, Model: "both", UsedTokens: 30}}}
	repair := &hybridRepairStub{}
	svc := NewModelTokenQuotaAdminService(repo, realtimeInvalidator{}).ConfigureCurrentTokenUsage(reader, repair)
	svc.now = func() time.Time { return day.Add(12 * time.Hour) }
	records, err := svc.List(context.Background())
	if err != nil || len(records) != 3 || records[0].UsedTokens != 30 || records[1].UsedTokens != 20 || records[2].UsedTokens != 0 || len(repair.rows) != 1 || repair.rows[0].Model != "mysql" {
		t.Fatalf("records=%+v repair=%+v err=%v", records, repair.rows, err)
	}
	record, err := svc.Set(context.Background(), "both", &limit)
	if err != nil || record.UsedTokens != 30 {
		t.Fatalf("record=%+v err=%v", record, err)
	}
}
