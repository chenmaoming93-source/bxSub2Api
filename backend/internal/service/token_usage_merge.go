package service

import "time"

type modelTokenUsageKey struct {
	usageDate string
	model     string
}

type routeTokenUsageKey struct {
	usageDate     string
	groupID       int64
	routeAlias    string
	upstreamModel string
}

type userTokenUsageKey struct {
	usageDate string
	userID    int64
	model     string
}

func modelUsageKey(row ModelTokenUsageRow) modelTokenUsageKey {
	return modelTokenUsageKey{usageDate: tokenUsageDateKey(row.UsageDate), model: row.Model}
}

func routeUsageKey(row RouteTokenUsageRow) routeTokenUsageKey {
	return routeTokenUsageKey{
		usageDate: tokenUsageDateKey(row.UsageDate), groupID: row.GroupID,
		routeAlias: row.RouteAlias, upstreamModel: row.UpstreamModel,
	}
}

func userUsageKey(row UserTokenUsageRow) userTokenUsageKey {
	return userTokenUsageKey{usageDate: tokenUsageDateKey(row.UsageDate), userID: row.UserID, model: row.Model}
}

func tokenUsageDateKey(value time.Time) string { return value.Format(time.DateOnly) }

// MergeModelTokenUsage combines current-day snapshots. Redis values are authoritative;
// MySQL-only rows are returned separately as read-repair candidates.
func MergeModelTokenUsage(mysqlRows, redisRows []ModelTokenUsageRow) ([]ModelTokenUsageRow, []ModelTokenUsageRow) {
	mysqlByKey := indexRows(mysqlRows, modelUsageKey)
	return mergeUsageRows(mysqlByKey, redisRows, modelUsageKey, func(redis, mysql ModelTokenUsageRow) ModelTokenUsageRow {
		redis.DailyLimitTokens = mysql.DailyLimitTokens
		return redis
	})
}

// MergeRouteTokenUsage combines current-day route-candidate snapshots.
func MergeRouteTokenUsage(mysqlRows, redisRows []RouteTokenUsageRow) ([]RouteTokenUsageRow, []RouteTokenUsageRow) {
	mysqlByKey := indexRows(mysqlRows, routeUsageKey)
	return mergeUsageRows(mysqlByKey, redisRows, routeUsageKey, func(redis, mysql RouteTokenUsageRow) RouteTokenUsageRow {
		redis.GroupName = mysql.GroupName
		redis.DailyLimitTokens = mysql.DailyLimitTokens
		redis.Priority = mysql.Priority
		return redis
	})
}

// MergeUserTokenUsage combines current-day user-model snapshots.
func MergeUserTokenUsage(mysqlRows, redisRows []UserTokenUsageRow) ([]UserTokenUsageRow, []UserTokenUsageRow) {
	mysqlByKey := indexRows(mysqlRows, userUsageKey)
	return mergeUsageRows(mysqlByKey, redisRows, userUsageKey, func(redis, mysql UserTokenUsageRow) UserTokenUsageRow {
		redis.Email = mysql.Email
		redis.Username = mysql.Username
		redis.UserDeleted = mysql.UserDeleted
		redis.DailyLimitTokens = mysql.DailyLimitTokens
		return redis
	})
}

func indexRows[T any, K comparable](rows []T, key func(T) K) map[K]T {
	indexed := make(map[K]T, len(rows))
	for _, row := range rows {
		indexed[key(row)] = row
	}
	return indexed
}

func mergeUsageRows[T any, K comparable](mysqlByKey map[K]T, redisRows []T, key func(T) K, enrich func(T, T) T) ([]T, []T) {
	final := make([]T, 0, len(mysqlByKey)+len(redisRows))
	seen := make(map[K]struct{}, len(mysqlByKey)+len(redisRows))
	for _, redisRow := range redisRows {
		rowKey := key(redisRow)
		if _, exists := seen[rowKey]; exists {
			continue
		}
		seen[rowKey] = struct{}{}
		if mysqlRow, exists := mysqlByKey[rowKey]; exists {
			redisRow = enrich(redisRow, mysqlRow)
		}
		final = append(final, redisRow)
	}

	repair := make([]T, 0)
	for rowKey, mysqlRow := range mysqlByKey {
		if _, exists := seen[rowKey]; exists {
			continue
		}
		seen[rowKey] = struct{}{}
		final = append(final, mysqlRow)
		repair = append(repair, mysqlRow)
	}
	return final, repair
}
