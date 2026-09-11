package tokenstat

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

func TestCurrentDirtyIdentityReaderScansOnlyCurrentSet(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	location := time.FixedZone("UTC+8", 8*60*60)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	mini.SetTime(at)
	period := domain.NaturalPeriods(at, location)[0]
	values := map[domain.DimensionCode]domain.DimensionValue{domain.DimensionUserID: domain.Int64Value(42), domain.DimensionGroupID: domain.Int64Value(7)}
	identity, err := domain.BuildDimensionIdentity([]domain.DimensionCode{domain.DimensionUserID, domain.DimensionGroupID}, values)
	require.NoError(t, err)
	require.NoError(t, NewRedisAccumulator(client, 16, 7).Add(context.Background(), []domain.AccountingOperation{{
		Period: period, ProjectionID: 5, DimensionHash: identity.Hash,
		DimensionValues: map[string]any{"user_id": int64(42), "group_id": int64(7)}, MetricCode: domain.MetricTotalTokens, Delta: 10,
	}}))
	require.NoError(t, client.SAdd(context.Background(), dynamicDirtyKey, "malformed").Err())

	var found []domain.StatisticIdentity
	err = NewCurrentDirtyIdentityReader(client).ScanCurrent(context.Background(), func(identity domain.StatisticIdentity) error {
		found = append(found, identity)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, int64(5), found[0].ProjectionID)
	require.Equal(t, domain.Int64Value(42), found[0].DimensionValues[domain.DimensionUserID])
	require.True(t, mini.Exists(dynamicDirtyKey))
	for _, key := range mini.Keys() {
		require.NotContains(t, key, "processing:")
		require.NotEqual(t, dynamicSyncLockKey, key)
	}
}

func TestRepositoryListsAggregateIdentitiesWithWhitelistedFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repository := NewRepository(db)
	period := domain.Period{Type: domain.PeriodDay, Start: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)}
	hash := [16]byte{1, 2, 3}
	queryPattern := regexp.QuoteMeta("SELECT period_type, period_start, period_end, projection_id, dimension_hash, dimension_values, metric_code\nFROM token_stat_aggregates\nWHERE period_type = ? AND period_start = ? AND metric_code = ? AND projection_id IN (?,?) AND user_id = ? AND group_id = ? ORDER BY projection_id ASC, dimension_hash ASC LIMIT ?")
	rows := sqlmock.NewRows([]string{"period_type", "period_start", "period_end", "projection_id", "dimension_hash", "dimension_values", "metric_code"}).
		AddRow("D", period.Start, period.End, int64(3), hash[:], []byte(`{"user_id":42,"group_id":7}`), "total_tokens")
	mock.ExpectQuery(queryPattern).
		WithArgs(asDriverValues("D", period.Start, "total_tokens", int64(3), int64(5), int64(42), int64(7), 10)...).
		WillReturnRows(rows)

	identities, next, err := repository.ListAggregateIdentities(context.Background(), domain.AggregateIdentityQuery{
		Period: period, ProjectionIDs: []int64{5, 3}, MetricCode: domain.MetricTotalTokens,
		Filters: map[domain.DimensionCode]domain.DimensionValue{domain.DimensionGroupID: domain.Int64Value(7), domain.DimensionUserID: domain.Int64Value(42)}, Limit: 10,
	})
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, identities, 1)
	require.Equal(t, int64(3), identities[0].ProjectionID)
	require.Equal(t, hash, identities[0].DimensionHash)
	require.Equal(t, domain.Int64Value(42), identities[0].DimensionValues[domain.DimensionUserID])
	require.NoError(t, mock.ExpectationsWereMet())
}

func asDriverValues(values ...any) []driver.Value {
	result := make([]driver.Value, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
