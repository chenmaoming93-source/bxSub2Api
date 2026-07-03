package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

type explicitRouteAccountRepo struct {
	AccountRepository
	accounts map[int64]*Account
}

func (r explicitRouteAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account := r.accounts[id]; account != nil {
			result = append(result, account)
		}
	}
	return result, nil
}

func TestMergeExplicitRouteAccountsDoesNotRequireGroupMembership(t *testing.T) {
	routed := &Account{
		ID:          2,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		// AccountGroups intentionally empty.
	}
	svc := &GatewayService{accountRepo: explicitRouteAccountRepo{accounts: map[int64]*Account{2: routed}}}

	accounts, err := svc.mergeExplicitRouteAccounts(context.Background(), nil, []domain.ModelRouteCandidate{
		{Model: "deepseek-v4-flash", AccountIDs: []int64{2}},
	})

	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, int64(2), accounts[0].ID)
}

func TestMergeExplicitRouteAccountsKeepsRuntimeSchedulability(t *testing.T) {
	disabled := &Account{ID: 2, Platform: PlatformOpenAI, Status: StatusDisabled, Schedulable: true}
	svc := &GatewayService{accountRepo: explicitRouteAccountRepo{accounts: map[int64]*Account{2: disabled}}}

	accounts, err := svc.mergeExplicitRouteAccounts(context.Background(), nil, []domain.ModelRouteCandidate{
		{Model: "deepseek-v4-flash", AccountIDs: []int64{2}},
	})

	require.NoError(t, err)
	require.Empty(t, accounts)
}
