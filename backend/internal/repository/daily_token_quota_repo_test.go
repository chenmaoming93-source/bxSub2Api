package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "modernc.org/sqlite"
)

func newDailyTokenQuotaRepoTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(driver)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestDailyTokenQuotaRepositoryUsesStartOfDayAndMissingIsUnlimited(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 18, 45, 0, 0, time.FixedZone("request", -7*60*60))
	day := timezone.StartOfDay(at)
	limit := int64(100)
	if _, err := client.ModelTokenDailyUsage.Create().
		SetModel("gpt-5").SetUsageDate(day).SetUsedTokens(100).SetDailyLimitTokens(limit).Save(ctx); err != nil {
		t.Fatalf("create model quota: %v", err)
	}

	snapshot, err := repo.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at})
	if err != nil {
		t.Fatalf("GetModelDailyTokenQuota: %v", err)
	}
	if !snapshot.Exists || !snapshot.UsageDate.Equal(day) || snapshot.UsedTokens != 100 || snapshot.DailyLimitTokens == nil || *snapshot.DailyLimitTokens != 100 {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	missing, err := repo.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "missing", At: at})
	if err != nil {
		t.Fatalf("missing GetModelDailyTokenQuota: %v", err)
	}
	if missing.Exists || missing.DailyLimitTokens != nil || missing.UsedTokens != 0 || !missing.UsageDate.Equal(day) {
		t.Fatalf("missing snapshot = %+v", missing)
	}
}

func TestDailyTokenQuotaRepositoryKeepsUsersIsolated(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := timezone.Today()
	limit := int64(25)
	user, err := client.User.Create().SetEmail("quota-41@example.com").SetPasswordHash("hash").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := client.UserModelTokenDailyUsage.Create().
		SetUserID(user.ID).SetModel("claude-sonnet").SetUsageDate(at).SetUsedTokens(25).SetDailyLimitTokens(limit).Save(ctx); err != nil {
		t.Fatalf("create user model quota: %v", err)
	}

	first, err := repo.GetUserModelDailyTokenQuota(ctx, service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "claude-sonnet", At: at})
	if err != nil || !first.Exists || first.UsedTokens != 25 {
		t.Fatalf("first user snapshot=%+v err=%v", first, err)
	}
	second, err := repo.GetUserModelDailyTokenQuota(ctx, service.UserModelDailyTokenQuotaKey{UserID: user.ID + 1, Model: "claude-sonnet", At: at})
	if err != nil || second.Exists {
		t.Fatalf("second user snapshot=%+v err=%v", second, err)
	}
}
