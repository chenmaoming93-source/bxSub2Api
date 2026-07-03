-- Add upstream error events list (JSON) to ops_error_logs for per-request correlation.
--
-- This is intentionally idempotent.

ALTER TABLE ops_error_logs
    ADD COLUMN upstream_errors JSON;
