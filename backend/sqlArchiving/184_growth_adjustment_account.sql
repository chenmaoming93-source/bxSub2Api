-- Separate manual point deductions from redemption spending.
-- Run once on environments that already applied 182_create_growth_tables.sql.

ALTER TABLE growth_budgets
  DROP CHECK chk_growth_budgets_nonnegative,
  ADD CONSTRAINT chk_growth_budgets_nonnegative CHECK (
    total_amount >= 0
    AND allocated_amount >= 0
    AND distributed_amount >= 0
    AND allocated_amount <= total_amount
    AND distributed_amount <= total_amount
  );

ALTER TABLE growth_transfers
  DROP CHECK chk_growth_transfers_accounts,
  ADD CONSTRAINT chk_growth_transfers_accounts CHECK (
    from_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM', 'ADJUST')
    AND to_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM', 'ADJUST')
    AND points > 0
  );

ALTER TABLE growth_account_ledger
  DROP CHECK chk_growth_account_ledger_values,
  ADD CONSTRAINT chk_growth_account_ledger_values CHECK (
    account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM', 'ADJUST')
    AND direction IN ('IN', 'OUT')
    AND points > 0
    AND (balance_after IS NULL OR balance_after >= 0)
  );
