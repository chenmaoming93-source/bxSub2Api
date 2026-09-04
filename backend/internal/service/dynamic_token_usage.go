package service

import (
	"strings"

	tokenstat "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

// submitDynamicTokenUsage is deliberately fail-open. It only performs a
// non-blocking in-memory enqueue and never returns an error to model handling.
func submitDynamicTokenUsage(log *UsageLog, department string) {
	if log == nil || log.UserID <= 0 {
		return
	}
	dimensions := map[tokenstat.DimensionCode]tokenstat.DimensionValue{
		tokenstat.DimensionUserID: tokenstat.Int64Value(log.UserID),
	}
	if department = strings.TrimSpace(department); department == "" {
		department = "未设置"
	}
	dimensions[tokenstat.DimensionDepartment] = tokenstat.StringValue(department)
	if log.APIKeyID > 0 {
		dimensions[tokenstat.DimensionAPIKeyID] = tokenstat.Int64Value(log.APIKeyID)
	}
	if log.GroupID != nil && *log.GroupID > 0 {
		dimensions[tokenstat.DimensionGroupID] = tokenstat.Int64Value(*log.GroupID)
	}
	if alias := strings.TrimSpace(log.RouteAlias); alias != "" {
		dimensions[tokenstat.DimensionRouteAlias] = tokenstat.StringValue(alias)
	}
	if log.AccountID > 0 {
		dimensions[tokenstat.DimensionAccountID] = tokenstat.Int64Value(log.AccountID)
	}
	model := strings.TrimSpace(log.Model)
	if log.UpstreamModel != nil && strings.TrimSpace(*log.UpstreamModel) != "" {
		model = strings.TrimSpace(*log.UpstreamModel)
	}
	if model != "" {
		dimensions[tokenstat.DimensionUpstreamModel] = tokenstat.StringValue(model)
	}
	tokenstat.TryEnqueueDefault(tokenstat.UsageEvent{
		OccurredAt:  log.CreatedAt,
		RequestType: log.EffectiveRequestType().String(),
		Dimensions:  dimensions,
		Metrics: map[tokenstat.MetricCode]int64{
			tokenstat.MetricTotalTokens: int64(log.TotalTokens()),
		},
	})
}
