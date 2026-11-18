//go:build unit

package growth

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisNonceCacheTake(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisNonceCache(client, &config.Config{
		Growth: config.GrowthConfig{Crypto: config.GrowthCryptoConfig{TimeWindow: "5m"}},
	})
	accepted, err := cache.Take(context.Background(), "nonce-1")
	require.NoError(t, err)
	require.True(t, accepted)

	accepted, err = cache.Take(context.Background(), "nonce-1")
	require.NoError(t, err)
	require.False(t, accepted)

	server.FastForward(6 * time.Minute)
	accepted, err = cache.Take(context.Background(), "nonce-1")
	require.NoError(t, err)
	require.True(t, accepted)
}
