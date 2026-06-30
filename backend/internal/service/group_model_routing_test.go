package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

func TestGroupedModelCandidateRoutingUsesLowestPriorityCandidate(t *testing.T) {
	group := &Group{
		ModelRoutingEnabled: true,
		ModelRouting: map[string][]domain.ModelRouteCandidate{
			"fast-code": {
				{Model: "expensive-model", AccountIDs: []int64{9}, Priority: 20},
				{Model: "cheap-model", AccountIDs: []int64{2, 4}, Priority: 10},
			},
		},
	}

	candidates := group.GetRoutingCandidates("fast-code")
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(candidates))
	}
	if candidates[0].Model != "cheap-model" {
		t.Fatalf("first candidate model = %q, want cheap-model", candidates[0].Model)
	}
	ids := group.GetRoutingAccountIDs("fast-code")
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 4 {
		t.Fatalf("routing account IDs = %v, want [2 4]", ids)
	}
}

func TestGroupedModelCandidateRoutingPreservesLegacyAccountIDs(t *testing.T) {
	group := &Group{
		ModelRoutingEnabled: true,
		ModelRouting: map[string][]int64{
			"claude-opus-*": {9, 3, 7},
		},
	}

	candidates := group.GetRoutingCandidates("claude-opus-4-1")
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].Model != "claude-opus-4-1" || !candidates[0].Legacy {
		t.Fatalf("legacy candidate = %+v", candidates[0])
	}
	ids := group.GetRoutingAccountIDs("claude-opus-4-1")
	if len(ids) != 3 || ids[0] != 9 || ids[1] != 3 || ids[2] != 7 {
		t.Fatalf("legacy routing account IDs = %v, want [9 3 7]", ids)
	}
}
