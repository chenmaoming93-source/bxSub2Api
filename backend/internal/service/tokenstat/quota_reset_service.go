package tokenstat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const defaultQuotaResetBatchSize = 100

var ErrTokenQuotaResetUnavailable = errors.New("token quota reset unavailable")

type QuotaResetStatus string

const (
	QuotaResetStatusReset        QuotaResetStatus = "RESET"
	QuotaResetStatusPartialReset QuotaResetStatus = "PARTIAL_RESET"
	QuotaResetStatusNoQuota      QuotaResetStatus = "NO_QUOTA"
	QuotaResetStatusNoUsage      QuotaResetStatus = "NO_USAGE"
)

type QuotaResetEntryStatus string

const (
	QuotaResetEntryReset   QuotaResetEntryStatus = "RESET"
	QuotaResetEntryNoUsage QuotaResetEntryStatus = "NO_USAGE"
	QuotaResetEntryFailed  QuotaResetEntryStatus = "FAILED"
)

type QuotaResetEntryResult struct {
	Status QuotaResetEntryStatus
	Err    error
}

type QuotaResetMatchedQuota struct {
	ID              int64                            `json:"id"`
	Name            string                           `json:"name"`
	ProjectionID    int64                            `json:"projection_id"`
	DimensionValues map[DimensionCode]DimensionValue `json:"dimension_values"`
	MetricCode      MetricCode                       `json:"metric_code"`
	PeriodType      PeriodType                       `json:"period_type"`
	LimitValue      int64                            `json:"limit_value"`
	Mode            QuotaMode                        `json:"mode"`
}

type QuotaResetMatchedEntry struct {
	ProjectionID    int64                            `json:"projection_id"`
	ProjectionName  string                           `json:"projection_name"`
	DimensionValues map[DimensionCode]DimensionValue `json:"dimension_values"`
	MetricCode      MetricCode                       `json:"metric_code"`
	PeriodType      PeriodType                       `json:"period_type"`
	PeriodStart     time.Time                        `json:"period_start"`
	PeriodEnd       time.Time                        `json:"period_end"`
	MatchedQuotaIDs []int64                          `json:"matched_quota_ids"`
	Status          QuotaResetEntryStatus            `json:"status"`
}

type QuotaResetResult struct {
	Status            QuotaResetStatus         `json:"status"`
	MatchedQuotaCount int                      `json:"matched_quota_count"`
	MatchedUsageCount int                      `json:"matched_usage_count"`
	ResetCount        int                      `json:"reset_count"`
	NoUsageCount      int                      `json:"no_usage_count"`
	FailedCount       int                      `json:"failed_count"`
	MatchedQuotas     []QuotaResetMatchedQuota `json:"matched_quotas,omitempty"`
	MatchedEntries    []QuotaResetMatchedEntry `json:"matched_entries,omitempty"`
}

type QuotaResetIdentityDiscoverer interface {
	Discover(ctx context.Context, request ResetIdentityDiscoveryRequest) (IdentityDiscoveryResult, error)
}

type QuotaResetSnapshotWriter interface {
	SnapshotBatch(ctx context.Context, identities []StatisticIdentity) ([]QuotaResetEntryResult, error)
}

type TokenStatisticsRuntime interface{ Enabled() bool }

type QuotaResetService struct {
	discovery QuotaResetIdentityDiscoverer
	writer    QuotaResetSnapshotWriter
	runtime   TokenStatisticsRuntime
	batchSize int
}

func NewQuotaResetService(discovery QuotaResetIdentityDiscoverer, writer QuotaResetSnapshotWriter, runtime TokenStatisticsRuntime, batchSize int) *QuotaResetService {
	if batchSize <= 0 {
		batchSize = defaultQuotaResetBatchSize
	}
	return &QuotaResetService{discovery: discovery, writer: writer, runtime: runtime, batchSize: batchSize}
}

func (s *QuotaResetService) ResetQuotaUsage(ctx context.Context, request ResetIdentityDiscoveryRequest) (QuotaResetResult, error) {
	started := time.Now()
	observability.quotaResetRequests.Add(1)
	if s == nil || s.runtime == nil || !s.runtime.Enabled() || s.discovery == nil || s.writer == nil {
		observability.quotaResetFailures.Add(1)
		return QuotaResetResult{}, ErrTokenQuotaResetUnavailable
	}

	discovery, err := s.discovery.Discover(ctx, request)
	if err != nil {
		if errors.Is(err, ErrInvalidQuotaResetRequest) {
			return QuotaResetResult{}, err
		}
		observability.quotaResetFailures.Add(1)
		return QuotaResetResult{}, fmt.Errorf("%w: %v", ErrTokenQuotaResetUnavailable, err)
	}
	result := QuotaResetResult{MatchedQuotaCount: discovery.MatchedQuotaCount, MatchedUsageCount: len(discovery.Identities)}
	if request.IncludeDetails {
		result.MatchedQuotas = quotaResetMatchedQuotas(discovery.MatchedQuotas)
		result.MatchedEntries = quotaResetMatchedEntries(discovery.Identities)
	}
	if discovery.Status == IdentityDiscoveryNoQuota {
		result.Status = QuotaResetStatusNoQuota
		observability.quotaResetNoQuota.Add(1)
		s.logResult(request, result, started)
		return result, nil
	}
	if discovery.Status == IdentityDiscoveryNoUsage || len(discovery.Identities) == 0 {
		result.Status = QuotaResetStatusNoUsage
		observability.quotaResetNoUsage.Add(1)
		s.logResult(request, result, started)
		return result, nil
	}

	for offset := 0; offset < len(discovery.Identities); offset += s.batchSize {
		if err := ctx.Err(); err != nil {
			result.FailedCount += len(discovery.Identities) - offset
			break
		}
		end := offset + s.batchSize
		if end > len(discovery.Identities) {
			end = len(discovery.Identities)
		}
		batch := discovery.Identities[offset:end]
		entries, batchErr := s.writer.SnapshotBatch(ctx, batch)
		if len(entries) != len(batch) {
			result.FailedCount += len(batch)
			slog.WarnContext(ctx, "dynamic token quota reset batch returned incomplete results", "batch_size", len(batch), "result_count", len(entries))
			continue
		}
		for index, entry := range entries {
			if request.IncludeDetails {
				result.MatchedEntries[offset+index].Status = entry.Status
			}
			switch entry.Status {
			case QuotaResetEntryReset:
				result.ResetCount++
			case QuotaResetEntryNoUsage:
				result.NoUsageCount++
			default:
				result.FailedCount++
			}
		}
		if batchErr != nil {
			slog.WarnContext(ctx, "dynamic token quota reset batch completed with errors", "batch_size", len(batch), "error", batchErr)
		}
	}

	observability.quotaResetEntries.Add(uint64(result.ResetCount))
	observability.quotaResetFailedEntries.Add(uint64(result.FailedCount))
	switch {
	case result.FailedCount > 0 && result.ResetCount > 0:
		result.Status = QuotaResetStatusPartialReset
		observability.quotaResetPartial.Add(1)
	case result.FailedCount > 0:
		observability.quotaResetFailures.Add(1)
		s.logResult(request, result, started)
		return result, ErrTokenQuotaResetUnavailable
	case result.ResetCount == 0:
		result.Status = QuotaResetStatusNoUsage
		observability.quotaResetNoUsage.Add(1)
	default:
		result.Status = QuotaResetStatusReset
	}
	s.logResult(request, result, started)
	return result, nil
}

func quotaResetMatchedQuotas(rules []QuotaRule) []QuotaResetMatchedQuota {
	result := make([]QuotaResetMatchedQuota, 0, len(rules))
	for _, rule := range rules {
		result = append(result, QuotaResetMatchedQuota{
			ID: rule.ID, Name: rule.Name, ProjectionID: rule.ProjectionID,
			DimensionValues: rule.DimensionValues, MetricCode: rule.MetricCode,
			PeriodType: rule.PeriodType, LimitValue: rule.LimitValue, Mode: rule.Mode,
		})
	}
	return result
}

func quotaResetMatchedEntries(identities []StatisticIdentity) []QuotaResetMatchedEntry {
	result := make([]QuotaResetMatchedEntry, 0, len(identities))
	for _, identity := range identities {
		result = append(result, QuotaResetMatchedEntry{
			ProjectionID: identity.ProjectionID, ProjectionName: identity.ProjectionName,
			DimensionValues: identity.DimensionValues, MetricCode: identity.MetricCode,
			PeriodType: identity.Period.Type, PeriodStart: identity.Period.Start, PeriodEnd: identity.Period.End,
			MatchedQuotaIDs: identity.MatchedQuotaIDs, Status: QuotaResetEntryFailed,
		})
	}
	return result
}

func (s *QuotaResetService) logResult(request ResetIdentityDiscoveryRequest, result QuotaResetResult, started time.Time) {
	slog.Info("dynamic token quota reset completed",
		"period_type", request.PeriodType, "metric_code", request.MetricCode,
		"dimension_count", len(request.DimensionValues), "status", result.Status,
		"matched_quota_count", result.MatchedQuotaCount, "matched_usage_count", result.MatchedUsageCount,
		"reset_count", result.ResetCount, "no_usage_count", result.NoUsageCount, "failed_count", result.FailedCount,
		"duration_ms", time.Since(started).Milliseconds())
}
