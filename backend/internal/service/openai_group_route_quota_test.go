package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

func TestOpenAIGroupRouteUsesSharedQuotaChecksAndFallsBack(t *testing.T) {
	limit := int64(100)
	repo := &quotaAwareRoutingRepoStub{
		userSnapshots: map[userModelQuotaTestKey]DailyTokenQuotaSnapshot{
			{userID: 1, model: "model-a"}: {Exists: true, UsedTokens: 100, DailyLimitTokens: &limit},
		},
	}
	svc := &OpenAIGatewayService{dailyTokenQuotaRepo: repo}
	group := &Group{ID: 16, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{Model: "model-a", AccountIDs: []int64{3}, Priority: 1},
			{Model: "model-b", AccountIDs: []int64{4}, Priority: 2},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if err != nil || !routed || model != "model-b" || len(accountIDs) != 1 || accountIDs[0] != 4 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}

func TestOpenAIGroupRouteReturnsQuotaExhaustedWhenEveryCandidateBlocked(t *testing.T) {
	limit := int64(100)
	repo := &quotaAwareRoutingRepoStub{modelSnapshots: map[string]DailyTokenQuotaSnapshot{
		"model-a": {Exists: true, UsedTokens: 100, DailyLimitTokens: &limit},
	}}
	svc := &OpenAIGatewayService{dailyTokenQuotaRepo: repo}
	group := &Group{ID: 16, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {{Model: "model-a", AccountIDs: []int64{3}}},
	}}

	_, _, _, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if !errors.Is(err, ErrRoutedTokenQuotaExhausted) {
		t.Fatalf("err=%v, want ErrRoutedTokenQuotaExhausted", err)
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

func TestOpenAIGroupRouteUsesCandidateLimitConfigTable(t *testing.T) {
	limit := int64(10)
	repo := &quotaAwareRoutingRepoStub{groupSnapshots: map[groupCandidateQuotaTestKey]DailyTokenQuotaSnapshot{
		{groupID: 3, route: "test", model: "deepseek-v4-flash"}: {
			Exists: true, UsedTokens: 725, DailyLimitTokens: &limit,
		},
	}}
	svc := &OpenAIGatewayService{dailyTokenQuotaRepo: repo}
	group := &Group{ID: 3, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{Model: "deepseek-v4-flash", AccountIDs: []int64{2}, Priority: 0},
			{Model: "deepseek-v4-pro", AccountIDs: []int64{1}, Priority: 1},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if err != nil || !routed || model != "deepseek-v4-pro" || len(accountIDs) != 1 || accountIDs[0] != 1 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}
