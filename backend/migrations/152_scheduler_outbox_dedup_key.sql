ALTER TABLE scheduler_outbox
    ADD COLUMN dedup_key TEXT;
