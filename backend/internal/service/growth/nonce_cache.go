package growth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/redis/go-redis/v9"
)

type NonceCache interface {
	Take(ctx context.Context, nonce string) (bool, error)
}

type RedisNonceCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewRedisNonceCache(client *redis.Client, cfg *config.Config) *RedisNonceCache {
	ttl := 5 * time.Minute
	if cfg != nil {
		ttl = cfg.Growth.Crypto.WindowDuration()
	}
	return &RedisNonceCache{
		client: client,
		ttl:    ttl,
		prefix: "growth:nonce:",
	}
}

func (c *RedisNonceCache) Take(ctx context.Context, nonce string) (bool, error) {
	if c == nil || c.client == nil {
		return false, fmt.Errorf("growth nonce cache unavailable")
	}
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return false, fmt.Errorf("growth nonce is empty")
	}
	return c.client.SetNX(ctx, c.prefix+nonce, "1", c.ttl).Result()
}
