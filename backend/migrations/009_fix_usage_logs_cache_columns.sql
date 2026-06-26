-- Ensure usage_logs cache token columns use the underscored names expected by code.

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS cache_creation_5m_tokens INT NOT NULL DEFAULT 0;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS cache_creation_1h_tokens INT NOT NULL DEFAULT 0;

-- Legacy PostgreSQL column-name backfill is intentionally skipped for GoldenDB.
-- Existing production data should be transformed by the PostgreSQL-to-GoldenDB
-- data migration job before these application migrations run.
