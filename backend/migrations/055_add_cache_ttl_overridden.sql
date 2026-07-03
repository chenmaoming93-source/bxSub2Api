-- Add cache_ttl_overridden flag to usage_logs for tracking cache TTL override per account.
ALTER TABLE usage_logs ADD COLUMN cache_ttl_overridden BOOLEAN NOT NULL DEFAULT FALSE;
