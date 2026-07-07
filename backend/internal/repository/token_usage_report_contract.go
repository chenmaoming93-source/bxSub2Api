package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TokenUsageDefaultPageSize = 20
	TokenUsageMaxPageSize     = 100
)

var ErrInvalidTokenUsageQuery = errors.New("invalid token usage query")

type TokenUsageSort struct {
	By    string
	Order string
}

type TokenUsagePage struct {
	Page     int
	PageSize int
}

type TokenUsageQueryContract struct {
	StartDate time.Time
	EndDate   time.Time
	Page      TokenUsagePage
	Sort      TokenUsageSort
}

func (c TokenUsageQueryContract) Validate() error {
	if c.StartDate.IsZero() || c.EndDate.IsZero() || c.EndDate.Before(c.StartDate) {
		return fmt.Errorf("%w: date range", ErrInvalidTokenUsageQuery)
	}
	if c.Page.Page < 1 || c.Page.PageSize < 1 || c.Page.PageSize > TokenUsageMaxPageSize {
		return fmt.Errorf("%w: pagination", ErrInvalidTokenUsageQuery)
	}
	if c.Sort.By != "usage_date" && c.Sort.By != "used_tokens" {
		return fmt.Errorf("%w: sort_by", ErrInvalidTokenUsageQuery)
	}
	if c.Sort.Order != "asc" && c.Sort.Order != "desc" {
		return fmt.Errorf("%w: sort_order", ErrInvalidTokenUsageQuery)
	}
	return nil
}

func tokenUsageOrderBy(sort TokenUsageSort) string {
	// Values must have passed Validate. Keeping this mapping explicit prevents
	// user-controlled identifiers from entering SQL.
	column := map[string]string{"usage_date": "u.usage_date", "used_tokens": "u.used_tokens"}[sort.By]
	return column + " " + strings.ToUpper(sort.Order) + ", u.id ASC"
}

func tokenUsageReportOrderBy(sort TokenUsageSort, columns map[string]string, tieBreakers ...string) string {
	column := columns[sort.By]
	return column + " " + strings.ToUpper(sort.Order) + ", " + strings.Join(tieBreakers, ", ")
}

var tokenUsageCommonSortColumns = map[string]string{
	"usage_date":         "r.usage_date",
	"used_tokens":        "r.used_tokens",
	"daily_limit_tokens": "r.daily_limit_tokens",
	"usage_rate":         "CASE WHEN r.daily_limit_tokens > 0 THEN r.used_tokens / r.daily_limit_tokens ELSE NULL END",
	"status":             "CASE WHEN r.daily_limit_tokens IS NULL OR r.daily_limit_tokens <= 0 THEN 0 WHEN r.used_tokens >= r.daily_limit_tokens THEN 3 WHEN r.used_tokens * 10 >= r.daily_limit_tokens * 8 THEN 2 ELSE 1 END",
}

func reportSortColumns(extra map[string]string) map[string]string {
	columns := make(map[string]string, len(tokenUsageCommonSortColumns)+len(extra))
	for key, value := range tokenUsageCommonSortColumns {
		columns[key] = value
	}
	for key, value := range extra {
		columns[key] = value
	}
	return columns
}

const tokenUsageDateSeriesSQL = `dates(usage_date) AS (
  SELECT DATE(?)
  UNION ALL
  SELECT DATE_ADD(usage_date, INTERVAL 1 DAY) FROM dates WHERE usage_date < DATE(?)
)`

func modelTokenUsageListSQL(sort TokenUsageSort) string {
	return `SELECT u.id, u.usage_date, u.model, u.used_tokens
FROM model_token_daily_usages u
WHERE u.model = ? AND u.usage_date BETWEEN ? AND ?
ORDER BY ` + tokenUsageOrderBy(sort) + ` LIMIT ? OFFSET ?`
}

func routeTokenUsageListSQL(sort TokenUsageSort, withModel bool) string {
	query := `SELECT u.id, u.usage_date, u.group_id, u.route_alias, u.upstream_model, u.used_tokens
FROM group_candidate_token_daily_usages u
WHERE u.group_id = ? AND u.route_alias = ? AND u.usage_date BETWEEN ? AND ?`
	if withModel {
		query += ` AND u.upstream_model = ?`
	}
	return query + ` ORDER BY ` + tokenUsageOrderBy(sort) + ` LIMIT ? OFFSET ?`
}

func userModelTokenUsageListSQL(sort TokenUsageSort, withModel bool) string {
	query := `SELECT u.id, u.usage_date, u.user_id, u.model, u.used_tokens
FROM user_model_token_daily_usages u
WHERE u.user_id = ? AND u.usage_date BETWEEN ? AND ?`
	if withModel {
		query += ` AND u.model = ?`
	}
	return query + ` ORDER BY ` + tokenUsageOrderBy(sort) + ` LIMIT ? OFFSET ?`
}
