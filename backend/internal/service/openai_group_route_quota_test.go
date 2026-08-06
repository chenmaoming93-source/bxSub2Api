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
			{AccountIDs: []int64{3}, Priority: 1},
			{AccountIDs: []int64{4}, Priority: 2},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(context.Background(), group, "test", 1, nil)
	if err != nil || !routed || model != "test" || len(accountIDs) != 1 || accountIDs[0] != 3 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}

func TestOpenAIGroupRouteSkipsCandidateWhoseAccountsFailedUpstream(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 3, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {
			{AccountIDs: []int64{2}, Priority: 0},
			{AccountIDs: []int64{1}, Priority: 1},
		},
	}}

	model, accountIDs, routed, err := svc.ResolveQuotaAllowedGroupRoute(
		context.Background(), group, "test", 1, map[int64]struct{}{2: {}},
	)
	if err != nil || !routed || model != "test" || len(accountIDs) != 1 || accountIDs[0] != 1 {
		t.Fatalf("model=%q accounts=%v routed=%v err=%v", model, accountIDs, routed, err)
	}
}

func TestOpenAIGroupRouteReturnsNoAccountsWhenAllCandidatesFailed(t *testing.T) {
	svc := &OpenAIGatewayService{}
	group := &Group{ID: 3, ModelRoutingEnabled: true, ModelRouting: map[string][]domain.ModelRouteCandidate{
		"test": {{AccountIDs: []int64{2}}},
	}}

	_, _, routed, err := svc.ResolveQuotaAllowedGroupRoute(
		context.Background(), group, "test", 1, map[int64]struct{}{2: {}},
	)
	if !routed || !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("routed=%v err=%v, want ErrNoAvailableAccounts", routed, err)
	}
}

// TestOpenAIRouteCandidateEligibleByOwnModel 验证：路由候选账号的模型支持由账号自身
// model_mapping 决定（路由别名不要求命中账号白名单），且选中后上游模型取第一个 value。
func TestOpenAIRouteCandidateEligibleByOwnModel(t *testing.T) {
	// 白名单账号：key == value
	whitelist := &Account{
		ID: 1, Name: "wl", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 5,
		Credentials: map[string]any{"model_mapping": map[string]any{"deepseek-v4-pro": "deepseek-v4-pro"}},
	}
	// 映射账号：key != value，value 为真实上游模型
	mapped := &Account{
		ID: 2, Name: "mapped", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 5,
		Credentials: map[string]any{"model_mapping": map[string]any{"alias": "claude-opus-4-6-thinking"}},
	}
	// 无 model_mapping 账号
	noMapping := &Account{
		ID: 3, Name: "none", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 5,
		Credentials: map[string]any{},
	}

	// 请求名是路由别名 "test"，不在任何账号白名单 key 中。
	// 路由候选账号仍应被判定为可用（用账号自身模型），非候选账号则被过滤。
	routeCtx := WithRouteAccountIDs(context.Background(), map[int64]struct{}{1: {}, 2: {}, 3: {}})

	if !isOpenAIAccountEligibleForRequest(routeCtx, whitelist, "test", false, OpenAIEndpointCapabilityChatCompletions) {
		t.Fatalf("whitelist route candidate should be eligible via own model")
	}
	if !isOpenAIAccountEligibleForRequest(routeCtx, mapped, "test", false, OpenAIEndpointCapabilityChatCompletions) {
		t.Fatalf("mapped route candidate should be eligible via own model value")
	}
	if isOpenAIAccountEligibleForRequest(routeCtx, noMapping, "test", false, OpenAIEndpointCapabilityChatCompletions) {
		t.Fatalf("route candidate without model_mapping must be ineligible")
	}

	// 非路由候选：仍按请求名白名单判断（"test" 不在白名单 → 不可用）
	plainCtx := context.Background()
	if isOpenAIAccountEligibleForRequest(plainCtx, whitelist, "test", false, OpenAIEndpointCapabilityChatCompletions) {
		t.Fatalf("non-route account must be judged by requested model whitelist")
	}
	// 请求名命中白名单时普通账号可用
	if !isOpenAIAccountEligibleForRequest(plainCtx, whitelist, "deepseek-v4-pro", false, OpenAIEndpointCapabilityChatCompletions) {
		t.Fatalf("non-route account should be eligible when requested model is whitelisted")
	}

	// 选中后上游模型 = 账号 model_mapping 第一个 value
	if got := whitelist.FirstModelMappingValue(); got != "deepseek-v4-pro" {
		t.Fatalf("whitelist upstream model = %q, want deepseek-v4-pro", got)
	}
	if got := mapped.FirstModelMappingValue(); got != "claude-opus-4-6-thinking" {
		t.Fatalf("mapped upstream model = %q, want claude-opus-4-6-thinking", got)
	}
}
