-- Ensure the legacy compatibility column exists before 007 backfills it.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allowed_groups JSON;
