-- Restore native JSON columns for environments that previously ran the
-- long-text interoperability version of 186.
-- Existing values must be valid JSON objects.

ALTER TABLE growth_task_rules
  MODIFY COLUMN conditions JSON NULL;

ALTER TABLE growth_product_versions
  MODIFY COLUMN benefit_detail JSON NULL;

ALTER TABLE growth_exchange_orders
  MODIFY COLUMN product_snapshot JSON NULL;

ALTER TABLE growth_admin_audit
  MODIFY COLUMN before_json JSON NULL,
  MODIFY COLUMN after_json JSON NULL;
