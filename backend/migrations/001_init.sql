-- Sub2API initial schema for MySQL/GoldenDB.

CREATE TABLE IF NOT EXISTS proxies (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    protocol        VARCHAR(20) NOT NULL,
    host            VARCHAR(255) NOT NULL,
    port            INT NOT NULL,
    username        VARCHAR(100),
    password        VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    deleted_at      DATETIME(6)
);

CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_deleted_at ON proxies(deleted_at);

CREATE TABLE IF NOT EXISTS groups (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT,
    rate_multiplier DECIMAL(10, 4) NOT NULL DEFAULT 1.0,
    is_exclusive    BOOLEAN NOT NULL DEFAULT FALSE,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    deleted_at      DATETIME(6)
);

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_status ON groups(status);
CREATE INDEX IF NOT EXISTS idx_groups_is_exclusive ON groups(is_exclusive);
CREATE INDEX IF NOT EXISTS idx_groups_deleted_at ON groups(deleted_at);

CREATE TABLE IF NOT EXISTS users (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'user',
    balance         DECIMAL(20, 8) NOT NULL DEFAULT 0,
    concurrency     INT NOT NULL DEFAULT 5,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    allowed_groups  JSON,
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    deleted_at      DATETIME(6)
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

CREATE TABLE IF NOT EXISTS accounts (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    platform        VARCHAR(50) NOT NULL,
    type            VARCHAR(20) NOT NULL,
    credentials     JSON,
    extra           JSON,
    proxy_id        BIGINT REFERENCES proxies(id) ON DELETE SET NULL,
    concurrency     INT NOT NULL DEFAULT 3,
    priority        INT NOT NULL DEFAULT 50,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    error_message   TEXT,
    last_used_at    DATETIME(6),
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    deleted_at      DATETIME(6)
);

CREATE INDEX IF NOT EXISTS idx_accounts_platform ON accounts(platform);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_proxy_id ON accounts(proxy_id);
CREATE INDEX IF NOT EXISTS idx_accounts_priority ON accounts(priority);
CREATE INDEX IF NOT EXISTS idx_accounts_last_used_at ON accounts(last_used_at);
CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at);

CREATE TABLE IF NOT EXISTS api_keys (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    `key`           VARCHAR(64) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    group_id        BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    deleted_at      DATETIME(6)
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(`key`);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_group_id ON api_keys(group_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys(deleted_at);

CREATE TABLE IF NOT EXISTS account_groups (
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    group_id        BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    priority        INT NOT NULL DEFAULT 50,
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (account_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_account_groups_group_id ON account_groups(group_id);
CREATE INDEX IF NOT EXISTS idx_account_groups_priority ON account_groups(priority);

CREATE TABLE IF NOT EXISTS redeem_codes (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code            VARCHAR(32) NOT NULL UNIQUE,
    type            VARCHAR(20) NOT NULL DEFAULT 'balance',
    value           DECIMAL(20, 8) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'unused',
    used_by         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    used_at         DATETIME(6),
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE INDEX IF NOT EXISTS idx_redeem_codes_code ON redeem_codes(code);
CREATE INDEX IF NOT EXISTS idx_redeem_codes_status ON redeem_codes(status);
CREATE INDEX IF NOT EXISTS idx_redeem_codes_used_by ON redeem_codes(used_by);

CREATE TABLE IF NOT EXISTS usage_logs (
    id                          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id                     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id                  BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    account_id                  BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    request_id                  VARCHAR(64),
    model                       VARCHAR(100) NOT NULL,
    input_tokens                INT NOT NULL DEFAULT 0,
    output_tokens               INT NOT NULL DEFAULT 0,
    cache_creation_tokens       INT NOT NULL DEFAULT 0,
    cache_read_tokens           INT NOT NULL DEFAULT 0,
    cache_creation_5m_tokens    INT NOT NULL DEFAULT 0,
    cache_creation_1h_tokens    INT NOT NULL DEFAULT 0,
    input_cost                  DECIMAL(20, 10) NOT NULL DEFAULT 0,
    output_cost                 DECIMAL(20, 10) NOT NULL DEFAULT 0,
    cache_creation_cost         DECIMAL(20, 10) NOT NULL DEFAULT 0,
    cache_read_cost             DECIMAL(20, 10) NOT NULL DEFAULT 0,
    total_cost                  DECIMAL(20, 10) NOT NULL DEFAULT 0,
    actual_cost                 DECIMAL(20, 10) NOT NULL DEFAULT 0,
    stream                      BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms                 INT,
    created_at                  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_account_id ON usage_logs(account_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model ON usage_logs(model);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_user_created ON usage_logs(user_id, created_at);
