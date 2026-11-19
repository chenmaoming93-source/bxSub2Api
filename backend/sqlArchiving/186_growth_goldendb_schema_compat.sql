-- Upgrade Growth tables created by the earlier 182 script.
-- Do not run this script on a database created from the latest 182 script.

ALTER TABLE growth_transfers
  DROP CHECK chk_growth_transfers_status;

ALTER TABLE growth_transfers
  CHANGE COLUMN status transfer_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

ALTER TABLE growth_transfers
  ADD CONSTRAINT chk_growth_transfers_status
    CHECK (transfer_status IN ('PENDING', 'SUCCEEDED', 'FAILED', 'REVERSED'));

ALTER TABLE growth_tasks
  DROP CHECK chk_growth_tasks_status;

ALTER TABLE growth_tasks
  CHANGE COLUMN description remark VARCHAR(2048) NOT NULL DEFAULT '',
  CHANGE COLUMN status task_status VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

ALTER TABLE growth_tasks
  ADD CONSTRAINT chk_growth_tasks_status
    CHECK (task_status IN ('DRAFT', 'ACTIVE', 'OFFLINE'));

ALTER TABLE growth_user_tasks
  CHANGE COLUMN status progress_status VARCHAR(32) NOT NULL DEFAULT 'IN_PROGRESS';

ALTER TABLE growth_reward_claims
  CHANGE COLUMN status claim_status VARCHAR(20) NOT NULL DEFAULT 'SUCCEEDED';

ALTER TABLE growth_products
  DROP CHECK chk_growth_products_values;

ALTER TABLE growth_products
  CHANGE COLUMN status product_status VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

ALTER TABLE growth_products
  ADD CONSTRAINT chk_growth_products_values CHECK (
    product_type IN ('TOKEN_QUOTA', 'SUBSCRIPTION', 'PHYSICAL')
    AND fulfillment_mode IN ('AUTOMATIC', 'MANUAL')
    AND product_status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'OFFLINE')
    AND current_version >= 0
  );

ALTER TABLE growth_task_rules
  MODIFY COLUMN conditions LONGTEXT NULL;

ALTER TABLE growth_product_versions
  MODIFY COLUMN benefit_detail LONGTEXT NULL;

ALTER TABLE growth_exchange_orders
  DROP CHECK chk_growth_exchange_orders_values;

ALTER TABLE growth_exchange_orders
  CHANGE COLUMN status order_status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
  MODIFY COLUMN product_snapshot LONGTEXT NULL;

ALTER TABLE growth_exchange_orders
  ADD CONSTRAINT chk_growth_exchange_orders_values CHECK (
    product_version > 0
    AND points_cost > 0
    AND order_status IN ('CREATED', 'FULFILLING', 'SUCCEEDED', 'FAILED', 'REFUNDED')
  );

ALTER TABLE growth_fulfillment_jobs
  DROP CHECK chk_growth_fulfillment_jobs_values;

ALTER TABLE growth_fulfillment_jobs
  CHANGE COLUMN status job_status VARCHAR(20) NOT NULL DEFAULT 'PENDING';

ALTER TABLE growth_fulfillment_jobs
  ADD CONSTRAINT chk_growth_fulfillment_jobs_values CHECK (
    job_status IN ('PENDING', 'PROCESSING', 'RETRY', 'SUCCEEDED', 'FAILED', 'CANCELLED')
    AND attempt_count >= 0
    AND max_attempts > 0
    AND attempt_count <= max_attempts
  );

ALTER TABLE growth_admin_audit
  CHANGE COLUMN action action_type VARCHAR(64) NOT NULL,
  MODIFY COLUMN before_json LONGTEXT NULL,
  MODIFY COLUMN after_json LONGTEXT NULL;
