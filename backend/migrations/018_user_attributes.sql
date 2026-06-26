-- Add user attribute definitions and values tables for custom user attributes.

-- User Attribute Definitions table (with soft delete support)
CREATE TABLE IF NOT EXISTS user_attribute_definitions (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    key             VARCHAR(100) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT DEFAULT '',
    type            VARCHAR(20) NOT NULL,
    options         JSON DEFAULT (JSON_ARRAY()),
    required        BOOLEAN NOT NULL DEFAULT FALSE,
    validation      JSON DEFAULT (JSON_OBJECT()),
    placeholder     VARCHAR(255) DEFAULT '',
    display_order   INT NOT NULL DEFAULT 0,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      DATETIME(6) NOT NULL DEFAULT NOW(),
    updated_at      DATETIME(6) NOT NULL DEFAULT NOW(),
    deleted_at      DATETIME(6)
);

-- Partial unique index for key (only for non-deleted records)
-- Allows reusing keys after soft delete
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_attribute_definitions_key_unique
    ON user_attribute_definitions(key);

CREATE INDEX IF NOT EXISTS idx_user_attribute_definitions_enabled
    ON user_attribute_definitions(enabled);
CREATE INDEX IF NOT EXISTS idx_user_attribute_definitions_display_order
    ON user_attribute_definitions(display_order);
CREATE INDEX IF NOT EXISTS idx_user_attribute_definitions_deleted_at
    ON user_attribute_definitions(deleted_at);

-- User Attribute Values table (hard delete only, no deleted_at)
CREATE TABLE IF NOT EXISTS user_attribute_values (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attribute_id    BIGINT NOT NULL REFERENCES user_attribute_definitions(id) ON DELETE CASCADE,
    value           TEXT DEFAULT '',
    created_at      DATETIME(6) NOT NULL DEFAULT NOW(),
    updated_at      DATETIME(6) NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, attribute_id)
);

CREATE INDEX IF NOT EXISTS idx_user_attribute_values_user_id
    ON user_attribute_values(user_id);
CREATE INDEX IF NOT EXISTS idx_user_attribute_values_attribute_id
    ON user_attribute_values(attribute_id);
