-- 1) Normalize historical affiliate rebate rate values.
-- Legacy compatibility treated 0<x<=1 as fractional inputs (e.g. 0.2 => 20%).
-- We now use pure percentage semantics, so convert persisted fractional values once.
UPDATE settings
SET value = TRIM(TRAILING '.' FROM TRIM(TRAILING '0' FROM CAST((CAST(value AS DECIMAL(20,8)) * 100) AS CHAR))),
    updated_at = NOW()
WHERE `key` = 'affiliate_rebate_rate'
  AND REGEXP_LIKE(value, '^-?[0-9]+(\\.[0-9]+)?$')
  AND CAST(value AS DECIMAL(20,8)) > 0
  AND CAST(value AS DECIMAL(20,8)) <= 1;

-- 2) Affiliate ledger for accrual/transfer traceability.
CREATE TABLE IF NOT EXISTS user_affiliate_ledger (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    amount DECIMAL(20,8) NOT NULL,
    source_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE INDEX idx_user_affiliate_ledger_user_id ON user_affiliate_ledger(user_id);
CREATE INDEX idx_user_affiliate_ledger_action ON user_affiliate_ledger(action);


-- 3) Enforce idempotency at DB layer for payment audit actions.
DELETE p
FROM payment_audit_logs p
JOIN (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY order_id, action ORDER BY id) AS rn
    FROM payment_audit_logs
) AS ranked ON ranked.id = p.id
WHERE ranked.rn > 1;

CREATE UNIQUE INDEX idx_payment_audit_logs_order_action_uniq
ON payment_audit_logs(order_id, action);

-- 4) Prevent retroactive affiliate rebate issuance for legacy completed balance orders.
INSERT INTO payment_audit_logs (order_id, action, detail, operator, created_at)
SELECT po.id,
       'AFFILIATE_REBATE_SKIPPED',
       '{"reason":"baseline before affiliate rebate idempotency rollout"}',
       'system',
       NOW()
FROM payment_orders po
WHERE po.order_type = 'balance'
  AND po.status = 'COMPLETED'
  AND NOT EXISTS (
      SELECT 1
      FROM payment_audit_logs pal
      WHERE pal.order_id = po.id
        AND pal.action IN ('AFFILIATE_REBATE_APPLIED', 'AFFILIATE_REBATE_SKIPPED')
  );
