package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/ent/groupcandidatetokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/modeltokendailyusage"
	"github.com/Wei-Shaw/sub2api/ent/usermodeltokendailyusage"
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
	db.SetMaxOpenConns(1)
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

func TestDailyTokenQuotaRepositoryIncrementConcurrent(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	day := timezone.StartOfDay(at)
	user, err := client.User.Create().SetEmail("quota-concurrent@example.com").SetPasswordHash("hash").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	group, err := client.Group.Create().SetName("quota-concurrent").Save(ctx)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	const workers = 16
	const tokensPerWorker = int64(7)
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- repo.IncrementDailyTokenQuotas(ctx, service.DailyTokenQuotaIncrement{
				ModelKey:          service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at},
				UserModelKey:      service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "gpt-5", At: at},
				GroupCandidateKey: service.GroupCandidateDailyTokenQuotaKey{GroupID: group.ID, RouteAlias: "chat", UpstreamModel: "gpt-5", At: at},
				Tokens:            tokensPerWorker,
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("IncrementDailyTokenQuotas: %v", err)
		}
	}
	want := int64(workers) * tokensPerWorker
	model := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	userModel := client.UserModelTokenDailyUsage.Query().Where(usermodeltokendailyusage.UserIDEQ(user.ID), usermodeltokendailyusage.ModelEQ("gpt-5"), usermodeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	groupCandidate := client.GroupCandidateTokenDailyUsage.Query().Where(groupcandidatetokendailyusage.GroupIDEQ(group.ID), groupcandidatetokendailyusage.RouteAliasEQ("chat"), groupcandidatetokendailyusage.UpstreamModelEQ("gpt-5"), groupcandidatetokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	if model.UsedTokens != want || userModel.UsedTokens != want || groupCandidate.UsedTokens != want {
		t.Fatalf("used tokens model=%d user=%d group=%d, want %d", model.UsedTokens, userModel.UsedTokens, groupCandidate.UsedTokens, want)
	}
}

func TestDailyTokenQuotaRepositoryIncrementRolloverUsesNewDay(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	yesterdayAt := time.Date(2026, 6, 29, 15, 0, 0, 0, time.UTC)
	todayAt := yesterdayAt.AddDate(0, 0, 1)
	yesterday := timezone.StartOfDay(yesterdayAt)
	today := timezone.StartOfDay(todayAt)
	modelLimit := int64(1000)
	userLimit := int64(500)
	groupLimit := int64(250)
	user, err := client.User.Create().SetEmail("quota-rollover@example.com").SetPasswordHash("hash").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	group, err := client.Group.Create().SetName("quota-rollover").Save(ctx)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := client.ModelTokenDailyUsage.Create().SetModel("gpt-5").SetUsageDate(yesterday).SetUsedTokens(33).SetDailyLimitTokens(modelLimit).Save(ctx); err != nil {
		t.Fatalf("create previous model usage: %v", err)
	}
	if _, err := client.UserModelTokenDailyUsage.Create().SetUserID(user.ID).SetModel("gpt-5").SetUsageDate(yesterday).SetUsedTokens(44).SetDailyLimitTokens(userLimit).Save(ctx); err != nil {
		t.Fatalf("create previous user model usage: %v", err)
	}

	if err := repo.IncrementDailyTokenQuotas(ctx, service.DailyTokenQuotaIncrement{
		ModelKey:                       service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: todayAt},
		UserModelKey:                   service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "gpt-5", At: todayAt},
		GroupCandidateKey:              service.GroupCandidateDailyTokenQuotaKey{GroupID: group.ID, RouteAlias: "chat", UpstreamModel: "gpt-5", At: todayAt},
		Tokens:                         12,
		GroupCandidateDailyLimitTokens: &groupLimit,
	}); err != nil {
		t.Fatalf("IncrementDailyTokenQuotas: %v", err)
	}

	previous := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(yesterday)).OnlyX(ctx)
	current := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(today)).OnlyX(ctx)
	currentUser := client.UserModelTokenDailyUsage.Query().Where(usermodeltokendailyusage.UserIDEQ(user.ID), usermodeltokendailyusage.ModelEQ("gpt-5"), usermodeltokendailyusage.UsageDateEQ(today)).OnlyX(ctx)
	currentGroup := client.GroupCandidateTokenDailyUsage.Query().Where(groupcandidatetokendailyusage.GroupIDEQ(group.ID), groupcandidatetokendailyusage.RouteAliasEQ("chat"), groupcandidatetokendailyusage.UpstreamModelEQ("gpt-5"), groupcandidatetokendailyusage.UsageDateEQ(today)).OnlyX(ctx)
	if previous.UsedTokens != 33 || current.UsedTokens != 12 {
		t.Fatalf("model rollover previous=%d current=%d", previous.UsedTokens, current.UsedTokens)
	}
	if current.DailyLimitTokens == nil || *current.DailyLimitTokens != modelLimit {
		t.Fatalf("model daily limit = %v, want %d", current.DailyLimitTokens, modelLimit)
	}
	if currentUser.DailyLimitTokens == nil || *currentUser.DailyLimitTokens != userLimit {
		t.Fatalf("user daily limit = %v, want %d", currentUser.DailyLimitTokens, userLimit)
	}
	if currentGroup.DailyLimitTokens == nil || *currentGroup.DailyLimitTokens != groupLimit {
		t.Fatalf("group daily limit = %v, want %d", currentGroup.DailyLimitTokens, groupLimit)
	}
}

func TestDailyTokenQuotaRepositoryIncrementRollbackOnFailure(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	day := timezone.StartOfDay(at)
	user, err := client.User.Create().SetEmail("quota-rollback@example.com").SetPasswordHash("hash").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	err = repo.IncrementDailyTokenQuotas(ctx, service.DailyTokenQuotaIncrement{
		ModelKey:          service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at},
		UserModelKey:      service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "gpt-5", At: at},
		GroupCandidateKey: service.GroupCandidateDailyTokenQuotaKey{GroupID: 999999, RouteAlias: "chat", UpstreamModel: "gpt-5", At: at},
		Tokens:            10,
	})
	if err == nil {
		t.Fatal("IncrementDailyTokenQuotas succeeded with missing group")
	}
	if count := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(day)).CountX(ctx); count != 0 {
		t.Fatalf("model usage rows after rollback = %d, want 0", count)
	}
	if count := client.UserModelTokenDailyUsage.Query().Where(usermodeltokendailyusage.UserIDEQ(user.ID), usermodeltokendailyusage.ModelEQ("gpt-5"), usermodeltokendailyusage.UsageDateEQ(day)).CountX(ctx); count != 0 {
		t.Fatalf("user model usage rows after rollback = %d, want 0", count)
	}
}
