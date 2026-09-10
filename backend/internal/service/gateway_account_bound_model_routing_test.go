package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type accountBoundRoutingConcurrencyCache struct {
	loads map[int64]*AccountLoadInfo
}

func (c *accountBoundRoutingConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}
func (c *accountBoundRoutingConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	return nil
}
func (c *accountBoundRoutingConcurrencyCache) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *accountBoundRoutingConcurrencyCache) GetAccountConcurrencyBatch(context.Context, []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}
func (c *accountBoundRoutingConcurrencyCache) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}
func (c *accountBoundRoutingConcurrencyCache) DecrementAccountWaitCount(context.Context, int64) error {
	return nil
}
func (c *accountBoundRoutingConcurrencyCache) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *accountBoundRoutingConcurrencyCache) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}
func (c *accountBoundRoutingConcurrencyCache) ReleaseUserSlot(context.Context, int64, string) error {
	return nil
}
func (c *accountBoundRoutingConcurrencyCache) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}
func (c *accountBoundRoutingConcurrencyCache) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}
func (c *accountBoundRoutingConcurrencyCache) DecrementWaitCount(context.Context, int64) error {
	return nil
}
func (c *accountBoundRoutingConcurrencyCache) GetAccountsLoadBatch(context.Context, []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return c.loads, nil
}
func (c *accountBoundRoutingConcurrencyCache) GetUsersLoadBatch(context.Context, []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	return map[int64]*UserLoadInfo{}, nil
}
func (c *accountBoundRoutingConcurrencyCache) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}
func (c *accountBoundRoutingConcurrencyCache) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

func accountBoundRoutingConfig() config.GatewaySchedulingConfig {
	return config.GatewaySchedulingConfig{StickySessionMaxWaiting: 3, FallbackMaxWaiting: 3}
}

func modelRoutingAccount(id int64, upstreamModels ...string) *Account {
	mapping := make(map[string]any, len(upstreamModels))
	for _, model := range upstreamModels {
		mapping[model] = model
	}
	return &Account{
		ID: id, Name: "route-account", Platform: PlatformAnthropic,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Concurrency: 5, Credentials: map[string]any{"model_mapping": mapping},
	}
}

func TestAccountFirstModelMappingKeyIsDeterministic(t *testing.T) {
	account := modelRoutingAccount(1, "zeta-model", "alpha-model", "middle-model")
	for i := 0; i < 20; i++ {
		require.Equal(t, "alpha-model", account.FirstModelMappingKey())
	}
	require.Empty(t, (&Account{Credentials: map[string]any{}}).FirstModelMappingKey())
}

func TestAccountFirstModelMappingValueResolvesUpstreamModel(t *testing.T) {
	// 白名单（key == value）：第一个 value 即模型名本身
	whitelist := modelRoutingAccount(1, "gpt-5.3-codex")
	require.Equal(t, "gpt-5.3-codex", whitelist.FirstModelMappingValue())

	// 映射（key != value）：第一个 value 为真实上游模型
	mapped := &Account{Credentials: map[string]any{"model_mapping": map[string]any{"claude-opus-4-6": "claude-opus-4-6-thinking"}}}
	require.Equal(t, "claude-opus-4-6-thinking", mapped.FirstModelMappingValue())

	// 多组 key-value（老账号）：取 key 字典序第一个条目的 value
	multi := &Account{Credentials: map[string]any{"model_mapping": map[string]any{
		"z-model": "z-upstream", "a-model": "a-upstream", "m-model": "m-upstream",
	}}}
	require.Equal(t, "a-upstream", multi.FirstModelMappingValue())

	// 空 mapping / nil 账号
	require.Empty(t, (&Account{Credentials: map[string]any{}}).FirstModelMappingValue())
	var nilAccount *Account
	require.Empty(t, nilAccount.FirstModelMappingValue())
}

func TestRouteCandidateCarriesSelectedAccountsOwnUpstreamModel(t *testing.T) {
	first := modelRoutingAccount(1, "model-a")
	first.Priority = 1
	second := modelRoutingAccount(2, "model-b")
	second.Priority = 99
	concurrencyCache := &accountBoundRoutingConcurrencyCache{loads: map[int64]*AccountLoadInfo{
		1: {AccountID: 1, LoadRate: 80},
		2: {AccountID: 2, LoadRate: 20},
	}}
	svc := &GatewayService{concurrencyService: NewConcurrencyService(concurrencyCache)}
	failures := []ModelCandidateFailure{}
	selection, ok, err := svc.trySelectRouteCandidateAccountsWithModel(
		context.Background(), nil, DynamicTokenRequestIdentity{}, "coding-alias", "obsolete-candidate-model", "", 0,
		[]int64{1, 2}, map[int64]*Account{1: first, 2: second}, func(int64) bool { return false },
		PlatformAnthropic, false, accountBoundRoutingConfig(), &failures,
	)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(2), selection.Account.ID)
	require.Equal(t, "coding-alias", selection.RequestedModel)
	require.Equal(t, "coding-alias", selection.RouteAlias)
	require.Equal(t, "model-b", selection.UpstreamModel)
}

func TestRouteCandidateSkipsAccountWithoutModel(t *testing.T) {
	missing := modelRoutingAccount(1)
	valid := modelRoutingAccount(2, "model-b")
	svc := &GatewayService{concurrencyService: NewConcurrencyService(&accountBoundRoutingConcurrencyCache{loads: map[int64]*AccountLoadInfo{}})}
	failures := []ModelCandidateFailure{}
	selection, ok, err := svc.trySelectRouteCandidateAccounts(
		context.Background(), nil, DynamicTokenRequestIdentity{}, "coding-alias", "", 0,
		[]int64{1, 2}, map[int64]*Account{1: missing, 2: valid}, func(int64) bool { return false },
		PlatformAnthropic, false, accountBoundRoutingConfig(), &failures,
	)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(2), selection.Account.ID)
	require.Equal(t, "model-b", selection.UpstreamModel)
}
