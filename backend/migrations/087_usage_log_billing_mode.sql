-- Add billing_mode to usage_logs (records the billing mode: token/per_request/image)
ALTER TABLE usage_logs ADD COLUMN billing_mode VARCHAR(20);
