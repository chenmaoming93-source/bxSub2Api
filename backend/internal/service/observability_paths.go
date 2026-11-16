package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

type ObservabilityPathQuery struct {
	Start       time.Time
	End         time.Time
	Window      observability.Window
	Bucket      observability.Bucket
	Filter      observability.Filter
	Search      string
	Status      string
	Sort        string
	Page        int
	PageSize    int
	IncludeIdle bool
}

type ObservabilityPathAggregate struct {
	UserID          int64
	APIKeyID        int64
	GroupID         int64
	RouteAlias      string
	AccountID       int64
	RequestCount    int64
	SuccessCount    int64
	ErrorCount      int64
	SLAErrorCount   int64
	BusinessLimited int64
	LegacyInferred  int64
	TotalTokens     int64
	Duration        observability.Histogram
	TTFT            observability.Histogram
	DataAt          time.Time
}

type APIKeyRelation struct {
	UserID  int64
	KeyID   int64
	KeyName string
	GroupID int64
}

type RouteRelation struct {
	GroupID    int64
	RouteAlias string
	AccountID  int64
	Limit      *int
}

type ObservabilityPathLoader interface {
	LoadPathAggregates(ctx context.Context, query ObservabilityPathQuery) ([]ObservabilityPathAggregate, int, error)
	LoadAPIKeyRelations(ctx context.Context, filter observability.Filter, search string) ([]APIKeyRelation, error)
	LoadRouteRelations(ctx context.Context, filter observability.Filter) ([]RouteRelation, error)
}

type ObservabilityPathErrorStageLoader interface {
	LoadPathErrorStages(ctx context.Context, start, end time.Time, refs []PathRef) (map[string]string, error)
}

type PathRef struct {
	UserID     int64
	APIKeyID   int64
	GroupID    int64
	RouteAlias string
	AccountID  int64
}

type PathWindowMetrics struct {
	AverageDurationMS   float64
	AverageFirstTokenMS float64
}

type ObservabilityPathService struct {
	loader      ObservabilityPathLoader
	userRepo    UserRepository
	accountRepo AccountRepository
	groupRepo   GroupRepository
	concurrency *ConcurrencyHistoryService
}

func NewObservabilityPathService(loader ObservabilityPathLoader, userRepo UserRepository, accountRepo AccountRepository, groupRepo GroupRepository, concurrency *ConcurrencyHistoryService) *ObservabilityPathService {
	return &ObservabilityPathService{loader: loader, userRepo: userRepo, accountRepo: accountRepo, groupRepo: groupRepo, concurrency: concurrency}
}

func (s *ObservabilityPathService) Paths(ctx context.Context, query ObservabilityPathQuery) (observability.PathsResponse, error) {
	if s == nil || s.loader == nil {
		return observability.PathsResponse{}, fmt.Errorf("observability path service is unavailable")
	}
	if query.Start.IsZero() || query.End.IsZero() {
		resolved, err := observability.ResolveRange(query.Window, query.Bucket, "", "", time.Now().UTC(), timezone.Location())
		if err != nil {
			return observability.PathsResponse{}, err
		}
		query.Window, query.Bucket, query.Start, query.End = resolved.Window, resolved.Bucket, resolved.Start, resolved.End
	}
	if query.End.Sub(query.Start) >= observability.MaxRange {
		query.Page = 1
		if query.PageSize <= 0 || query.PageSize > 100 {
			query.PageSize = 100
		}
	}
	if query.Page <= 0 || query.PageSize <= 0 || query.PageSize > 100 {
		return observability.PathsResponse{}, fmt.Errorf("invalid path pagination")
	}
	rows, total, err := s.loader.LoadPathAggregates(ctx, query)
	if err != nil {
		return observability.PathsResponse{}, err
	}
	currentByAccount, err := s.loadPathPressure(ctx, rows, query.End)
	pressureAvailable := err == nil
	if err != nil {
		currentByAccount = map[int64]observability.PressureRow{}
	}
	users, accounts, groups := s.loadPathLabels(ctx, rows)
	previousByPath := s.loadPreviousPathMetrics(ctx, query, rows)
	errorStages, errorStageAvailable := s.loadPathErrorStages(ctx, query, rows)
	keyIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		keyIDs = append(keyIDs, row.APIKeyID)
	}
	apiKeys := make(map[int64]string)
	if relations, keyErr := s.loader.LoadAPIKeyRelations(ctx, observability.Filter{APIKeyIDs: uniqueInt64(keyIDs)}, ""); keyErr == nil {
		for _, relation := range relations {
			apiKeys[relation.KeyID] = relation.KeyName
		}
	}
	out := make([]observability.PathRow, 0, len(rows))
	for _, row := range rows {
		account := accounts[row.AccountID]
		model := ""
		modelAttribution := "unknown"
		if account != nil {
			model = account.BoundModel()
			if model != "" {
				modelAttribution = "derived"
			}
		}
		user := users[row.UserID]
		group := groups[row.GroupID]
		pressure := currentByAccount[row.AccountID]
		avgDuration := 0.0
		if row.Duration.SampleCount > 0 {
			avgDuration = float64(row.Duration.SumMS) / float64(row.Duration.SampleCount)
		}
		avgFirstToken := 0.0
		if row.TTFT.SampleCount > 0 {
			avgFirstToken = float64(row.TTFT.SumMS) / float64(row.TTFT.SampleCount)
		}
		var dataAt *time.Time
		if !row.DataAt.IsZero() {
			value := row.DataAt.UTC()
			dataAt = &value
		}
		out = append(out, observability.PathRow{
			UserID: row.UserID, UserEmail: user.Email, Department: user.Department,
			APIKeyID: row.APIKeyID, APIKeyName: apiKeys[row.APIKeyID],
			GroupID: row.GroupID, GroupName: group,
			RouteAlias: row.RouteAlias, AccountID: row.AccountID, AccountName: accountName(account),
			BoundModel: model, Tokens: row.TotalTokens, RequestCount: row.RequestCount,
			SuccessRate:       ratio(row.SuccessCount, row.RequestCount),
			P95FirstTokenMS:   row.TTFT.Quantiles(0.95)[0.95],
			AverageDurationMS: avgDuration, AverageFirstTokenMS: avgFirstToken,
			DurationSampleCount: row.Duration.SampleCount, TTFTTotalSampleCount: row.TTFT.SampleCount,
			DataAt: dataAt, ErrorStage: errorStages[pathRefKey(PathRef{
				UserID: row.UserID, APIKeyID: row.APIKeyID, GroupID: row.GroupID,
				RouteAlias: row.RouteAlias, AccountID: row.AccountID,
			})],
			Current: pressure.Current, Limit: pressure.Limit,
			LegacyInferred: row.LegacyInferred > 0, Status: pathStatus(row),
			ModelAttribution: modelAttribution,
		})
		if previous, ok := previousByPath[pathRefKey(PathRef{
			UserID: row.UserID, APIKeyID: row.APIKeyID, GroupID: row.GroupID,
			RouteAlias: row.RouteAlias, AccountID: row.AccountID,
		})]; ok {
			previousDuration := previous.AverageDurationMS
			previousTTFT := previous.AverageFirstTokenMS
			out[len(out)-1].PreviousAverageDurationMS = &previousDuration
			out[len(out)-1].PreviousAverageFirstTokenMS = &previousTTFT
		}
	}
	warnings := make([]string, 0)
	if query.End.Sub(query.Start) >= observability.MaxRange {
		warnings = append(warnings, "coverage_top_n")
	}
	complete := true
	if !pressureAvailable {
		warnings = append(warnings, "concurrency_snapshot_unavailable")
		complete = false
	}
	if !errorStageAvailable {
		warnings = append(warnings, "error_stage_unavailable")
	}
	meta := observability.Meta{
		AsOf: time.Now().UTC(), StartAt: query.Start.UTC(), EndAt: query.End.UTC(), Window: query.Window, Bucket: query.Bucket,
		Timezone: timezone.Location().String(), Sources: []string{"observability_request_bucket", "observability_concurrency_bucket"},
		Consistency: "eventual", Complete: complete, Warnings: warnings,
		Coverage: coverageForRange(query),
	}
	return observability.PathsResponse{Meta: meta, Total: total, Rows: out}, nil
}

func (s *ObservabilityPathService) loadPathErrorStages(
	ctx context.Context,
	query ObservabilityPathQuery,
	rows []ObservabilityPathAggregate,
) (map[string]string, bool) {
	loader, ok := s.loader.(ObservabilityPathErrorStageLoader)
	if !ok {
		return map[string]string{}, true
	}
	refs := make([]PathRef, 0, len(rows))
	for _, row := range rows {
		if row.ErrorCount <= 0 {
			continue
		}
		refs = append(refs, PathRef{
			UserID: row.UserID, APIKeyID: row.APIKeyID, GroupID: row.GroupID,
			RouteAlias: row.RouteAlias, AccountID: row.AccountID,
		})
	}
	if len(refs) == 0 {
		return map[string]string{}, true
	}
	values, err := loader.LoadPathErrorStages(ctx, query.Start, query.End, refs)
	if err != nil {
		return map[string]string{}, false
	}
	return values, true
}

func (s *ObservabilityPathService) loadPreviousPathMetrics(
	ctx context.Context,
	query ObservabilityPathQuery,
	rows []ObservabilityPathAggregate,
) map[string]PathWindowMetrics {
	if len(rows) == 0 || !query.Start.Before(query.End) {
		return map[string]PathWindowMetrics{}
	}
	loader, ok := s.loader.(interface {
		LoadPathWindowMetrics(context.Context, time.Time, time.Time, observability.Bucket, []PathRef) (map[string]PathWindowMetrics, error)
	})
	if !ok {
		return map[string]PathWindowMetrics{}
	}
	duration := query.End.Sub(query.Start)
	refs := make([]PathRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, PathRef{
			UserID: row.UserID, APIKeyID: row.APIKeyID, GroupID: row.GroupID,
			RouteAlias: row.RouteAlias, AccountID: row.AccountID,
		})
	}
	values, err := loader.LoadPathWindowMetrics(
		ctx, query.Start.Add(-duration), query.Start, query.Bucket, refs,
	)
	if err != nil {
		return map[string]PathWindowMetrics{}
	}
	return values
}

func pathRefKey(ref PathRef) string {
	return fmt.Sprintf("%d|%d|%d|%s|%d", ref.UserID, ref.APIKeyID, ref.GroupID, ref.RouteAlias, ref.AccountID)
}

func (s *ObservabilityPathService) Relations(ctx context.Context, query ObservabilityPathQuery) (observability.RelationsResponse, error) {
	paths, err := s.Paths(ctx, query)
	if err != nil {
		return observability.RelationsResponse{}, err
	}
	keyRelations, err := s.loader.LoadAPIKeyRelations(ctx, query.Filter, query.Search)
	if err != nil {
		return observability.RelationsResponse{}, err
	}
	routeRelations, err := s.loader.LoadRouteRelations(ctx, query.Filter)
	if err != nil {
		return observability.RelationsResponse{}, err
	}
	traffic := pathTrafficKeys(paths.Rows)
	nodes := make(map[string]observability.RelationNode)
	edges := make(map[string]observability.RelationEdge)
	qualityErrors := make([]string, 0)
	addEdge := func(edge observability.RelationEdge) {
		if !query.IncludeIdle && edge.Idle {
			return
		}
		edges[edge.From+"|"+edge.Kind+"|"+edge.To] = edge
	}
	for _, relation := range keyRelations {
		userID := relationNodeID("user", relation.UserID)
		keyID := relationNodeID("api_key", relation.KeyID)
		groupID := relationNodeID("group", relation.GroupID)
		nodes[userID] = observability.RelationNode{ID: userID, Kind: "user", UserID: relation.UserID}
		nodes[keyID] = observability.RelationNode{ID: keyID, Kind: "api_key", APIKeyID: relation.KeyID, Label: relation.KeyName}
		nodes[groupID] = observability.RelationNode{ID: groupID, Kind: "group", GroupID: relation.GroupID}
		addEdge(observability.RelationEdge{From: userID, To: keyID, Kind: "owns_api_key", Idle: !traffic.userKey(relation.UserID, relation.KeyID)})
		addEdge(observability.RelationEdge{From: keyID, To: groupID, Kind: "uses_group", Idle: !traffic.keyGroup(relation.KeyID, relation.GroupID)})
	}
	for _, relation := range routeRelations {
		groupID := relationNodeID("group", relation.GroupID)
		routeID := routeNodeID(relation.GroupID, relation.RouteAlias)
		accountID := relationNodeID("account", relation.AccountID)
		nodes[groupID] = observability.RelationNode{ID: groupID, Kind: "group", GroupID: relation.GroupID}
		nodes[routeID] = observability.RelationNode{ID: routeID, Kind: "route_alias", RouteAlias: relation.RouteAlias, RoutePattern: relation.RouteAlias, GroupID: relation.GroupID, Label: relation.RouteAlias}
		nodes[accountID] = observability.RelationNode{ID: accountID, Kind: "account", AccountID: relation.AccountID}
		addEdge(observability.RelationEdge{From: groupID, To: routeID, Kind: "defines_route", Idle: !traffic.groupRoute(relation.GroupID, relation.RouteAlias)})
		addEdge(observability.RelationEdge{From: routeID, To: accountID, Kind: "routes_to_account", Idle: !traffic.routeAccount(relation.GroupID, relation.RouteAlias, relation.AccountID)})
	}
	accountIDs := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		if node.Kind == "account" {
			accountIDs = append(accountIDs, node.AccountID)
		}
	}
	if len(accountIDs) > 0 && s.accountRepo != nil {
		accounts, _ := s.accountRepo.GetByIDs(ctx, uniqueInt64(accountIDs))
		for _, account := range accounts {
			if account == nil {
				continue
			}
			if account.ModelBindingLegacyMulti() {
				qualityErrors = append(qualityErrors, "multiple_bound_models:account:"+formatInt(account.ID))
			}
			nodeID := relationNodeID("account", account.ID)
			node := nodes[nodeID]
			node.Label = account.Name
			node.BoundModel = account.BoundModel()
			node.ModelAttribution = "unknown"
			if node.BoundModel != "" {
				node.ModelAttribution = "derived"
			}
			nodes[nodeID] = node
			modelID := "model:" + node.BoundModel
			if node.BoundModel != "" {
				nodes[modelID] = observability.RelationNode{ID: modelID, Kind: "model", BoundModel: node.BoundModel, Label: node.BoundModel}
				addEdge(observability.RelationEdge{From: nodeID, To: modelID, Kind: "bound_to_model", Idle: false})
			}
		}
	}
	for nodeID, node := range nodes {
		switch node.Kind {
		case "user":
			if s.userRepo != nil {
				if user, lookupErr := s.userRepo.GetByID(ctx, node.UserID); lookupErr == nil && user != nil {
					node.Label = user.Email
					node.UserEmail = user.Email
					node.Department = user.Department
				}
			}
		case "group":
			if s.groupRepo != nil {
				if group, lookupErr := s.groupRepo.GetByID(ctx, node.GroupID); lookupErr == nil && group != nil {
					node.Label = group.Name
				}
			}
		}
		nodes[nodeID] = node
	}
	nodeList := make([]observability.RelationNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	sort.Slice(nodeList, func(i, j int) bool { return nodeList[i].ID < nodeList[j].ID })
	edgeList := make([]observability.RelationEdge, 0, len(edges))
	for _, edge := range edges {
		edgeList = append(edgeList, edge)
	}
	sort.Slice(edgeList, func(i, j int) bool {
		if edgeList[i].From != edgeList[j].From {
			return edgeList[i].From < edgeList[j].From
		}
		if edgeList[i].Kind != edgeList[j].Kind {
			return edgeList[i].Kind < edgeList[j].Kind
		}
		return edgeList[i].To < edgeList[j].To
	})
	if !query.IncludeIdle {
		activeNodes := make(map[string]struct{}, len(edgeList)*2)
		for _, edge := range edgeList {
			activeNodes[edge.From] = struct{}{}
			activeNodes[edge.To] = struct{}{}
		}
		filtered := nodeList[:0]
		for _, node := range nodeList {
			if _, ok := activeNodes[node.ID]; ok {
				filtered = append(filtered, node)
			}
		}
		nodeList = filtered
	}
	sort.Strings(qualityErrors)
	return observability.RelationsResponse{Meta: paths.Meta, Nodes: nodeList, Edges: edgeList, DataQualityErrors: uniqueStrings(qualityErrors)}, nil
}

func (s *ObservabilityPathService) loadPathPressure(ctx context.Context, rows []ObservabilityPathAggregate, end time.Time) (map[int64]observability.PressureRow, error) {
	if s.concurrency == nil {
		return nil, fmt.Errorf("concurrency history unavailable")
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.AccountID > 0 {
			ids = append(ids, row.AccountID)
		}
	}
	ids = uniqueInt64(ids)
	accounts := make([]AccountWithConcurrency, 0, len(ids))
	for _, id := range ids {
		accounts = append(accounts, AccountWithConcurrency{ID: id})
	}
	values, _, err := s.concurrency.CurrentBatch(ctx, accounts, nil, end)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]observability.PressureRow, len(values))
	for id, value := range values {
		out[id] = observability.PressureRow{AccountID: id, Current: value.Current, Limit: value.Limit}
	}
	return out, nil
}

func (s *ObservabilityPathService) loadPathLabels(ctx context.Context, rows []ObservabilityPathAggregate) (map[int64]User, map[int64]*Account, map[int64]string) {
	users := make(map[int64]User)
	accounts := make(map[int64]*Account)
	groups := make(map[int64]string)
	for _, row := range rows {
		if _, exists := users[row.UserID]; !exists && s.userRepo != nil {
			if user, err := s.userRepo.GetByID(ctx, row.UserID); err == nil && user != nil {
				users[row.UserID] = *user
			}
		}
		if _, exists := groups[row.GroupID]; !exists && s.groupRepo != nil {
			if group, err := s.groupRepo.GetByID(ctx, row.GroupID); err == nil && group != nil {
				groups[row.GroupID] = group.Name
			}
		}
	}
	ids := make([]int64, 0)
	for _, row := range rows {
		ids = append(ids, row.AccountID)
	}
	if s.accountRepo != nil {
		values, _ := s.accountRepo.GetByIDs(ctx, uniqueInt64(ids))
		for _, account := range values {
			if account != nil {
				accounts[account.ID] = account
			}
		}
	}
	return users, accounts, groups
}

type trafficKeys struct {
	userKeys      map[string]struct{}
	keyGroups     map[string]struct{}
	groupAliases  map[string]struct{}
	routeAccounts map[string]struct{}
}

func pathTrafficKeys(rows []observability.PathRow) trafficKeys {
	out := trafficKeys{
		userKeys: make(map[string]struct{}), keyGroups: make(map[string]struct{}),
		groupAliases: make(map[string]struct{}), routeAccounts: make(map[string]struct{}),
	}
	for _, row := range rows {
		if row.RequestCount <= 0 {
			continue
		}
		out.userKeys[formatInt(row.UserID)+"|"+formatInt(row.APIKeyID)] = struct{}{}
		out.keyGroups[formatInt(row.APIKeyID)+"|"+formatInt(row.GroupID)] = struct{}{}
		out.groupAliases[formatInt(row.GroupID)+"|"+row.RouteAlias] = struct{}{}
		out.routeAccounts[formatInt(row.GroupID)+"|"+row.RouteAlias+"|"+formatInt(row.AccountID)] = struct{}{}
	}
	return out
}

func (t trafficKeys) userKey(userID, keyID int64) bool {
	_, ok := t.userKeys[formatInt(userID)+"|"+formatInt(keyID)]
	return ok
}
func (t trafficKeys) keyGroup(keyID, groupID int64) bool {
	_, ok := t.keyGroups[formatInt(keyID)+"|"+formatInt(groupID)]
	return ok
}
func (t trafficKeys) groupRoute(groupID int64, alias string) bool {
	if _, ok := t.groupAliases[formatInt(groupID)+"|"+alias]; ok {
		return true
	}
	for key := range t.groupAliases {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) == 2 && parts[0] == formatInt(groupID) && routePatternMatches(alias, parts[1]) {
			return true
		}
	}
	return false
}
func (t trafficKeys) routeAccount(groupID int64, alias string, accountID int64) bool {
	if t.groupRoute(groupID, alias) {
		for key := range t.routeAccounts {
			parts := strings.Split(key, "|")
			if len(parts) == 3 && parts[0] == formatInt(groupID) && parts[2] == formatInt(accountID) && routePatternMatches(alias, parts[1]) {
				return true
			}
		}
	}
	return false
}

func routePatternMatches(pattern, alias string) bool {
	if pattern == alias {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(alias, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

func pathStatus(row ObservabilityPathAggregate) string {
	rate := ratio(row.ErrorCount, row.RequestCount)
	switch {
	case rate >= 0.5:
		return "error"
	case rate >= 0.2:
		return "degraded"
	default:
		return "healthy"
	}
}

func coverageForRange(query ObservabilityPathQuery) string {
	if query.End.Sub(query.Start) >= observability.MaxRange {
		return "top_n"
	}
	return "full"
}

func accountName(account *Account) string {
	if account == nil {
		return ""
	}
	return account.Name
}

func relationNodeID(kind string, id int64) string {
	return kind + ":" + formatInt(id)
}
func routeNodeID(groupID int64, alias string) string {
	return "route_alias:" + formatInt(groupID) + ":" + alias
}
func formatInt(value int64) string {
	return fmt.Sprintf("%d", value)
}
func uniqueInt64(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
