-- Store the normalized account references derived from groups.model_routing.
-- This table is a query/configuration projection; the JSON remains the routing source of truth.
CREATE TABLE IF NOT EXISTS group_model_route_accounts (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    route_alias VARCHAR(255) NOT NULL,
    account_id BIGINT NOT NULL,
    max_concurrency INT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY uq_group_route_account (group_id, route_alias, account_id),
    KEY idx_group_model_route_accounts_account (account_id),
    KEY idx_group_model_route_accounts_group_route (group_id, route_alias),
    CONSTRAINT fk_group_model_route_accounts_group FOREIGN KEY (group_id) REFERENCES `groups` (id),
    CONSTRAINT fk_group_model_route_accounts_account FOREIGN KEY (account_id) REFERENCES accounts (id)
);
