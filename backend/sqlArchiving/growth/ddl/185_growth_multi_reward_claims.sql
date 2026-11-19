-- Upgrade an existing single-reward Growth schema to support multiple rewards.

ALTER TABLE growth_reward_claims
  ADD COLUMN sequence INT NOT NULL DEFAULT 1 AFTER cycle_key;

ALTER TABLE growth_reward_claims
  DROP INDEX uq_growth_reward_claims_cycle;

ALTER TABLE growth_reward_claims
  ADD UNIQUE KEY uq_growth_reward_claims_cycle
    (user_id, task_id, cycle_key, sequence);
