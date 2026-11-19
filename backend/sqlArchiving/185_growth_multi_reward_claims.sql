-- Allow multiple task rewards in one business cycle.
-- Safe to run after both the legacy and latest versions of 182.

SET @growth_reward_sequence_exists = (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'growth_reward_claims'
    AND column_name = 'sequence'
);
SET @growth_reward_sequence_ddl = IF(
  @growth_reward_sequence_exists = 0,
  'ALTER TABLE growth_reward_claims ADD COLUMN sequence INT NOT NULL DEFAULT 1 AFTER cycle_key',
  'DO 0'
);
PREPARE growth_reward_sequence_stmt FROM @growth_reward_sequence_ddl;
EXECUTE growth_reward_sequence_stmt;
DEALLOCATE PREPARE growth_reward_sequence_stmt;

SET @growth_reward_cycle_index = (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'growth_reward_claims'
    AND index_name = 'uq_growth_reward_claims_cycle'
);
SET @growth_reward_old_shape = (
  SELECT IF(
    COUNT(*) = 3
      AND SUM(column_name IN ('user_id', 'task_id', 'cycle_key')) = 3,
    1,
    0
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'growth_reward_claims'
    AND index_name = 'uq_growth_reward_claims_cycle'
);
SET @growth_reward_rebuild_cycle_index_ddl = IF(
  @growth_reward_old_shape = 1,
  'ALTER TABLE growth_reward_claims DROP INDEX uq_growth_reward_claims_cycle, ADD UNIQUE KEY uq_growth_reward_claims_cycle (user_id, task_id, cycle_key, sequence)',
  IF(
    @growth_reward_cycle_index = 0,
    'ALTER TABLE growth_reward_claims ADD UNIQUE KEY uq_growth_reward_claims_cycle (user_id, task_id, cycle_key, sequence)',
    'DO 0'
  )
);
PREPARE growth_reward_rebuild_cycle_index_stmt FROM @growth_reward_rebuild_cycle_index_ddl;
EXECUTE growth_reward_rebuild_cycle_index_stmt;
DEALLOCATE PREPARE growth_reward_rebuild_cycle_index_stmt;

SET @growth_reward_sequence_check = (
  SELECT COUNT(*)
  FROM information_schema.check_constraints
  WHERE constraint_schema = DATABASE()
    AND constraint_name = 'chk_growth_reward_claims_sequence'
);
SET @growth_reward_sequence_check_ddl = IF(
  @growth_reward_sequence_check = 0,
  'ALTER TABLE growth_reward_claims ADD CONSTRAINT chk_growth_reward_claims_sequence CHECK (sequence > 0)',
  'DO 0'
);
PREPARE growth_reward_sequence_check_stmt FROM @growth_reward_sequence_check_ddl;
EXECUTE growth_reward_sequence_check_stmt;
DEALLOCATE PREPARE growth_reward_sequence_check_stmt;
