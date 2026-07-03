ALTER TABLE auth_identity_migration_reports
    ADD COLUMN resolved_at DATETIME(6) NULL;

ALTER TABLE auth_identity_migration_reports
    ADD COLUMN resolved_by_user_id BIGINT NULL;

ALTER TABLE auth_identity_migration_reports
    ADD COLUMN resolution_note TEXT NULL;

CREATE INDEX idx_auth_identity_migration_reports_resolved_at
    ON auth_identity_migration_reports (resolved_at);
