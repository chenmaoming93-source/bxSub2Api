package handler

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelCandidatesExhaustedDetailsIncludesEveryReason(t *testing.T) {
	status, message := modelCandidatesExhaustedDetails([]service.ModelCandidateFailure{
		{AccountID: 1, AccountName: "primary", Model: "model-a", Reason: "upstream_error", Message: "upstream overloaded"},
		{AccountID: 2, AccountName: "fallback", Model: "model-b", Reason: "token_quota", Message: "Token quota exceeded"},
	})
	require.Equal(t, http.StatusBadGateway, status)
	require.Contains(t, message, "primary(1)")
	require.Contains(t, message, "model-a")
	require.Contains(t, message, "upstream overloaded")
	require.Contains(t, message, "fallback(2)")
	require.Contains(t, message, "token_quota")
}

func TestEveryModelRoutingHandlerAggregatesCandidateFailures(t *testing.T) {
	files := []string{
		"gateway_handler.go",
		"gateway_handler_chat_completions.go",
		"gateway_handler_responses.go",
		"gemini_v1beta_handler.go",
	}
	for _, file := range files {
		source, err := os.ReadFile(file)
		require.NoError(t, err)
		text := string(source)
		require.Contains(t, text, "SelectAccountWithLoadAwarenessForRequest")
		require.Contains(t, text, "AddSelectionFailures", file)
		require.Contains(t, text, "RecordUpstreamFailure", file)
		require.True(t, strings.Contains(text, "handleModelCandidatesExhausted") || strings.Contains(text, "modelCandidatesExhaustedDetails"), file)
	}

	openAIChat, err := os.ReadFile("openai_chat_completions.go")
	require.NoError(t, err)
	text := string(openAIChat)
	require.True(t, strings.Contains(text, "ResolveQuotaAllowedGroupRoute"))
	require.Contains(t, text, "candidateFailures")
	require.Contains(t, text, "all_model_candidates_failed")
}

func TestModelCandidatesExhaustedDetailsUses429WhenAllQuotaLimited(t *testing.T) {
	status, _ := modelCandidatesExhaustedDetails([]service.ModelCandidateFailure{
		{AccountID: 1, Model: "model-a", Reason: "token_quota", Message: "Token quota exceeded"},
	})
	require.Equal(t, http.StatusTooManyRequests, status)
}

func TestModelCandidatesExhaustedDetailsUses429WhenAllRouteConcurrencyLimited(t *testing.T) {
	status, message := modelCandidatesExhaustedDetails([]service.ModelCandidateFailure{
		{AccountID: 1, Model: "model-a", Reason: "route_concurrency", Message: "candidate concurrency limit reached"},
	})
	require.Equal(t, http.StatusTooManyRequests, status)
	require.Contains(t, message, "candidate concurrency limit reached")
}

func TestFailoverStateCombinesUpstreamAndSelectionFailures(t *testing.T) {
	state := NewFailoverState(2, false)
	state.RecordUpstreamFailure(&service.Account{ID: 1, Name: "primary"}, "model-a", &service.UpstreamFailoverError{
		StatusCode:   http.StatusServiceUnavailable,
		ResponseBody: []byte(`{"error":{"message":"capacity exhausted"}}`),
	})
	state.AddSelectionFailures([]service.ModelCandidateFailure{{
		AccountID: 2, AccountName: "fallback", Model: "model-b", Reason: "token_quota", Message: "Token quota exceeded",
	}})
	require.Len(t, state.CandidateFailures, 2)
	require.Equal(t, "HTTP 503: capacity exhausted", state.CandidateFailures[0].Message)
}
