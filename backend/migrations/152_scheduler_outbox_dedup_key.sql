ALTER TABLE scheduler_outbox
    ADD COLUMN dedup_key VARCHAR(128) NULL;
