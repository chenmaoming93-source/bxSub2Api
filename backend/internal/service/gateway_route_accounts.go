package service

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// mergeExplicitRouteAccounts adds accounts explicitly referenced by the
// matched model_routing rule. Group membership is intentionally not checked:
// model_routing.account_ids is itself the assignment. Runtime account health
// and candidate-specific platform/model/quota checks remain in the selector.
func (s *GatewayService) mergeExplicitRouteAccounts(ctx context.Context, accounts []Account, candidates []domain.ModelRouteCandidate) ([]Account, error) {
	if s == nil || s.accountRepo == nil || len(candidates) == 0 {
		return accounts, nil
	}

	seen := make(map[int64]struct{}, len(accounts))
	for i := range accounts {
		seen[accounts[i].ID] = struct{}{}
	}

	ids := make([]int64, 0)
	for _, candidate := range candidates {
		for _, accountID := range candidate.AccountIDs {
			if accountID <= 0 {
				continue
			}
			if _, ok := seen[accountID]; ok {
				continue
			}
			seen[accountID] = struct{}{}
			ids = append(ids, accountID)
		}
	}
	if len(ids) == 0 {
		return accounts, nil
	}

	routed, err := s.accountRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("query model routing accounts failed: %w", err)
	}
	for _, account := range routed {
		if account == nil || !account.IsSchedulable() {
			continue
		}
		accounts = append(accounts, *account)
	}
	return accounts, nil
}
