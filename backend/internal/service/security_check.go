package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

const (
	SingGuardQueryTask       = "query"
	SingGuardMaxTextChars    = 100000
	SingGuardMaxResponseSize = 4 << 20
)

// SingGuardRisk is one dimension's classifier result.
type SingGuardRisk struct {
	RiskProb float64 `json:"risk_prob"`
	Label    string  `json:"label"`
}

// SingGuardResponse is the Query response returned by SingGuard.
type SingGuardResponse struct {
	Label         string                   `json:"label"`
	MaxRiskDomain string                   `json:"max_risk_domain"`
	MaxRiskProb   float64                  `json:"max_risk_prob"`
	Risks         map[string]SingGuardRisk `json:"risks"`
	Task          string                   `json:"task"`
	Threshold     float64                  `json:"threshold"`
	LatencyMS     float64                  `json:"latency_ms"`
}

// SingGuardClient calls the internal /classify endpoint.
type SingGuardClient struct {
	baseURL string
	client  *http.Client
}

func NewSingGuardClient(baseURL string, client *http.Client) (*SingGuardClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("singguard base URL is empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &SingGuardClient{baseURL: baseURL, client: client}, nil
}

func (c *SingGuardClient) Classify(ctx context.Context, text string) (SingGuardResponse, []byte, time.Duration, error) {
	started := time.Now()
	var result SingGuardResponse
	if strings.TrimSpace(text) == "" {
		return result, nil, time.Since(started), errors.New("singguard text is empty")
	}
	if len([]rune(text)) > SingGuardMaxTextChars {
		return result, nil, time.Since(started), fmt.Errorf("singguard text exceeds %d characters", SingGuardMaxTextChars)
	}
	payload, err := json.Marshal(struct {
		Text string `json:"text"`
		Task string `json:"task"`
	}{Text: text, Task: SingGuardQueryTask})
	if err != nil {
		return result, nil, time.Since(started), err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/classify", bytes.NewReader(payload))
	if err != nil {
		return result, nil, time.Since(started), err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return result, nil, time.Since(started), err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, SingGuardMaxResponseSize+1))
	if err != nil {
		return result, raw, time.Since(started), err
	}
	if len(raw) > SingGuardMaxResponseSize {
		return result, raw[:SingGuardMaxResponseSize], time.Since(started), errors.New("singguard response exceeds size limit")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, raw, time.Since(started), fmt.Errorf("singguard returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, raw, time.Since(started), fmt.Errorf("decode singguard response: %w", err)
	}
	if result.Task != "" && result.Task != SingGuardQueryTask {
		return result, raw, time.Since(started), fmt.Errorf("unexpected singguard task %q", result.Task)
	}
	if len(result.Risks) == 0 {
		return result, raw, time.Since(started), errors.New("singguard response contains no risks")
	}
	return result, raw, time.Since(started), nil
}

// SecurityCheckTriggeredRule identifies a matching configured rule.
type SecurityCheckTriggeredRule struct {
	Dimension string                         `json:"dimension"`
	Threshold float64                        `json:"threshold"`
	Action    domain.SecurityCheckRuleAction `json:"action"`
	RiskProb  float64                        `json:"risk_prob"`
}

// SecurityCheckResult is the protocol-neutral result consumed by Gateway handlers and collectors.
type SecurityCheckResult struct {
	Status           domain.SecurityCheckStatus
	Decision         domain.SecurityCheckDecision
	IsUnsafe         bool
	TriggeredRules   []SecurityCheckTriggeredRule
	Response         *SingGuardResponse
	RawResponse      []byte
	Latency          time.Duration
	SingGuardLatency time.Duration
	Err              error
}

// SecurityCheckService formats a request, calls SingGuard and evaluates group rules.
type SecurityCheckService struct {
	client *SingGuardClient
}

func NewSecurityCheckService(client *SingGuardClient) *SecurityCheckService {
	return &SecurityCheckService{client: client}
}

func (s *SecurityCheckService) Check(ctx context.Context, protocol string, body []byte, config domain.SecurityCheckConfig) SecurityCheckResult {
	started := time.Now()
	result := SecurityCheckResult{Status: domain.SecurityCheckStatusSkipped, Decision: domain.SecurityCheckDecisionAllow}
	config = domain.NormalizeSecurityCheckConfig(config)
	if !config.Enabled || len(config.Rules) == 0 {
		return result
	}
	text, err := FormatSecurityCheckText(protocol, body)
	if err != nil {
		return securityCheckErrorResult(result, started, config, err)
	}
	if strings.TrimSpace(text) == "" {
		return result
	}
	if s == nil || s.client == nil {
		return securityCheckErrorResult(result, started, config, errors.New("singguard client is not configured"))
	}
	if config.TimeoutMS <= 0 {
		return securityCheckErrorResult(result, started, config, errors.New("security check timeout_ms must be greater than zero"))
	}
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMS)*time.Millisecond)
	defer cancel()
	response, raw, singGuardLatency, err := s.client.Classify(checkCtx, text)
	result.RawResponse = raw
	result.SingGuardLatency = singGuardLatency
	result.Latency = time.Since(started)
	if err != nil {
		result.Err = err
		result.Status = domain.SecurityCheckStatusError
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
			result.Status = domain.SecurityCheckStatusTimeout
		}
		result.Decision = exceptionDecision(config)
		return result
	}
	result.Status = domain.SecurityCheckStatusSuccess
	result.Response = &response
	for _, rule := range config.Rules {
		risk, ok := response.Risks[rule.Dimension]
		if !ok {
			return securityCheckErrorResult(result, started, config, fmt.Errorf("singguard response missing configured dimension %q", rule.Dimension))
		}
		if risk.RiskProb <= rule.Threshold {
			continue
		}
		result.IsUnsafe = true
		result.TriggeredRules = append(result.TriggeredRules, SecurityCheckTriggeredRule{
			Dimension: rule.Dimension,
			Threshold: rule.Threshold,
			Action:    rule.Action,
			RiskProb:  risk.RiskProb,
		})
		if rule.Action == domain.SecurityCheckRuleActionBlock {
			result.Decision = domain.SecurityCheckDecisionBlock
			result.Latency = time.Since(started)
			return result
		}
	}
	if result.IsUnsafe {
		result.Decision = domain.SecurityCheckDecisionWarn
	}
	result.Latency = time.Since(started)
	return result
}

func securityCheckErrorResult(result SecurityCheckResult, started time.Time, config domain.SecurityCheckConfig, err error) SecurityCheckResult {
	result.Status = domain.SecurityCheckStatusError
	result.Decision = exceptionDecision(config)
	result.Err = err
	result.Latency = time.Since(started)
	return result
}

func exceptionDecision(config domain.SecurityCheckConfig) domain.SecurityCheckDecision {
	if config.ExceptionAction == domain.SecurityCheckExceptionActionBlock {
		return domain.SecurityCheckDecisionBlock
	}
	return domain.SecurityCheckDecisionAllow
}

// FormatSecurityCheckText converts supported request bodies into readable role-formatted text.
func FormatSecurityCheckText(protocol string, body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode %s request: %w", protocol, err)
	}
	var items any
	switch strings.ToLower(protocol) {
	case "anthropic", "anthropic_messages", "anthropic-messages", "openai_chat", "openai-chat", "chat_completions", "openai_chat_completions":
		items = payload["messages"]
	case "openai_responses", "openai-responses", "responses":
		items = payload["input"]
	case "gemini", "gemini_v1beta", "gemini-v1beta":
		items = payload["contents"]
	default:
		return "", fmt.Errorf("unsupported security check protocol %q", protocol)
	}
	return formatConversation(items), nil
}

func formatConversation(items any) string {
	if text, ok := items.(string); ok {
		return "[user]\n" + text
	}
	list, ok := items.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(item["role"])
		if role == "" {
			role = "user"
		}
		content := formatMessageContent(item["content"])
		if content == "" {
			content = formatMessageContent(item["parts"])
		}
		if content == "" {
			content = formatToolMessage(item)
		}
		if content != "" {
			parts = append(parts, "["+role+"]\n"+content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func formatMessageContent(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	list, ok := value.([]any)
	if !ok {
		return ""
	}
	var out []string
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ := stringValue(item["type"])
		switch typ {
		case "text", "input_text", "output_text":
			if text := stringValue(item["text"]); text != "" {
				out = append(out, text)
			}
		case "tool_use", "function_call":
			name := stringValue(item["name"])
			args := item["input"]
			if args == nil {
				args = item["arguments"]
			}
			out = append(out, formatToolCall(name, args))
		case "tool_result", "function_call_output":
			out = append(out, formatToolResult(item["content"], item["output"]))
		case "image", "input_image", "image_url":
			out = append(out, "[image]")
		default:
			if call, ok := item["functionCall"].(map[string]any); ok {
				out = append(out, formatToolCall(stringValue(call["name"]), call["args"]))
				continue
			}
			if response, ok := item["functionResponse"].(map[string]any); ok {
				out = append(out, formatToolResult(nil, response["response"]))
				continue
			}
			if text := stringValue(item["text"]); text != "" {
				out = append(out, text)
			}
		}
	}
	return strings.Join(out, "\n")
}

func formatToolMessage(item map[string]any) string {
	if calls, ok := item["tool_calls"].([]any); ok {
		var out []string
		for _, raw := range calls {
			call, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			function, _ := call["function"].(map[string]any)
			out = append(out, formatToolCall(stringValue(function["name"]), function["arguments"]))
		}
		return strings.Join(out, "\n")
	}
	if name := stringValue(item["name"]); name != "" {
		if args := item["arguments"]; args != nil {
			return formatToolCall(name, args)
		}
		if output := item["output"]; output != nil {
			return formatToolResult(nil, output)
		}
	}
	if output := item["output"]; output != nil {
		return formatToolResult(nil, output)
	}
	return ""
}

func formatToolCall(name string, args any) string {
	if name == "" {
		name = "unknown"
	}
	data, _ := json.Marshal(args)
	return "[tool_call]\nname: " + name + "\narguments: " + string(data)
}

func formatToolResult(content, output any) string {
	if content == nil {
		content = output
	}
	data, _ := json.Marshal(content)
	return "[tool_result]\ncontent: " + string(data)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
