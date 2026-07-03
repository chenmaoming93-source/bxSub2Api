-- Ops error logs: add endpoint, model mapping, and request_type fields
-- to match usage_logs observability coverage.
--
-- All columns are nullable with no default to preserve backward compatibility
-- with existing rows.

-- 1) Standardized endpoint paths (analogous to usage_logs.inbound_endpoint / upstream_endpoint)
ALTER TABLE ops_error_logs
    ADD COLUMN inbound_endpoint VARCHAR(256),
    ADD COLUMN upstream_endpoint VARCHAR(256);

-- 2) Model mapping fields (analogous to usage_logs.requested_model / upstream_model)
ALTER TABLE ops_error_logs
    ADD COLUMN requested_model VARCHAR(100),
    ADD COLUMN upstream_model VARCHAR(100);

-- 3) Granular request type enum (analogous to usage_logs.request_type: 0=unknown, 1=sync, 2=stream, 3=ws_v2)
ALTER TABLE ops_error_logs
    ADD COLUMN request_type SMALLINT;
