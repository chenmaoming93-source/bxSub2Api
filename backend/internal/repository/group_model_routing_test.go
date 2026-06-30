package repository

import (
	"encoding/json"
	"testing"
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
