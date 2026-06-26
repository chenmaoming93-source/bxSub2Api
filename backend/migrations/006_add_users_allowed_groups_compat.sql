-- Compatibility column for legacy allowed group data. GoldenDB stores the old
-- array-shaped value as JSON so 007 can backfill through JSON_TABLE.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allowed_groups JSON;
