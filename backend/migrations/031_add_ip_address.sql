-- Add IP address field to usage_logs table for request tracking (admin-only visibility)
ALTER TABLE usage_logs ADD COLUMN ip_address VARCHAR(45);

-- Create index for IP address queries
CREATE INDEX idx_usage_logs_ip_address ON usage_logs(ip_address);
