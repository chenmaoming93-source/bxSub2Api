package service

import "testing"

func TestModelRoutingCandidateAndLegacyRegression(t *testing.T) {
	t.Run("priority candidate", TestGroupedModelCandidateRoutingUsesLowestPriorityCandidate)
	t.Run("legacy account order", TestGroupedModelCandidateRoutingPreservesLegacyAccountIDs)
}
