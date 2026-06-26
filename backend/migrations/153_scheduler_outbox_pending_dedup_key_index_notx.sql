CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduler_outbox_pending_dedup_key
    ON scheduler_outbox (dedup_key);
