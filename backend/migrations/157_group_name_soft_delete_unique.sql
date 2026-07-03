-- Remove the non-soft-delete-aware unique indexes on groups.name.
--
-- Two indexes exist:
-- 1. The column-level UNIQUE from 001_init.sql (named `name` by MySQL)
-- 2. groups_name_unique_active from 016_soft_delete_partial_unique_indexes.sql
--
-- Neither filters deleted_at IS NULL, so soft-deleted groups block name reuse.
-- Application layer now handles uniqueness via ExistsByName(DeletedAtIsNil)
-- wrapped in a mutex-protected CreateGroup.

DROP INDEX `name` ON `groups`;
DROP INDEX groups_name_unique_active ON `groups`;
