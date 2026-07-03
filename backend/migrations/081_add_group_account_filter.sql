ALTER TABLE `groups` ADD COLUMN require_oauth_only BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE `groups` ADD COLUMN require_privacy_set BOOLEAN NOT NULL DEFAULT false;
