package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type tokenQuotaAccountingRepoStub struct {
	increments []DailyTokenQuotaIncrement
	err        error
}

func (s *tokenQuotaAccountingRepoStub) GetModelDailyTokenQuota(context.Context, ModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *tokenQuotaAccountingRepoStub) GetUserModelDailyTokenQuota(context.Context, UserModelDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *tokenQuotaAccountingRepoStub) GetGroupCandidateDailyTokenQuota(context.Context, GroupCandidateDailyTokenQuotaKey) (DailyTokenQuotaSnapshot, error) {
	return DailyTokenQuotaSnapshot{}, nil
}

func (s *tokenQuotaAccountingRepoStub) IncrementDailyTokenQuotas(_ context.Context, increment DailyTokenQuotaIncrement) error {
	s.increments = append(s.increments, increment)
	return s.err
}

func newTokenQuotaGatewayService(usageRepo UsageLogRepository, billingRepo UsageBillingRepository, quotaRepo DailyTokenQuotaRepository) *GatewayService {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	return &GatewayService{
		usageLogRepo:        usageRepo,
		usageBillingRepo:    billingRepo,
		cfg:                 cfg,
		billingService:      NewBillingService(cfg, nil),
		billingCacheService: &BillingCacheService{},
		deferredService:     &DeferredService{},
		dailyTokenQuotaRepo: quotaRepo,
	}
}

func tokenQuotaTestAPIKey(groupID int64, routeAlias, upstreamModel string, limit int64) *APIKey {
	return &APIKey{
		ID:      10,
		GroupID: &groupID,
		Group: &Group{
			ID:                  groupID,
			RateMultiplier:      1,
			ModelRoutingEnabled: true,
			ModelRouting: map[string]any{
				routeAlias: []map[string]any{{
					"model":             upstreamModel,
					"account_ids":       []int64{30},
					"priority":          0,
					"daily_token_limit": limit,
				}},
			},
		},
	}
}

func TestRecordUsageTokenQuotaAccountingCountsAllClaudeTokensOnce(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: false}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	quotaRepo := &tokenQuotaAccountingRepoStub{}
	svc := newTokenQuotaGatewayService(usageRepo, billingRepo, quotaRepo)
	apiKey := tokenQuotaTestAPIKey(20, "claude-route", "claude-sonnet-4", 500)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID:     "claude-token-quota",
			Model:         "claude-route",
			UpstreamModel: "claude-sonnet-4",
			Usage: ClaudeUsage{
				InputTokens:              11,
				OutputTokens:             13,
				CacheCreationInputTokens: 17,
				CacheReadInputTokens:     19,
			},
			Duration: time.Second,
		},
		APIKey: apiKey,
		User:   &User{ID: 40},
		Account: &Account{
			ID: 30,
		},
	})

	require.NoError(t, err)
	require.Len(t, quotaRepo.increments, 1)
	increment := quotaRepo.increments[0]
	require.Equal(t, int64(60), increment.Tokens)
	require.Equal(t, "claude-sonnet-4", increment.ModelKey.Model)
	require.Equal(t, int64(40), increment.UserModelKey.UserID)
	require.Equal(t, int64(20), increment.GroupCandidateKey.GroupID)
	require.Equal(t, "claude-route", increment.GroupCandidateKey.RouteAlias)
	require.Equal(t, "claude-sonnet-4", increment.GroupCandidateKey.UpstreamModel)
}

func TestRecordUsageTokenQuotaAccountingCountsAllOpenAITokensOnce(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: false}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	quotaRepo := &tokenQuotaAccountingRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	svc.usageBillingRepo = billingRepo
	svc.dailyTokenQuotaRepo = quotaRepo
	apiKey := tokenQuotaTestAPIKey(21, "gpt-route", "gpt-5", 700)

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:     "openai-token-quota",
			Model:         "gpt-5",
			UpstreamModel: "gpt-5",
			Usage: OpenAIUsage{
				InputTokens:              30,
				OutputTokens:             7,
				CacheCreationInputTokens: 5,
				CacheReadInputTokens:     11,
			},
			Duration: time.Second,
		},
		APIKey:     apiKey,
		RouteAlias: "gpt-route",
		User:       &User{ID: 41},
		Account: &Account{
			ID:       31,
			Platform: PlatformOpenAI,
		},
	})

	require.NoError(t, err)
	require.Len(t, quotaRepo.increments, 1)
	require.Equal(t, int64(42), quotaRepo.increments[0].Tokens)
	require.Equal(t, "gpt-5", quotaRepo.increments[0].ModelKey.Model)
	require.Equal(t, "gpt-route", quotaRepo.increments[0].GroupCandidateKey.RouteAlias)
	require.Equal(t, "gpt-5", quotaRepo.increments[0].GroupCandidateKey.UpstreamModel)
}

func TestTokenQuotaAccountingSkipsFailedOrDuplicateUsagePersistence(t *testing.T) {
	quotaRepo := &tokenQuotaAccountingRepoStub{}
	apiKey := tokenQuotaTestAPIKey(22, "route", "model", 100)
	input := &RecordUsageInput{
		Result:  &ForwardResult{RequestID: "failed-usage", Model: "route", UpstreamModel: "model", Usage: ClaudeUsage{InputTokens: 3}, Duration: time.Second},
		APIKey:  apiKey,
		User:    &User{ID: 42},
		Account: &Account{ID: 32},
	}

	billingFailure := &openAIRecordUsageBillingRepoStub{err: errors.New("usage transaction failed")}
	svc := newTokenQuotaGatewayService(&openAIRecordUsageLogRepoStub{}, billingFailure, quotaRepo)
	require.Error(t, svc.RecordUsage(context.Background(), input))
	require.Empty(t, quotaRepo.increments)

	duplicate := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: false}}
	svc = newTokenQuotaGatewayService(&openAIRecordUsageLogRepoStub{}, duplicate, quotaRepo)
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	require.Empty(t, quotaRepo.increments)
}

func TestTokenQuotaAccountingSimpleModeRequiresUsageInsert(t *testing.T) {
	quotaRepo := &tokenQuotaAccountingRepoStub{}
	usageRepo := &openAIRecordUsageLogRepoStub{err: errors.New("usage insert failed")}
	svc := newTokenQuotaGatewayService(usageRepo, nil, quotaRepo)
	svc.cfg.RunMode = config.RunModeSimple
	apiKey := tokenQuotaTestAPIKey(23, "route", "model", 100)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result:  &ForwardResult{RequestID: "simple-write-failure", Model: "route", UpstreamModel: "model", Usage: ClaudeUsage{InputTokens: 3}, Duration: time.Second},
		APIKey:  apiKey,
		User:    &User{ID: 43},
		Account: &Account{ID: 33},
	})

	require.NoError(t, err)
	require.Empty(t, quotaRepo.increments)
}

func TestTokenQuotaAccountingReturnsIncrementFailureWithoutRetry(t *testing.T) {
	quotaRepo := &tokenQuotaAccountingRepoStub{err: errors.New("quota database unavailable")}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newTokenQuotaGatewayService(&openAIRecordUsageLogRepoStub{}, billingRepo, quotaRepo)
	apiKey := tokenQuotaTestAPIKey(24, "route", "model", 100)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result:  &ForwardResult{RequestID: "quota-failure", Model: "route", UpstreamModel: "model", Usage: ClaudeUsage{InputTokens: 3}, Duration: time.Second},
		APIKey:  apiKey,
		User:    &User{ID: 44},
		Account: &Account{ID: 34},
	})

	require.ErrorContains(t, err, "increment daily token quotas after usage persisted")
	require.Len(t, quotaRepo.increments, 1)
}
