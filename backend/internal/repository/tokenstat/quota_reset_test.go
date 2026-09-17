package tokenstat

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

func TestDynamicQuotaResetKeyMatchesCountKeySuffix(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	periods := domain.NaturalPeriods(time.Date(2026, 9, 9, 12, 0, 0, 0, location), location)
	var hash [16]byte
	hash[0], hash[1] = 3, 7
	shard := RedisShard(hash, 16)

	for _, period := range periods {
		rawKey := DynamicCountKey(period, 12, shard)
		resetKey := DynamicQuotaResetKey(period, 12, shard)
		derived, deriveErr := DynamicQuotaResetKeyFromCountKey(rawKey)
		require.NoError(t, deriveErr)
		require.Equal(t, resetKey, derived)
		require.Equal(t, rawKey[len(dynamicCountPrefix):], resetKey[len(dynamicQuotaResetPrefix):])
	}
}

func TestRedisQuotaResetStoreSnapshotAndEffectiveUsage(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	store := NewRedisQuotaResetStore(client, 16, 7)
	reader := NewQuotaReader(client)
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	mini.SetTime(at)
	periods := domain.NaturalPeriods(at, location)
	var hash [16]byte
	hash[0], hash[1] = 3, 7
	field := RedisField(hash, domain.MetricTotalTokens)

	for _, period := range periods {
		identity := QuotaResetIdentity{
			Period: period, ProjectionID: 12, DimensionHash: hash, MetricCode: domain.MetricTotalTokens,
		}
		shard := RedisShard(hash, 16)
		rawKey := DynamicCountKey(period, identity.ProjectionID, shard)
		resetKey := DynamicQuotaResetKey(period, identity.ProjectionID, shard)
		versionKey := dynamicVersionPrefix + rawKey[len(dynamicCountPrefix):]
		dirtyMember := "unchanged-" + string(period.Type)
		require.NoError(t, client.HSet(ctx, rawKey, field, 100_000).Err())
		require.NoError(t, client.HSet(ctx, versionKey, field, 9).Err())
		require.NoError(t, client.SAdd(ctx, dynamicDirtyKey, dirtyMember).Err())

		used, readErr := reader.Read(ctx, rawKey, field)
		require.NoError(t, readErr)
		require.Equal(t, int64(100_000), used)

		status, snapshotErr := store.Snapshot(ctx, identity)
		require.NoError(t, snapshotErr)
		require.Equal(t, QuotaResetSnapshotReset, status)
		require.Equal(t, "100000", mini.HGet(resetKey, field))
		baselines, baselineErr := store.ReadBaselines(ctx, []domain.StatisticIdentity{{Period: period, ProjectionID: 12, DimensionHash: hash, MetricCode: domain.MetricTotalTokens}})
		require.NoError(t, baselineErr)
		require.Equal(t, []int64{100_000}, baselines)
		require.Equal(t, "100000", mini.HGet(rawKey, field))
		require.Equal(t, "9", mini.HGet(versionKey, field))
		dirtyUnchanged, dirtyErr := mini.SIsMember(dynamicDirtyKey, dirtyMember)
		require.NoError(t, dirtyErr)
		require.True(t, dirtyUnchanged)
		require.Equal(t, period.End.Add(7*24*time.Hour).Sub(at), mini.TTL(resetKey))

		require.NoError(t, client.HSet(ctx, rawKey, field, 120_000).Err())
		used, readErr = reader.Read(ctx, rawKey, field)
		require.NoError(t, readErr)
		require.Equal(t, int64(20_000), used)

		status, snapshotErr = store.Snapshot(ctx, identity)
		require.NoError(t, snapshotErr)
		require.Equal(t, QuotaResetSnapshotReset, status)
		require.Equal(t, "120000", mini.HGet(resetKey, field), "a later reset must replace the baseline")
	}
}

func TestRedisQuotaResetStoreSnapshotBatchKeepsIndependentResults(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	location := time.FixedZone("UTC+8", 8*60*60)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	mini.SetTime(at)
	period := domain.NaturalPeriods(at, location)[0]
	identities := []domain.StatisticIdentity{
		{Period: period, ProjectionID: 1, DimensionHash: [16]byte{1}, MetricCode: domain.MetricTotalTokens},
		{Period: period, ProjectionID: 2, DimensionHash: [16]byte{2}, MetricCode: domain.MetricTotalTokens},
		{Period: period, ProjectionID: 3, DimensionHash: [16]byte{3}, MetricCode: domain.MetricTotalTokens},
	}
	for _, identity := range identities[:2] {
		shard := RedisShard(identity.DimensionHash, 16)
		require.NoError(t, client.HSet(context.Background(), DynamicCountKey(period, identity.ProjectionID, shard), RedisField(identity.DimensionHash, identity.MetricCode), 99).Err())
	}
	bad := identities[1]
	require.NoError(t, client.Set(context.Background(), DynamicQuotaResetKey(period, bad.ProjectionID, RedisShard(bad.DimensionHash, 16)), "wrong-type", 0).Err())

	results, err := NewRedisQuotaResetStore(client, 16, 7).SnapshotBatch(context.Background(), identities)
	require.Error(t, err, "pipeline reports the per-command WRONGTYPE error")
	require.Len(t, results, 3)
	require.Equal(t, domain.QuotaResetEntryReset, results[0].Status)
	require.Equal(t, domain.QuotaResetEntryFailed, results[1].Status)
	require.Error(t, results[1].Err)
	require.Equal(t, domain.QuotaResetEntryNoUsage, results[2].Status)
	first := identities[0]
	value, readErr := client.HGet(context.Background(), DynamicQuotaResetKey(period, first.ProjectionID, RedisShard(first.DimensionHash, 16)), RedisField(first.DimensionHash, first.MetricCode)).Int64()
	require.NoError(t, readErr)
	require.Equal(t, int64(99), value, "a sibling error must not roll back successful snapshots")
}

func TestRedisQuotaResetNoUsageAndEffectiveUsageClamp(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	store := NewRedisQuotaResetStore(client, 16, 7)
	reader := NewQuotaReader(client)
	location := time.FixedZone("UTC+8", 8*60*60)
	period := domain.NaturalPeriods(time.Date(2026, 9, 9, 12, 0, 0, 0, location), location)[0]
	identity := QuotaResetIdentity{Period: period, ProjectionID: 12, MetricCode: domain.MetricTotalTokens}
	field := RedisField(identity.DimensionHash, identity.MetricCode)
	rawKey := DynamicCountKey(period, identity.ProjectionID, RedisShard(identity.DimensionHash, 16))
	resetKey := DynamicQuotaResetKey(period, identity.ProjectionID, RedisShard(identity.DimensionHash, 16))

	status, err := store.Snapshot(ctx, identity)
	require.NoError(t, err)
	require.Equal(t, QuotaResetSnapshotNoUsage, status)
	require.False(t, mini.Exists(resetKey))

	require.NoError(t, client.HSet(ctx, rawKey, field, 100).Err())
	require.NoError(t, client.HSet(ctx, resetKey, field, 120).Err())
	used, err := reader.Read(ctx, rawKey, field)
	require.NoError(t, err)
	require.Zero(t, used)
}

func TestQuotaResetBaselineSurvivesServiceReconstructionAndLossFallsBackToRaw(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	location := time.FixedZone("UTC+8", 8*60*60)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, location)
	mini.SetTime(at)
	period := domain.NaturalPeriods(at, location)[0]
	identity := QuotaResetIdentity{Period: period, ProjectionID: 8, DimensionHash: [16]byte{8}, MetricCode: domain.MetricTotalTokens}
	shard := RedisShard(identity.DimensionHash, 16)
	rawKey := DynamicCountKey(period, identity.ProjectionID, shard)
	field := RedisField(identity.DimensionHash, identity.MetricCode)
	require.NoError(t, client.HSet(context.Background(), rawKey, field, 120).Err())
	status, err := NewRedisQuotaResetStore(client, 16, 7).Snapshot(context.Background(), identity)
	require.NoError(t, err)
	require.Equal(t, QuotaResetSnapshotReset, status)

	// Reconstructing application services does not affect the Redis baseline.
	restartedReader := NewQuotaReader(client)
	used, err := restartedReader.Read(context.Background(), rawKey, field)
	require.NoError(t, err)
	require.Zero(t, used)
	require.NoError(t, client.HIncrBy(context.Background(), rawKey, field, 20).Err())
	used, err = NewQuotaReader(client).Read(context.Background(), rawKey, field)
	require.NoError(t, err)
	require.Equal(t, int64(20), used)

	// Redis snapshot loss never changes raw/MySQL truth; absence restores legacy raw usage semantics.
	require.NoError(t, client.Del(context.Background(), DynamicQuotaResetKey(period, identity.ProjectionID, shard)).Err())
	used, err = NewQuotaReader(client).Read(context.Background(), rawKey, field)
	require.NoError(t, err)
	require.Equal(t, int64(140), used)
	raw, err := client.HGet(context.Background(), rawKey, field).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(140), raw)
}

func TestQuotaReaderReturnsRedisErrors(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	reader := NewQuotaReader(client)
	mini.Close()

	_, err := reader.Read(context.Background(), dynamicCountPrefix+"D:any:1:0", "field")
	require.Error(t, err)
}
