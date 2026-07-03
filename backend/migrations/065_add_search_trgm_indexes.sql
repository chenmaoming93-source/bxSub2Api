-- Improve admin fuzzy-search performance on large datasets.
-- PostgreSQL trigram/GIN indexes are not available in GoldenDB/MySQL mode.
-- Keep lightweight prefix/filter helper indexes instead.
CREATE INDEX idx_users_email_search ON users(email);
CREATE INDEX idx_users_username_search ON users(username);
CREATE INDEX idx_accounts_name_search ON accounts(name);
CREATE INDEX idx_api_keys_name_search ON api_keys(name);
