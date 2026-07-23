package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCheckBillingEligibilityTokenStatisticsOnlySkipsMonetaryChecks(t *testing.T) {
	s := &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeStandard}}
	user := &User{ID: 42}
	apiKey := &APIKey{ID: 7, RateLimit5h: 1, RateLimit1d: 1, RateLimit7d: 1}
	group := &Group{ID: 9, SubscriptionType: "subscription", Status: "active"}
	subscription := &UserSubscription{Status: "inactive"}

	require.NoError(t, s.CheckBillingEligibility(context.Background(), user, apiKey, group, subscription, "anthropic"))
	// Concrete monetary implementations remain callable and therefore retained.
	require.NotNil(t, s.checkBalanceEligibility)
	require.NotNil(t, s.checkSubscriptionEligibility)
	require.NotNil(t, s.checkUserPlatformQuotaEligibility)
	require.NotNil(t, s.checkAPIKeyRateLimits)
}
