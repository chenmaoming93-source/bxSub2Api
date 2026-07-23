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

func TestGetModel_ConfigAndUsage(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := &dailyTokenQuotaRepository{client: client}
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 18, 45, 0, 0, time.FixedZone("request", -7*60*60))
	day := timezone.StartOfDay(at)
	limit := int64(100)
	if _, err := client.ModelTokenDailyLimitConfig.Create().SetModel("gpt-5").SetDailyLimitTokens(limit).Save(ctx); err != nil {
		t.Fatalf("create config: %v", err)
	}
	if _, err := client.ModelTokenDailyUsage.Create().SetModel("gpt-5").SetUsageDate(day).SetUsedTokens(100).Save(ctx); err != nil {
		t.Fatalf("create usage: %v", err)
	}
	snapshot, err := repo.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at})
	if err != nil || !snapshot.Exists || snapshot.UsedTokens != 100 || snapshot.DailyLimitTokens == nil || *snapshot.DailyLimitTokens != 100 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	missing, err := repo.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "missing", At: at})
	if err != nil || missing.Exists || missing.DailyLimitTokens != nil {
		t.Fatalf("missing=%+v err=%v", missing, err)
	}
}

func TestGetUserModel_KeepsUsersIsolated(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := timezone.Today()
	limit := int64(25)
	user, err := client.User.Create().SetEmail("u41@test.com").SetPasswordHash("h").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := client.UserModelTokenDailyLimitConfig.Create().SetUserID(user.ID).SetModel("claude-sonnet").SetDailyLimitTokens(limit).Save(ctx); err != nil {
		t.Fatalf("create config: %v", err)
	}
	if _, err := client.UserModelTokenDailyUsage.Create().SetUserID(user.ID).SetModel("claude-sonnet").SetUsageDate(at).SetUsedTokens(25).Save(ctx); err != nil {
		t.Fatalf("create usage: %v", err)
	}
	first, err := repo.GetUserModelDailyTokenQuota(ctx, service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "claude-sonnet", At: at})
	if err != nil || !first.Exists || first.UsedTokens != 25 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := repo.GetUserModelDailyTokenQuota(ctx, service.UserModelDailyTokenQuotaKey{UserID: user.ID + 1, Model: "claude-sonnet", At: at})
	if err != nil || second.Exists {
		t.Fatalf("second=%+v err=%v", second, err)
	}
}

func TestIncrement_Concurrent(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	day := timezone.StartOfDay(at)
	user, _ := client.User.Create().SetEmail("qc@test.com").SetPasswordHash("h").Save(ctx)
	group, _ := client.Group.Create().SetName("qc").Save(ctx)
	const workers = 16
	const each = int64(7)
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
				Tokens:            each,
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Increment: %v", err)
		}
	}
	want := int64(workers) * each
	m := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	u := client.UserModelTokenDailyUsage.Query().Where(usermodeltokendailyusage.UserIDEQ(user.ID), usermodeltokendailyusage.ModelEQ("gpt-5"), usermodeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	g := client.GroupCandidateTokenDailyUsage.Query().Where(groupcandidatetokendailyusage.GroupIDEQ(group.ID), groupcandidatetokendailyusage.RouteAliasEQ("chat"), groupcandidatetokendailyusage.UpstreamModelEQ("gpt-5"), groupcandidatetokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	if m.UsedTokens != want || u.UsedTokens != want || g.UsedTokens != want {
		t.Fatalf("tokens m=%d u=%d g=%d want=%d", m.UsedTokens, u.UsedTokens, g.UsedTokens, want)
	}
}

func TestIncrement_RollbackOnFailure(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenQuotaRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	day := timezone.StartOfDay(at)
	user, _ := client.User.Create().SetEmail("qrb@test.com").SetPasswordHash("h").Save(ctx)
	err := repo.IncrementDailyTokenQuotas(ctx, service.DailyTokenQuotaIncrement{
		ModelKey:          service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at},
		UserModelKey:      service.UserModelDailyTokenQuotaKey{UserID: user.ID, Model: "gpt-5", At: at},
		GroupCandidateKey: service.GroupCandidateDailyTokenQuotaKey{GroupID: 999999, RouteAlias: "chat", UpstreamModel: "gpt-5", At: at},
		Tokens:            10,
	})
	if err == nil {
		t.Fatal("expected error with missing group")
	}
	if c := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(day)).CountX(ctx); c != 0 {
		t.Fatalf("model rows=%d want=0", c)
	}
}

func TestDailyTokenQuotaAbsoluteUpsertBatch(t *testing.T) {
	client := newDailyTokenQuotaRepoTestClient(t)
	repo := NewDailyTokenUsageAbsoluteRepository(client)
	ctx := context.Background()
	at := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	day := timezone.StartOfDay(at)
	user, err := client.User.Create().SetEmail("absolute@test.com").SetPasswordHash("h").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	group, err := client.Group.Create().SetName("absolute").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.ModelTokenDailyLimitConfig.Create().SetModel("gpt-5").SetDailyLimitTokens(999).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UserModelTokenDailyLimitConfig.Create().SetUserID(user.ID).SetModel("gpt-5").SetDailyLimitTokens(888).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GroupCandidateTokenDailyLimitConfig.Create().SetGroupID(group.ID).SetRouteAlias("chat").SetUpstreamModel("gpt-5").SetDailyLimitTokens(777).Save(ctx); err != nil {
		t.Fatal(err)
	}

	write := func(tokens int64) {
		if err := repo.UpsertModelDailyTokenUsageAbsolute(ctx, []service.ModelDailyTokenUsageAbsolute{{Model: "gpt-5", UsageDate: at, UsedTokens: tokens}}); err != nil {
			t.Fatal(err)
		}
		if err := repo.UpsertUserModelDailyTokenUsageAbsolute(ctx, []service.UserModelDailyTokenUsageAbsolute{{UserID: user.ID, Model: "gpt-5", UsageDate: at, UsedTokens: tokens}}); err != nil {
			t.Fatal(err)
		}
		if err := repo.UpsertGroupCandidateDailyTokenUsageAbsolute(ctx, []service.GroupCandidateDailyTokenUsageAbsolute{{GroupID: group.ID, RouteAlias: "chat", UpstreamModel: "gpt-5", UsageDate: at, UsedTokens: tokens}}); err != nil {
			t.Fatal(err)
		}
	}
	write(100)
	write(100)
	write(50)

	m := client.ModelTokenDailyUsage.Query().Where(modeltokendailyusage.ModelEQ("gpt-5"), modeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	u := client.UserModelTokenDailyUsage.Query().Where(usermodeltokendailyusage.UserIDEQ(user.ID), usermodeltokendailyusage.ModelEQ("gpt-5"), usermodeltokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	g := client.GroupCandidateTokenDailyUsage.Query().Where(groupcandidatetokendailyusage.GroupIDEQ(group.ID), groupcandidatetokendailyusage.RouteAliasEQ("chat"), groupcandidatetokendailyusage.UpstreamModelEQ("gpt-5"), groupcandidatetokendailyusage.UsageDateEQ(day)).OnlyX(ctx)
	if m.UsedTokens != 100 || u.UsedTokens != 100 || g.UsedTokens != 100 {
		t.Fatalf("absolute values m=%d u=%d g=%d", m.UsedTokens, u.UsedTokens, g.UsedTokens)
	}
	if client.ModelTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens == nil || *client.ModelTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens != 999 {
		t.Fatal("model limit changed")
	}
	if client.UserModelTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens == nil || *client.UserModelTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens != 888 {
		t.Fatal("user limit changed")
	}
	if client.GroupCandidateTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens == nil || *client.GroupCandidateTokenDailyLimitConfig.Query().OnlyX(ctx).DailyLimitTokens != 777 {
		t.Fatal("group limit changed")
	}
}
