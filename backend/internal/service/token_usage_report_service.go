package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var ErrInvalidTokenUsageReportQuery = errors.New("invalid token usage report query")

type TokenUsageReportQuery struct {
	Model              string
	StartDate, EndDate time.Time
	Page, PageSize     int
	SortBy, SortOrder  string
}

type ModelTokenUsageRow struct {
	UsageDate        time.Time
	Model            string
	UsedTokens       int64
	DailyLimitTokens *int64
}

type ModelTokenUsageRepository interface {
	ListModelTokenUsage(context.Context, TokenUsageReportQuery) ([]ModelTokenUsageRow, int64, int64, error)
	ListRouteTokenUsage(context.Context, RouteTokenUsageReportQuery) ([]RouteTokenUsageRow, int64, int64, error)
	ListUserTokenUsage(context.Context, UserTokenUsageReportQuery) ([]UserTokenUsageRow, int64, int64, error)
	SearchTokenUsageOptions(context.Context, string, int64, string, int) ([]TokenUsageOption, error)
	FindDefaultTokenUsageTarget(context.Context, string, time.Time) (*TokenUsageOption, error)
}

type TodayTokenUsageRepository interface {
	ListTodayModelTokenUsage(context.Context, TokenUsageReportQuery, time.Time) ([]ModelTokenUsageRow, error)
	ListTodayRouteTokenUsage(context.Context, RouteTokenUsageReportQuery, time.Time) ([]RouteTokenUsageRow, error)
	ListTodayUserTokenUsage(context.Context, UserTokenUsageReportQuery, time.Time) ([]UserTokenUsageRow, error)
}

type TokenUsageOption struct {
	ID         int64  `json:"id,omitempty"`
	Label      string `json:"label"`
	Model      string `json:"model,omitempty"`
	GroupID    int64  `json:"group_id,omitempty"`
	RouteAlias string `json:"route_alias,omitempty"`
	UserID     int64  `json:"user_id,omitempty"`
}

type ModelTokenUsageItem struct {
	UsageDate        time.Time
	Model            string
	UsedTokens       int64
	DailyLimitTokens *int64
	UsageRate        *float64
	Status           string
}

type ModelTokenUsageReport struct {
	Items          []ModelTokenUsageItem
	UsedTokens     int64
	Page, PageSize int
	Total          int64
}

type RouteTokenUsageReportQuery struct {
	TokenUsageReportQuery
	GroupID                   int64
	RouteAlias, UpstreamModel string
}
type RouteTokenUsageRow struct {
	UsageDate                            time.Time
	GroupID                              int64
	GroupName, RouteAlias, UpstreamModel string
	UsedTokens                           int64
	DailyLimitTokens                     *int64
	Priority                             *int
}
type RouteTokenUsageItem struct {
	RouteTokenUsageRow
	UsageRate *float64
	Status    string
}
type RouteTokenUsageReport struct {
	Items          []RouteTokenUsageItem
	UsedTokens     int64
	Page, PageSize int
	Total          int64
}

type UserTokenUsageReportQuery struct {
	TokenUsageReportQuery
	UserID         int64
	IncludeDeleted bool
}
type UserTokenUsageRow struct {
	UsageDate              time.Time
	UserID                 int64
	Email, Username, Model string
	UserDeleted            bool
	UsedTokens             int64
	DailyLimitTokens       *int64
}
type UserTokenUsageItem struct {
	UserTokenUsageRow
	UsageRate *float64
	Status    string
}
type UserTokenUsageReport struct {
	Items          []UserTokenUsageItem
	UsedTokens     int64
	Page, PageSize int
	Total          int64
}

type TokenUsageReportService struct {
	repo      ModelTokenUsageRepository
	todayRepo TodayTokenUsageRepository
	reader    CurrentTokenUsageReader
	repairer  CurrentTokenUsageRepairer
	now       func() time.Time
}

func NewTokenUsageReportService(repo ModelTokenUsageRepository) *TokenUsageReportService {
	today, _ := repo.(TodayTokenUsageRepository)
	return &TokenUsageReportService{repo: repo, todayRepo: today, now: time.Now}
}

func (s *TokenUsageReportService) ConfigureCurrentTokenUsage(reader CurrentTokenUsageReader, repairer CurrentTokenUsageRepairer) *TokenUsageReportService {
	s.reader = reader
	s.repairer = repairer
	return s
}
func (s *TokenUsageReportService) SetNowForTest(now func() time.Time) { s.now = now }

func (s *TokenUsageReportService) GetModelReport(ctx context.Context, query TokenUsageReportQuery) (ModelTokenUsageReport, error) {
	query.Model = strings.TrimSpace(query.Model)
	if query.StartDate.IsZero() || query.EndDate.IsZero() || query.EndDate.Before(query.StartDate) || query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		return ModelTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	if !validTokenUsageSort(query.SortBy, query.SortOrder, "model") {
		return ModelTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	today := businessTokenUsageDay(s.now())
	if !query.EndDate.Before(today) && s.todayRepo != nil && s.reader != nil {
		return s.getHybridModelReport(ctx, query, today)
	}
	rows, total, used, err := s.repo.ListModelTokenUsage(ctx, query)
	if err != nil {
		return ModelTokenUsageReport{}, fmt.Errorf("get model token usage report: %w", err)
	}
	items := make([]ModelTokenUsageItem, 0, len(rows))
	for _, row := range rows {
		item := ModelTokenUsageItem{UsageDate: row.UsageDate, Model: row.Model, UsedTokens: row.UsedTokens, DailyLimitTokens: row.DailyLimitTokens, Status: "unlimited"}
		if row.DailyLimitTokens != nil && *row.DailyLimitTokens > 0 {
			rate := float64(row.UsedTokens) / float64(*row.DailyLimitTokens)
			item.UsageRate = &rate
			item.Status = "normal"
			if rate >= 1 {
				item.Status = "exceeded"
			} else if rate >= .8 {
				item.Status = "warning"
			}
		}
		items = append(items, item)
	}
	return ModelTokenUsageReport{Items: items, UsedTokens: used, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func businessTokenUsageDay(now time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

func (s *TokenUsageReportService) getHybridModelReport(ctx context.Context, q TokenUsageReportQuery, today time.Time) (ModelTokenUsageReport, error) {
	all := []ModelTokenUsageRow{}
	if q.StartDate.Before(today) {
		hq := q
		hq.EndDate = today.AddDate(0, 0, -1)
		hq.Page = 1
		hq.PageSize = 100
		for {
			rows, total, _, err := s.repo.ListModelTokenUsage(ctx, hq)
			if err != nil {
				return ModelTokenUsageReport{}, err
			}
			all = append(all, rows...)
			if int64(len(all)) >= total || len(rows) == 0 {
				break
			}
			hq.Page++
		}
	}
	mysqlRows, err := s.todayRepo.ListTodayModelTokenUsage(ctx, q, today)
	if err != nil {
		return ModelTokenUsageReport{}, err
	}
	filters := []string(nil)
	if q.Model != "" {
		filters = []string{q.Model}
	}
	redisResult, redisErr := s.reader.ReadModelUsage(ctx, today, filters)
	todayRows := mysqlRows
	if redisErr == nil {
		var repair []ModelTokenUsageRow
		todayRows, repair = MergeModelTokenUsage(mysqlRows, redisResult.Rows)
		positive := repair[:0]
		for _, row := range repair {
			if row.UsedTokens > 0 {
				positive = append(positive, row)
			}
		}
		if len(positive) > 0 && s.repairer != nil {
			_ = s.repairer.RepairModelUsage(ctx, today, positive)
		}
	}
	all = append(all, todayRows...)
	items := make([]ModelTokenUsageItem, 0, len(all))
	var used int64
	for _, row := range all {
		rate, status := usageState(row.UsedTokens, row.DailyLimitTokens)
		items = append(items, ModelTokenUsageItem{UsageDate: row.UsageDate, Model: row.Model, UsedTokens: row.UsedTokens, DailyLimitTokens: row.DailyLimitTokens, UsageRate: rate, Status: status})
		used += row.UsedTokens
	}
	sort.SliceStable(items, func(i, j int) bool { return modelUsageItemLess(items[i], items[j], q.SortBy, q.SortOrder) })
	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := min(start+q.PageSize, total)
	return ModelTokenUsageReport{Items: items[start:end], UsedTokens: used, Page: q.Page, PageSize: q.PageSize, Total: int64(total)}, nil
}

func modelUsageItemLess(a, b ModelTokenUsageItem, sortBy, sortOrder string) bool {
	fields, orders := strings.Split(sortBy, ","), strings.Split(sortOrder, ",")
	for i, field := range fields {
		field = strings.TrimSpace(field)
		cmp := 0
		switch field {
		case "model":
			cmp = strings.Compare(a.Model, b.Model)
		case "used_tokens":
			cmp = cmpInt64(a.UsedTokens, b.UsedTokens)
		case "usage_date":
			if a.UsageDate.Before(b.UsageDate) {
				cmp = -1
			} else if a.UsageDate.After(b.UsageDate) {
				cmp = 1
			}
		case "status":
			cmp = strings.Compare(a.Status, b.Status)
		case "usage_rate":
			cmp = cmpOptionalFloat(a.UsageRate, b.UsageRate)
		case "daily_limit_tokens":
			cmp = cmpOptionalInt64(a.DailyLimitTokens, b.DailyLimitTokens)
		}
		if cmp != 0 {
			if strings.TrimSpace(orders[i]) == "desc" {
				return cmp > 0
			}
			return cmp < 0
		}
	}
	return false
}

func usageState(used int64, limit *int64) (*float64, string) {
	if limit == nil || *limit <= 0 {
		return nil, "unlimited"
	}
	rate := float64(used) / float64(*limit)
	status := "normal"
	if rate >= 1 {
		status = "exceeded"
	} else if rate >= .8 {
		status = "warning"
	}
	return &rate, status
}

func validateBaseQuery(q TokenUsageReportQuery) error {
	if q.StartDate.IsZero() || q.EndDate.IsZero() || q.EndDate.Before(q.StartDate) || q.Page < 1 || q.PageSize < 1 || q.PageSize > 100 {
		return ErrInvalidTokenUsageReportQuery
	}
	return nil
}

func validTokenUsageSort(sortBy, sortOrder string, extra ...string) bool {
	common := map[string]bool{"usage_date": true, "used_tokens": true, "daily_limit_tokens": true, "usage_rate": true, "status": true}
	for _, field := range extra {
		common[field] = true
	}
	return repositorySortListValid(sortBy, sortOrder, common)
}

func repositorySortListValid(sortBy, sortOrder string, allowed map[string]bool) bool {
	fields, orders := strings.Split(sortBy, ","), strings.Split(sortOrder, ",")
	if len(fields) == 0 || len(fields) != len(orders) {
		return false
	}
	seen := map[string]bool{}
	for i := range fields {
		field, order := strings.TrimSpace(fields[i]), strings.TrimSpace(orders[i])
		if field == "" || !allowed[field] || seen[field] || order != "asc" && order != "desc" {
			return false
		}
		seen[field] = true
	}
	return true
}

func (s *TokenUsageReportService) GetRouteReport(ctx context.Context, q RouteTokenUsageReportQuery) (RouteTokenUsageReport, error) {
	q.RouteAlias, q.UpstreamModel = strings.TrimSpace(q.RouteAlias), strings.TrimSpace(q.UpstreamModel)
	if q.GroupID < 0 || validateBaseQuery(q.TokenUsageReportQuery) != nil || !validTokenUsageSort(q.SortBy, q.SortOrder, "group", "route_alias", "upstream_model", "priority") {
		return RouteTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	today := businessTokenUsageDay(s.now())
	if !q.EndDate.Before(today) && s.todayRepo != nil && s.reader != nil {
		return s.getHybridRouteReport(ctx, q, today)
	}
	rows, total, used, err := s.repo.ListRouteTokenUsage(ctx, q)
	if err != nil {
		return RouteTokenUsageReport{}, fmt.Errorf("get route token usage report: %w", err)
	}
	items := make([]RouteTokenUsageItem, 0, len(rows))
	for _, row := range rows {
		rate, status := usageState(row.UsedTokens, row.DailyLimitTokens)
		items = append(items, RouteTokenUsageItem{RouteTokenUsageRow: row, UsageRate: rate, Status: status})
	}
	return RouteTokenUsageReport{Items: items, UsedTokens: used, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

func (s *TokenUsageReportService) getHybridRouteReport(ctx context.Context, q RouteTokenUsageReportQuery, today time.Time) (RouteTokenUsageReport, error) {
	all := []RouteTokenUsageRow{}
	if q.StartDate.Before(today) {
		hq := q
		hq.EndDate = today.AddDate(0, 0, -1)
		hq.Page = 1
		hq.PageSize = 100
		for {
			rows, total, _, err := s.repo.ListRouteTokenUsage(ctx, hq)
			if err != nil {
				return RouteTokenUsageReport{}, err
			}
			all = append(all, rows...)
			if int64(len(all)) >= total || len(rows) == 0 {
				break
			}
			hq.Page++
		}
	}
	mysqlRows, err := s.todayRepo.ListTodayRouteTokenUsage(ctx, q, today)
	if err != nil {
		return RouteTokenUsageReport{}, err
	}
	filters := []RouteTokenUsageRow(nil)
	if q.GroupID > 0 && q.RouteAlias != "" && q.UpstreamModel != "" {
		filters = []RouteTokenUsageRow{{GroupID: q.GroupID, RouteAlias: q.RouteAlias, UpstreamModel: q.UpstreamModel}}
	}
	redisResult, redisErr := s.reader.ReadRouteUsage(ctx, today, filters)
	todayRows := mysqlRows
	if redisErr == nil {
		var repair []RouteTokenUsageRow
		todayRows, repair = MergeRouteTokenUsage(mysqlRows, redisResult.Rows)
		valid := todayRows[:0]
		for _, row := range todayRows {
			if row.GroupName != "" {
				valid = append(valid, row)
			}
		}
		todayRows = valid
		positive := repair[:0]
		for _, row := range repair {
			if row.UsedTokens > 0 {
				positive = append(positive, row)
			}
		}
		if len(positive) > 0 && s.repairer != nil {
			_ = s.repairer.RepairRouteUsage(ctx, today, positive)
		}
	}
	all = append(all, todayRows...)
	items := make([]RouteTokenUsageItem, 0, len(all))
	var used int64
	for _, row := range all {
		rate, status := usageState(row.UsedTokens, row.DailyLimitTokens)
		items = append(items, RouteTokenUsageItem{RouteTokenUsageRow: row, UsageRate: rate, Status: status})
		used += row.UsedTokens
	}
	sort.SliceStable(items, func(i, j int) bool { return routeUsageItemLess(items[i], items[j], q.SortBy, q.SortOrder) })
	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := min(start+q.PageSize, total)
	return RouteTokenUsageReport{Items: items[start:end], UsedTokens: used, Page: q.Page, PageSize: q.PageSize, Total: int64(total)}, nil
}

func routeUsageItemLess(a, b RouteTokenUsageItem, sortBy, sortOrder string) bool {
	fields, orders := strings.Split(sortBy, ","), strings.Split(sortOrder, ",")
	for i, field := range fields {
		field = strings.TrimSpace(field)
		cmp := 0
		switch field {
		case "group":
			cmp = strings.Compare(a.GroupName, b.GroupName)
		case "route_alias":
			cmp = strings.Compare(a.RouteAlias, b.RouteAlias)
		case "upstream_model":
			cmp = strings.Compare(a.UpstreamModel, b.UpstreamModel)
		case "used_tokens":
			cmp = cmpInt64(a.UsedTokens, b.UsedTokens)
		case "priority":
			cmp = cmpOptionalInt(a.Priority, b.Priority)
		case "usage_date":
			if a.UsageDate.Before(b.UsageDate) {
				cmp = -1
			} else if a.UsageDate.After(b.UsageDate) {
				cmp = 1
			}
		case "status":
			cmp = strings.Compare(a.Status, b.Status)
		case "usage_rate":
			cmp = cmpOptionalFloat(a.UsageRate, b.UsageRate)
		case "daily_limit_tokens":
			cmp = cmpOptionalInt64(a.DailyLimitTokens, b.DailyLimitTokens)
		}
		if cmp != 0 {
			if strings.TrimSpace(orders[i]) == "desc" {
				return cmp > 0
			}
			return cmp < 0
		}
	}
	return false
}
func cmpInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
func cmpOptionalInt(a, b *int) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if *a < *b {
		return -1
	}
	if *a > *b {
		return 1
	}
	return 0
}
func cmpOptionalInt64(a, b *int64) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return cmpInt64(*a, *b)
}
func cmpOptionalFloat(a, b *float64) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if *a < *b {
		return -1
	}
	if *a > *b {
		return 1
	}
	return 0
}

func (s *TokenUsageReportService) GetUserReport(ctx context.Context, q UserTokenUsageReportQuery) (UserTokenUsageReport, error) {
	q.Model = strings.TrimSpace(q.Model)
	if q.UserID < 0 || validateBaseQuery(q.TokenUsageReportQuery) != nil || !validTokenUsageSort(q.SortBy, q.SortOrder, "user", "model", "user_deleted") {
		return UserTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	today := businessTokenUsageDay(s.now())
	if !q.EndDate.Before(today) && s.todayRepo != nil && s.reader != nil {
		return s.getHybridUserReport(ctx, q, today)
	}
	rows, total, used, err := s.repo.ListUserTokenUsage(ctx, q)
	if err != nil {
		return UserTokenUsageReport{}, fmt.Errorf("get user token usage report: %w", err)
	}
	items := make([]UserTokenUsageItem, 0, len(rows))
	for _, row := range rows {
		rate, status := usageState(row.UsedTokens, row.DailyLimitTokens)
		items = append(items, UserTokenUsageItem{UserTokenUsageRow: row, UsageRate: rate, Status: status})
	}
	return UserTokenUsageReport{Items: items, UsedTokens: used, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

func (s *TokenUsageReportService) getHybridUserReport(ctx context.Context, q UserTokenUsageReportQuery, today time.Time) (UserTokenUsageReport, error) {
	all := []UserTokenUsageRow{}
	if q.StartDate.Before(today) {
		hq := q
		hq.EndDate = today.AddDate(0, 0, -1)
		hq.Page = 1
		hq.PageSize = 100
		for {
			rows, total, _, err := s.repo.ListUserTokenUsage(ctx, hq)
			if err != nil {
				return UserTokenUsageReport{}, err
			}
			all = append(all, rows...)
			if int64(len(all)) >= total || len(rows) == 0 {
				break
			}
			hq.Page++
		}
	}
	mysqlRows, err := s.todayRepo.ListTodayUserTokenUsage(ctx, q, today)
	if err != nil {
		return UserTokenUsageReport{}, err
	}
	filters := []UserTokenUsageRow(nil)
	if q.UserID > 0 && q.Model != "" {
		filters = []UserTokenUsageRow{{UserID: q.UserID, Model: q.Model}}
	}
	redisResult, redisErr := s.reader.ReadUserModelUsage(ctx, today, filters)
	todayRows := mysqlRows
	if redisErr == nil {
		var repair []UserTokenUsageRow
		todayRows, repair = MergeUserTokenUsage(mysqlRows, redisResult.Rows)
		filtered := todayRows[:0]
		for _, row := range todayRows {
			if q.UserID > 0 && row.UserID != q.UserID || q.Model != "" && row.Model != q.Model || !q.IncludeDeleted && row.UserDeleted {
				continue
			}
			filtered = append(filtered, row)
		}
		todayRows = filtered
		positive := repair[:0]
		for _, row := range repair {
			if row.UsedTokens > 0 {
				positive = append(positive, row)
			}
		}
		if len(positive) > 0 && s.repairer != nil {
			_ = s.repairer.RepairUserModelUsage(ctx, today, positive)
		}
	}
	all = append(all, todayRows...)
	items := make([]UserTokenUsageItem, 0, len(all))
	var used int64
	for _, row := range all {
		rate, status := usageState(row.UsedTokens, row.DailyLimitTokens)
		items = append(items, UserTokenUsageItem{UserTokenUsageRow: row, UsageRate: rate, Status: status})
		used += row.UsedTokens
	}
	sort.SliceStable(items, func(i, j int) bool { return userUsageItemLess(items[i], items[j], q.SortBy, q.SortOrder) })
	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := min(start+q.PageSize, total)
	return UserTokenUsageReport{Items: items[start:end], UsedTokens: used, Page: q.Page, PageSize: q.PageSize, Total: int64(total)}, nil
}

func userUsageItemLess(a, b UserTokenUsageItem, sortBy, sortOrder string) bool {
	fields, orders := strings.Split(sortBy, ","), strings.Split(sortOrder, ",")
	for i, field := range fields {
		field = strings.TrimSpace(field)
		cmp := 0
		switch field {
		case "user":
			cmp = strings.Compare(a.Email+a.Username, b.Email+b.Username)
		case "model":
			cmp = strings.Compare(a.Model, b.Model)
		case "user_deleted":
			if !a.UserDeleted && b.UserDeleted {
				cmp = -1
			} else if a.UserDeleted && !b.UserDeleted {
				cmp = 1
			}
		case "used_tokens":
			cmp = cmpInt64(a.UsedTokens, b.UsedTokens)
		case "usage_date":
			if a.UsageDate.Before(b.UsageDate) {
				cmp = -1
			} else if a.UsageDate.After(b.UsageDate) {
				cmp = 1
			}
		case "status":
			cmp = strings.Compare(a.Status, b.Status)
		case "usage_rate":
			cmp = cmpOptionalFloat(a.UsageRate, b.UsageRate)
		case "daily_limit_tokens":
			cmp = cmpOptionalInt64(a.DailyLimitTokens, b.DailyLimitTokens)
		}
		if cmp != 0 {
			if strings.TrimSpace(orders[i]) == "desc" {
				return cmp > 0
			}
			return cmp < 0
		}
	}
	return false
}

func (s *TokenUsageReportService) SearchOptions(ctx context.Context, kind string, parentID int64, query string, limit int) ([]TokenUsageOption, error) {
	allowed := map[string]bool{"models": true, "groups": true, "routes": true, "route_models": true, "users": true, "user_models": true}
	if !allowed[kind] || limit < 1 {
		return nil, ErrInvalidTokenUsageReportQuery
	}
	if limit > 20 {
		limit = 20
	}
	if (kind == "routes" || kind == "route_models" || kind == "user_models") && parentID <= 0 {
		return nil, ErrInvalidTokenUsageReportQuery
	}
	query = strings.TrimSpace(query)
	base, err := s.repo.SearchTokenUsageOptions(ctx, kind, parentID, query, limit)
	if err != nil || s.reader == nil || kind == "groups" || kind == "users" {
		return base, err
	}
	today := businessTokenUsageDay(s.now())
	seen := map[string]bool{}
	for _, item := range base {
		seen[item.Label] = true
	}
	add := func(label string, item TokenUsageOption) {
		if len(base) < limit && !seen[label] && strings.Contains(strings.ToLower(label), strings.ToLower(query)) {
			item.Label = label
			base = append(base, item)
			seen[label] = true
		}
	}
	switch kind {
	case "models":
		result, e := s.reader.ReadModelUsage(ctx, today, nil)
		if e == nil {
			for _, row := range result.Rows {
				add(row.Model, TokenUsageOption{Model: row.Model})
			}
		}
	case "routes":
		result, e := s.reader.ReadRouteUsage(ctx, today, nil)
		if e == nil {
			for _, row := range result.Rows {
				if row.GroupID == parentID {
					add(row.RouteAlias, TokenUsageOption{GroupID: parentID, RouteAlias: row.RouteAlias})
				}
			}
		}
	case "route_models":
		parts := strings.SplitN(query, "\x00", 2)
		if len(parts) == 2 {
			alias, search := parts[0], parts[1]
			query = search
			result, e := s.reader.ReadRouteUsage(ctx, today, nil)
			if e == nil {
				for _, row := range result.Rows {
					if row.GroupID == parentID && row.RouteAlias == alias {
						add(row.UpstreamModel, TokenUsageOption{GroupID: parentID, RouteAlias: alias, Model: row.UpstreamModel})
					}
				}
			}
		}
	case "user_models":
		result, e := s.reader.ReadUserModelUsage(ctx, today, nil)
		if e == nil {
			for _, row := range result.Rows {
				if row.UserID == parentID {
					add(row.Model, TokenUsageOption{UserID: parentID, Model: row.Model})
				}
			}
		}
	}
	return base, nil
}
func (s *TokenUsageReportService) DefaultTarget(ctx context.Context, dimension string, date time.Time) (*TokenUsageOption, error) {
	if dimension != "model" && dimension != "route" && dimension != "user" || date.IsZero() {
		return nil, ErrInvalidTokenUsageReportQuery
	}
	today := businessTokenUsageDay(s.now())
	if tokenUsageDateKey(date) != tokenUsageDateKey(today) || s.reader == nil || s.todayRepo == nil {
		return s.repo.FindDefaultTokenUsageTarget(ctx, dimension, date)
	}
	base := TokenUsageReportQuery{StartDate: today, EndDate: today, Page: 1, PageSize: 100, SortBy: "used_tokens", SortOrder: "desc"}
	switch dimension {
	case "model":
		report, err := s.GetModelReport(ctx, base)
		if err != nil || len(report.Items) == 0 {
			return nil, err
		}
		x := report.Items[0]
		return &TokenUsageOption{Label: x.Model, Model: x.Model}, nil
	case "route":
		report, err := s.GetRouteReport(ctx, RouteTokenUsageReportQuery{TokenUsageReportQuery: base})
		if err != nil || len(report.Items) == 0 {
			return nil, err
		}
		x := report.Items[0]
		return &TokenUsageOption{Label: fmt.Sprintf("%d:%s", x.GroupID, x.RouteAlias), GroupID: x.GroupID, RouteAlias: x.RouteAlias, Model: x.UpstreamModel}, nil
	default:
		report, err := s.GetUserReport(ctx, UserTokenUsageReportQuery{TokenUsageReportQuery: base})
		if err != nil || len(report.Items) == 0 {
			return nil, err
		}
		x := report.Items[0]
		label := x.Email
		if label == "" {
			label = x.Username
		}
		if label == "" {
			label = fmt.Sprintf("%d", x.UserID)
		}
		return &TokenUsageOption{ID: x.UserID, Label: label, UserID: x.UserID, Model: x.Model}, nil
	}
}
