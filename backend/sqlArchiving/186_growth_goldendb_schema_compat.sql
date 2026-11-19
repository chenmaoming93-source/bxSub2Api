-- Upgrade Growth tables created by the earlier 182 script.
-- Do not run this script on a database created from the latest 182 script.

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

ALTER TABLE growth_task_rules
  MODIFY COLUMN conditions LONGTEXT NULL;

ALTER TABLE growth_product_versions
  MODIFY COLUMN benefit_detail LONGTEXT NULL;

ALTER TABLE growth_exchange_orders
  CHANGE COLUMN status order_status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
  MODIFY COLUMN product_snapshot LONGTEXT NULL;

ALTER TABLE growth_fulfillment_jobs
  CHANGE COLUMN status job_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

ALTER TABLE growth_admin_audit
  CHANGE COLUMN action action_type VARCHAR(64) NOT NULL,
  MODIFY COLUMN before_json LONGTEXT NULL,
  MODIFY COLUMN after_json LONGTEXT NULL;
