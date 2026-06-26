SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE channels ADD COLUMN IF NOT EXISTS model_mapping JSON DEFAULT (JSON_OBJECT());
