-- Add group-level SingGuard request security configuration and audit records.

ALTER TABLE `groups`
    ADD COLUMN `security_check_config` JSON NULL;

UPDATE `groups`
SET `security_check_config` = '{"enabled":false,"rules":[],"timeout_ms":500,"exception_action":"allow","collect_enabled":false,"sample_rate":10,"version":1}'
WHERE `security_check_config` IS NULL;

ALTER TABLE `groups`
    MODIFY COLUMN `security_check_config` JSON NOT NULL;

CREATE TABLE IF NOT EXISTS `security_check_logs` (
    id                          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id                    VARCHAR(64) NOT NULL,
    request_id                  VARCHAR(64),
    client_request_id           VARCHAR(64),
    user_id                     BIGINT,
    api_key_id                  BIGINT,
    api_key_name                VARCHAR(100),
    group_id                    BIGINT,
    group_name                  VARCHAR(100),
    model                       VARCHAR(100),
    provider                    VARCHAR(50),
    protocol                    VARCHAR(32),
    endpoint                    VARCHAR(255),
    config_version              BIGINT NOT NULL DEFAULT 1,
    rules_snapshot              JSON NOT NULL,
    request_body                MEDIUMBLOB,
    request_body_original_bytes BIGINT NOT NULL DEFAULT 0,
    request_body_stored_bytes   BIGINT NOT NULL DEFAULT 0,
    request_body_truncated      BOOLEAN NOT NULL DEFAULT FALSE,
    singguard_response          MEDIUMTEXT,
    check_status                VARCHAR(16) NOT NULL,
    decision                    VARCHAR(16) NOT NULL,
    is_unsafe                   BOOLEAN NOT NULL DEFAULT FALSE,
    triggered_rules             JSON,
    latency_ms                  INT,
    singguard_latency_ms        INT,
    queue_delay_ms              INT,
    exception_type              VARCHAR(32),
    exception_message           TEXT,
    created_at                  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_security_check_logs_event_id (event_id),
    KEY idx_security_check_logs_created_at (created_at),
    KEY idx_security_check_logs_group_created (group_id, created_at),
    KEY idx_security_check_logs_decision_created (decision, created_at)
);
