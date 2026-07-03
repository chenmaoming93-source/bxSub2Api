-- Add per-user per-group RPM override.
ALTER TABLE user_group_rate_multipliers
    ADD COLUMN rpm_override INT NULL;

ALTER TABLE user_group_rate_multipliers
    MODIFY COLUMN rate_multiplier DECIMAL(10,4) NULL;