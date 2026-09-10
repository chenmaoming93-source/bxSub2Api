package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestParseModelRoutingConfigLegacyPreservesAccountOrderAndWildcard(t *testing.T) {
	config, err := ParseModelRoutingConfig([]byte(`{"claude-opus-*":[9,3,7]}`))
	if err != nil {
		t.Fatalf("ParseModelRoutingConfig() error = %v", err)
	}

	candidates := config.Match("claude-opus-4-1")
	if len(candidates) != 1 {
		t.Fatalf("Match() candidate count = %d, want 1", len(candidates))
	}
	if !candidates[0].Legacy {
		t.Fatalf("legacy candidate = %+v", candidates[0])
	}
	want := []int64{9, 3, 7}
	for i := range want {
		if candidates[0].AccountIDs[i] != want[i] {
			t.Fatalf("account order = %v, want %v", candidates[0].AccountIDs, want)
		}
	}
}

func TestParseModelRoutingConfigCandidatesStableSort(t *testing.T) {
	config, err := ParseModelRoutingConfig([]byte(`{
		"fast-code":[
			{"model":"third","account_ids":[3],"priority":20,"daily_token_limit":null},
			{"model":"first","account_ids":[1],"priority":10,"daily_token_limit":0},
			{"model":"second","account_ids":[2],"priority":10,"daily_token_limit":500}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseModelRoutingConfig() error = %v", err)
	}

	candidates := config.Match("fast-code")
	if got := []int64{candidates[0].AccountIDs[0], candidates[1].AccountIDs[0], candidates[2].AccountIDs[0]}; got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("stable priority order = %v", got)
	}
	encoded, err := json.Marshal(candidates)
	if err != nil || bytes.Contains(encoded, []byte(`"model"`)) {
		t.Fatalf("candidate JSON = %s, err = %v; model must not be emitted", encoded, err)
	}
}

func TestParseModelRoutingConfigRejectsInvalidCandidates(t *testing.T) {
	tests := []string{
		``,
		`{"bad*pattern":[1]}`,
		`{"route":null}`,
		`{"route":[]}`,
		`{"route":[{"account_ids":[],"priority":1}]}`,
		`{"route":[{"model":"model","account_ids":[],"priority":1}]}`,
		`{"route":[{"model":"model","account_ids":[1],"priority":-1}]}`,
	}
	for _, input := range tests {
		_, err := ParseModelRoutingConfig([]byte(input))
		if !errors.Is(err, ErrInvalidModelRouting) {
			t.Errorf("ParseModelRoutingConfig(%q) error = %v, want ErrInvalidModelRouting", input, err)
		}
	}
}

func TestParseModelRoutingConfigNullIsEmpty(t *testing.T) {
	config, err := ParseModelRoutingConfig([]byte(`null`))
	if err != nil {
		t.Fatalf("ParseModelRoutingConfig(null) error = %v", err)
	}
	if len(config) != 0 || config.Match("anything") != nil {
		t.Fatalf("null config = %#v, want empty", config)
	}
}

func TestGroupCandidatesByPriorityDeduplicatesWithinTier(t *testing.T) {
	candidates := []ModelRouteCandidate{
		{AccountIDs: []int64{1, 2}, Priority: 1},
		{AccountIDs: []int64{2, 3}, Priority: 1},
		{AccountIDs: []int64{4}, Priority: 2},
	}
	tiers, err := GroupCandidatesByPriority(candidates)
	if err != nil {
		t.Fatalf("GroupCandidatesByPriority() error = %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("tier count = %d, want 2", len(tiers))
	}
	if got := tiers[0].AccountIDs; len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("tier accounts = %v, want [1 2 3]", got)
	}
}

func TestGroupCandidatesByPriorityRejectsCrossPriorityDuplicate(t *testing.T) {
	_, err := GroupCandidatesByPriority([]ModelRouteCandidate{
		{AccountIDs: []int64{1}, Priority: 1},
		{AccountIDs: []int64{1}, Priority: 2},
	})
	if !errors.Is(err, ErrInvalidModelRouting) {
		t.Fatalf("error = %v, want ErrInvalidModelRouting", err)
	}
}
