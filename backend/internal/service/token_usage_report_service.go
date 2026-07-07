package service

import (
	"context"
	"errors"
	"fmt"
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

type TokenUsageReportService struct{ repo ModelTokenUsageRepository }

func NewTokenUsageReportService(repo ModelTokenUsageRepository) *TokenUsageReportService {
	return &TokenUsageReportService{repo: repo}
}

func (s *TokenUsageReportService) GetModelReport(ctx context.Context, query TokenUsageReportQuery) (ModelTokenUsageReport, error) {
	query.Model = strings.TrimSpace(query.Model)
	if query.StartDate.IsZero() || query.EndDate.IsZero() || query.EndDate.Before(query.StartDate) || query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		return ModelTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	if !validTokenUsageSort(query.SortBy, "model") {
		return ModelTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
	}
	if query.SortOrder != "asc" && query.SortOrder != "desc" {
		return ModelTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
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
	if q.SortOrder != "asc" && q.SortOrder != "desc" {
		return ErrInvalidTokenUsageReportQuery
	}
	return nil
}

func validTokenUsageSort(sortBy string, extra ...string) bool {
	common := map[string]bool{"usage_date": true, "used_tokens": true, "daily_limit_tokens": true, "usage_rate": true, "status": true}
	if common[sortBy] {
		return true
	}
	for _, field := range extra {
		if sortBy == field {
			return true
		}
	}
	return false
}

func (s *TokenUsageReportService) GetRouteReport(ctx context.Context, q RouteTokenUsageReportQuery) (RouteTokenUsageReport, error) {
	q.RouteAlias, q.UpstreamModel = strings.TrimSpace(q.RouteAlias), strings.TrimSpace(q.UpstreamModel)
	if q.GroupID < 0 || validateBaseQuery(q.TokenUsageReportQuery) != nil || !validTokenUsageSort(q.SortBy, "group", "route_alias", "upstream_model", "priority") {
		return RouteTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
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

func (s *TokenUsageReportService) GetUserReport(ctx context.Context, q UserTokenUsageReportQuery) (UserTokenUsageReport, error) {
	q.Model = strings.TrimSpace(q.Model)
	if q.UserID < 0 || validateBaseQuery(q.TokenUsageReportQuery) != nil || !validTokenUsageSort(q.SortBy, "user", "model", "user_deleted") {
		return UserTokenUsageReport{}, ErrInvalidTokenUsageReportQuery
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
	return s.repo.SearchTokenUsageOptions(ctx, kind, parentID, strings.TrimSpace(query), limit)
}
func (s *TokenUsageReportService) DefaultTarget(ctx context.Context, dimension string, date time.Time) (*TokenUsageOption, error) {
	if dimension != "model" && dimension != "route" && dimension != "user" || date.IsZero() {
		return nil, ErrInvalidTokenUsageReportQuery
	}
	return s.repo.FindDefaultTokenUsageTarget(ctx, dimension, date)
}
