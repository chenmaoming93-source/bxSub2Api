package domain

import "fmt"

// SecurityCheckRuleAction is the action taken when a configured risk rule matches.
type SecurityCheckRuleAction string

const (
	SecurityCheckRuleActionBlock SecurityCheckRuleAction = "block"
	SecurityCheckRuleActionWarn  SecurityCheckRuleAction = "warn"
)

// SecurityCheckExceptionAction is the decision used when the checker cannot produce a result.
type SecurityCheckExceptionAction string

const (
	SecurityCheckExceptionActionAllow SecurityCheckExceptionAction = "allow"
	SecurityCheckExceptionActionBlock SecurityCheckExceptionAction = "block"
)

// SecurityCheckDecision is the final request decision.
type SecurityCheckDecision string

const (
	SecurityCheckDecisionAllow SecurityCheckDecision = "allow"
	SecurityCheckDecisionWarn  SecurityCheckDecision = "warn"
	SecurityCheckDecisionBlock SecurityCheckDecision = "block"
)

// SecurityCheckStatus describes how the security check completed.
type SecurityCheckStatus string

const (
	SecurityCheckStatusSkipped SecurityCheckStatus = "skipped"
	SecurityCheckStatusSuccess SecurityCheckStatus = "success"
	SecurityCheckStatusTimeout SecurityCheckStatus = "timeout"
	SecurityCheckStatusError   SecurityCheckStatus = "error"
)

// SecurityCheckRule is one ordered group-level risk rule.
type SecurityCheckRule struct {
	Dimension string                  `json:"dimension"`
	Threshold float64                 `json:"threshold"`
	Action    SecurityCheckRuleAction `json:"action"`
}

// SecurityCheckTriggeredRule identifies one matching configured risk rule.
type SecurityCheckTriggeredRule struct {
	Dimension string                  `json:"dimension"`
	Threshold float64                 `json:"threshold"`
	Action    SecurityCheckRuleAction `json:"action"`
	RiskProb  float64                 `json:"risk_prob"`
}

// SecurityCheckConfig is stored as one JSON value on groups.
type SecurityCheckConfig struct {
	Enabled         bool                         `json:"enabled"`
	Rules           []SecurityCheckRule          `json:"rules"`
	TimeoutMS       int                          `json:"timeout_ms"`
	ExceptionAction SecurityCheckExceptionAction `json:"exception_action"`
	CollectEnabled  bool                         `json:"collect_enabled"`
	SampleRate      int                          `json:"sample_rate"`
	Version         int64                        `json:"version"`
}

// DefaultSecurityCheckConfig returns the backward-compatible disabled default.
func DefaultSecurityCheckConfig() SecurityCheckConfig {
	return SecurityCheckConfig{
		Enabled:         false,
		Rules:           []SecurityCheckRule{},
		TimeoutMS:       500,
		ExceptionAction: SecurityCheckExceptionActionAllow,
		CollectEnabled:  false,
		SampleRate:      10,
		Version:         1,
	}
}

// SingGuardQueryDimensions are the five query-side dimensions supported by SingGuard.
var SingGuardQueryDimensions = []string{
	"Dangerous_Operations_Tool_Abuse",
	"Malicious_Code_and_Cyberattack",
	"Prompt_Injection_and_Jailbreak",
	"Resource_Abuse",
	"Sensitive_Information_Stealing",
}

// NormalizeSecurityCheckConfig returns the backward-compatible default for a zero value.
func NormalizeSecurityCheckConfig(config SecurityCheckConfig) SecurityCheckConfig {
	if !config.Enabled && len(config.Rules) == 0 && config.TimeoutMS == 0 && config.ExceptionAction == "" && !config.CollectEnabled && config.SampleRate == 0 && config.Version == 0 {
		return DefaultSecurityCheckConfig()
	}
	if config.Rules == nil {
		config.Rules = []SecurityCheckRule{}
	}
	if config.TimeoutMS == 0 {
		config.TimeoutMS = DefaultSecurityCheckConfig().TimeoutMS
	}
	if config.ExceptionAction == "" {
		config.ExceptionAction = SecurityCheckExceptionActionAllow
	}
	if config.Version == 0 {
		config.Version = DefaultSecurityCheckConfig().Version
	}
	return config
}

// ValidateSecurityCheckConfig validates the persisted group-level policy.
func ValidateSecurityCheckConfig(config SecurityCheckConfig) error {
	if config.TimeoutMS <= 0 {
		return fmt.Errorf("security check timeout_ms must be greater than zero")
	}
	if config.SampleRate < 0 || config.SampleRate > 100 {
		return fmt.Errorf("security check sample_rate must be between 0 and 100")
	}
	if config.ExceptionAction != SecurityCheckExceptionActionAllow && config.ExceptionAction != SecurityCheckExceptionActionBlock {
		return fmt.Errorf("security check exception_action must be allow or block")
	}
	allowed := make(map[string]struct{}, len(SingGuardQueryDimensions))
	for _, dimension := range SingGuardQueryDimensions {
		allowed[dimension] = struct{}{}
	}
	for i, rule := range config.Rules {
		if _, ok := allowed[rule.Dimension]; !ok {
			return fmt.Errorf("security check rule %d has unsupported dimension %q", i, rule.Dimension)
		}
		if rule.Threshold < 0 || rule.Threshold > 1 {
			return fmt.Errorf("security check rule %d threshold must be between 0 and 1", i)
		}
		if rule.Action != SecurityCheckRuleActionBlock && rule.Action != SecurityCheckRuleActionWarn {
			return fmt.Errorf("security check rule %d action must be block or warn", i)
		}
	}
	return nil
}
