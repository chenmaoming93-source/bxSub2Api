package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func incrementDailyTokenQuotasForUsage(ctx context.Context, repo DailyTokenQuotaRepository, usageLog *UsageLog, apiKey *APIKey) error {
	if repo == nil || usageLog == nil {
		return nil
	}
	tokens := usageLog.TotalTokens()
	if tokens <= 0 {
		return nil
	}

	groupID := int64(0)
	if usageLog.GroupID != nil {
		groupID = *usageLog.GroupID
	} else if apiKey != nil && apiKey.GroupID != nil {
		groupID = *apiKey.GroupID
	}
	routeAlias := strings.TrimSpace(usageLog.Model)
	if routeAlias == "" {
		routeAlias = strings.TrimSpace(usageLog.RequestedModel)
	}
	upstreamModel := routeAlias
	if usageLog.UpstreamModel != nil && strings.TrimSpace(*usageLog.UpstreamModel) != "" {
		upstreamModel = strings.TrimSpace(*usageLog.UpstreamModel)
	}
	if usageLog.UserID <= 0 || groupID <= 0 || routeAlias == "" || upstreamModel == "" {
		return fmt.Errorf("increment daily token quotas after usage: incomplete identity")
	}

	at := usageLog.CreatedAt
	if at.IsZero() {
		at = time.Now()
	}
	increment := DailyTokenQuotaIncrement{
		ModelKey:          ModelDailyTokenQuotaKey{Model: upstreamModel, At: at},
		UserModelKey:      UserModelDailyTokenQuotaKey{UserID: usageLog.UserID, Model: upstreamModel, At: at},
		GroupCandidateKey: GroupCandidateDailyTokenQuotaKey{GroupID: groupID, RouteAlias: routeAlias, UpstreamModel: upstreamModel, At: at},
		Tokens:            int64(tokens),
	}
	if apiKey != nil && apiKey.Group != nil {
		for _, candidate := range apiKey.Group.GetRoutingCandidates(routeAlias) {
			candidateModel := strings.TrimSpace(candidate.Model)
			if candidateModel == "" {
				candidateModel = routeAlias
			}
			if candidateModel == upstreamModel {
				increment.GroupCandidateDailyLimitTokens = candidate.DailyTokenLimit
				break
			}
		}
	}

	quotaCtx, cancel := detachedBillingContext(ctx)
	defer cancel()
	if err := repo.IncrementDailyTokenQuotas(quotaCtx, increment); err != nil {
		return fmt.Errorf("increment daily token quotas after usage persisted: %w", err)
	}
	return nil
}

func createUsageLogForTokenQuota(ctx context.Context, repo UsageLogRepository, usageLog *UsageLog) (bool, error) {
	if repo == nil || usageLog == nil {
		return false, nil
	}
	usageCtx, cancel := detachedBillingContext(ctx)
	defer cancel()
	return repo.Create(usageCtx, usageLog)
}
