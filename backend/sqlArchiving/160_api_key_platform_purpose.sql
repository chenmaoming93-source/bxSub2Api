-- Extend API keys with optional external-platform ownership and an explicit purpose.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS platform VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS purpose VARCHAR(20) NOT NULL DEFAULT 'user_created';
