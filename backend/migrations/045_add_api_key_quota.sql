-- Migration: Add quota fields to api_keys table
-- This migration adds independent quota and expiration support for API keys

-- Add quota limit field (0 = unlimited)
ALTER TABLE api_keys ADD COLUMN quota DECIMAL(20, 8) NOT NULL DEFAULT 0;

-- Add used quota amount field
ALTER TABLE api_keys ADD COLUMN quota_used DECIMAL(20, 8) NOT NULL DEFAULT 0;

-- Add expiration time field (NULL = never expires)
ALTER TABLE api_keys ADD COLUMN expires_at DATETIME(6);

-- Add indexes for efficient quota queries
CREATE INDEX idx_api_keys_quota_quota_used ON api_keys(quota, quota_used);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- Comment on columns for documentation
