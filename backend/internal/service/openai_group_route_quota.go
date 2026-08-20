package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// ResolveQuotaAllowedGroupRoute resolves a client-facing alias to the first
// priority-ordered candidate that passes the shared daily-token quota gates.
// excludedAccountIDs contains account IDs that have already failed upstream and
// should be skipped; candidates whose accounts are all excluded will be bypassed.
func (s *OpenAIGatewayService) ResolveQuotaAllowedGroupRoute(ctx context.Context, group *Group, requestedModel string, userID int64, excludedAccountIDs map[int64]struct{}) (string, []int64, bool, error) {
	_ = ctx
	_ = userID
	if group == nil {
		return requestedModel, nil, false, nil
	}
	candidates := group.GetRoutingCandidates(requestedModel)
	if len(candidates) == 0 {
		return requestedModel, nil, false, nil
	}
	if !candidates[0].Legacy {
		tiers, err := domain.GroupCandidatesByPriority(candidates)
		if err != nil {
			return "", nil, true, err
		}
		for _, tier := range tiers {
			if len(tier.AccountIDs) > 0 && !allAccountIDsExcluded(tier.AccountIDs, excludedAccountIDs) {
				return requestedModel, tier.AccountIDs, true, nil
			}
		}
		return "", nil, true, ErrNoAvailableAccounts
	}
	allExcluded := 0
	for _, candidate := range candidates {
		// Skip candidates whose all accounts have failed upstream.
		if len(candidate.AccountIDs) > 0 && allAccountIDsExcluded(candidate.AccountIDs, excludedAccountIDs) {
			allExcluded++
			continue
		}
		return requestedModel, candidate.AccountIDs, true, nil
	}
	return "", nil, true, ErrNoAvailableAccounts
}

// allAccountIDsExcluded returns true when every id in accountIDs is present in excluded.
func allAccountIDsExcluded(accountIDs []int64, excluded map[int64]struct{}) bool {
	if len(excluded) == 0 {
		return false
	}
	for _, id := range accountIDs {
		if _, ok := excluded[id]; !ok {
			return false
		}
	}
	return true
}
