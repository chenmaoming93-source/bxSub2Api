package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDefaultSecurityCheckConfigIsValid(t *testing.T) {
	if err := ValidateSecurityCheckConfig(DefaultSecurityCheckConfig()); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestSecurityCheckConfigRoundTripsAsJSON(t *testing.T) {
	want := SecurityCheckConfig{
		Enabled:         true,
		Rules:           []SecurityCheckRule{{Dimension: SingGuardQueryDimensions[2], Threshold: 0.8, Action: SecurityCheckRuleActionBlock}},
		TimeoutMS:       500,
		ExceptionAction: SecurityCheckExceptionActionAllow,
		CollectEnabled:  true,
		SampleRate:      10,
		Version:         1,
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var got SecurityCheckConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if err := ValidateSecurityCheckConfig(got); err != nil {
		t.Fatalf("round-tripped config should be valid: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-tripped config mismatch: got %#v, want %#v", got, want)
	}
}

func TestValidateSecurityCheckConfigRejectsInvalidValues(t *testing.T) {
	base := DefaultSecurityCheckConfig()
	tests := []struct {
		name   string
		mutate func(*SecurityCheckConfig)
	}{
		{"timeout", func(c *SecurityCheckConfig) { c.TimeoutMS = 0 }},
		{"sample rate", func(c *SecurityCheckConfig) { c.SampleRate = 101 }},
		{"exception action", func(c *SecurityCheckConfig) { c.ExceptionAction = "warn" }},
		{"dimension", func(c *SecurityCheckConfig) {
			c.Rules = []SecurityCheckRule{{Dimension: "unknown", Threshold: 0.5, Action: SecurityCheckRuleActionWarn}}
		}},
		{"threshold", func(c *SecurityCheckConfig) {
			c.Rules = []SecurityCheckRule{{Dimension: SingGuardQueryDimensions[0], Threshold: 1.1, Action: SecurityCheckRuleActionWarn}}
		}},
		{"rule action", func(c *SecurityCheckConfig) {
			c.Rules = []SecurityCheckRule{{Dimension: SingGuardQueryDimensions[0], Threshold: 0.5, Action: "allow"}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := base
			tt.mutate(&config)
			if err := ValidateSecurityCheckConfig(config); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
