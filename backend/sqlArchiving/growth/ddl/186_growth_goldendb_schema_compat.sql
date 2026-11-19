-- Rename reserved-word columns in Growth tables created by the earlier 182 script.
-- JSON fields are restored to the native JSON type by 189.

ALTER TABLE growth_transfers
  CHANGE COLUMN status transfer_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

ALTER TABLE growth_tasks
  CHANGE COLUMN description remark VARCHAR(2048) NOT NULL DEFAULT '',
  CHANGE COLUMN status task_status VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

ALTER TABLE growth_user_tasks
  CHANGE COLUMN status progress_status VARCHAR(32) NOT NULL DEFAULT 'IN_PROGRESS';

ALTER TABLE growth_reward_claims
  CHANGE COLUMN status claim_status VARCHAR(20) NOT NULL DEFAULT 'SUCCEEDED';

ALTER TABLE growth_products
  CHANGE COLUMN status product_status VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

ALTER TABLE growth_exchange_orders
  CHANGE COLUMN status order_status VARCHAR(20) NOT NULL DEFAULT 'CREATED';

ALTER TABLE growth_fulfillment_jobs
  CHANGE COLUMN status job_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

ALTER TABLE growth_admin_audit
  CHANGE COLUMN action action_type VARCHAR(64) NOT NULL;
