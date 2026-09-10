-- Optional business-facing scene name for groups.
ALTER TABLE `groups`
    ADD COLUMN scene_name VARCHAR(100) NULL;
