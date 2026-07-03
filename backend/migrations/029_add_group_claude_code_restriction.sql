-- Add group-level Claude Code client restrictions.

ALTER TABLE `groups`
ADD COLUMN claude_code_only BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE `groups`
ADD COLUMN fallback_group_id BIGINT REFERENCES `groups`(id) ON DELETE SET NULL;

CREATE INDEX idx_groups_claude_code_only
ON `groups`(claude_code_only);

CREATE INDEX idx_groups_fallback_group_id
ON `groups`(fallback_group_id);