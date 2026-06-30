package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type quotaAwareRoutingRepoStub struct {
	modelSnapshots map[string]DailyTokenQuotaSnapshot
	userSnapshots  map[userModelQuotaTestKey]DailyTokenQuotaSnapshot
	groupSnapshots map[groupCandidateQuotaTestKey]DailyTokenQuotaSnapshot
	modelErr       error
	userErr        error
	groupErr       error
}

type userModelQuotaTestKey struct {
	userID int64
	model  string
}

type groupCandidateQuotaTestKey struct {
	groupID int64
	route   string
	model   string
}

func (s *quotaAwareRoutingRepoStub) GetModelDailyTokenQuota(_ context.Context, key ModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	if s.modelErr != nil {
		return DailyTokenQuotaSnapshot{}, s.modelErr
	}
	if snapshot, ok := s.modelSnapshots[key.Model]; ok {
		return snapshot, nil
	}
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *quotaAwareRoutingRepoStub) GetUserModelDailyTokenQuota(_ context.Context, key UserModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	if s.userErr != nil {
		return DailyTokenQuotaSnapshot{}, s.userErr
	}
	if snapshot, ok := s.userSnapshots[userModelQuotaTestKey{userID: key.UserID, model: key.Model}]; ok {
		return snapshot, nil
	}
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *quotaAwareRoutingRepoStub) GetGroupCandidateDailyTokenQuota(_ context.Context, key GroupCandidateDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	if s.groupErr != nil {
		return DailyTokenQuotaSnapshot{}, s.groupErr
	}
	if snapshot, ok := s.groupSnapshots[groupCandidateQuotaTestKey{groupID: key.GroupID, route: key.RouteAlias, model: key.UpstreamModel}]; ok {
		return snapshot, nil
	}
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *quotaAwareRoutingRepoStub) IncrementDailyTokenQuotas(context.Context, DailyTokenQuotaIncrement) error {
	return nil
}

type quotaAwareRoutingConcurrencyCache struct{}

func (c *quotaAwareRoutingConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (c *quotaAwareRoutingConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	return nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = 0
	}
	return result, nil
}

func (c *quotaAwareRoutingConcurrencyCache) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *quotaAwareRoutingConcurrencyCache) DecrementAccountWaitCount(context.Context, int64) error {
	return nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *quotaAwareRoutingConcurrencyCache) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (c *quotaAwareRoutingConcurrencyCache) ReleaseUserSlot(context.Context, int64, string) error {
	return nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *quotaAwareRoutingConcurrencyCache) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *quotaAwareRoutingConcurrencyCache) DecrementWaitCount(context.Context, int64) error {
	return nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetAccountsLoadBatch(_ context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	result := make(map[int64]*AccountLoadInfo, len(accounts))
	for _, account := range accounts {
		result[account.ID] = &AccountLoadInfo{AccountID: account.ID, LoadRate: 0}
	}
	return result, nil
}

func (c *quotaAwareRoutingConcurrencyCache) GetUsersLoadBatch(_ context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	result := make(map[int64]*UserLoadInfo, len(users))
	for _, user := range users {
		result[user.ID] = &UserLoadInfo{UserID: user.ID, LoadRate: 0}
	}
	return result, nil
}

func (c *quotaAwareRoutingConcurrencyCache) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}

func (c *quotaAwareRoutingConcurrencyCache) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

func TestQuotaAwareRoutingSkipsGroupCandidateExhausted(t *testing.T) {
	svc := &GatewayService{dailyTokenQuotaRepo: &quotaAwareRoutingRepoStub{
		groupSnapshots: map[groupCandidateQuotaTestKey]DailyTokenQuotaSnapshot{
			{groupID: 7, route: "fast-code", model: "model-a"}: exhaustedTokenQuotaSnapshot(),
		},
	}}

	candidate, model, ok, err := svc.selectQuotaAllowedRouteCandidate(context.Background(), 7, "fast-code", 42, quotaAwareRoutingCandidates())
	if err != nil || !ok {
		t.Fatalf("selectQuotaAllowedRouteCandidate ok=%v err=%v", ok, err)
	}
	if model != "model-b" || candidate.AccountIDs[0] != 2 {
		t.Fatalf("selected model=%q candidate=%+v, want model-b account 2", model, candidate)
	}
}

func TestQuotaAwareRoutingSkipsGlobalModelExhausted(t *testing.T) {
	svc := &GatewayService{dailyTokenQuotaRepo: &quotaAwareRoutingRepoStub{
		modelSnapshots: map[string]DailyTokenQuotaSnapshot{
			"model-a": exhaustedTokenQuotaSnapshot(),
		},
	}}

	candidate, model, ok, err := svc.selectQuotaAllowedRouteCandidate(context.Background(), 7, "fast-code", 42, quotaAwareRoutingCandidates())
	if err != nil || !ok {
		t.Fatalf("selectQuotaAllowedRouteCandidate ok=%v err=%v", ok, err)
	}
	if model != "model-b" || candidate.AccountIDs[0] != 2 {
		t.Fatalf("selected model=%q candidate=%+v, want model-b account 2", model, candidate)
	}
}

func TestQuotaAwareRoutingUserModelExhaustionIsUserScoped(t *testing.T) {
	repo := &quotaAwareRoutingRepoStub{
		userSnapshots: map[userModelQuotaTestKey]DailyTokenQuotaSnapshot{
			{userID: 100, model: "model-a"}: exhaustedTokenQuotaSnapshot(),
		},
	}
	svc := &GatewayService{dailyTokenQuotaRepo: repo}

	candidate, model, ok, err := svc.selectQuotaAllowedRouteCandidate(context.Background(), 7, "fast-code", 100, quotaAwareRoutingCandidates())
	if err != nil || !ok {
		t.Fatalf("select user 100 ok=%v err=%v", ok, err)
	}
	if model != "model-b" || candidate.AccountIDs[0] != 2 {
		t.Fatalf("user 100 selected model=%q candidate=%+v, want model-b account 2", model, candidate)
	}

	candidate, model, ok, err = svc.selectQuotaAllowedRouteCandidate(context.Background(), 7, "fast-code", 200, quotaAwareRoutingCandidates())
	if err != nil || !ok {
		t.Fatalf("select user 200 ok=%v err=%v", ok, err)
	}
	if model != "model-a" || candidate.AccountIDs[0] != 1 {
		t.Fatalf("user 200 selected model=%q candidate=%+v, want model-a account 1", model, candidate)
	}
}

func TestQuotaAwareRoutingInfrastructureErrorIsNotQuotaExhaustion(t *testing.T) {
	dbErr := errors.New("quota store unavailable")
	svc := &GatewayService{dailyTokenQuotaRepo: &quotaAwareRoutingRepoStub{modelErr: dbErr}}

	_, _, ok, err := svc.selectQuotaAllowedRouteCandidate(context.Background(), 7, "fast-code", 42, quotaAwareRoutingCandidates())
	if ok {
		t.Fatal("selectQuotaAllowedRouteCandidate ok=true, want false")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v, want dbErr", err)
	}
	if isDailyTokenQuotaExhausted(err) {
		t.Fatalf("infrastructure error was classified as quota exhaustion: %v", err)
	}
}

func TestGroupedModelCandidateFailoverKeepsSameCandidateWhenAnotherAccountAvailable(t *testing.T) {
	svc := quotaAwareRoutingGatewayService()
	accounts := quotaAwareRoutingAccountMap()
	excluded := map[int64]struct{}{1: {}}

	result, ok, err := svc.trySelectRouteCandidateAccounts(context.Background(), nil, "fast-code", "model-a", "", 0, []int64{1, 2}, accounts, quotaAwareRoutingExcluded(excluded), PlatformAnthropic, false, quotaAwareRoutingSchedulingConfig())
	if err != nil {
		t.Fatalf("trySelectRouteCandidateAccounts: %v", err)
	}
	if !ok || result == nil || result.Account == nil {
		t.Fatalf("selection ok=%v result=%+v, want account 2", ok, result)
	}
	if result.Account.ID != 2 {
		t.Fatalf("selected account = %d, want 2", result.Account.ID)
	}
	if result.RequestedModel != "fast-code" || result.UpstreamModel != "model-a" {
		t.Fatalf("selection identity requested=%q upstream=%q, want fast-code/model-a", result.RequestedModel, result.UpstreamModel)
	}
}

func TestGroupedModelCandidateFailoverTriesNextCandidateAfterAllAccountsExcluded(t *testing.T) {
	svc := quotaAwareRoutingGatewayService()
	accounts := quotaAwareRoutingAccountMap()
	excluded := map[int64]struct{}{1: {}, 2: {}}

	first, ok, err := svc.trySelectRouteCandidateAccounts(context.Background(), nil, "fast-code", "model-a", "", 0, []int64{1, 2}, accounts, quotaAwareRoutingExcluded(excluded), PlatformAnthropic, false, quotaAwareRoutingSchedulingConfig())
	if err != nil {
		t.Fatalf("first candidate: %v", err)
	}
	if ok || first != nil {
		t.Fatalf("first candidate ok=%v result=%+v, want exhausted candidate", ok, first)
	}

	second, ok, err := svc.trySelectRouteCandidateAccounts(context.Background(), nil, "fast-code", "model-b", "", 0, []int64{3}, accounts, quotaAwareRoutingExcluded(excluded), PlatformAnthropic, false, quotaAwareRoutingSchedulingConfig())
	if err != nil {
		t.Fatalf("second candidate: %v", err)
	}
	if !ok || second == nil || second.Account == nil {
		t.Fatalf("second candidate ok=%v result=%+v, want account 3", ok, second)
	}
	if second.Account.ID != 3 {
		t.Fatalf("selected account = %d, want 3", second.Account.ID)
	}
	if second.RequestedModel != "fast-code" || second.UpstreamModel != "model-b" {
		t.Fatalf("selection identity requested=%q upstream=%q, want fast-code/model-b", second.RequestedModel, second.UpstreamModel)
	}
}

func TestGroupedModelCandidateFailoverUnschedulableReturnsNoAvailable(t *testing.T) {
	svc := &GatewayService{
		concurrencyService: NewConcurrencyService(&quotaAwareRoutingConcurrencyCache{}),
		cfg:                &config.Config{},
	}
	accounts := map[int64]*Account{
		1: quotaAwareRoutingAccount(1, "model-a", false),
	}

	result, ok, err := svc.trySelectRouteCandidateAccounts(context.Background(), nil, "fast-code", "model-a", "", 0, []int64{1}, accounts, quotaAwareRoutingExcluded(nil), PlatformAnthropic, false, quotaAwareRoutingSchedulingConfig())
	if err != nil {
		t.Fatalf("trySelectRouteCandidateAccounts: %v", err)
	}
	if ok || result != nil {
		t.Fatalf("ok=%v result=%+v, want no schedulable routed accounts", ok, result)
	}
}

func quotaAwareRoutingCandidates() []domain.ModelRouteCandidate {
	return []domain.ModelRouteCandidate{
		{Model: "model-a", AccountIDs: []int64{1}, Priority: 1},
		{Model: "model-b", AccountIDs: []int64{2}, Priority: 2},
	}
}

func quotaAwareRoutingGatewayService() *GatewayService {
	return &GatewayService{
		concurrencyService: NewConcurrencyService(&quotaAwareRoutingConcurrencyCache{}),
		cfg:                &config.Config{},
	}
}

func quotaAwareRoutingAccountMap() map[int64]*Account {
	return map[int64]*Account{
		1: quotaAwareRoutingAccount(1, "model-a", true),
		2: quotaAwareRoutingAccount(2, "model-a", true),
		3: quotaAwareRoutingAccount(3, "model-b", true),
	}
}

func quotaAwareRoutingAccount(id int64, model string, schedulable bool) *Account {
	return &Account{
		ID:          id,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: schedulable,
		Concurrency: 1,
		Priority:    int(id),
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				model: model,
			},
		},
	}
}

func quotaAwareRoutingExcluded(excluded map[int64]struct{}) func(int64) bool {
	return func(accountID int64) bool {
		_, ok := excluded[accountID]
		return ok
	}
}

func quotaAwareRoutingSchedulingConfig() config.GatewaySchedulingConfig {
	return config.GatewaySchedulingConfig{
		StickySessionMaxWaiting:  3,
		StickySessionWaitTimeout: time.Second,
		FallbackWaitTimeout:      time.Second,
		FallbackMaxWaiting:       10,
		LoadBatchEnabled:         true,
	}
}

func exhaustedTokenQuotaSnapshot() DailyTokenQuotaSnapshot {
	limit := int64(10)
	return DailyTokenQuotaSnapshot{
		Exists:           true,
		UsageDate:        time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		UsedTokens:       10,
		DailyLimitTokens: &limit,
	}
}
