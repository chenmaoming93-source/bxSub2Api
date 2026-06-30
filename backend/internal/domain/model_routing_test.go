package domain

import (
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
	if candidates[0].Model != "claude-opus-4-1" || !candidates[0].Legacy {
		t.Fatalf("legacy candidate = %+v", candidates[0])
	}
	want := []int64{9, 3, 7}
	for i := range want {
		if candidates[0].AccountIDs[i] != want[i] {
			t.Fatalf("account order = %v, want %v", candidates[0].AccountIDs, want)
		}
	}
}

func TestParseModelRoutingConfigCandidatesStableSortAndUnlimitedNormalization(t *testing.T) {
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
	if got := []string{candidates[0].Model, candidates[1].Model, candidates[2].Model}; got[0] != "first" || got[1] != "second" || got[2] != "third" {
		t.Fatalf("stable priority order = %v", got)
	}
	if candidates[0].DailyTokenLimit != nil || candidates[2].DailyTokenLimit != nil {
		t.Fatalf("zero/null limits were not normalized: %+v", candidates)
	}
	if candidates[1].DailyTokenLimit == nil || *candidates[1].DailyTokenLimit != 500 {
		t.Fatalf("positive limit = %v, want 500", candidates[1].DailyTokenLimit)
	}
}

func TestParseModelRoutingConfigRejectsInvalidCandidates(t *testing.T) {
	tests := []string{
		``,
		`{"bad*pattern":[1]}`,
		`{"route":null}`,
		`{"route":[]}`,
		`{"route":[{"model":"","account_ids":[1],"priority":1}]}`,
		`{"route":[{"model":"model","account_ids":[],"priority":1}]}`,
		`{"route":[{"model":"model","account_ids":[1],"priority":-1}]}`,
		`{"route":[{"model":"model","account_ids":[1],"priority":1,"daily_token_limit":-1}]}`,
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
