-- MySQL 8 / GoldenDB (MySQL compatibility mode)
-- Growth plan accounting, catalog, exchange, fulfillment and audit tables.
-- This archive is manually executed and is not part of the runtime migration set.

CREATE TABLE IF NOT EXISTS growth_budgets (
    id BIGINT NOT NULL AUTO_INCREMENT,
    total_amount BIGINT NOT NULL DEFAULT 0,
    allocated_amount BIGINT NOT NULL DEFAULT 0,
    distributed_amount BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    CONSTRAINT chk_growth_budgets_nonnegative CHECK (
        total_amount >= 0
        AND allocated_amount >= 0
        AND distributed_amount >= 0
        AND allocated_amount <= total_amount
        AND distributed_amount <= allocated_amount
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_budget_entries (
    id BIGINT NOT NULL AUTO_INCREMENT,
    budget_id BIGINT NOT NULL DEFAULT 1,
    entry_type VARCHAR(16) NOT NULL,
    points BIGINT NOT NULL,
    budget_after BIGINT NOT NULL,
    allocated_after BIGINT NOT NULL,
    distributed_after BIGINT NOT NULL DEFAULT 0,
    task_id BIGINT NULL,
    operator_id BIGINT NOT NULL,
    reason VARCHAR(512) NOT NULL,
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_growth_budget_entries_task (task_id),
    KEY idx_growth_budget_entries_type_time (entry_type, created_at),
    KEY idx_growth_budget_entries_operator_time (operator_id, created_at),
    CONSTRAINT chk_growth_budget_entries_type CHECK (entry_type IN ('TOPUP', 'REDUCE', 'ALLOCATE', 'REVOKE')),
    CONSTRAINT chk_growth_budget_entries_amounts CHECK (
        points > 0
        AND budget_after >= 0
        AND allocated_after >= 0
        AND distributed_after >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_wallets (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    available_points BIGINT NOT NULL DEFAULT 0,
    earned_total BIGINT NOT NULL DEFAULT 0,
    spent_total BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    balance_updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_wallets_user (user_id),
    CONSTRAINT fk_growth_wallets_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT chk_growth_wallets_balances CHECK (
        available_points >= 0
        AND earned_total >= 0
        AND spent_total >= 0
        AND version > 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_transfers (
    id BIGINT NOT NULL AUTO_INCREMENT,
    biz_type VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    from_account_type VARCHAR(32) NOT NULL,
    to_account_type VARCHAR(32) NOT NULL,
    from_user_id BIGINT NULL,
    to_user_id BIGINT NULL,
    task_id BIGINT NULL,
    order_id BIGINT NULL,
    points BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    failure_reason VARCHAR(512) NOT NULL DEFAULT '',
    completed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_transfers_business (biz_type, idempotency_key),
    KEY idx_growth_transfers_from_user_time (from_user_id, created_at),
    KEY idx_growth_transfers_to_user_time (to_user_id, created_at),
    KEY idx_growth_transfers_task (task_id),
    KEY idx_growth_transfers_order (order_id),
    KEY idx_growth_transfers_status (status),
    CONSTRAINT chk_growth_transfers_accounts CHECK (
        from_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
        AND to_account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
        AND points > 0
    ),
    CONSTRAINT chk_growth_transfers_status CHECK (status IN ('PENDING', 'SUCCEEDED', 'FAILED', 'REVERSED'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_account_ledger (
    id BIGINT NOT NULL AUTO_INCREMENT,
    transfer_id BIGINT NOT NULL,
    user_id BIGINT NULL,
    account_type VARCHAR(32) NOT NULL,
    direction VARCHAR(8) NOT NULL,
    points BIGINT NOT NULL,
    balance_after BIGINT NULL,
    biz_type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    reference_id BIGINT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_growth_account_ledger_transfer (transfer_id),
    KEY idx_growth_account_ledger_user_time (user_id, created_at, id),
    KEY idx_growth_account_ledger_biz_time (biz_type, created_at),
    CONSTRAINT chk_growth_account_ledger_values CHECK (
        account_type IN ('TASK', 'WALLET', 'REDEEMED', 'SYSTEM')
        AND direction IN ('IN', 'OUT')
        AND points > 0
        AND (balance_after IS NULL OR balance_after >= 0)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_tasks (
    id BIGINT NOT NULL AUTO_INCREMENT,
    task_code VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description VARCHAR(2048) NOT NULL DEFAULT '',
    category VARCHAR(32) NOT NULL,
    task_type VARCHAR(32) NOT NULL,
    icon VARCHAR(128) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    effective_from DATETIME(6) NULL,
    effective_to DATETIME(6) NULL,
    allocated_points BIGINT NOT NULL DEFAULT 0,
    distributed_points BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_tasks_code (task_code),
    KEY idx_growth_tasks_status_sort (status, sort_order),
    KEY idx_growth_tasks_category_status (category, status),
    CONSTRAINT chk_growth_tasks_quota CHECK (
        allocated_points >= 0
        AND distributed_points >= 0
        AND distributed_points <= allocated_points
    ),
    CONSTRAINT chk_growth_tasks_status CHECK (status IN ('DRAFT', 'ACTIVE', 'OFFLINE')),
    CONSTRAINT chk_growth_tasks_category CHECK (category IN ('CHECKIN', 'DAILY', 'ACHIEVEMENT')),
    CONSTRAINT chk_growth_tasks_type CHECK (task_type IN ('CHECKIN', 'DAILY', 'ACHIEVEMENT'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_task_rules (
    id BIGINT NOT NULL AUTO_INCREMENT,
    task_id BIGINT NOT NULL,
    version INT NOT NULL,
    trigger_event VARCHAR(64) NOT NULL DEFAULT 'COMPLETION',
    conditions JSON NULL,
    target BIGINT NOT NULL DEFAULT 1,
    reward_points BIGINT NOT NULL,
    period_type VARCHAR(20) NOT NULL,
    daily_limit INT NOT NULL DEFAULT 0,
    per_user_limit INT NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL,
    effective_from DATETIME(6) NOT NULL,
    effective_to DATETIME(6) NULL,
    published_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_task_rules_version (task_id, version),
    KEY idx_growth_task_rules_effective (task_id, effective_from, effective_to),
    KEY idx_growth_task_rules_published (published_at),
    CONSTRAINT chk_growth_task_rules_values CHECK (
        version > 0
        AND target > 0
        AND reward_points > 0
        AND daily_limit >= 0
        AND per_user_limit >= 0
        AND period_type IN ('DAILY', 'ONCE', 'WEEKLY')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_user_tasks (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL,
    rule_id BIGINT NOT NULL,
    cycle_key VARCHAR(64) NOT NULL,
    progress BIGINT NOT NULL DEFAULT 0,
    target BIGINT NOT NULL DEFAULT 1,
    completed_count BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'IN_PROGRESS',
    completed_at DATETIME(6) NULL,
    rewarded_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_user_tasks_cycle (user_id, task_id, rule_id, cycle_key),
    KEY idx_growth_user_tasks_user_time (user_id, created_at),
    KEY idx_growth_user_tasks_task_cycle (task_id, cycle_key),
    CONSTRAINT fk_growth_user_tasks_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT chk_growth_user_tasks_values CHECK (
        progress >= 0
        AND target > 0
        AND completed_count >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_completion_requests (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL,
    completion_ref VARCHAR(128) NOT NULL,
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    result VARCHAR(32) NOT NULL,
    transfer_id BIGINT NULL,
    awarded_points BIGINT NOT NULL DEFAULT 0,
    rule_version INT NOT NULL,
    cycle_key VARCHAR(64) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_completion_business (user_id, task_id, completion_ref),
    KEY idx_growth_completion_user_time (user_id, created_at),
    KEY idx_growth_completion_task_cycle (task_id, cycle_key),
    KEY idx_growth_completion_transfer (transfer_id),
    CONSTRAINT fk_growth_completion_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT chk_growth_completion_result CHECK (
        result IN ('AWARDED', 'ALREADY_COMPLETED', 'NOT_ELIGIBLE')
        AND awarded_points >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_reward_claims (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL,
    rule_id BIGINT NOT NULL,
    cycle_key VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    transfer_id BIGINT NOT NULL,
    points BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'SUCCEEDED',
    claimed_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_reward_claims_key (user_id, idempotency_key),
    UNIQUE KEY uq_growth_reward_claims_cycle (user_id, task_id, cycle_key),
    KEY idx_growth_reward_claims_transfer (transfer_id),
    KEY idx_growth_reward_claims_user_time (user_id, claimed_at),
    CONSTRAINT fk_growth_reward_claims_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT chk_growth_reward_claims_points CHECK (points > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_products (
    id BIGINT NOT NULL AUTO_INCREMENT,
    product_code VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    product_type VARCHAR(32) NOT NULL,
    icon VARCHAR(128) NOT NULL DEFAULT '',
    fulfillment_mode VARCHAR(32) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    current_version INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_products_code (product_code),
    KEY idx_growth_products_status_sort (status, sort_order),
    KEY idx_growth_products_type_status (product_type, status),
    CONSTRAINT chk_growth_products_values CHECK (
        product_type IN ('TOKEN_QUOTA', 'SUBSCRIPTION', 'PHYSICAL')
        AND fulfillment_mode IN ('AUTOMATIC', 'MANUAL')
        AND status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'OFFLINE')
        AND current_version >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_product_versions (
    id BIGINT NOT NULL AUTO_INCREMENT,
    product_id BIGINT NOT NULL,
    version INT NOT NULL,
    points_cost BIGINT NOT NULL,
    benefit_summary VARCHAR(512) NOT NULL DEFAULT '',
    benefit_detail JSON NULL,
    token_amount BIGINT NOT NULL DEFAULT 0,
    per_user_limit INT NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL,
    published_at DATETIME(6) NULL,
    effective_from DATETIME(6) NULL,
    effective_to DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_product_versions_version (product_id, version),
    KEY idx_growth_product_versions_published (product_id, published_at),
    CONSTRAINT chk_growth_product_versions_values CHECK (
        version > 0
        AND points_cost > 0
        AND token_amount >= 0
        AND per_user_limit >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_product_inventory (
    id BIGINT NOT NULL AUTO_INCREMENT,
    product_id BIGINT NOT NULL,
    stock_mode VARCHAR(20) NOT NULL DEFAULT 'LIMITED',
    available BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0,
    sold BIGINT NOT NULL DEFAULT 0,
    lock_version BIGINT NOT NULL DEFAULT 1,
    token_quota_total BIGINT NOT NULL DEFAULT 0,
    token_quota_distributed BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_product_inventory_product (product_id),
    CONSTRAINT chk_growth_product_inventory_values CHECK (
        stock_mode IN ('LIMITED', 'UNLIMITED')
        AND available >= 0
        AND reserved >= 0
        AND sold >= 0
        AND lock_version > 0
        AND token_quota_total >= 0
        AND token_quota_distributed >= 0
        AND token_quota_distributed <= token_quota_total
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_exchange_orders (
    id BIGINT NOT NULL AUTO_INCREMENT,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    product_version INT NOT NULL,
    product_snapshot JSON NOT NULL,
    points_cost BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
    idempotency_key VARCHAR(128) NOT NULL,
    expected_version INT NULL,
    transfer_id BIGINT NULL,
    refund_transfer_id BIGINT NULL,
    failure_reason VARCHAR(1024) NOT NULL DEFAULT '',
    fulfilled_at DATETIME(6) NULL,
    refunded_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_exchange_orders_no (order_no),
    UNIQUE KEY uq_growth_exchange_orders_key (user_id, idempotency_key),
    KEY idx_growth_exchange_orders_user_time (user_id, created_at),
    KEY idx_growth_exchange_orders_product_status (product_id, status),
    KEY idx_growth_exchange_orders_status_time (status, created_at),
    CONSTRAINT fk_growth_exchange_orders_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT chk_growth_exchange_orders_values CHECK (
        product_version > 0
        AND points_cost > 0
        AND status IN ('CREATED', 'FULFILLING', 'SUCCEEDED', 'FAILED', 'REFUNDED')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_fulfillment_jobs (
    id BIGINT NOT NULL AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    job_type VARCHAR(32) NOT NULL DEFAULT 'FULFILL',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_retry_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    locked_until DATETIME(6) NULL,
    last_error VARCHAR(1024) NOT NULL DEFAULT '',
    provider_reference VARCHAR(255) NOT NULL DEFAULT '',
    completed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_fulfillment_jobs_order (order_id),
    KEY idx_growth_fulfillment_jobs_due (status, next_retry_at),
    KEY idx_growth_fulfillment_jobs_lock (locked_until),
    CONSTRAINT chk_growth_fulfillment_jobs_values CHECK (
        status IN ('PENDING', 'PROCESSING', 'RETRY', 'SUCCEEDED', 'FAILED', 'CANCELLED')
        AND attempt_count >= 0
        AND max_attempts > 0
        AND attempt_count <= max_attempts
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_inventory_ledger (
    id BIGINT NOT NULL AUTO_INCREMENT,
    business_key VARCHAR(128) NOT NULL,
    product_id BIGINT NOT NULL,
    inventory_id BIGINT NOT NULL,
    operation_type VARCHAR(32) NOT NULL,
    delta_available BIGINT NOT NULL DEFAULT 0,
    delta_reserved BIGINT NOT NULL DEFAULT 0,
    delta_sold BIGINT NOT NULL DEFAULT 0,
    available_after BIGINT NOT NULL,
    reserved_after BIGINT NOT NULL,
    sold_after BIGINT NOT NULL,
    order_id BIGINT NULL,
    operator_id BIGINT NULL,
    reason VARCHAR(512) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_growth_inventory_ledger_business (business_key),
    KEY idx_growth_inventory_ledger_product_time (product_id, created_at),
    KEY idx_growth_inventory_ledger_order (order_id),
    CONSTRAINT chk_growth_inventory_ledger_values CHECK (
        operation_type IN ('STOCK_IN', 'RESERVE', 'SELL', 'RELEASE', 'ADJUST')
        AND available_after >= 0
        AND reserved_after >= 0
        AND sold_after >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS growth_admin_audit (
    id BIGINT NOT NULL AUTO_INCREMENT,
    operator_id BIGINT NOT NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id BIGINT NULL,
    before_json JSON NULL,
    after_json JSON NULL,
    reason VARCHAR(512) NOT NULL DEFAULT '',
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_growth_admin_audit_operator_time (operator_id, created_at),
    KEY idx_growth_admin_audit_resource_time (resource_type, resource_id, created_at),
    KEY idx_growth_admin_audit_action_time (action, created_at),
    CONSTRAINT fk_growth_admin_audit_operator FOREIGN KEY (operator_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Verification
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN (
    'growth_budgets',
    'growth_budget_entries',
    'growth_wallets',
    'growth_transfers',
    'growth_account_ledger',
    'growth_tasks',
    'growth_task_rules',
    'growth_user_tasks',
    'growth_completion_requests',
    'growth_reward_claims',
    'growth_products',
    'growth_product_versions',
    'growth_product_inventory',
    'growth_exchange_orders',
    'growth_fulfillment_jobs',
    'growth_inventory_ledger',
    'growth_admin_audit'
  )
ORDER BY table_name;
