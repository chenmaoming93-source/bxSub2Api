package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	entaccount "github.com/Wei-Shaw/sub2api/ent/account"
	entgroup "github.com/Wei-Shaw/sub2api/ent/group"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

const sceneUsageDateLayout = "2006-01-02"

var (
	ErrSceneUsageStatisticsNotConfigured = infraerrors.Conflict(
		"SCENE_USAGE_STATISTICS_NOT_CONFIGURED",
		"缺少必要的动态 Token 统计项：group_id + account_id + upstream_model，指标为 total_tokens，周期为日统计",
	)
	ErrSceneUsageStatisticsNotActive = infraerrors.Conflict(
		"SCENE_USAGE_STATISTICS_NOT_ACTIVE",
		"场景 Token 统计项尚未启用，请先发布并启用该统计项",
	)
	ErrSceneUsageStatisticsDisabled = infraerrors.ServiceUnavailable(
		"TOKEN_STATISTICS_DISABLED",
		"动态 Token 统计功能当前未开启",
	)
)

var sceneUsageDimensions = []tokenstat.DimensionCode{
	tokenstat.DimensionGroupID,
	tokenstat.DimensionAccountID,
	tokenstat.DimensionUpstreamModel,
}

type SceneAccountDailyUsageInput struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
	GroupName string `json:"group_name,omitempty" form:"group_name"`
}

type SceneAccountDailyUsageResult struct {
	Timezone            string                      `json:"timezone"`
	StartDate           string                      `json:"start_date"`
	EndDate             string                      `json:"end_date"`
	Complete            bool                        `json:"complete"`
	Consistency         string                      `json:"consistency"`
	ProjectionID        int64                       `json:"projection_id"`
	ProjectionEnabledAt *time.Time                  `json:"projection_enabled_at,omitempty"`
	LastSyncedAt        *time.Time                  `json:"last_synced_at,omitempty"`
	Days                []SceneAccountDailyUsageDay `json:"days"`
}

type SceneAccountDailyUsageDay struct {
	Date   string                        `json:"date"`
	Scenes []SceneAccountDailyUsageScene `json:"scenes"`
}

type SceneAccountDailyUsageScene struct {
	GroupID     int64                           `json:"group_id"`
	GroupName   string                          `json:"group_name"`
	SceneName   string                          `json:"scene_name"`
	TotalTokens int64                           `json:"total_tokens"`
	Accounts    []SceneAccountDailyUsageAccount `json:"accounts"`
}

type SceneAccountDailyUsageAccount struct {
	AccountID     int64  `json:"account_id"`
	AccountName   string `json:"account_name"`
	UpstreamModel string `json:"upstream_model"`
	TotalTokens   int64  `json:"total_tokens"`
}

type SceneAccountDailyUsageService struct {
	client      *ent.Client
	projections *tokenstat.ProjectionAdminService
	runtime     *tokenstat.RuntimeController
	location    *time.Location
}

func NewSceneAccountDailyUsageService(client *ent.Client, projections *tokenstat.ProjectionAdminService, runtime *tokenstat.RuntimeController, location *time.Location) *SceneAccountDailyUsageService {
	if location == nil {
		location = time.UTC
	}
	return &SceneAccountDailyUsageService{client: client, projections: projections, runtime: runtime, location: location}
}

func (s *SceneAccountDailyUsageService) QuerySceneAccountDailyUsage(ctx context.Context, input SceneAccountDailyUsageInput) (*SceneAccountDailyUsageResult, error) {
	start, endExclusive, err := parseSceneUsageDateRange(input.StartDate, input.EndDate, s.location)
	if err != nil {
		return nil, infraerrors.BadRequest("SCENE_USAGE_INVALID_DATE_RANGE", err.Error())
	}
	if !s.dynamicStatisticsEnabled() {
		return nil, ErrSceneUsageStatisticsDisabled
	}
	if s.projections == nil || s.client == nil {
		return nil, infraerrors.InternalServer("SCENE_USAGE_SERVICE_UNAVAILABLE", "场景 Token 统计服务未初始化")
	}

	projection, err := s.projections.ResolveUsageProjection(ctx, sceneUsageDimensions, tokenstat.MetricTotalTokens, tokenstat.PeriodDay)
	if err != nil {
		switch {
		case errors.Is(err, tokenstat.ErrUsageProjectionNotConfigured):
			return nil, ErrSceneUsageStatisticsNotConfigured
		case errors.Is(err, tokenstat.ErrUsageProjectionNotActive):
			return nil, ErrSceneUsageStatisticsNotActive
		default:
			return nil, fmt.Errorf("resolve scene usage projection: %w", err)
		}
	}

	result := &SceneAccountDailyUsageResult{
		Timezone: s.location.String(), StartDate: start.Format(sceneUsageDateLayout), EndDate: endExclusive.AddDate(0, 0, -1).Format(sceneUsageDateLayout),
		Complete: true, Consistency: "mysql_eventual", ProjectionID: projection.ID,
		ProjectionEnabledAt: projection.EnabledAt, Days: []SceneAccountDailyUsageDay{},
	}

	filters := map[tokenstat.DimensionCode]tokenstat.DimensionValue{}
	if name := strings.TrimSpace(input.GroupName); name != "" {
		group, findErr := s.client.Group.Query().Where(entgroup.NameEQ(name)).Only(ctx)
		if findErr != nil {
			if ent.IsNotFound(findErr) {
				return result, nil
			}
			return nil, fmt.Errorf("resolve group_name: %w", findErr)
		}
		filters[tokenstat.DimensionGroupID] = tokenstat.Int64Value(group.ID)
	}

	rows, complete, lastSynced, err := s.queryAllUsageRows(ctx, projection.ID, start, endExclusive, filters)
	if err != nil {
		return nil, fmt.Errorf("query scene usage projection: %w", err)
	}
	result.Complete = complete
	result.LastSyncedAt = lastSynced
	return s.enrichAndAggregate(ctx, result, rows)
}

func (s *SceneAccountDailyUsageService) dynamicStatisticsEnabled() bool {
	if s.runtime != nil {
		return s.runtime.Enabled()
	}
	return tokenstat.RuntimeEnabled()
}

func (s *SceneAccountDailyUsageService) queryAllUsageRows(ctx context.Context, projectionID int64, start, end time.Time, filters map[tokenstat.DimensionCode]tokenstat.DimensionValue) ([]tokenstat.UsageQueryRow, bool, *time.Time, error) {
	const pageSize = 1000
	var rows []tokenstat.UsageQueryRow
	complete := true
	var lastSynced *time.Time
	for page := 1; ; page++ {
		pageResult, err := s.projections.QueryUsage(ctx, tokenstat.UsageQueryInput{
			ProjectionID: projectionID, MetricCode: tokenstat.MetricTotalTokens, PeriodType: tokenstat.PeriodDay,
			Start: start, End: end, Filters: filters, GroupBy: sceneUsageDimensions,
			Sort: "time_asc", Page: page, PageSize: pageSize,
		})
		if err != nil {
			return nil, false, nil, err
		}
		rows = append(rows, pageResult.Rows...)
		complete = complete && pageResult.Complete
		if pageResult.LastSyncedAt != nil && (lastSynced == nil || pageResult.LastSyncedAt.After(*lastSynced)) {
			value := *pageResult.LastSyncedAt
			lastSynced = &value
		}
		if len(pageResult.Rows) == 0 || page*pageSize >= pageResult.Total {
			return rows, complete, lastSynced, nil
		}
	}
}

type sceneUsageAccountKey struct {
	accountID int64
	model     string
}

type sceneUsageSceneAccumulator struct {
	groupID     int64
	groupName   string
	sceneName   string
	totalTokens int64
	accounts    map[sceneUsageAccountKey]int64
}

func (s *SceneAccountDailyUsageService) enrichAndAggregate(ctx context.Context, result *SceneAccountDailyUsageResult, rows []tokenstat.UsageQueryRow) (*SceneAccountDailyUsageResult, error) {
	if len(rows) == 0 {
		result.Days = []SceneAccountDailyUsageDay{}
		return result, nil
	}
	groupIDs := make(map[int64]struct{})
	accountIDs := make(map[int64]struct{})
	for _, row := range rows {
		groupValue, groupOK := row.Dimensions[tokenstat.DimensionGroupID]
		accountValue, accountOK := row.Dimensions[tokenstat.DimensionAccountID]
		modelValue, modelOK := row.Dimensions[tokenstat.DimensionUpstreamModel]
		if !groupOK || !accountOK || !modelOK || groupValue.Type != tokenstat.ValueTypeInt64 || accountValue.Type != tokenstat.ValueTypeInt64 || modelValue.Type != tokenstat.ValueTypeString {
			return nil, fmt.Errorf("aggregate row is missing required scene usage dimensions")
		}
		groupIDs[groupValue.Int64] = struct{}{}
		accountIDs[accountValue.Int64] = struct{}{}
	}

	groupIDsList := sortedIDs(groupIDs)
	groups, err := s.client.Group.Query().Where(entgroup.IDIn(groupIDsList...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load scene groups: %w", err)
	}
	accountIDsList := sortedIDs(accountIDs)
	accounts, err := s.client.Account.Query().Where(entaccount.IDIn(accountIDsList...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load usage accounts: %w", err)
	}
	groupByID := make(map[int64]*ent.Group, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	accountByID := make(map[int64]*ent.Account, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}

	sceneByDay := make(map[string]map[int64]*sceneUsageSceneAccumulator)
	for _, row := range rows {
		groupID := row.Dimensions[tokenstat.DimensionGroupID].Int64
		accountID := row.Dimensions[tokenstat.DimensionAccountID].Int64
		model := row.Dimensions[tokenstat.DimensionUpstreamModel].String
		group := groupByID[groupID]
		account := accountByID[accountID]
		if group == nil || account == nil {
			return nil, fmt.Errorf("usage references missing group or account: group_id=%d account_id=%d", groupID, accountID)
		}
		day := row.PeriodStart.In(s.location).Format(sceneUsageDateLayout)
		byGroup := sceneByDay[day]
		if byGroup == nil {
			byGroup = make(map[int64]*sceneUsageSceneAccumulator)
			sceneByDay[day] = byGroup
		}
		scene := byGroup[groupID]
		if scene == nil {
			sceneName := ""
			if group.SceneName != nil {
				sceneName = *group.SceneName
			}
			scene = &sceneUsageSceneAccumulator{groupID: groupID, groupName: group.Name, sceneName: sceneName, accounts: make(map[sceneUsageAccountKey]int64)}
			byGroup[groupID] = scene
		}
		scene.totalTokens += row.Value
		scene.accounts[sceneUsageAccountKey{accountID: accountID, model: model}] += row.Value
	}

	dayKeys := make([]string, 0, len(sceneByDay))
	for day := range sceneByDay {
		dayKeys = append(dayKeys, day)
	}
	sort.Strings(dayKeys)
	result.Days = make([]SceneAccountDailyUsageDay, 0, len(dayKeys))
	for _, day := range dayKeys {
		groupMap := sceneByDay[day]
		groupIDs := make([]int64, 0, len(groupMap))
		for groupID := range groupMap {
			groupIDs = append(groupIDs, groupID)
		}
		sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
		dayResult := SceneAccountDailyUsageDay{Date: day, Scenes: make([]SceneAccountDailyUsageScene, 0, len(groupIDs))}
		for _, groupID := range groupIDs {
			scene := groupMap[groupID]
			keys := make([]sceneUsageAccountKey, 0, len(scene.accounts))
			for key := range scene.accounts {
				keys = append(keys, key)
			}
			sort.Slice(keys, func(i, j int) bool {
				if keys[i].accountID != keys[j].accountID {
					return keys[i].accountID < keys[j].accountID
				}
				return keys[i].model < keys[j].model
			})
			accountsResult := make([]SceneAccountDailyUsageAccount, 0, len(keys))
			for _, key := range keys {
				accountsResult = append(accountsResult, SceneAccountDailyUsageAccount{AccountID: key.accountID, AccountName: accountByID[key.accountID].Name, UpstreamModel: key.model, TotalTokens: scene.accounts[key]})
			}
			dayResult.Scenes = append(dayResult.Scenes, SceneAccountDailyUsageScene{GroupID: scene.groupID, GroupName: scene.groupName, SceneName: scene.sceneName, TotalTokens: scene.totalTokens, Accounts: accountsResult})
		}
		result.Days = append(result.Days, dayResult)
	}
	return result, nil
}

func parseSceneUsageDateRange(startRaw, endRaw string, location *time.Location) (time.Time, time.Time, error) {
	startRaw = strings.TrimSpace(startRaw)
	endRaw = strings.TrimSpace(endRaw)
	if startRaw == "" || endRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("start_date and end_date are required in YYYY-MM-DD format")
	}
	start, err := time.ParseInLocation(sceneUsageDateLayout, startRaw, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("start_date must use YYYY-MM-DD format")
	}
	end, err := time.ParseInLocation(sceneUsageDateLayout, endRaw, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end_date must use YYYY-MM-DD format")
	}
	startUTCDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTCDate := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if endUTCDate.Before(startUTCDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("end_date must be on or after start_date")
	}
	days := int(endUTCDate.Sub(startUTCDate)/(24*time.Hour)) + 1
	if days > 366 {
		return time.Time{}, time.Time{}, fmt.Errorf("query range exceeds 366 days")
	}
	return start, end.AddDate(0, 0, 1), nil
}

func sortedIDs(values map[int64]struct{}) []int64 {
	ids := make([]int64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
