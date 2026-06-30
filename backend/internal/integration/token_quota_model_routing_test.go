package integration

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestTokenQuotaModelRoutingIntegrationContract(t *testing.T) {
	legacy, err := domain.ParseModelRoutingConfig([]byte(`{"chat":[9,3,7]}`))
	if err != nil {
		t.Fatalf("parse legacy routing: %v", err)
	}
	legacyCandidates := legacy.Match("chat")
	if len(legacyCandidates) != 1 || len(legacyCandidates[0].AccountIDs) != 3 || legacyCandidates[0].AccountIDs[0] != 9 {
		t.Fatalf("legacy routing order changed: %+v", legacyCandidates)
	}

	modern, err := domain.ParseModelRoutingConfig([]byte(`{"chat":[{"model":"gpt-5","account_ids":[4],"priority":20,"daily_token_limit":1000},{"model":"claude-sonnet","account_ids":[8],"priority":10}]}`))
	if err != nil {
		t.Fatalf("parse candidate routing: %v", err)
	}
	candidates := modern.Match("chat")
	if len(candidates) != 2 || candidates[0].Model != "claude-sonnet" || candidates[1].Model != "gpt-5" {
		t.Fatalf("candidate priority changed: %+v", candidates)
	}

	upstream := candidates[0].Model
	usage := &service.UsageLog{
		Model:               "chat",
		RequestedModel:      "chat",
		UpstreamModel:       &upstream,
		InputTokens:         11,
		OutputTokens:        13,
		CacheCreationTokens: 17,
		CacheReadTokens:     19,
	}
	if usage.TotalTokens() != 60 || usage.RequestedModel != "chat" || *usage.UpstreamModel != "claude-sonnet" {
		t.Fatalf("usage identity/accounting contract changed: %+v", usage)
	}
}
