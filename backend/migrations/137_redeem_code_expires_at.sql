-- Add optional expiry time for redeem codes themselves.
-- `validity_days` remains the subscription duration granted after redeeming.

ALTER TABLE redeem_codes
    ADD COLUMN expires_at DATETIME(6);

CREATE INDEX idx_redeem_codes_expires_at
    ON redeem_codes (expires_at);
