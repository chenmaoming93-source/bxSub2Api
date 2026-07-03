-- Add per-group Requests-Per-Minute limit.
ALTER TABLE `groups` ADD COLUMN rpm_limit INT NOT NULL DEFAULT 0;
