package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

func TestFormatSecurityCheckTextSupportsChatResponsesAndGemini(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		body     string
		want     []string
	}{
		{
			name:     "anthropic messages with tool",
			protocol: "anthropic",
			body:     `{"messages":[{"role":"user","content":"find order"},{"role":"assistant","content":[{"type":"tool_use","name":"query_order","input":{"id":1}}]}]}`,
			want:     []string{"[user]\nfind order", "[assistant]", "[tool_call]", "query_order"},
		},
		{
			name:     "openai chat blocks",
			protocol: "openai_chat",
			body:     `{"messages":[{"role":"system","content":"be safe"},{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"data:secret"}}]}]}`,
			want:     []string{"[system]\nbe safe", "[user]", "hello", "[image]"},
		},
		{
			name:     "responses input",
			protocol: "openai_responses",
			body:     `{"input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]},{"type":"function_call","name":"lookup","arguments":"{\"id\":1}"},{"type":"function_call_output","output":"ok"}]}`,
			want:     []string{"[user]", "hello", "[user]", "[tool_call]", "lookup", "[tool_result]"},
		},
		{
			name:     "gemini contents",
			protocol: "gemini",
			body:     `{"contents":[{"role":"user","parts":[{"text":"hello"}]},{"role":"model","parts":[{"functionCall":{"name":"lookup","args":{"id":1}}}]},{"role":"user","parts":[{"functionResponse":{"name":"lookup","response":{"ok":true}}}]}]}`,
			want:     []string{"[user]\nhello", "[model]", "[user]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatSecurityCheckText(tt.protocol, []byte(tt.body))
			if err != nil {
				t.Fatalf("format: %v", err)
			}
			for _, part := range tt.want {
				if !strings.Contains(got, part) {
					t.Errorf("formatted text missing %q: %s", part, got)
				}
			}
			if strings.Contains(got, "data:secret") {
				t.Error("formatted text leaked image data")
			}
		})
	}
}

func TestSingGuardClientSendsStringTextAndQueryTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/classify" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := request["text"].(string); !ok {
			t.Errorf("text must be string: %#v", request["text"])
		}
		if request["task"] != SingGuardQueryTask {
			t.Errorf("task = %#v", request["task"])
		}
		if _, ok := request["threshold"]; ok {
			t.Error("threshold must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task":"query","risks":{"Dangerous_Operations_Tool_Abuse":{"risk_prob":0.1},"Malicious_Code_and_Cyberattack":{"risk_prob":0.1},"Prompt_Injection_and_Jailbreak":{"risk_prob":0.1},"Resource_Abuse":{"risk_prob":0.1},"Sensitive_Information_Stealing":{"risk_prob":0.1}}}`))
	}))
	defer server.Close()
	client, err := NewSingGuardClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	response, _, _, err := client.Classify(context.Background(), "[user]\nhello")
	if err != nil || response.Task != SingGuardQueryTask || len(response.Risks) != 5 {
		t.Fatalf("classify response=%#v err=%v", response, err)
	}
}

func TestNewSingGuardClientUsesPerRequestContextTimeout(t *testing.T) {
	client, err := NewSingGuardClient("http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client.client.Timeout != 0 {
		t.Fatalf("default client timeout = %v, want no shared deadline", client.client.Timeout)
	}
}

func TestSecurityCheckServiceEvaluatesStrictThresholdAndActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"task":"query","risks":{"Dangerous_Operations_Tool_Abuse":{"risk_prob":0.8},"Malicious_Code_and_Cyberattack":{"risk_prob":0.2},"Prompt_Injection_and_Jailbreak":{"risk_prob":0.9},"Resource_Abuse":{"risk_prob":0.1},"Sensitive_Information_Stealing":{"risk_prob":0.1}}}`))
	}))
	defer server.Close()
	client, _ := NewSingGuardClient(server.URL, server.Client())
	service := NewSecurityCheckService(client)
	config := domain.DefaultSecurityCheckConfig()
	config.Enabled = true
	config.Rules = []domain.SecurityCheckRule{
		{Dimension: domain.SingGuardQueryDimensions[0], Threshold: 0.7, Action: domain.SecurityCheckRuleActionWarn},
		{Dimension: domain.SingGuardQueryDimensions[2], Threshold: 0.8, Action: domain.SecurityCheckRuleActionBlock},
	}
	result := service.Check(context.Background(), "openai_chat", []byte(`{"messages":[{"role":"user","content":"hello"}]}`), config)
	if result.Status != domain.SecurityCheckStatusSuccess || result.Decision != domain.SecurityCheckDecisionBlock || !result.IsUnsafe || len(result.TriggeredRules) != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.TriggeredRules[0].RiskProb != 0.9 && result.TriggeredRules[0].RiskProb != 0.8 {
		t.Fatalf("unexpected trigger: %#v", result.TriggeredRules)
	}
}

func TestSecurityCheckServiceHandlesMissingDimensionAndTimeout(t *testing.T) {
	missingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"task":"query","risks":{"Prompt_Injection_and_Jailbreak":{"risk_prob":0.9}}}`))
	}))
	defer missingServer.Close()
	client, _ := NewSingGuardClient(missingServer.URL, missingServer.Client())
	service := NewSecurityCheckService(client)
	config := domain.DefaultSecurityCheckConfig()
	config.Enabled = true
	config.ExceptionAction = domain.SecurityCheckExceptionActionBlock
	config.Rules = []domain.SecurityCheckRule{{Dimension: domain.SingGuardQueryDimensions[0], Threshold: 0.5, Action: domain.SecurityCheckRuleActionBlock}}
	missing := service.Check(context.Background(), "anthropic", []byte(`{"messages":[{"role":"user","content":"hello"}]}`), config)
	if missing.Status != domain.SecurityCheckStatusError || missing.Decision != domain.SecurityCheckDecisionBlock {
		t.Fatalf("missing dimension result: %#v", missing)
	}

	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"task":"query","risks":{}}`))
	}))
	defer timeoutServer.Close()
	timeoutClient, _ := NewSingGuardClient(timeoutServer.URL, timeoutServer.Client())
	timeoutConfig := config
	timeoutConfig.ExceptionAction = domain.SecurityCheckExceptionActionAllow
	timeoutConfig.TimeoutMS = 10
	timed := NewSecurityCheckService(timeoutClient).Check(context.Background(), "anthropic", []byte(`{"messages":[{"role":"user","content":"hello"}]}`), timeoutConfig)
	if timed.Status != domain.SecurityCheckStatusTimeout || timed.Decision != domain.SecurityCheckDecisionAllow {
		t.Fatalf("timeout result: %#v err=%v", timed, timed.Err)
	}
}

func TestSingGuardClientRejectsHTTPAndInvalidJSON(t *testing.T) {
	for _, body := range []struct {
		status int
		body   string
	}{
		{status: http.StatusServiceUnavailable, body: `{"error":"down"}`},
		{status: http.StatusOK, body: `not-json`},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(body.status)
			_, _ = w.Write([]byte(body.body))
		}))
		client, _ := NewSingGuardClient(server.URL, server.Client())
		_, _, _, err := client.Classify(context.Background(), "hello")
		server.Close()
		if err == nil {
			t.Fatalf("expected error for status=%d", body.status)
		}
	}
}

func TestSecurityCheckServiceReturnsSkippedForDisabledOrEmptyContent(t *testing.T) {
	service := NewSecurityCheckService(nil)
	config := domain.DefaultSecurityCheckConfig()
	result := service.Check(context.Background(), "anthropic", []byte(`{"messages":[]}`), config)
	if result.Status != domain.SecurityCheckStatusSkipped || result.Decision != domain.SecurityCheckDecisionAllow {
		t.Fatalf("unexpected skipped result: %#v", result)
	}
	if !reflect.DeepEqual(result.TriggeredRules, []SecurityCheckTriggeredRule(nil)) {
		t.Fatalf("unexpected triggers: %#v", result.TriggeredRules)
	}
}
