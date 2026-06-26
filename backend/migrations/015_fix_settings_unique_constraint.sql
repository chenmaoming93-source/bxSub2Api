-- Ensure settings.key has a unique index for MySQL ON DUPLICATE KEY semantics.
CREATE UNIQUE INDEX IF NOT EXISTS settings_key_key ON settings (`key`);
