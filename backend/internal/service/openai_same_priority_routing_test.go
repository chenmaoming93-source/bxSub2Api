package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type openAIRouteLoadCache struct {
	stubConcurrencyCache
	routeCounts map[string]int
	routeLimits map[string]*int
}

type openAISamePriorityAccountRepo struct {
	stubOpenAIAccountRepo
}

func (r openAISamePriorityAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		for index := range r.accounts {
			if r.accounts[index].ID == id {
				account := r.accounts[index]
				result = append(result, &account)
				break
			}
		}
	}
	return result, nil
}

func (c openAIRouteLoadCache) GetRouteConcurrencyBatch(_ context.Context, keys []string) (map[string]int, error) {
	result := make(map[string]int, len(keys))
	for _, key := range keys {
		result[key] = c.routeCounts[key]
	}
	return result, nil
}

func (c openAIRouteLoadCache) GetAccountConcurrencyBatch(context.Context, []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (c openAIRouteLoadCache) AcquireRouteSlot(context.Context, string, int, string) (bool, error) {
	return true, nil
}

func (c openAIRouteLoadCache) ReleaseRouteSlot(context.Context, string, string) error {
	return nil
}

func (c openAIRouteLoadCache) GetRouteConcurrencyLimit(_ context.Context, key string) (*int, bool, error) {
	limit, ok := c.routeLimits[key]
	return limit, ok, nil
}

func (c openAIRouteLoadCache) SetRouteConcurrencyLimit(context.Context, string, *int) error {
	return nil
}

func TestOpenAIGroupRouteReturnsAllAccountsFromSamePriorityTier(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 16, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{AccountIDs: []int64{3}, Priority: 1},
			{AccountIDs: []int64{4}, Priority: 1},
			{AccountIDs: []int64{5}, Priority: 2},
		},
	}}

	_, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if err != nil || !routed || len(accountIDs) != 2 || accountIDs[0] != 3 || accountIDs[1] != 4 {
		t.Fatalf("accounts=%v routed=%v err=%v, want same-priority accounts [3 4]", accountIDs, routed, err)
	}

	_, accountIDs, routed, err = svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, map[int64]struct{}{3: {}, 4: {}})
	if err != nil || !routed || len(accountIDs) != 1 || accountIDs[0] != 5 {
		t.Fatalf("accounts=%v routed=%v err=%v, want next-priority account [5]", accountIDs, routed, err)
	}
}

func TestOpenAIRouteSamePrioritySelectsLowestRouteLoadRate(t *testing.T) {
	groupID := int64(16)
	group := &Group{ID: groupID, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{AccountIDs: []int64{1}, Priority: 1},
			{AccountIDs: []int64{2}, Priority: 1},
			{AccountIDs: []int64{3}, Priority: 2},
		},
	}}
	accounts := []Account{
		openAIRouteTestAccount(1),
		openAIRouteTestAccount(2),
		openAIRouteTestAccount(3),
	}
	cache := openAIRouteLoadCache{
		routeCounts: map[string]int{
			routeConcurrencyKey(&groupID, "test", 1): 8,
			routeConcurrencyKey(&groupID, "test", 2): 2,
			routeConcurrencyKey(&groupID, "test", 3): 0,
		},
		routeLimits: map[string]*int{
			routeConcurrencyKey(&groupID, "test", 1): allocationPtr(10),
			routeConcurrencyKey(&groupID, "test", 2): allocationPtr(10),
			routeConcurrencyKey(&groupID, "test", 3): allocationPtr(10),
		},
	}
	repo := openAISamePriorityAccountRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: accounts}}
	svc := &OpenAIGatewayService{
		accountRepo:        repo,
		cache:              &stubGatewayCache{},
		concurrencyService: NewConcurrencyService(cache),
	}
	ctx := context.WithValue(context.Background(), ctxkey.Group, group)

	selection, err := svc.SelectAccountWithLoadAwareness(ctx, &groupID, "", "test", nil)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if selection == nil || selection.Account == nil || selection.Account.ID != 2 {
		t.Fatalf("selected account=%v, want account 2 with lower same-priority route LoadRate", selection)
	}

	cache.routeCounts[routeConcurrencyKey(&groupID, "test", 1)] = 10
	cache.routeCounts[routeConcurrencyKey(&groupID, "test", 2)] = 10
	selection, err = svc.SelectAccountWithLoadAwareness(ctx, &groupID, "", "test", nil)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness fallback error: %v", err)
	}
	if selection == nil || selection.Account == nil || selection.Account.ID != 3 {
		t.Fatalf("selected account=%v, want lower-priority account 3 after same-priority tier exhaustion", selection)
	}
}

func openAIRouteTestAccount(id int64) Account {
	return Account{
		ID:          id,
		Name:        "openai-route-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 10,
		Credentials: map[string]any{"model_mapping": map[string]any{"test": "upstream-test-model"}},
	}
}
