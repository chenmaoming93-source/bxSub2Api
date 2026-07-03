package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type openAIRouteAccountRepo struct {
	AccountRepository
	routed *Account
}

func (r openAIRouteAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}

func (r openAIRouteAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	if len(ids) == 1 && r.routed != nil && ids[0] == r.routed.ID {
		return []*Account{r.routed}, nil
	}
	return nil, nil
}

func TestOpenAIListSchedulableAccountsIncludesExplicitRouteAccount(t *testing.T) {
	groupID := int64(16)
	routed := &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	group := &Group{
		ID:                  groupID,
		Hydrated:            true,
		ModelRoutingEnabled: true,
		ModelRouting: map[string][]domain.ModelRouteCandidate{
			"test": {{Model: "deepseek-v4-pro", AccountIDs: []int64{3}}},
		},
	}
	ctx := context.WithValue(context.Background(), ctxkey.Group, group)
	svc := &OpenAIGatewayService{
		accountRepo: openAIRouteAccountRepo{routed: routed},
		cfg:         &config.Config{RunMode: config.RunModeStandard},
	}

	accounts, err := svc.listSchedulableAccounts(ctx, &groupID)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, int64(3), accounts[0].ID)
}
