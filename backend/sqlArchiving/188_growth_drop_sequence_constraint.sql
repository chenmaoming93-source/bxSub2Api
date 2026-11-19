-- Optional cleanup for environments that previously executed the old 185 script.
-- Run only when growth_reward_claims still contains this constraint.

ALTER TABLE growth_reward_claims
  DROP CONSTRAINT chk_growth_reward_claims_sequence;
