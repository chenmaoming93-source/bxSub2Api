package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type GroupModelRouteRebuildResult struct {
	GroupsProcessed int64 `json:"groups_processed"`
}

type GroupModelRouteReference = service.ModelRouteReference

type groupModelRouteAccountKey struct {
	routeAlias string
	accountID  int64
}

func rescaleAccountModelRouteAllocations(ctx context.Context, exec sqlExecutor, accountID int64, oldConcurrency, newConcurrency int) error {
	rows, err := exec.QueryContext(ctx, "SELECT group_id, route_alias, max_concurrency FROM group_model_route_accounts WHERE account_id = ? ORDER BY group_id, route_alias", accountID)
	if err != nil {
		return err
	}
	type allocationRow struct {
		groupID    int64
		routeAlias string
		value      *int
	}
	items := make([]allocationRow, 0)
	values := make([]*int, 0)
	for rows.Next() {
		var item allocationRow
		if err := rows.Scan(&item.groupID, &item.routeAlias, &item.value); err != nil {
			_ = rows.Close()
			return err
		}
		items = append(items, item)
		values = append(values, item.value)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	updated := service.ScaleModelRouteConcurrencyAllocations(oldConcurrency, newConcurrency, values)
	for i, item := range items {
		if item.value == nil || updated[i] == nil || (item.value != nil && *item.value == *updated[i]) {
			continue
		}
		if _, err := exec.ExecContext(ctx, "UPDATE group_model_route_accounts SET max_concurrency = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE group_id = ? AND route_alias = ? AND account_id = ?", *updated[i], item.groupID, item.routeAlias, accountID); err != nil {
			return err
		}
	}
	return rescaleAccountModelRouteConcurrencySchedules(ctx, exec, accountID, oldConcurrency, newConcurrency)
}

// rescaleAccountModelRouteConcurrencySchedules keeps explicit daily schedule
// values proportional to the account's finite concurrency. NULL values mean
// unlimited and are intentionally left unchanged. Redis is not touched here;
// the minute refresh task publishes the committed database values later.
func rescaleAccountModelRouteConcurrencySchedules(ctx context.Context, exec sqlExecutor, accountID int64, oldConcurrency, newConcurrency int) error {
	if oldConcurrency <= 0 || newConcurrency <= 0 {
		return nil
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT id, max_concurrency
		FROM group_model_route_account_concurrency_schedules
		WHERE account_id = ? AND max_concurrency IS NOT NULL
		ORDER BY id`, accountID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type scheduleValue struct {
		id    int64
		value *int
	}
	items := make([]scheduleValue, 0)
	for rows.Next() {
		var item scheduleValue
		if err := rows.Scan(&item.id, &item.value); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		updated := service.ScaleModelRouteConcurrencyValue(oldConcurrency, newConcurrency, item.value)
		if updated == nil || item.value == nil || *updated == *item.value {
			continue
		}
		if _, err := exec.ExecContext(ctx, `
			UPDATE group_model_route_account_concurrency_schedules
			SET max_concurrency = ?, updated_at = CURRENT_TIMESTAMP(6)
			WHERE id = ? AND account_id = ?`, *updated, item.id, accountID); err != nil {
			return err
		}
	}
	return nil
}

// syncGroupModelRouteAccounts rebuilds one group's normalized routing projection.
// Existing concurrency settings are preserved for unchanged relations.
func syncGroupModelRouteAccounts(ctx context.Context, exec sqlExecutor, groupID int64, raw domain.ModelRoutingJSON) error {
	config, err := domain.ParseModelRoutingConfig(raw.RawMessage())
	if err != nil {
		return fmt.Errorf("parse model routing for relation sync: %w", err)
	}

	existing := make(map[groupModelRouteAccountKey]*int)
	rows, err := exec.QueryContext(ctx, "SELECT route_alias, account_id, max_concurrency FROM group_model_route_accounts WHERE group_id = ?", groupID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var key groupModelRouteAccountKey
		var limit *int
		if err := rows.Scan(&key.routeAlias, &key.accountID, &limit); err != nil {
			_ = rows.Close()
			return err
		}
		existing[key] = limit
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := exec.ExecContext(ctx, "DELETE FROM group_model_route_accounts WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for routeAlias, candidates := range config {
		for _, candidate := range candidates {
			for _, accountID := range candidate.AccountIDs {
				key := groupModelRouteAccountKey{routeAlias: routeAlias, accountID: accountID}
				if _, err := exec.ExecContext(ctx, `INSERT INTO group_model_route_accounts (group_id, route_alias, account_id, max_concurrency) VALUES (?, ?, ?, ?)`, groupID, routeAlias, accountID, existing[key]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *groupRepository) RebuildGroupModelRouteAccounts(ctx context.Context, groupID *int64) (any, error) {
	result := &GroupModelRouteRebuildResult{}
	if groupID != nil {
		g, err := r.client.Group.Query().Where(group.IDEQ(*groupID)).Only(ctx)
		if err != nil {
			return nil, err
		}
		if err := syncGroupModelRouteAccounts(ctx, r.sql, g.ID, g.ModelRouting); err != nil {
			return nil, err
		}
		result.GroupsProcessed = 1
		return result, nil
	}

	groups, err := r.client.Group.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if err := syncGroupModelRouteAccounts(ctx, r.sql, g.ID, g.ModelRouting); err != nil {
			return nil, err
		}
		result.GroupsProcessed++
	}
	return result, nil
}

func (r *groupRepository) ListGroupModelRouteReferences(ctx context.Context, accountID int64) (any, error) {
	rows, err := r.sql.QueryContext(ctx, "SELECT r.group_id, g.name, r.route_alias, r.account_id, r.max_concurrency, a.concurrency, COALESCE((SELECT SUM(r2.max_concurrency) FROM group_model_route_accounts r2 WHERE r2.account_id = r.account_id AND r2.max_concurrency IS NOT NULL), 0) FROM group_model_route_accounts r JOIN `groups` AS g ON g.id = r.group_id JOIN accounts AS a ON a.id = r.account_id WHERE r.account_id = ? AND g.deleted_at IS NULL ORDER BY g.name, r.route_alias", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GroupModelRouteReference, 0)
	for rows.Next() {
		var item GroupModelRouteReference
		if err := rows.Scan(&item.GroupID, &item.GroupName, &item.RouteAlias, &item.AccountID, &item.MaxConcurrency, &item.AccountConcurrency, &item.AllocatedConcurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *groupRepository) ListGroupModelRouteReferencesByGroup(ctx context.Context, groupID int64) (any, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT r.group_id, '' AS group_name, r.route_alias, r.account_id, r.max_concurrency, a.concurrency, COALESCE((SELECT SUM(r2.max_concurrency) FROM group_model_route_accounts r2 WHERE r2.account_id = r.account_id AND r2.group_id <> r.group_id AND r2.max_concurrency IS NOT NULL), 0) FROM group_model_route_accounts r JOIN accounts AS a ON a.id = r.account_id WHERE r.group_id = ? ORDER BY r.route_alias, r.account_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GroupModelRouteReference, 0)
	for rows.Next() {
		var item GroupModelRouteReference
		if err := rows.Scan(&item.GroupID, &item.GroupName, &item.RouteAlias, &item.AccountID, &item.MaxConcurrency, &item.AccountConcurrency, &item.AllocatedConcurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *groupRepository) UpdateGroupModelRouteConcurrency(ctx context.Context, groupID int64, routeAlias string, accountID int64, maxConcurrency *int) error {
	return r.UpdateGroupModelRouteConcurrencyBatch(ctx, groupID, []service.ModelRouteConcurrencyUpdate{{RouteAlias: routeAlias, AccountID: accountID, MaxConcurrency: maxConcurrency}})
}

func (r *groupRepository) UpdateGroupModelRouteConcurrencyBatch(ctx context.Context, groupID int64, updates []service.ModelRouteConcurrencyUpdate) error {
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
	if err := updateGroupModelRouteConcurrencyOnExec(ctx, exec, groupID, updates); err != nil {
		return err
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}

func updateGroupModelRouteConcurrencyOnExec(ctx context.Context, exec sqlExecutor, groupID int64, updates []service.ModelRouteConcurrencyUpdate) error {
	if len(updates) == 0 {
		return fmt.Errorf("concurrency updates are required")
	}
	byAccount := make(map[int64][]service.ModelRouteConcurrencyUpdate)
	for _, update := range updates {
		if update.MaxConcurrency != nil && *update.MaxConcurrency <= 0 {
			return infraerrors.BadRequest("INVALID_MODEL_ROUTE_CONCURRENCY", "候选最大并发数必须为正整数或留空")
		}
		byAccount[update.AccountID] = append(byAccount[update.AccountID], update)
	}
	for accountID, accountUpdates := range byAccount {
		accountName, accountConcurrency, err := queryAccountNameAndConcurrency(ctx, exec, accountID)
		if err != nil {
			return err
		}
		current, err := querySingleInt(ctx, exec, "SELECT COALESCE(SUM(max_concurrency), 0) FROM group_model_route_accounts WHERE account_id = ?", accountID)
		if err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(accountUpdates))
		for _, update := range accountUpdates {
			if _, ok := seen[update.RouteAlias]; ok {
				return fmt.Errorf("duplicate route alias in concurrency updates: %s", update.RouteAlias)
			}
			seen[update.RouteAlias] = struct{}{}
			old, err := querySingleInt(ctx, exec, "SELECT COALESCE(max_concurrency, 0) FROM group_model_route_accounts WHERE group_id = ? AND route_alias = ? AND account_id = ?", groupID, update.RouteAlias, accountID)
			if err != nil {
				return err
			}
			current -= old
			if update.MaxConcurrency != nil {
				current += *update.MaxConcurrency
			}
		}
		if accountConcurrency > 0 && current > accountConcurrency {
			return infraerrors.BadRequest(
				"MODEL_ROUTE_CONCURRENCY_EXCEEDED",
				fmt.Sprintf("账号「%s」的候选并发分配合计为 %d，超过账号总并发 %d", accountName, current, accountConcurrency),
			)
		}
		for _, update := range accountUpdates {
			result, err := exec.ExecContext(ctx, "UPDATE group_model_route_accounts SET max_concurrency = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE group_id = ? AND route_alias = ? AND account_id = ?", update.MaxConcurrency, groupID, update.RouteAlias, accountID)
			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if count == 0 {
				return fmt.Errorf("model-route account relation not found")
			}
		}
	}
	return nil
}

func queryAccountNameAndConcurrency(ctx context.Context, exec sqlExecutor, accountID int64) (string, int, error) {
	rows, err := exec.QueryContext(ctx, "SELECT name, concurrency FROM accounts WHERE id = ?", accountID)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", 0, fmt.Errorf("account %d not found", accountID)
	}
	var name string
	var concurrency int
	if err := rows.Scan(&name, &concurrency); err != nil {
		return "", 0, err
	}
	return name, concurrency, rows.Err()
}

func querySingleInt(ctx context.Context, exec sqlExecutor, query string, args ...any) (int, error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, fmt.Errorf("query returned no rows")
	}
	var value int
	if err := rows.Scan(&value); err != nil {
		return 0, err
	}
	return value, rows.Err()
}

func (r *groupRepository) GetGroupModelRouteConcurrency(ctx context.Context, groupID int64, routeAlias string, accountID int64) (*int, error) {
	var value *int
	rows, err := r.sql.QueryContext(ctx, "SELECT max_concurrency FROM group_model_route_accounts WHERE group_id = ? AND route_alias = ? AND account_id = ?", groupID, routeAlias, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	err = rows.Scan(&value)
	return value, err
}
