-- Ensure settings.key has a unique index for MySQL ON DUPLICATE KEY semantics.
CREATE UNIQUE INDEX settings_key_key ON settings (`key`);
