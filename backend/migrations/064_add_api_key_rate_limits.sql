-- Add rate limit fields to api_keys table
-- Rate limit configuration (0 = unlimited)
ALTER TABLE api_keys ADD COLUMN rate_limit_5h decimal(20,8) NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN rate_limit_1d decimal(20,8) NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN rate_limit_7d decimal(20,8) NOT NULL DEFAULT 0;

-- Rate limit usage tracking
ALTER TABLE api_keys ADD COLUMN usage_5h decimal(20,8) NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN usage_1d decimal(20,8) NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN usage_7d decimal(20,8) NOT NULL DEFAULT 0;

-- Window start times (nullable)
ALTER TABLE api_keys ADD COLUMN window_5h_start DATETIME(6);
ALTER TABLE api_keys ADD COLUMN window_1d_start DATETIME(6);
ALTER TABLE api_keys ADD COLUMN window_7d_start DATETIME(6);
