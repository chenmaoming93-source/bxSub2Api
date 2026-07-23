package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type tokenUsageReportRepository struct{ db *sql.DB }

func NewTokenUsageReportRepository(db *sql.DB) service.ModelTokenUsageRepository {
	return &tokenUsageReportRepository{db: db}
}

func (r *tokenUsageReportRepository) ListTodayModelTokenUsage(ctx context.Context, q service.TokenUsageReportQuery, day time.Time) ([]service.ModelTokenUsageRow, error) {
	filter, args := "", []any{}
	if q.Model != "" {
		filter = " WHERE model = ?"
		args = append(args, q.Model)
	}
	query := `WITH targets AS (SELECT model FROM model_token_daily_limit_configs` + filter + ` UNION SELECT model FROM model_token_daily_usages WHERE usage_date = ?`
	usageArgs := []any{day}
	if q.Model != "" {
		query += ` AND model = ?`
		usageArgs = append(usageArgs, q.Model)
	}
	query += `) SELECT t.model,COALESCE(u.used_tokens,0),c.daily_limit_tokens FROM targets t LEFT JOIN model_token_daily_usages u ON u.model=t.model AND u.usage_date=? LEFT JOIN model_token_daily_limit_configs c ON c.model=t.model`
	all := append(args, usageArgs...)
	all = append(all, day)
	rows, err := r.db.QueryContext(ctx, query, all...)
	if err != nil {
		return nil, fmt.Errorf("list today model token usage: %w", err)
	}
	defer rows.Close()
	out := []service.ModelTokenUsageRow{}
	for rows.Next() {
		var x service.ModelTokenUsageRow
		var limit sql.NullInt64
		x.UsageDate = day
		if err := rows.Scan(&x.Model, &x.UsedTokens, &limit); err != nil {
			return nil, err
		}
		if limit.Valid {
			v := limit.Int64
			x.DailyLimitTokens = &v
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *tokenUsageReportRepository) ListTodayRouteTokenUsage(ctx context.Context, q service.RouteTokenUsageReportQuery, day time.Time) ([]service.RouteTokenUsageRow, error) {
	filters := []string{"1=1"}
	configArgs := []any{}
	usageArgs := []any{day}
	add := func(column string, value any, active bool) {
		if active {
			filters = append(filters, column+" = ?")
			configArgs = append(configArgs, value)
			usageArgs = append(usageArgs, value)
		}
	}
	add("group_id", q.GroupID, q.GroupID > 0)
	add("route_alias", q.RouteAlias, q.RouteAlias != "")
	add("upstream_model", q.UpstreamModel, q.UpstreamModel != "")
	f := strings.Join(filters, " AND ")
	query := `WITH targets AS (SELECT group_id,route_alias,upstream_model FROM group_candidate_token_daily_limit_configs WHERE ` + f + ` UNION SELECT group_id,route_alias,upstream_model FROM group_candidate_token_daily_usages WHERE usage_date = ? AND ` + f + `) SELECT t.group_id,COALESCE(g.name,''),t.route_alias,t.upstream_model,COALESCE(u.used_tokens,0),c.daily_limit_tokens,rc.priority FROM targets t LEFT JOIN group_candidate_token_daily_usages u ON u.group_id=t.group_id AND u.route_alias=t.route_alias AND u.upstream_model=t.upstream_model AND u.usage_date=? LEFT JOIN group_candidate_token_daily_limit_configs c ON c.group_id=t.group_id AND c.route_alias=t.route_alias AND c.upstream_model=t.upstream_model LEFT JOIN ` + "`groups`" + ` g ON g.id=t.group_id LEFT JOIN JSON_TABLE(JSON_EXTRACT(g.model_routing,CONCAT('$."',t.route_alias,'"')),'$[*]' COLUMNS(model VARCHAR(255) PATH '$.model',priority INT PATH '$.priority')) rc ON rc.model=t.upstream_model`
	args := append([]any{}, configArgs...)
	args = append(args, usageArgs...)
	args = append(args, day)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list today route token usage: %w", err)
	}
	defer rows.Close()
	out := []service.RouteTokenUsageRow{}
	for rows.Next() {
		var x service.RouteTokenUsageRow
		var limit, priority sql.NullInt64
		x.UsageDate = day
		if err := rows.Scan(&x.GroupID, &x.GroupName, &x.RouteAlias, &x.UpstreamModel, &x.UsedTokens, &limit, &priority); err != nil {
			return nil, err
		}
		if limit.Valid {
			v := limit.Int64
			x.DailyLimitTokens = &v
		}
		if priority.Valid {
			v := int(priority.Int64)
			x.Priority = &v
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *tokenUsageReportRepository) ListTodayUserTokenUsage(ctx context.Context, q service.UserTokenUsageReportQuery, day time.Time) ([]service.UserTokenUsageRow, error) {
	filters := []string{"1=1"}
	configArgs := []any{}
	usageArgs := []any{day}
	add := func(column string, value any, active bool) {
		if active {
			filters = append(filters, column+" = ?")
			configArgs = append(configArgs, value)
			usageArgs = append(usageArgs, value)
		}
	}
	add("user_id", q.UserID, q.UserID > 0)
	add("model", q.Model, q.Model != "")
	f := strings.Join(filters, " AND ")
	query := `WITH targets AS (SELECT user_id,model FROM user_model_token_daily_limit_configs WHERE ` + f + ` UNION SELECT user_id,model FROM user_model_token_daily_usages WHERE usage_date = ? AND ` + f + `) SELECT t.user_id,COALESCE(us.email,''),COALESCE(us.username,''),us.deleted_at IS NOT NULL,t.model,COALESCE(u.used_tokens,0),c.daily_limit_tokens FROM targets t LEFT JOIN user_model_token_daily_usages u ON u.user_id=t.user_id AND u.model=t.model AND u.usage_date=? LEFT JOIN user_model_token_daily_limit_configs c ON c.user_id=t.user_id AND c.model=t.model LEFT JOIN users us ON us.id=t.user_id`
	if !q.IncludeDeleted {
		query += ` WHERE us.deleted_at IS NULL`
	}
	args := append([]any{}, configArgs...)
	args = append(args, usageArgs...)
	args = append(args, day)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list today user token usage: %w", err)
	}
	defer rows.Close()
	out := []service.UserTokenUsageRow{}
	for rows.Next() {
		var x service.UserTokenUsageRow
		var limit sql.NullInt64
		x.UsageDate = day
		if err := rows.Scan(&x.UserID, &x.Email, &x.Username, &x.UserDeleted, &x.Model, &x.UsedTokens, &limit); err != nil {
			return nil, err
		}
		if limit.Valid {
			v := limit.Int64
			x.DailyLimitTokens = &v
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *tokenUsageReportRepository) ListModelTokenUsage(ctx context.Context, q service.TokenUsageReportQuery) ([]service.ModelTokenUsageRow, int64, int64, error) {
	configFilter, usageFilter := "1=1", "usage_date BETWEEN ? AND ?"
	baseArgs := []any{q.StartDate, q.EndDate}
	if q.Model != "" {
		configFilter = "model = ?"
		baseArgs = append(baseArgs, q.Model)
	}
	usageArgs := []any{q.StartDate, q.EndDate}
	if q.Model != "" {
		usageFilter += " AND model = ?"
		usageArgs = append(usageArgs, q.Model)
	}
	baseArgs = append(baseArgs, usageArgs...)
	reportCTE := `WITH RECURSIVE ` + tokenUsageDateSeriesSQL + `,
targets(model) AS (
  SELECT model FROM model_token_daily_limit_configs WHERE ` + configFilter + `
  UNION
  SELECT model FROM model_token_daily_usages WHERE ` + usageFilter + `
),
report AS (
  SELECT d.usage_date, t.model, COALESCE(u.used_tokens, 0) AS used_tokens, c.daily_limit_tokens
  FROM targets t CROSS JOIN dates d
  LEFT JOIN model_token_daily_usages u ON u.model = t.model AND u.usage_date = d.usage_date
  LEFT JOIN model_token_daily_limit_configs c ON c.model = t.model
) `
	var total, used int64
	if err := r.db.QueryRowContext(ctx, reportCTE+`SELECT COUNT(*), COALESCE(SUM(used_tokens), 0) FROM report`, baseArgs...).Scan(&total, &used); err != nil {
		return nil, 0, 0, fmt.Errorf("summarize model token usage: %w", err)
	}
	sort := TokenUsageSort{By: q.SortBy, Order: q.SortOrder}
	listArgs := append(append([]any{}, baseArgs...), q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, reportCTE+`SELECT r.usage_date, r.model, r.used_tokens, r.daily_limit_tokens
FROM report r ORDER BY `+tokenUsageReportOrderBy(sort, reportSortColumns(map[string]string{"model": "r.model"}), "r.model ASC", "r.usage_date ASC")+` LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list model token usage: %w", err)
	}
	defer rows.Close()
	items := make([]service.ModelTokenUsageRow, 0)
	for rows.Next() {
		var item service.ModelTokenUsageRow
		var limit sql.NullInt64
		if err := rows.Scan(&item.UsageDate, &item.Model, &item.UsedTokens, &limit); err != nil {
			return nil, 0, 0, err
		}
		if limit.Valid {
			value := limit.Int64
			item.DailyLimitTokens = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}
	return items, total, used, nil
}

func (r *tokenUsageReportRepository) ListRouteTokenUsage(ctx context.Context, q service.RouteTokenUsageReportQuery) ([]service.RouteTokenUsageRow, int64, int64, error) {
	configFilters := []string{"1=1"}
	usageFilters := []string{"usage_date BETWEEN ? AND ?"}
	args := []any{q.StartDate, q.EndDate}
	usageArgs := []any{q.StartDate, q.EndDate}
	if q.GroupID > 0 {
		configFilters = append(configFilters, "group_id = ?")
		usageFilters = append(usageFilters, "group_id = ?")
		args = append(args, q.GroupID)
		usageArgs = append(usageArgs, q.GroupID)
	}
	if q.RouteAlias != "" {
		configFilters = append(configFilters, "route_alias = ?")
		usageFilters = append(usageFilters, "route_alias = ?")
		args = append(args, q.RouteAlias)
		usageArgs = append(usageArgs, q.RouteAlias)
	}
	if q.UpstreamModel != "" {
		configFilters = append(configFilters, "upstream_model = ?")
		usageFilters = append(usageFilters, "upstream_model = ?")
		args = append(args, q.UpstreamModel)
		usageArgs = append(usageArgs, q.UpstreamModel)
	}
	args = append(args, usageArgs...)
	configFilter := strings.Join(configFilters, " AND ")
	usageFilter := strings.Join(usageFilters, " AND ")
	reportCTE := `WITH RECURSIVE ` + tokenUsageDateSeriesSQL + `,
targets(group_id, route_alias, upstream_model) AS (
  SELECT group_id, route_alias, upstream_model FROM group_candidate_token_daily_limit_configs WHERE ` + configFilter + `
  UNION
  SELECT group_id, route_alias, upstream_model FROM group_candidate_token_daily_usages WHERE ` + usageFilter + `
),
report AS (
  SELECT d.usage_date, t.group_id, t.route_alias, t.upstream_model, COALESCE(u.used_tokens, 0) AS used_tokens, c.daily_limit_tokens
  FROM targets t CROSS JOIN dates d
  LEFT JOIN group_candidate_token_daily_usages u ON u.group_id=t.group_id AND u.route_alias=t.route_alias AND u.upstream_model=t.upstream_model AND u.usage_date=d.usage_date
  LEFT JOIN group_candidate_token_daily_limit_configs c ON c.group_id=t.group_id AND c.route_alias=t.route_alias AND c.upstream_model=t.upstream_model
) `
	var total, used int64
	if err := r.db.QueryRowContext(ctx, reportCTE+`SELECT COUNT(*), COALESCE(SUM(used_tokens), 0) FROM report`, args...).Scan(&total, &used); err != nil {
		return nil, 0, 0, err
	}
	listArgs := append(append([]any{}, args...), q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, reportCTE+`SELECT r.usage_date, r.group_id, COALESCE(g.name, ''), r.route_alias, r.upstream_model, r.used_tokens, r.daily_limit_tokens, rc.priority
FROM report r
LEFT JOIN `+"`groups`"+` g ON g.id = r.group_id
LEFT JOIN JSON_TABLE(JSON_EXTRACT(g.model_routing, CONCAT('$."', r.route_alias, '"')), '$[*]' COLUMNS(model VARCHAR(255) PATH '$.model', priority INT PATH '$.priority')) rc ON rc.model=r.upstream_model
ORDER BY `+tokenUsageReportOrderBy(TokenUsageSort{By: q.SortBy, Order: q.SortOrder}, reportSortColumns(map[string]string{"group": "COALESCE(g.name, '')", "route_alias": "r.route_alias", "upstream_model": "r.upstream_model", "priority": "rc.priority"}), "r.group_id ASC", "r.route_alias ASC", "r.upstream_model ASC", "r.usage_date ASC")+` LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	items := []service.RouteTokenUsageRow{}
	for rows.Next() {
		var x service.RouteTokenUsageRow
		var limit sql.NullInt64
		var priority sql.NullInt64
		if err := rows.Scan(&x.UsageDate, &x.GroupID, &x.GroupName, &x.RouteAlias, &x.UpstreamModel, &x.UsedTokens, &limit, &priority); err != nil {
			return nil, 0, 0, err
		}
		if limit.Valid {
			v := limit.Int64
			x.DailyLimitTokens = &v
		}
		if priority.Valid {
			v := int(priority.Int64)
			x.Priority = &v
		}
		items = append(items, x)
	}
	return items, total, used, rows.Err()
}

func (r *tokenUsageReportRepository) ListUserTokenUsage(ctx context.Context, q service.UserTokenUsageReportQuery) ([]service.UserTokenUsageRow, int64, int64, error) {
	configFilters := []string{"1=1"}
	usageFilters := []string{"usage_date BETWEEN ? AND ?"}
	args := []any{q.StartDate, q.EndDate}
	usageArgs := []any{q.StartDate, q.EndDate}
	if !q.IncludeDeleted {
		configFilters = append(configFilters, "EXISTS (SELECT 1 FROM users target_user WHERE target_user.id = user_model_token_daily_limit_configs.user_id AND target_user.deleted_at IS NULL)")
		usageFilters = append(usageFilters, "EXISTS (SELECT 1 FROM users target_user WHERE target_user.id = user_model_token_daily_usages.user_id AND target_user.deleted_at IS NULL)")
	}
	if q.UserID > 0 {
		configFilters = append(configFilters, "user_id = ?")
		usageFilters = append(usageFilters, "user_id = ?")
		args = append(args, q.UserID)
		usageArgs = append(usageArgs, q.UserID)
	}
	if q.Model != "" {
		configFilters = append(configFilters, "model = ?")
		usageFilters = append(usageFilters, "model = ?")
		args = append(args, q.Model)
		usageArgs = append(usageArgs, q.Model)
	}
	args = append(args, usageArgs...)
	configFilter := strings.Join(configFilters, " AND ")
	usageFilter := strings.Join(usageFilters, " AND ")
	reportCTE := `WITH RECURSIVE ` + tokenUsageDateSeriesSQL + `,
targets(user_id, model) AS (
  SELECT user_id, model FROM user_model_token_daily_limit_configs WHERE ` + configFilter + `
  UNION
  SELECT user_id, model FROM user_model_token_daily_usages WHERE ` + usageFilter + `
),
report AS (
  SELECT d.usage_date, t.user_id, t.model, COALESCE(u.used_tokens, 0) AS used_tokens, c.daily_limit_tokens
  FROM targets t CROSS JOIN dates d
  LEFT JOIN user_model_token_daily_usages u ON u.user_id=t.user_id AND u.model=t.model AND u.usage_date=d.usage_date
  LEFT JOIN user_model_token_daily_limit_configs c ON c.user_id=t.user_id AND c.model=t.model
) `
	var total, used int64
	if err := r.db.QueryRowContext(ctx, reportCTE+`SELECT COUNT(*), COALESCE(SUM(used_tokens), 0) FROM report`, args...).Scan(&total, &used); err != nil {
		return nil, 0, 0, err
	}
	listArgs := append(append([]any{}, args...), q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, reportCTE+`SELECT r.usage_date,r.user_id,COALESCE(us.email,''),COALESCE(us.username,''),us.deleted_at IS NOT NULL,r.model,r.used_tokens,r.daily_limit_tokens
FROM report r LEFT JOIN users us ON us.id=r.user_id
ORDER BY `+tokenUsageReportOrderBy(TokenUsageSort{By: q.SortBy, Order: q.SortOrder}, reportSortColumns(map[string]string{"user": "COALESCE(NULLIF(us.email,''),NULLIF(us.username,''),CAST(r.user_id AS CHAR))", "model": "r.model", "user_deleted": "us.deleted_at IS NOT NULL"}), "r.user_id ASC", "r.model ASC", "r.usage_date ASC")+` LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	items := []service.UserTokenUsageRow{}
	for rows.Next() {
		var x service.UserTokenUsageRow
		var limit sql.NullInt64
		if err := rows.Scan(&x.UsageDate, &x.UserID, &x.Email, &x.Username, &x.UserDeleted, &x.Model, &x.UsedTokens, &limit); err != nil {
			return nil, 0, 0, err
		}
		if limit.Valid {
			v := limit.Int64
			x.DailyLimitTokens = &v
		}
		items = append(items, x)
	}
	return items, total, used, rows.Err()
}

func (r *tokenUsageReportRepository) SearchTokenUsageOptions(ctx context.Context, kind string, parentID int64, q string, limit int) ([]service.TokenUsageOption, error) {
	like := "%" + q + "%"
	var query string
	var args []any
	switch kind {
	case "models":
		query = `SELECT 0,model FROM model_token_daily_limit_configs WHERE model LIKE ? UNION SELECT 0,model FROM model_token_daily_usages WHERE model LIKE ? LIMIT ?`
		args = []any{like, like, limit}
	case "groups":
		query = "SELECT id,name FROM `groups` WHERE deleted_at IS NULL AND (name LIKE ? OR CAST(id AS CHAR) LIKE ?) ORDER BY name LIMIT ?"
		args = []any{like, like, limit}
	case "routes":
		query = `SELECT 0,route_alias FROM group_candidate_token_daily_limit_configs WHERE group_id=? AND route_alias LIKE ? UNION SELECT 0,route_alias FROM group_candidate_token_daily_usages WHERE group_id=? AND route_alias LIKE ? ORDER BY route_alias LIMIT ?`
		args = []any{parentID, like, parentID, like, limit}
	case "route_models":
		query = `SELECT 0,upstream_model FROM group_candidate_token_daily_limit_configs WHERE group_id=? AND route_alias=? AND upstream_model LIKE ? UNION SELECT 0,upstream_model FROM group_candidate_token_daily_usages WHERE group_id=? AND route_alias=? AND upstream_model LIKE ? ORDER BY upstream_model LIMIT ?`
		parts := strings.SplitN(q, "\x00", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, service.ErrInvalidTokenUsageReportQuery
		}
		routeAlias, modelLike := parts[0], "%"+parts[1]+"%"
		args = []any{parentID, routeAlias, modelLike, parentID, routeAlias, modelLike, limit}
	case "users":
		query = `SELECT id,CONCAT(COALESCE(NULLIF(email,''),NULLIF(username,''),CAST(id AS CHAR)),IF(deleted_at IS NULL,'',' (deleted)')) FROM users WHERE email LIKE ? OR username LIKE ? OR CAST(id AS CHAR) LIKE ? ORDER BY deleted_at IS NOT NULL,id LIMIT ?`
		args = []any{like, like, like, limit}
	case "user_models":
		query = `SELECT DISTINCT 0,model FROM user_model_token_daily_limit_configs WHERE user_id=? AND model LIKE ? UNION SELECT DISTINCT 0,model FROM user_model_token_daily_usages WHERE user_id=? AND model LIKE ? LIMIT ?`
		args = []any{parentID, like, parentID, like, limit}
	default:
		return nil, service.ErrInvalidTokenUsageReportQuery
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.TokenUsageOption{}
	for rows.Next() {
		var x service.TokenUsageOption
		if err := rows.Scan(&x.ID, &x.Label); err != nil {
			return nil, err
		}
		switch kind {
		case "models", "user_models", "route_models":
			x.Model = x.Label
		case "groups":
			x.GroupID = x.ID
		case "routes":
			x.GroupID = parentID
			x.RouteAlias = x.Label
		case "users":
			x.UserID = x.ID
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *tokenUsageReportRepository) FindDefaultTokenUsageTarget(ctx context.Context, dimension string, date time.Time) (*service.TokenUsageOption, error) {
	queries := map[string][2]string{"model": {`SELECT 0,model,model,0,'',0 FROM model_token_daily_usages WHERE usage_date=? ORDER BY used_tokens DESC LIMIT 1`, `SELECT 0,model,model,0,'',0 FROM model_token_daily_limit_configs ORDER BY model LIMIT 1`}, "route": {`SELECT 0,CONCAT(group_id,':',route_alias),'',group_id,route_alias,0 FROM group_candidate_token_daily_usages WHERE usage_date=? ORDER BY used_tokens DESC LIMIT 1`, `SELECT 0,CONCAT(group_id,':',route_alias),'',group_id,route_alias,0 FROM group_candidate_token_daily_limit_configs ORDER BY group_id,route_alias LIMIT 1`}, "user": {`SELECT u.user_id,COALESCE(us.email,CAST(u.user_id AS CHAR)),'',0,'',u.user_id FROM user_model_token_daily_usages u LEFT JOIN users us ON us.id=u.user_id WHERE u.usage_date=? ORDER BY u.used_tokens DESC LIMIT 1`, `SELECT c.user_id,COALESCE(us.email,CAST(c.user_id AS CHAR)),'',0,'',c.user_id FROM user_model_token_daily_limit_configs c LEFT JOIN users us ON us.id=c.user_id ORDER BY c.user_id LIMIT 1`}}
	pair, ok := queries[dimension]
	if !ok {
		return nil, service.ErrInvalidTokenUsageReportQuery
	}
	scan := func(query string, args ...any) (*service.TokenUsageOption, error) {
		var x service.TokenUsageOption
		err := r.db.QueryRowContext(ctx, query, args...).Scan(&x.ID, &x.Label, &x.Model, &x.GroupID, &x.RouteAlias, &x.UserID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return &x, err
	}
	x, err := scan(pair[0], date)
	if err != nil || x != nil {
		return x, err
	}
	return scan(pair[1])
}
