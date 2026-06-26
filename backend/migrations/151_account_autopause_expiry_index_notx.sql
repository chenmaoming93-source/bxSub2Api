CREATE INDEX idx_accounts_autopause_expiry_due
    ON accounts (expires_at);
