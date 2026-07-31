package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

func TestOpenAIGroupRouteSelectsFirstCandidate(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 16, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{Model: "model-a", AccountIDs: []int64{3}, Priority: 1},
			{Model: "model-b", AccountIDs: []int64{4}, Priority: 2},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if err != nil || !routed || model != "model-a" || len(accountIDs) != 1 || accountIDs[0] != 3 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}

func TestOpenAIGroupRouteSkipsCandidateWhoseAccountsFailedUpstream(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 3, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{Model: "deepseek-v4-flash", AccountIDs: []int64{2}, Priority: 0},
			{Model: "deepseek-v4-pro", AccountIDs: []int64{1}, Priority: 1},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(
		context.Background(), group, "test", 1, map[int64]struct{}{2: {}},
	)
	if err != nil || !routed || model != "deepseek-v4-pro" || len(accountIDs) != 1 || accountIDs[0] != 1 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}

func TestOpenAIGroupRouteReturnsNoAccountsWhenAllCandidatesFailed(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 3, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {{Model: "model-a", AccountIDs: []int64{2}}},
	}}

	_, _, routed, err := svc.ResolveQuotaAllowedGroupRoute(
		context.Background(), group, "test", 1, map[int64]struct{}{2: {}},
	)
	if !routed || !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("routed=%v err=%v, want ErrNoAvailableAccounts", routed, err)
	}
}
