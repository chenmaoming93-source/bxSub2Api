package tokenstat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type QuotaReader struct{ redis *redis.Client }

func NewQuotaReader(client *redis.Client) *QuotaReader { return &QuotaReader{redis: client} }

// Read returns effective quota usage: max(0, raw usage - reset baseline).
// The reset key is derived from the validated statistics-key namespace so the
// quota checker remains unaware of reset snapshot persistence details.
func (r *QuotaReader) Read(ctx context.Context, key, field string) (int64, error) {
	if r == nil || r.redis == nil {
		return 0, fmt.Errorf("dynamic token quota redis client is required")
	}
	resetKey, err := DynamicQuotaResetKeyFromCountKey(key)
	if err != nil {
		return 0, err
	}
	result, err := effectiveQuotaUsageScript.Run(ctx, r.redis, []string{key, resetKey}, field).Result()
	if err != nil {
		return 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, fmt.Errorf("invalid dynamic token quota usage response")
	}
	raw, err := parseQuotaUsageValue(values[0])
	if err != nil {
		return 0, fmt.Errorf("parse dynamic token raw usage: %w", err)
	}
	baseline, err := parseQuotaUsageValue(values[1])
	if err != nil {
		return 0, fmt.Errorf("parse dynamic token reset baseline: %w", err)
	}
	if raw <= baseline {
		return 0, nil
	}
	return raw - baseline, nil
}

// DynamicQuotaResetKeyFromCountKey preserves the complete period/projection/
// shard suffix while moving a statistics key into the reset namespace.
func DynamicQuotaResetKeyFromCountKey(countKey string) (string, error) {
	if !strings.HasPrefix(countKey, dynamicCountPrefix) {
		return "", fmt.Errorf("invalid dynamic token statistics key")
	}
	return dynamicQuotaResetPrefix + strings.TrimPrefix(countKey, dynamicCountPrefix), nil
}

func parseQuotaUsageValue(value interface{}) (int64, error) {
	text, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected Redis value type %T", value)
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("usage value must not be negative")
	}
	return parsed, nil
}

var effectiveQuotaUsageScript = redis.NewScript(`
local raw = redis.call('HGET', KEYS[1], ARGV[1]) or '0'
local baseline = redis.call('HGET', KEYS[2], ARGV[1]) or '0'
return {raw, baseline}
`)
