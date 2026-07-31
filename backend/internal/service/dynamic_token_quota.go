package service

import (
	"context"
	"errors"
	"strings"
	"time"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

var ErrDynamicTokenQuotaExceeded = errors.New("dynamic token quota exceeded")

// DynamicTokenRequestIdentity contains dimensions known at authentication time.
// Route resolution appends route_alias, account_id and upstream_model before
// checking candidate-specific quotas.
type DynamicTokenRequestIdentity struct {
	UserID   int64
	APIKeyID int64
}

func checkDynamicTokenQuota(ctx context.Context, values map[tokenstat.DimensionCode]tokenstat.DimensionValue) error {
	decisions := tokenstat.CheckDefaultQuota(ctx, time.Now(), values)
	if tokenstat.HasEnforcedDecision(decisions) {
		return ErrDynamicTokenQuotaExceeded
	}
	return nil
}

func checkDynamicQuotaBeforeScheduling(ctx context.Context, groupID *int64, identity DynamicTokenRequestIdentity) error {
	values := make(map[tokenstat.DimensionCode]tokenstat.DimensionValue, 3)
	if groupID != nil && *groupID > 0 {
		values[tokenstat.DimensionGroupID] = tokenstat.Int64Value(*groupID)
	}
	if identity.UserID > 0 {
		values[tokenstat.DimensionUserID] = tokenstat.Int64Value(identity.UserID)
	}
	if identity.APIKeyID > 0 {
		values[tokenstat.DimensionAPIKeyID] = tokenstat.Int64Value(identity.APIKeyID)
	}
	return checkDynamicTokenQuota(ctx, values)
}

func checkDynamicQuotaAfterSelection(ctx context.Context, groupID *int64, identity DynamicTokenRequestIdentity, routeAlias string, account *Account, upstreamModel string) error {
	values := make(map[tokenstat.DimensionCode]tokenstat.DimensionValue, 6)
	if groupID != nil && *groupID > 0 {
		values[tokenstat.DimensionGroupID] = tokenstat.Int64Value(*groupID)
	}
	if identity.UserID > 0 {
		values[tokenstat.DimensionUserID] = tokenstat.Int64Value(identity.UserID)
	}
	if identity.APIKeyID > 0 {
		values[tokenstat.DimensionAPIKeyID] = tokenstat.Int64Value(identity.APIKeyID)
	}
	if alias := strings.TrimSpace(routeAlias); alias != "" {
		values[tokenstat.DimensionRouteAlias] = tokenstat.StringValue(alias)
	}
	if account != nil && account.ID > 0 {
		values[tokenstat.DimensionAccountID] = tokenstat.Int64Value(account.ID)
	}
	if model := strings.TrimSpace(upstreamModel); model != "" {
		values[tokenstat.DimensionUpstreamModel] = tokenstat.StringValue(model)
	}
	return checkDynamicTokenQuota(ctx, values)
}

// CheckDynamicTokenRouteCandidate is the shared model_routing candidate gate
// used by schedulers that live outside GatewayService.
func CheckDynamicTokenRouteCandidate(ctx context.Context, groupID *int64, identity DynamicTokenRequestIdentity, routeAlias string, account *Account, upstreamModel string) error {
	return checkDynamicQuotaAfterSelection(ctx, groupID, identity, routeAlias, account, upstreamModel)
}
