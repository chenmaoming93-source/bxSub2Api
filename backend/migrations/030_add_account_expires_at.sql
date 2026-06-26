-- Add expires_at for account expiration configuration
ALTER TABLE accounts ADD COLUMN expires_at DATETIME(6);
-- Document expires_at meaning

-- Add auto_pause_on_expired for account expiration scheduling control
ALTER TABLE accounts ADD COLUMN auto_pause_on_expired boolean NOT NULL DEFAULT true;
-- Document auto_pause_on_expired meaning

-- Ensure existing accounts are enabled by default
UPDATE accounts SET auto_pause_on_expired = true;
