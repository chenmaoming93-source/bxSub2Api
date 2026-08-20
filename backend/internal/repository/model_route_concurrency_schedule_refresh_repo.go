package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ListModelRouteConcurrencyScheduleCandidates reads only candidates that have
// schedule rows. Candidates without rows intentionally remain on the legacy
// request-path key and therefore do not create a new Redis key.
func (r *groupRepository) ListModelRouteConcurrencyScheduleCandidates(ctx context.Context) ([]service.ModelRouteConcurrencyScheduleCandidate, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT s.group_id, s.route_alias, s.account_id,
		       r.max_concurrency,
		       s.id, s.start_minute, s.end_minute, s.max_concurrency,
		       s.created_at, s.updated_at
		FROM group_model_route_account_concurrency_schedules AS s
		JOIN group_model_route_accounts AS r
		  ON r.group_id = s.group_id
		 AND r.route_alias = s.route_alias
		 AND r.account_id = s.account_id
		JOIN `+"`groups`"+` AS g ON g.id = s.group_id
		WHERE g.deleted_at IS NULL
		ORDER BY s.group_id, s.route_alias, s.account_id, s.start_minute, s.end_minute, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]service.ModelRouteConcurrencyScheduleCandidate, 0)
	indexByKey := make(map[string]int)
	for rows.Next() {
		var candidate service.ModelRouteConcurrencyScheduleCandidate
		var schedule service.ModelRouteConcurrencySchedule
		if err := rows.Scan(
			&candidate.GroupID, &candidate.RouteAlias, &candidate.AccountID,
			&candidate.DefaultMaxConcurrency,
			&schedule.ID, &schedule.StartMinute, &schedule.EndMinute,
			&schedule.MaxConcurrency, &schedule.CreatedAt, &schedule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedule.GroupID = candidate.GroupID
		schedule.RouteAlias = candidate.RouteAlias
		schedule.AccountID = candidate.AccountID
		key := routeKey(candidate.GroupID, candidate.RouteAlias, candidate.AccountID)
		idx, ok := indexByKey[key]
		if !ok {
			idx = len(items)
			indexByKey[key] = idx
			items = append(items, candidate)
		}
		items[idx].Schedules = append(items[idx].Schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func routeKey(groupID int64, routeAlias string, accountID int64) string {
	return fmt.Sprintf("%d|%s|%d", groupID, routeAlias, accountID)
}
