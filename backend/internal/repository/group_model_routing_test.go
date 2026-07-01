package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	groupcandidateconfig "github.com/Wei-Shaw/sub2api/ent/groupcandidatetokendailylimitconfig"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

func TestModelRoutingRawPreservesLegacyAndCandidateShapes(t *testing.T) {
	legacy, err := modelRoutingRaw(map[string][]int64{"claude-opus-*": {9, 3, 7}})
	if err != nil {
		t.Fatalf("modelRoutingRaw(legacy) error = %v", err)
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("json.Marshal(legacy) error = %v", err)
	}
	var legacyRoundTrip map[string][]int64
	if err := json.Unmarshal(legacyJSON, &legacyRoundTrip); err != nil {
		t.Fatalf("json.Unmarshal(legacy) error = %v", err)
	}
	if got := legacyRoundTrip["claude-opus-*"]; len(got) != 3 || got[0] != 9 || got[1] != 3 || got[2] != 7 {
		t.Fatalf("legacy round trip = %v", got)
	}

	raw := json.RawMessage(`{"fast-code":[{"model":"gpt-5","account_ids":[4,5],"priority":2,"daily_token_limit":600}]}`)
	candidates, err := modelRoutingRaw(raw)
	if err != nil {
		t.Fatalf("modelRoutingRaw(candidates) error = %v", err)
	}
	candidateJSON, err := json.Marshal(candidates)
	if err != nil {
		t.Fatalf("json.Marshal(candidates) error = %v", err)
	}
	var decoded map[string][]map[string]any
	if err := json.Unmarshal(candidateJSON, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(candidates) error = %v", err)
	}
	got := decoded["fast-code"][0]
	if got["model"] != "gpt-5" || got["priority"] != float64(2) || got["daily_token_limit"] != float64(600) {
		t.Fatalf("candidate scalar fields = %#v", got)
	}
	accounts, ok := got["account_ids"].([]any)
	if !ok || len(accounts) != 2 || accounts[0] != float64(4) || accounts[1] != float64(5) {
		t.Fatalf("candidate account_ids = %#v", got["account_ids"])
	}
}

func TestReplaceGroupCandidateTokenLimitsWritesConfigTable(t *testing.T) {
	ctx := context.Background()
	client := newDailyTokenQuotaRepoTestClient(t)
	group, err := client.Group.Create().SetName("route-group").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := &groupRepository{client: client}
	routing := domain.NewModelRoutingJSON([]byte(`{"test":[{"model":"deepseek-v4-flash","account_ids":[2],"priority":0,"daily_token_limit":10},{"model":"deepseek-v4-pro","account_ids":[1],"priority":1,"daily_token_limit":20}]}`))
	cleanRouting, err := stripDeprecatedRouteDailyTokenLimits(routing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Group.UpdateOneID(group.ID).SetModelRouting(cleanRouting).Save(ctx); err != nil {
		t.Fatal(err)
	}

	if err := repo.replaceGroupCandidateTokenLimits(ctx, group.ID, routing); err != nil {
		t.Fatal(err)
	}
	rows, err := client.GroupCandidateTokenDailyLimitConfig.Query().
		Where(groupcandidateconfig.GroupIDEQ(group.ID)).All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("config row count = %d, want 2", len(rows))
	}
	limits := map[string]int64{}
	for _, row := range rows {
		if row.DailyLimitTokens != nil {
			limits[row.UpstreamModel] = *row.DailyLimitTokens
		}
	}
	if limits["deepseek-v4-flash"] != 10 || limits["deepseek-v4-pro"] != 20 {
		t.Fatalf("stored limits = %v", limits)
	}
	hydrated, err := repo.GetByIDLite(ctx, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	hydratedRouting, err := modelRoutingRaw(hydrated.ModelRouting)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(hydratedRouting.RawMessage(), &response); err != nil {
		t.Fatal(err)
	}
	if response["test"][0]["daily_token_limit"] != float64(10) || response["test"][1]["daily_token_limit"] != float64(20) {
		t.Fatalf("hydrated routing = %s", hydratedRouting.RawMessage())
	}
}

func TestStripDeprecatedRouteDailyTokenLimits(t *testing.T) {
	routing := domain.NewModelRoutingJSON([]byte(`{"test":[{"model":"deepseek-v4-flash","account_ids":[2],"priority":0,"daily_token_limit":10}]}`))
	clean, err := stripDeprecatedRouteDailyTokenLimits(routing)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(clean.RawMessage()), "daily_token_limit") {
		t.Fatalf("deprecated limit persisted in model_routing: %s", clean.RawMessage())
	}
}
