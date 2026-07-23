package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTokenStatisticsFallbackReturnsMySQLWithoutRepair(t *testing.T) {
	today := time.Date(2026, 7, 15, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo := &hybridModelRepoStub{today: []ModelTokenUsageRow{{UsageDate: today, Model: "mysql", UsedTokens: 12}}}
	reader := &hybridReaderStub{err: errors.New("redis unavailable")}
	repair := &hybridRepairStub{}
	svc := NewTokenUsageReportService(repo).ConfigureCurrentTokenUsage(reader, repair)
	svc.SetNowForTest(func() time.Time { return today })
	report, err := svc.GetModelReport(context.Background(), TokenUsageReportQuery{StartDate: today, EndDate: today, Page: 1, PageSize: 20, SortBy: "used_tokens", SortOrder: "desc"})
	if err != nil || report.UsedTokens != 12 || report.Total != 1 || len(repair.rows) != 0 {
		t.Fatalf("report=%+v repair=%+v err=%v", report, repair.rows, err)
	}
}
