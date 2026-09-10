package tokenstat

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

type Aggregate struct {
	PeriodType      string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	ProjectionID    int64
	DimensionHash   [16]byte
	DimensionValues map[string]any
	MetricCode      string
	MetricValue     int64
	SourceVersion   int64
	UserID          *int64
	APIKeyID        *int64
	GroupID         *int64
	RouteAlias      *string
	AccountID       *int64
	UpstreamModel   *string
	Department      *string
	LastSyncedAt    time.Time
}

func (r *Repository) SetPeriodState(ctx context.Context, period domain.Period, state, lastError string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO token_stat_period_states
    (period_type, period_start, period_end, state, final_sync_version, last_error, created_at, updated_at)
VALUES (?, ?, ?, ?, 0, NULLIF(?, ''), NOW(6), NOW(6))
AS new
ON DUPLICATE KEY UPDATE
    period_end = new.period_end, state = new.state, last_error = new.last_error,
    closed_at = IF(new.state = 'CLOSING', NOW(6), closed_at),
    persisted_at = IF(new.state = 'PERSISTED', NOW(6), persisted_at),
    deleted_at = IF(new.state = 'DELETED', NOW(6), deleted_at),
    updated_at = NOW(6)`,
		period.Type, period.Start, period.End, state, lastError,
	)
	return err
}

func (r *Repository) VerifyPeriodVersions(ctx context.Context, period domain.Period, redisVersions map[string]int64) error {
	for identity, redisVersion := range redisVersions {
		parts := strings.Split(identity, "|")
		if len(parts) != 2 {
			return fmt.Errorf("invalid redis version identity")
		}
		keyParts := strings.Split(parts[0], ":")
		if len(keyParts) < 3 {
			return fmt.Errorf("invalid redis version key")
		}
		projectionID, err := strconv.ParseInt(keyParts[len(keyParts)-2], 10, 64)
		if err != nil {
			return err
		}
		fieldParts := strings.SplitN(parts[1], ":", 2)
		if len(fieldParts) != 2 {
			return fmt.Errorf("invalid redis version field")
		}
		hash, err := hex.DecodeString(fieldParts[0])
		if err != nil {
			return err
		}
		var mysqlVersion int64
		err = r.db.QueryRowContext(ctx, `
SELECT source_version FROM token_stat_aggregates
WHERE period_type = ? AND period_start = ? AND projection_id = ?
  AND dimension_hash = ? AND metric_code = ?`,
			period.Type, period.Start, projectionID, hash, fieldParts[1],
		).Scan(&mysqlVersion)
		if err != nil {
			return err
		}
		if mysqlVersion != redisVersion {
			return fmt.Errorf("period version mismatch: redis=%d mysql=%d", redisVersion, mysqlVersion)
		}
	}
	return nil
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UpsertAggregate only applies a snapshot when its source version is newer.
// Replaying the same version and delivering an older version are both no-ops.
func (r *Repository) UpsertAggregate(ctx context.Context, aggregate Aggregate) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("tokenstat repository database is required")
	}
	values, err := json.Marshal(aggregate.DimensionValues)
	if err != nil {
		return fmt.Errorf("marshal dimension values: %w", err)
	}
	_, err = r.db.ExecContext(ctx, upsertAggregateSQL,
		aggregate.PeriodType, aggregate.PeriodStart, aggregate.PeriodEnd,
		aggregate.ProjectionID, aggregate.DimensionHash[:], values,
		aggregate.MetricCode, aggregate.MetricValue, aggregate.SourceVersion,
		aggregate.UserID, aggregate.APIKeyID, aggregate.GroupID, aggregate.RouteAlias,
		aggregate.AccountID, aggregate.UpstreamModel, aggregate.Department, aggregate.LastSyncedAt,
		aggregate.LastSyncedAt, aggregate.LastSyncedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert token statistic aggregate: %w", err)
	}
	return nil
}

const upsertAggregateSQL = `
INSERT INTO token_stat_aggregates (
    period_type, period_start, period_end, projection_id, dimension_hash,
    dimension_values, metric_code, metric_value, source_version,
    user_id, api_key_id, group_id, route_alias, account_id, upstream_model, department,
    last_synced_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, CAST(? AS JSON), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
AS new
ON DUPLICATE KEY UPDATE
    period_end = IF(new.source_version > token_stat_aggregates.source_version, new.period_end, token_stat_aggregates.period_end),
    dimension_values = IF(new.source_version > token_stat_aggregates.source_version, new.dimension_values, token_stat_aggregates.dimension_values),
    metric_value = IF(new.source_version > token_stat_aggregates.source_version, new.metric_value, token_stat_aggregates.metric_value),
    user_id = IF(new.source_version > token_stat_aggregates.source_version, new.user_id, token_stat_aggregates.user_id),
    api_key_id = IF(new.source_version > token_stat_aggregates.source_version, new.api_key_id, token_stat_aggregates.api_key_id),
    group_id = IF(new.source_version > token_stat_aggregates.source_version, new.group_id, token_stat_aggregates.group_id),
    route_alias = IF(new.source_version > token_stat_aggregates.source_version, new.route_alias, token_stat_aggregates.route_alias),
    account_id = IF(new.source_version > token_stat_aggregates.source_version, new.account_id, token_stat_aggregates.account_id),
    upstream_model = IF(new.source_version > token_stat_aggregates.source_version, new.upstream_model, token_stat_aggregates.upstream_model),
    department = IF(new.source_version > token_stat_aggregates.source_version, new.department, token_stat_aggregates.department),
    last_synced_at = IF(new.source_version > token_stat_aggregates.source_version, new.last_synced_at, token_stat_aggregates.last_synced_at),
    updated_at = IF(new.source_version > token_stat_aggregates.source_version, new.updated_at, token_stat_aggregates.updated_at),
    source_version = GREATEST(token_stat_aggregates.source_version, new.source_version)`
