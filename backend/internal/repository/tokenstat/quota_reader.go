package tokenstat

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type QuotaReader struct{ redis *redis.Client }

func NewQuotaReader(client *redis.Client) *QuotaReader { return &QuotaReader{redis: client} }

func (r *QuotaReader) Read(ctx context.Context, key, field string) (int64, error) {
	value, err := r.redis.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(value, 10, 64)
}
