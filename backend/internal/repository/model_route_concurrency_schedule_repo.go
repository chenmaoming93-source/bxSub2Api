package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *groupRepository) ListModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64) ([]service.ModelRouteConcurrencySchedule, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, group_id, route_alias, account_id, start_minute, end_minute,
		       max_concurrency, created_at, updated_at
		FROM group_model_route_account_concurrency_schedules
		WHERE group_id = ? AND route_alias = ? AND account_id = ?
		ORDER BY start_minute, end_minute, id`, groupID, routeAlias, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]service.ModelRouteConcurrencySchedule, 0)
	for rows.Next() {
		var item service.ModelRouteConcurrencySchedule
		if err := rows.Scan(
			&item.ID, &item.GroupID, &item.RouteAlias, &item.AccountID,
			&item.StartMinute, &item.EndMinute, &item.MaxConcurrency,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *groupRepository) ListAllModelRouteConcurrencySchedules(ctx context.Context) ([]service.ModelRouteConcurrencySchedule, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, group_id, route_alias, account_id, start_minute, end_minute,
		       max_concurrency, created_at, updated_at
		FROM group_model_route_account_concurrency_schedules
		ORDER BY group_id, route_alias, account_id, start_minute, end_minute, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]service.ModelRouteConcurrencySchedule, 0)
	for rows.Next() {
		var item service.ModelRouteConcurrencySchedule
		if err := rows.Scan(
			&item.ID, &item.GroupID, &item.RouteAlias, &item.AccountID,
			&item.StartMinute, &item.EndMinute, &item.MaxConcurrency,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *groupRepository) ReplaceModelRouteConcurrencySchedules(ctx context.Context, groupID int64, routeAlias string, accountID int64, schedules []service.ModelRouteConcurrencySchedule) error {
	if groupID <= 0 || accountID <= 0 || strings.TrimSpace(routeAlias) == "" {
		return fmt.Errorf("%w: candidate identity is required", service.ErrInvalidModelRouteConcurrencySchedule)
	}
	for i := range schedules {
		schedules[i].GroupID = groupID
		schedules[i].RouteAlias = routeAlias
		schedules[i].AccountID = accountID
	}
	if err := service.ValidateModelRouteConcurrencySchedules(schedules); err != nil {
		return err
	}

	var exec sqlExecutor = r.sql
	var tx *sql.Tx
	if db, ok := r.sql.(*sql.DB); ok {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		exec = tx
		defer tx.Rollback()
	}

	if _, err := exec.ExecContext(ctx,
		"DELETE FROM group_model_route_account_concurrency_schedules WHERE group_id = ? AND route_alias = ? AND account_id = ?",
		groupID, routeAlias, accountID,
	); err != nil {
		return err
	}
	for _, schedule := range schedules {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO group_model_route_account_concurrency_schedules
				(group_id, route_alias, account_id, start_minute, end_minute, max_concurrency)
			VALUES (?, ?, ?, ?, ?, ?)`,
			groupID, routeAlias, accountID, schedule.StartMinute, schedule.EndMinute, schedule.MaxConcurrency,
		); err != nil {
			return err
		}
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}
