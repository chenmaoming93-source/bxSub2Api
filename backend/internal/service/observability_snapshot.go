package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

type SnapshotRequest struct {
	Window      observability.Window
	Bucket      observability.Bucket
	Start       time.Time
	End         time.Time
	Filter      observability.Filter
	Scope       string
	Search      string
	OptionLimit int
}

type SnapshotResult struct {
	Response observability.SnapshotResponse
	ETag     string
	CacheHit bool
}

type SnapshotRollupTotals struct {
	RequestCount         int64
	SuccessCount         int64
	ErrorCount           int64
	SLAErrorCount        int64
	BusinessLimitedCount int64
	RateLimitedCount     int64
	LegacyInferredCount  int64
	InputTokens          int64
	OutputTokens         int64
	CacheCreationTokens  int64
	CacheReadTokens      int64
	TotalTokens          int64
	Duration             observability.Histogram
	TTFT                 observability.Histogram
}

type SnapshotDimensionRow struct {
	ID              int64
	RouteAlias      string
	RequestCount    int64
	ErrorCount      int64
	SLAErrorCount   int64
	TotalTokens     int64
	Duration        observability.Histogram
	TTFT            observability.Histogram
	BusinessLimited int64
}

type ObservabilitySnapshotRollup struct {
	Totals      SnapshotRollupTotals
	Users       []SnapshotDimensionRow
	Groups      []SnapshotDimensionRow
	Aliases     []SnapshotDimensionRow
	Accounts    []SnapshotDimensionRow
	SourceMaxAt *time.Time
}

type SnapshotRollupLoader interface {
	LoadSnapshotRollup(ctx context.Context, start, end time.Time, bucket observability.Bucket, filter observability.Filter) (ObservabilitySnapshotRollup, error)
	LoadSnapshotSeries(ctx context.Context, start, end time.Time, bucket observability.Bucket, filter observability.Filter, dimension string, selectedIDs []int64, selectedAliases []string) ([]SnapshotSeriesBucket, error)
}

type SnapshotDimensionNameLoader interface {
	LoadSnapshotDimensionNames(ctx context.Context, dimension string, ids []int64) (map[int64]string, error)
}

type SnapshotConcurrencyReader interface {
	CollectCurrent(ctx context.Context, sampledAt time.Time) ([]ConcurrencyCurrentValue, error)
	QueryHistory(ctx context.Context, filter ConcurrencyHistoryFilter) (ConcurrencyHistoryResult, error)
}

type SnapshotSeriesBucket struct {
	BucketStart time.Time
	DimensionID int64
	RouteAlias  string
	Totals      SnapshotRollupTotals
}

type snapshotCacheEntry struct {
	result    SnapshotResult
	expiresAt time.Time
}

type ObservabilitySnapshotService struct {
	loader      SnapshotRollupLoader
	accountRepo AccountRepository
	userRepo    UserRepository
	groupRepo   GroupRepository
	concurrency SnapshotConcurrencyReader
	tokenstat   *tokenstat.ProjectionAdminService

	cacheMu sync.Mutex
	cache   map[string]snapshotCacheEntry
}

func NewObservabilitySnapshotService(
	loader SnapshotRollupLoader,
	accountRepo AccountRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	concurrency SnapshotConcurrencyReader,
	tokenstatProvider *tokenstat.ProjectionAdminService,
) *ObservabilitySnapshotService {
	return &ObservabilitySnapshotService{
		loader:      loader,
		accountRepo: accountRepo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		concurrency: concurrency,
		tokenstat:   tokenstatProvider,
		cache:       make(map[string]snapshotCacheEntry),
	}
}

func (s *ObservabilitySnapshotService) Snapshot(ctx context.Context, request SnapshotRequest) (SnapshotResult, error) {
	if s == nil || s.loader == nil {
		return SnapshotResult{}, fmt.Errorf("observability snapshot service is unavailable")
	}
	if request.OptionLimit <= 0 {
		request.OptionLimit = 20
	}
	if request.OptionLimit > observability.MaxFilterValues {
		request.OptionLimit = observability.MaxFilterValues
	}
	request.Search = strings.TrimSpace(request.Search)
	key := snapshotRequestCacheKey(request)
	now := time.Now()
	s.cacheMu.Lock()
	if cached, ok := s.cache[key]; ok && now.Before(cached.expiresAt) {
		cached.result.CacheHit = true
		s.cacheMu.Unlock()
		return cached.result, nil
	}
	s.cacheMu.Unlock()

	start, end := request.Start, request.End
	if start.IsZero() || end.IsZero() {
		resolved, resolveErr := observability.ResolveRange(request.Window, request.Bucket, "", "", now, timezone.Location())
		if resolveErr != nil {
			return SnapshotResult{}, resolveErr
		}
		request.Window, request.Bucket, start, end = resolved.Window, resolved.Bucket, resolved.Start, resolved.End
	} else {
		if request.Bucket == "" {
			request.Bucket = observability.DefaultBucketForDuration(end.Sub(start))
		}
		if !start.Before(end) || end.Sub(start) > observability.MaxRange || end.After(now.UTC()) {
			return SnapshotResult{}, fmt.Errorf("invalid observability range")
		}
	}
	rollup, err := s.loader.LoadSnapshotRollup(ctx, start, end, request.Bucket, request.Filter)
	if err != nil {
		return SnapshotResult{}, err
	}
	response, err := s.buildSnapshot(ctx, request, rollup, start, end, now)
	if err != nil {
		return SnapshotResult{}, err
	}
	body, err := json.Marshal(response)
	if err != nil {
		return SnapshotResult{}, err
	}
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:12]) + `"`
	response.Meta.ETag = etag
	result := SnapshotResult{Response: response, ETag: etag}
	s.cacheMu.Lock()
	for cacheKey, entry := range s.cache {
		if !now.Before(entry.expiresAt) {
			delete(s.cache, cacheKey)
		}
	}
	s.cache[key] = snapshotCacheEntry{result: result, expiresAt: now.Add(5 * time.Second)}
	s.cacheMu.Unlock()
	return result, nil
}

func (s *ObservabilitySnapshotService) buildSnapshot(
	ctx context.Context,
	request SnapshotRequest,
	rollup ObservabilitySnapshotRollup,
	start, end, now time.Time,
) (observability.SnapshotResponse, error) {
	warnings := make([]string, 0)
	metrics := make([]observability.MetricValue, 0, 8)
	metrics = append(metrics,
		observability.MetricValue{Code: "total_tokens", Value: float64(rollup.Totals.TotalTokens), Unit: "token"},
		observability.MetricValue{Code: "request_count", Value: float64(rollup.Totals.RequestCount), Unit: "request"},
		observability.MetricValue{Code: "token_rate", Value: float64(rollup.Totals.TotalTokens) / end.Sub(start).Seconds(), Unit: "token/s"},
		observability.MetricValue{Code: "client_failure_rate", Value: ratio(rollup.Totals.ErrorCount, rollup.Totals.RequestCount), Unit: "ratio"},
		observability.MetricValue{Code: "sla_error_rate", Value: ratio(rollup.Totals.SLAErrorCount, rollup.Totals.RequestCount), Unit: "ratio"},
		observability.MetricValue{Code: "business_limited_rate", Value: ratio(rollup.Totals.BusinessLimitedCount, rollup.Totals.RequestCount), Unit: "ratio"},
	)
	durationP95 := rollup.Totals.Duration.Quantiles(0.95)[0.95]
	ttftP95 := rollup.Totals.TTFT.Quantiles(0.95)[0.95]
	avgDuration := 0.0
	if rollup.Totals.Duration.SampleCount > 0 {
		avgDuration = float64(rollup.Totals.Duration.SumMS) / float64(rollup.Totals.Duration.SampleCount)
	}
	avgTTFT := 0.0
	if rollup.Totals.TTFT.SampleCount > 0 {
		avgTTFT = float64(rollup.Totals.TTFT.SumMS) / float64(rollup.Totals.TTFT.SampleCount)
	}
	metrics = append(metrics,
		observability.MetricValue{Code: "avg_duration_ms", Value: avgDuration, Unit: "ms"},
		observability.MetricValue{Code: "avg_first_token_ms", Value: avgTTFT, Unit: "ms"},
		observability.MetricValue{Code: "p95_duration_ms", Value: float64(durationP95), Unit: "ms"},
		observability.MetricValue{Code: "p95_first_token_ms", Value: float64(ttftP95), Unit: "ms"},
	)

	var tokenstatCurrent *int64
	if s.tokenstat != nil {
		currentTokens, tokenWarnings := s.loadTokenStatCurrent(ctx, request.Filter, now)
		warnings = append(warnings, tokenWarnings...)
		if currentTokens != nil {
			tokenstatCurrent = currentTokens
			metrics = append(metrics, observability.MetricValue{Code: "tokenstat_current_tokens", Value: float64(*currentTokens), Unit: "token"})
		}
	} else {
		warnings = append(warnings, "tokenstat_projection_unavailable")
	}
	if tokenstatCurrent != nil && rollup.Totals.TotalTokens > 0 {
		denominator := float64(rollup.Totals.TotalTokens)
		if float64(*tokenstatCurrent) > denominator {
			denominator = float64(*tokenstatCurrent)
		}
		difference := float64(*tokenstatCurrent - rollup.Totals.TotalTokens)
		if difference < 0 {
			difference = -difference
		}
		if denominator > 0 && difference/denominator > 0.05 {
			warnings = append(warnings, "tokenstat_rollup_reconciliation_difference")
		}
	}

	users, groups, accounts := s.loadDimensionLabels(ctx, rollup)
	topUsers := mapUserRows(rollup.Users, users)
	topGroups := mapGroupRows(rollup.Groups, groups)
	topAliases := mapAliasRows(rollup.Aliases)
	topAccounts := mapAccountRows(rollup.Accounts, accounts)
	topModels := modelRows(rollup.Accounts, accounts)

	pressureAccounts, pressureRoutes, capacity, pressureErr := s.loadPressure(ctx, now, request.Filter, accounts, groups)
	if pressureErr != nil {
		warnings = append(warnings, "concurrency_snapshot_unavailable")
	}
	pressureAccounts, pressureRoutes = s.enrichPressureLabels(ctx, pressureAccounts, pressureRoutes, accounts, groups)
	capacity, capacityWarnings := s.enrichCapacity(ctx, request, now, capacity)
	warnings = append(warnings, capacityWarnings...)
	filterOptions := loadFilterOptions(
		request.Search, request.OptionLimit, rollup, users, groups, accounts, pressureAccounts, pressureRoutes,
	)
	if request.Bucket == observability.Bucket5m && request.Window == observability.Window7d {
		warnings = append(warnings, "bucket_window_invalid")
	}
	lag := int64(0)
	complete := true
	if rollup.SourceMaxAt == nil {
		warnings = append(warnings, "request_rollup_unavailable")
		complete = false
	} else if now.Sub(*rollup.SourceMaxAt) > 2*time.Minute {
		warnings = append(warnings, "request_rollup_lagging")
		lag = int64(now.Sub(*rollup.SourceMaxAt).Seconds())
		complete = false
	}
	warnings = uniqueStrings(warnings)
	meta := observability.Meta{
		AsOf: now.UTC(), StartAt: start.UTC(), EndAt: end.UTC(), Window: request.Window, Bucket: request.Bucket,
		Timezone:    timezone.Location().String(),
		Sources:     []string{"observability_request_bucket", "observability_concurrency_bucket", "tokenstat"},
		Consistency: "eventual", LagSeconds: lag, Complete: complete,
		Warnings: warnings, Coverage: "full",
	}
	return observability.SnapshotResponse{
		Meta: meta, Metrics: metrics,
		TopUsers: topUsers, TopGroups: topGroups, TopAliases: topAliases,
		TopAccounts: topAccounts, TopModels: topModels,
		Pressure:      observability.SnapshotPressure{Accounts: pressureAccounts, Candidates: pressureRoutes},
		Capacity:      capacity,
		FilterOptions: filterOptions,
	}, nil
}

func (s *ObservabilitySnapshotService) loadTokenStatCurrent(ctx context.Context, filter observability.Filter, now time.Time) (*int64, []string) {
	projection, err := s.tokenstat.ResolveUsageProjection(ctx,
		[]tokenstat.DimensionCode{tokenstat.DimensionUserID},
		tokenstat.MetricTotalTokens,
		tokenstat.PeriodDay,
	)
	if err != nil || projection == nil {
		return nil, []string{"tokenstat_projection_unavailable"}
	}
	start := time.Date(now.In(timezone.Location()).Year(), now.In(timezone.Location()).Month(), now.In(timezone.Location()).Day(), 0, 0, 0, 0, timezone.Location())
	filters := make(map[tokenstat.DimensionCode]tokenstat.DimensionValue)
	if len(filter.UserIDs) == 1 {
		filters[tokenstat.DimensionUserID] = tokenstat.Int64Value(filter.UserIDs[0])
	}
	result, err := s.tokenstat.QueryUsage(ctx, tokenstat.UsageQueryInput{
		ProjectionID: projection.ID,
		MetricCode:   tokenstat.MetricTotalTokens,
		PeriodType:   tokenstat.PeriodDay,
		Start:        start, End: start.AddDate(0, 0, 1),
		Filters: filters,
		GroupBy: []tokenstat.DimensionCode{tokenstat.DimensionUserID},
		Page:    1, PageSize: 1,
	})
	if err != nil {
		return nil, []string{"tokenstat_projection_unavailable"}
	}
	value := result.Summary
	warnings := make([]string, 0)
	if !result.Complete {
		warnings = append(warnings, "tokenstat_projection_lagging")
	}
	return &value, warnings
}

func (s *ObservabilitySnapshotService) loadDimensionLabels(
	ctx context.Context,
	rollup ObservabilitySnapshotRollup,
) (map[int64]User, map[int64]*Group, map[int64]*Account) {
	users := make(map[int64]User)
	if s.userRepo != nil {
		userIDs := snapshotDimensionIDs(rollup.Users)
		if batch, ok := s.userRepo.(interface {
			GetByIDs(context.Context, []int64) ([]User, error)
		}); ok {
			if values, err := batch.GetByIDs(ctx, userIDs); err == nil {
				for index := range values {
					users[values[index].ID] = values[index]
				}
			}
		} else {
			for _, id := range userIDs {
				if user, err := s.userRepo.GetByID(ctx, id); err == nil && user != nil {
					users[id] = *user
				}
			}
		}
	}
	groups := make(map[int64]*Group)
	if s.groupRepo != nil {
		groupIDs := snapshotDimensionIDs(rollup.Groups)
		if batch, ok := s.groupRepo.(interface {
			GetByIDsLite(context.Context, []int64) ([]Group, error)
		}); ok {
			if values, err := batch.GetByIDsLite(ctx, groupIDs); err == nil {
				for index := range values {
					groups[values[index].ID] = &values[index]
				}
			}
		} else {
			for _, id := range groupIDs {
				if group, err := s.groupRepo.GetByID(ctx, id); err == nil && group != nil {
					groups[id] = group
				}
			}
		}
	}
	accounts := make(map[int64]*Account)
	ids := make([]int64, 0, len(rollup.Accounts))
	for _, row := range rollup.Accounts {
		if row.ID > 0 {
			ids = append(ids, row.ID)
		}
	}
	if len(ids) > 0 && s.accountRepo != nil {
		if values, err := s.accountRepo.GetByIDs(ctx, ids); err == nil {
			for _, account := range values {
				if account != nil {
					accounts[account.ID] = account
				}
			}
		}
	}
	return users, groups, accounts
}

func snapshotDimensionIDs(rows []SnapshotDimensionRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.ID > 0 {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func (s *ObservabilitySnapshotService) loadPressure(
	ctx context.Context,
	now time.Time,
	filter observability.Filter,
	accounts map[int64]*Account,
	groups map[int64]*Group,
) ([]observability.PressureRow, []observability.PressureRow, observability.CapacitySummary, error) {
	if s.concurrency == nil {
		return nil, nil, observability.CapacitySummary{}, fmt.Errorf("concurrency history unavailable")
	}
	rows, err := s.concurrency.CollectCurrent(ctx, now)
	if err != nil {
		return nil, nil, observability.CapacitySummary{}, err
	}
	accountRows := make([]observability.PressureRow, 0)
	routeRows := make([]observability.PressureRow, 0)
	seenAccounts := make(map[int64]struct{})
	var globalCurrent, globalLimit, globalWaiting int
	for _, row := range rows {
		if len(filter.AccountIDs) > 0 && !containsInt64(filter.AccountIDs, row.AccountID) {
			continue
		}
		if row.Kind == ConcurrencySampleKindRoute && len(filter.GroupIDs) > 0 && !containsInt64(filter.GroupIDs, row.GroupID) {
			continue
		}
		if row.Kind == ConcurrencySampleKindRoute && len(filter.RouteAliases) > 0 && !containsString(filter.RouteAliases, row.RouteAlias) {
			continue
		}
		item := observability.PressureRow{
			AccountID: row.AccountID, GroupID: row.GroupID, RouteAlias: row.RouteAlias,
			Current: row.Current, Limit: row.Limit, Waiting: row.Waiting,
		}
		if row.Limit > 0 {
			item.Utilization = float64(row.Current) / float64(row.Limit)
		}
		if account := accounts[row.AccountID]; account != nil {
			item.AccountName = account.Name
		}
		if group := groups[row.GroupID]; group != nil {
			item.GroupName = group.Name
		}
		if row.Kind == ConcurrencySampleKindAccount {
			if _, exists := seenAccounts[row.AccountID]; exists {
				continue
			}
			seenAccounts[row.AccountID] = struct{}{}
			accountRows = append(accountRows, item)
			globalCurrent += row.Current
			globalLimit += row.Limit
			globalWaiting += row.Waiting
		} else {
			routeRows = append(routeRows, item)
		}
	}
	sortPressureRows(accountRows)
	sortPressureRows(routeRows)
	capacity := observability.CapacitySummary{
		Global:       capacityLevel("全局并发槽", globalCurrent, globalLimit),
		QueueCurrent: globalWaiting,
	}
	if len(accountRows) > 0 {
		capacity.Account = capacityLevel(accountRows[0].AccountName, accountRows[0].Current, accountRows[0].Limit)
	}
	if len(routeRows) > 0 {
		route := routeRows[0]
		routeName := route.RouteAlias
		if routeName == "" {
			routeName = formatInt(route.AccountID)
		}
		capacity.Route = capacityLevel(routeName, route.Current, route.Limit)
		capacity.User = capacityLevel(route.GroupName, route.Current, route.Limit)
	}
	if len(accountRows) > 10 {
		accountRows = accountRows[:10]
	}
	if len(routeRows) > 10 {
		routeRows = routeRows[:10]
	}
	return accountRows, routeRows, capacity, nil
}

func (s *ObservabilitySnapshotService) enrichPressureLabels(
	ctx context.Context,
	accountRows, routeRows []observability.PressureRow,
	accounts map[int64]*Account,
	groups map[int64]*Group,
) ([]observability.PressureRow, []observability.PressureRow) {
	missingAccountIDs := make([]int64, 0)
	seenAccounts := make(map[int64]struct{})
	missingGroupIDs := make([]int64, 0)
	seenGroups := make(map[int64]struct{})
	addGroup := func(groupID int64) {
		if groupID <= 0 || groups[groupID] != nil {
			return
		}
		if _, exists := seenGroups[groupID]; exists {
			return
		}
		seenGroups[groupID] = struct{}{}
		missingGroupIDs = append(missingGroupIDs, groupID)
	}
	for _, row := range append(append([]observability.PressureRow{}, accountRows...), routeRows...) {
		if row.AccountID > 0 && accounts[row.AccountID] == nil {
			if _, exists := seenAccounts[row.AccountID]; !exists {
				seenAccounts[row.AccountID] = struct{}{}
				missingAccountIDs = append(missingAccountIDs, row.AccountID)
			}
		}
		addGroup(row.GroupID)
	}
	if len(missingAccountIDs) > 0 && s.accountRepo != nil {
		if values, err := s.accountRepo.GetByIDs(ctx, missingAccountIDs); err == nil {
			for _, account := range values {
				if account != nil {
					accounts[account.ID] = account
				}
			}
		}
	}
	if len(missingGroupIDs) > 0 && s.groupRepo != nil {
		if batch, ok := s.groupRepo.(interface {
			GetByIDsLite(context.Context, []int64) ([]Group, error)
		}); ok {
			if values, err := batch.GetByIDsLite(ctx, missingGroupIDs); err == nil {
				for index := range values {
					groups[values[index].ID] = &values[index]
				}
			}
		}
	}
	for index := range accountRows {
		if account := accounts[accountRows[index].AccountID]; account != nil {
			accountRows[index].AccountName = account.Name
		}
	}
	for index := range routeRows {
		if account := accounts[routeRows[index].AccountID]; account != nil {
			routeRows[index].AccountName = account.Name
		}
		if group := groups[routeRows[index].GroupID]; group != nil {
			routeRows[index].GroupName = group.Name
		}
	}
	return accountRows, routeRows
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func capacityLevel(name string, current, limit int) observability.CapacityLevel {
	level := observability.CapacityLevel{Name: name, Current: current, Limit: limit, Available: true}
	if limit > 0 {
		level.Utilization = float64(current) / float64(limit)
	}
	return level
}

func (s *ObservabilitySnapshotService) enrichCapacity(
	ctx context.Context,
	request SnapshotRequest,
	now time.Time,
	capacity observability.CapacitySummary,
) (observability.CapacitySummary, []string) {
	warnings := make([]string, 0)
	if s.concurrency != nil {
		filter := ConcurrencyHistoryFilter{
			Start: now.Add(-5 * time.Minute), End: now, WindowSeconds: ObservabilityWindowFiveMinute,
		}
		if len(request.Filter.AccountIDs) == 1 {
			filter.AccountID = request.Filter.AccountIDs[0]
		}
		if len(request.Filter.GroupIDs) == 1 {
			filter.GroupID = request.Filter.GroupIDs[0]
		}
		if len(request.Filter.RouteAliases) == 1 {
			filter.RouteAlias = request.Filter.RouteAliases[0]
		}
		history, err := s.concurrency.QueryHistory(ctx, filter)
		if err != nil {
			warnings = append(warnings, "wait_duration_unavailable")
		} else {
			var waitSum, waitSamples int64
			for _, row := range history.Rows {
				if row.SampleKind == ConcurrencySampleKindAccount {
					waitSum += row.WaitSumMS
					waitSamples += row.WaitSampleCount
				}
			}
			capacity.WaitSamples = waitSamples
			if waitSamples > 0 {
				average := float64(waitSum) / float64(waitSamples)
				capacity.AverageWaitMS = &average
			} else {
				warnings = append(warnings, "wait_duration_unavailable")
			}
		}
	} else {
		warnings = append(warnings, "wait_duration_unavailable")
	}

	localNow := now.In(timezone.Location())
	startOfDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, timezone.Location())
	var todayRateLimited int64
	var rateErr error
	if loader, ok := s.loader.(interface {
		LoadRateLimitedCount(context.Context, time.Time, time.Time, observability.Bucket, observability.Filter) (int64, error)
	}); ok {
		todayRateLimited, rateErr = loader.LoadRateLimitedCount(
			ctx, startOfDay.UTC(), now.UTC(), observability.Bucket1h, request.Filter,
		)
	} else {
		var todayRollup ObservabilitySnapshotRollup
		todayRollup, rateErr = s.loader.LoadSnapshotRollup(
			ctx, startOfDay.UTC(), now.UTC(), observability.Bucket1h, request.Filter,
		)
		todayRateLimited = todayRollup.Totals.RateLimitedCount
	}
	if rateErr != nil {
		warnings = append(warnings, "today_rate_limit_unavailable")
	} else {
		capacity.TodayRateLimited = todayRateLimited
		capacity.RateLimitAvailable = true
	}
	return capacity, warnings
}

func loadFilterOptions(
	search string,
	limit int,
	rollup ObservabilitySnapshotRollup,
	users map[int64]User,
	groups map[int64]*Group,
	accounts map[int64]*Account,
	pressureAccounts, pressureRoutes []observability.PressureRow,
) observability.SnapshotFilterOptions {
	result := observability.SnapshotFilterOptions{}
	for _, row := range rollup.Users {
		user, ok := users[row.ID]
		if !ok || (!matchesSearch(user.Email, search) && !matchesSearch(user.Department, search)) {
			continue
		}
		result.Users = append(result.Users, observability.UserFilterOption{
			UserID: user.ID, Email: user.Email, Department: user.Department,
		})
	}
	for _, row := range rollup.Groups {
		group, ok := groups[row.ID]
		if !ok || !matchesSearch(group.Name, search) {
			continue
		}
		result.Groups = append(result.Groups, observability.NamedFilterOption{ID: group.ID, Name: group.Name})
		if len(result.Groups) >= limit {
			break
		}
	}
	seenGroups := make(map[int64]struct{})
	for _, row := range rollup.Groups {
		seenGroups[row.ID] = struct{}{}
	}
	for _, row := range append(append([]observability.PressureRow{}, pressureAccounts...), pressureRoutes...) {
		groupID := row.GroupID
		if groupID <= 0 || len(result.Groups) >= limit {
			continue
		}
		if _, exists := seenGroups[groupID]; exists {
			continue
		}
		group, ok := groups[groupID]
		if !ok || !matchesSearch(group.Name, search) {
			continue
		}
		seenGroups[groupID] = struct{}{}
		result.Groups = append(result.Groups, observability.NamedFilterOption{ID: group.ID, Name: group.Name})
	}
	for _, row := range rollup.Accounts {
		account, ok := accounts[row.ID]
		if !ok || !matchesSearch(account.Name, search) {
			continue
		}
		result.Accounts = append(result.Accounts, observability.AccountFilterOption{
			AccountID: account.ID, Name: account.Name, BoundModel: account.BoundModel(),
		})
		if len(result.Accounts) >= limit {
			break
		}
	}
	seenAccounts := make(map[int64]struct{})
	for _, row := range rollup.Accounts {
		seenAccounts[row.ID] = struct{}{}
	}
	for _, row := range pressureAccounts {
		accountID := row.AccountID
		if accountID <= 0 || len(result.Accounts) >= limit {
			continue
		}
		if _, exists := seenAccounts[accountID]; exists {
			continue
		}
		account, ok := accounts[accountID]
		if !ok || !matchesSearch(account.Name, search) {
			continue
		}
		seenAccounts[accountID] = struct{}{}
		result.Accounts = append(result.Accounts, observability.AccountFilterOption{
			AccountID: account.ID, Name: account.Name, BoundModel: account.BoundModel(),
		})
	}
	for _, row := range rollup.Aliases {
		if strings.TrimSpace(row.RouteAlias) == "" || !matchesSearch(row.RouteAlias, search) {
			continue
		}
		result.Aliases = append(result.Aliases, observability.NamedFilterOption{Name: row.RouteAlias})
		if len(result.Aliases) >= limit {
			break
		}
	}
	seenAliases := make(map[string]struct{})
	for _, row := range rollup.Aliases {
		seenAliases[row.RouteAlias] = struct{}{}
	}
	for _, row := range pressureRoutes {
		alias := strings.TrimSpace(row.RouteAlias)
		if alias == "" || len(result.Aliases) >= limit || !matchesSearch(alias, search) {
			continue
		}
		if _, exists := seenAliases[alias]; exists {
			continue
		}
		seenAliases[alias] = struct{}{}
		result.Aliases = append(result.Aliases, observability.NamedFilterOption{Name: alias})
	}
	return result
}

func mapUserRows(rows []SnapshotDimensionRow, labels map[int64]User) []observability.DimensionRow {
	out := make([]observability.DimensionRow, 0, len(rows))
	for _, row := range rows {
		name := strconv.FormatInt(row.ID, 10)
		if user, ok := labels[row.ID]; ok {
			name = "#" + strconv.FormatInt(row.ID, 10) + " · " + user.Email
			if user.Department != "" {
				name += " · " + user.Department
			}
		}
		out = append(out, observability.DimensionRow{Key: strconv.FormatInt(row.ID, 10), Name: name, Value: row.TotalTokens})
	}
	return out
}

func mapGroupRows(rows []SnapshotDimensionRow, labels map[int64]*Group) []observability.DimensionRow {
	out := make([]observability.DimensionRow, 0, len(rows))
	for _, row := range rows {
		name := strconv.FormatInt(row.ID, 10)
		if group := labels[row.ID]; group != nil {
			name = group.Name
		}
		out = append(out, observability.DimensionRow{Key: strconv.FormatInt(row.ID, 10), Name: name, Value: row.TotalTokens})
	}
	return out
}

func mapAliasRows(rows []SnapshotDimensionRow) []observability.DimensionRow {
	out := make([]observability.DimensionRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, observability.DimensionRow{Key: row.RouteAlias, Name: row.RouteAlias, Value: row.TotalTokens})
	}
	return out
}

func mapAccountRows(rows []SnapshotDimensionRow, labels map[int64]*Account) []observability.DimensionRow {
	out := make([]observability.DimensionRow, 0, len(rows))
	for _, row := range rows {
		name := strconv.FormatInt(row.ID, 10)
		if account := labels[row.ID]; account != nil {
			name = account.Name
		}
		out = append(out, observability.DimensionRow{Key: strconv.FormatInt(row.ID, 10), Name: name, Value: row.TotalTokens})
	}
	return out
}

func modelRows(rows []SnapshotDimensionRow, labels map[int64]*Account) []observability.DimensionRow {
	totals := make(map[string]int64)
	for _, row := range rows {
		if account := labels[row.ID]; account != nil && account.BoundModel() != "" {
			totals[account.BoundModel()] += row.TotalTokens
		}
	}
	out := make([]observability.DimensionRow, 0, len(totals))
	for model, tokens := range totals {
		out = append(out, observability.DimensionRow{Key: model, Name: model, Value: tokens})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

func sortPressureRows(rows []observability.PressureRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Utilization != rows[j].Utilization {
			return rows[i].Utilization > rows[j].Utilization
		}
		return rows[i].Waiting > rows[j].Waiting
	})
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func matchesSearch(value, search string) bool {
	search = strings.TrimSpace(strings.ToLower(search))
	return search == "" || strings.Contains(strings.ToLower(value), search)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func snapshotRequestCacheKey(request SnapshotRequest) string {
	cacheRequest := request
	switch request.Window {
	case observability.Window1h, observability.Window6h, observability.Window24h, observability.Window7d,
		observability.WindowToday, observability.WindowYesterday:
		// Rolling presets share one short cache slot; the absolute bounds still
		// flow through the service when the five-second entry is rebuilt.
		cacheRequest.Start = time.Time{}
		cacheRequest.End = time.Time{}
	}
	body, _ := json.Marshal(struct {
		Request SnapshotRequest
	}{Request: cacheRequest})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
