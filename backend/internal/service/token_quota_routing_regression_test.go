package service

import "testing"

func TestTokenQuotaRoutingDegradationRegression(t *testing.T) {
	t.Run("group candidate exhausted", TestQuotaAwareRoutingSkipsGroupCandidateExhausted)
	t.Run("global model exhausted", TestQuotaAwareRoutingSkipsGlobalModelExhausted)
	t.Run("user model isolated", TestQuotaAwareRoutingUserModelExhaustionIsUserScoped)
	t.Run("repository failure", TestQuotaAwareRoutingInfrastructureErrorIsNotQuotaExhaustion)
}

func TestModelRoutingCandidateAndLegacyRegression(t *testing.T) {
	t.Run("priority candidate", TestGroupedModelCandidateRoutingUsesLowestPriorityCandidate)
	t.Run("legacy account order", TestGroupedModelCandidateRoutingPreservesLegacyAccountIDs)
	t.Run("same candidate account failover", TestGroupedModelCandidateFailoverKeepsSameCandidateWhenAnotherAccountAvailable)
	t.Run("next candidate failover", TestGroupedModelCandidateFailoverTriesNextCandidateAfterAllAccountsExcluded)
	t.Run("unschedulable remains unavailable", TestGroupedModelCandidateFailoverUnschedulableReturnsNoAvailable)
}
